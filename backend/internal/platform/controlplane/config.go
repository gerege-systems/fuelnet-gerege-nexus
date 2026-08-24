package controlplane

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/operator"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/flags"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/settings"
	"github.com/jackc/pgx/v5"
)

// The console's half of CP-3: changing how the platform behaves, without a
// deployment.
//
// Everything here writes through Do, so every change to a setting, a flag, a
// maintenance window or an announcement is one transaction with an
// operator_audit row and a reason. That is what makes a dynamic configuration
// safe to have at all: the argument against one has always been that it makes
// the running system differ from the repository in ways nobody can reconstruct
// — and the answer is not to avoid it but to record it.

var (
	// ErrNoSettingsStore is a deployment whose console was built without one.
	// It cannot happen in the server, and it is checked so a test that builds a
	// bare Service gets a sentence rather than a nil dereference.
	ErrNoSettingsStore = errors.New("this console has no settings store")
	// ErrNoFlagStore is the same for flags.
	ErrNoFlagStore = errors.New("this console has no feature flag store")
	// ErrHistoryNotFound is a rollback naming a change that is not there.
	ErrHistoryNotFound = errors.New("no such change")
)

// ListSettings returns every registered setting with its current value.
func (s *Service) ListSettings(ctx context.Context) ([]settings.Value, error) {
	if s.settings == nil {
		return nil, ErrNoSettingsStore
	}
	return s.settings.List(operator.Scoped(ctx))
}

// SetSetting writes a value.
//
// The audit row carries both values, so the trail answers "what was it before"
// without anybody having to open the history — the two questions arrive
// together in an incident.
func (s *Service) SetSetting(ctx context.Context, sess operator.Session, key, value, reason string) error {
	if s.settings == nil {
		return ErrNoSettingsStore
	}
	spec, known := settings.Lookup(key)
	if !known {
		return fmt.Errorf("%w: %s", settings.ErrUnknownSetting, key)
	}

	before := settings.Get(key)
	err := s.op.Do(ctx, sess, operator.Change{
		Action:     "settings.set",
		TargetType: "setting",
		TargetID:   key,
		Reason:     reason,
		Before:     map[string]any{"value": before},
		After:      map[string]any{"value": value, "kind": string(spec.Kind)},
	}, func(ctx context.Context, tx pgx.Tx) error {
		return s.settings.Set(ctx, tx, key, value, sess.ID, reason)
	})
	if err != nil {
		return err
	}
	// After the commit: the caches — this replica's and every other one's —
	// must not be told about a value that then failed to land.
	s.settings.Changed(ctx)
	return nil
}

// SettingHistory returns what a setting has been. An empty key returns every
// setting's changes, which is the screen an operator wants after an incident:
// "what did we change this afternoon".
func (s *Service) SettingHistory(ctx context.Context, key string) ([]settings.Change, error) {
	if s.settings == nil {
		return nil, ErrNoSettingsStore
	}
	return s.settings.History(operator.Scoped(ctx), key)
}

// RollbackSetting puts a setting back to what a named change moved it from.
//
// A rollback is itself a change: it writes a new history row rather than
// removing the one it undoes. A history that could be rewound would be a
// history somebody could edit, and the value of this table is that it cannot.
func (s *Service) RollbackSetting(ctx context.Context, sess operator.Session, changeID, reason string) error {
	if s.settings == nil {
		return ErrNoSettingsStore
	}

	var key string
	var previous *string
	err := s.db.QueryRow(operator.Scoped(ctx),
		`SELECT key, previous_value FROM platform_settings_history WHERE id = $1::uuid`,
		changeID).Scan(&key, &previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrHistoryNotFound
	}
	if err != nil {
		if operator.IsInvalidUUID(err) {
			return ErrHistoryNotFound
		}
		return fmt.Errorf("control plane: read the change: %w", err)
	}

	// A change whose previous value was NULL is the first time the setting was
	// written at all, so undoing it means going back to the environment or the
	// default — which is what the spec's own default is.
	target := ""
	if previous != nil {
		target = *previous
	} else if spec, known := settings.Lookup(key); known {
		target = spec.Default
	}

	return s.SetSetting(ctx, sess, key, target,
		reason+" (буцаалт: "+changeID+")")
}

// ListFlags returns every feature flag.
func (s *Service) ListFlags(ctx context.Context) ([]flags.Flag, error) {
	if s.flags == nil {
		return nil, ErrNoFlagStore
	}
	return s.flags.List(operator.Scoped(ctx))
}

// FlagInput is a flag as the console writes it.
type FlagInput struct {
	Key         string     `json:"key"`
	Description string     `json:"description"`
	Owner       string     `json:"owner"`
	Kind        string     `json:"kind"`
	Enabled     bool       `json:"enabled"`
	Rollout     int        `json:"rollout"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Reason      string     `json:"reason"`
}

// SaveFlag creates or updates a flag.
func (s *Service) SaveFlag(ctx context.Context, sess operator.Session, input FlagInput) error {
	if s.flags == nil {
		return ErrNoFlagStore
	}
	if input.Key == "" {
		return errors.New("a flag needs a key")
	}
	switch input.Kind {
	case flags.KindRelease, flags.KindKillSwitch, flags.KindExperiment:
	case "":
		input.Kind = flags.KindRelease
	default:
		return fmt.Errorf("%q is not a kind of flag", input.Kind)
	}
	if input.Rollout < 0 || input.Rollout > 100 {
		return errors.New("a rollout is 0 to 100")
	}

	err := s.op.Do(ctx, sess, operator.Change{
		Action:     "flag.save",
		TargetType: "flag",
		TargetID:   input.Key,
		Reason:     input.Reason,
		After: map[string]any{
			"enabled": input.Enabled, "rollout": input.Rollout, "kind": input.Kind,
		},
	}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO feature_flags (key, description, owner, kind, enabled, rollout, expires_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			 ON CONFLICT (key) DO UPDATE
			    SET description = EXCLUDED.description, owner = EXCLUDED.owner,
			        kind = EXCLUDED.kind, enabled = EXCLUDED.enabled,
			        rollout = EXCLUDED.rollout, expires_at = EXCLUDED.expires_at,
			        updated_at = NOW()`,
			input.Key, input.Description, input.Owner, input.Kind,
			input.Enabled, input.Rollout, input.ExpiresAt)
		return err
	})
	if err != nil {
		return err
	}
	s.flags.Changed(ctx)
	return nil
}

// DeleteFlag removes a flag.
//
// The console can do this, unlike almost everything else, because a flag that
// cannot be removed is flag debt by construction: the expiry warning would
// name flags nobody could act on.
func (s *Service) DeleteFlag(ctx context.Context, sess operator.Session, key, reason string) error {
	if s.flags == nil {
		return ErrNoFlagStore
	}
	err := s.op.Do(ctx, sess, operator.Change{
		Action:     "flag.delete",
		TargetType: "flag",
		TargetID:   key,
		Reason:     reason,
	}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM feature_flags WHERE key = $1`, key)
		return err
	})
	if err != nil {
		return err
	}
	s.flags.Changed(ctx)
	return nil
}

// SetFlagOverride decides a flag for one organisation, or removes the decision.
func (s *Service) SetFlagOverride(ctx context.Context, sess operator.Session, key, tenantID string, enabled *bool, reason string) error {
	if s.flags == nil {
		return ErrNoFlagStore
	}
	err := s.op.Do(ctx, sess, operator.Change{
		Action:     "flag.override",
		TargetType: "flag",
		TargetID:   key,
		Reason:     reason,
		After:      map[string]any{"tenant_id": tenantID, "enabled": enabled},
	}, func(ctx context.Context, tx pgx.Tx) error {
		if enabled == nil {
			_, err := tx.Exec(ctx,
				`DELETE FROM feature_flag_overrides WHERE flag_key = $1 AND tenant_id = $2::uuid`,
				key, tenantID)
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO feature_flag_overrides (flag_key, tenant_id, enabled, updated_at)
			 VALUES ($1, $2::uuid, $3, NOW())
			 ON CONFLICT (flag_key, tenant_id) DO UPDATE
			    SET enabled = EXCLUDED.enabled, updated_at = NOW()`,
			key, tenantID, *enabled)
		return err
	})
	if err != nil {
		return err
	}
	s.flags.Changed(ctx)
	return nil
}

// SetTenantMaintenance opens or closes one organisation for writing.
func (s *Service) SetTenantMaintenance(ctx context.Context, sess operator.Session, tenantID string, on bool, message, reason string) error {
	before, err := s.op.StateOf(ctx, tenantID)
	if err != nil {
		return err
	}
	defer s.changed(tenantID)
	return s.op.Do(ctx, sess, operator.Change{
		Action:     "tenant.maintenance",
		TargetType: "tenant",
		TargetID:   tenantID,
		Reason:     reason,
		Before:     before,
		After:      map[string]any{"maintenance": on, "message": message},
	}, func(ctx context.Context, tx pgx.Tx) error {
		if !on {
			_, err := tx.Exec(ctx,
				`UPDATE tenants SET maintenance_at = NULL, maintenance_message = '' WHERE id = $1::uuid`,
				tenantID)
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE tenants SET maintenance_at = NOW(), maintenance_message = $2 WHERE id = $1::uuid`,
			tenantID, message)
		return err
	})
}

// Announcement is one thing to tell people.
type Announcement struct {
	ID        string     `json:"id"`
	TenantID  *string    `json:"tenant_id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListAnnouncements returns them, newest first.
func (s *Service) ListAnnouncements(ctx context.Context) ([]Announcement, error) {
	rows, err := s.db.Query(operator.Scoped(ctx),
		`SELECT id::text, tenant_id::text, kind, title, body, starts_at, ends_at, created_at
		   FROM announcements ORDER BY starts_at DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("control plane: list the announcements: %w", err)
	}
	defer rows.Close()

	announcements := make([]Announcement, 0, 8)
	for rows.Next() {
		var announcement Announcement
		if err := rows.Scan(&announcement.ID, &announcement.TenantID, &announcement.Kind,
			&announcement.Title, &announcement.Body, &announcement.StartsAt,
			&announcement.EndsAt, &announcement.CreatedAt); err != nil {
			return nil, fmt.Errorf("control plane: read an announcement: %w", err)
		}
		announcements = append(announcements, announcement)
	}
	return announcements, rows.Err()
}

// Announce broadcasts something, to everybody or to one organisation.
func (s *Service) Announce(ctx context.Context, sess operator.Session, announcement Announcement, reason string) error {
	if announcement.Title == "" {
		return errors.New("an announcement needs something to say")
	}
	switch announcement.Kind {
	case "info", "warning", "maintenance":
	case "":
		announcement.Kind = "info"
	default:
		return fmt.Errorf("%q is not a kind of announcement", announcement.Kind)
	}

	return s.op.Do(ctx, sess, operator.Change{
		Action:     "announcement.create",
		TargetType: "announcement",
		TargetID:   valueOr(announcement.TenantID, "all"),
		Reason:     reason,
		After:      map[string]any{"title": announcement.Title, "kind": announcement.Kind},
	}, func(ctx context.Context, tx pgx.Tx) error {
		starts := announcement.StartsAt
		if starts.IsZero() {
			starts = time.Now()
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO announcements (tenant_id, kind, title, body, starts_at, ends_at, created_by)
			 VALUES (NULLIF($1, '')::uuid, $2, $3, $4, $5, $6, $7::uuid)`,
			valueOr(announcement.TenantID, ""), announcement.Kind, announcement.Title,
			announcement.Body, starts, announcement.EndsAt, sess.ID)
		return err
	})
}

// WithdrawAnnouncement removes one.
func (s *Service) WithdrawAnnouncement(ctx context.Context, sess operator.Session, id, reason string) error {
	return s.op.Do(ctx, sess, operator.Change{
		Action:     "announcement.withdraw",
		TargetType: "announcement",
		TargetID:   id,
		Reason:     reason,
	}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM announcements WHERE id = $1::uuid`, id)
		return err
	})
}

// valueOr dereferences a pointer or gives a fallback, which the two callers
// above would otherwise each write for themselves.
func valueOr(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}
