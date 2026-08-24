package controlplane

import (
	"context"
	"fmt"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/flags"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/settings"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CP-3 from the console's side: a setting changes and can be put back, a flag
// can be aimed at one organisation, and — the fourth of the access-mode tests
// the plan asks for — an invitation still creates an account while the
// platform is closed to strangers.

func configService(t *testing.T, pool *pgxpool.Pool) *Service {
	t.Helper()
	store := settings.NewStore(pool)
	settings.UseStore(store)
	t.Cleanup(func() { settings.UseStore(nil) })
	if err := store.Load(context.Background()); err != nil {
		t.Fatalf("load the settings: %v", err)
	}

	flagStore := flags.NewStore(pool)
	if err := flagStore.Load(context.Background()); err != nil {
		t.Fatalf("load the flags: %v", err)
	}

	return &Service{db: pool, sessions: NewSessionStore(pool), settings: store, flags: flagStore}
}

// The line the access mode does not cross: somebody an operator invited is
// already registered, so creating them is not just-in-time provisioning and is
// not what private mode refuses.
func TestAnInvitationStillWorksWhileThePlatformIsPrivate(t *testing.T) {
	pool := openPool(t)
	service := configService(t, pool)
	operator, _ := newOperator(t, pool, RoleOperator)
	sess := sessionFor(operator)
	ctx := context.Background()

	if err := service.SetSetting(ctx, sess, settings.AccessMode, settings.AccessPrivate,
		"closing the platform"); err != nil {
		t.Fatalf("set the access mode: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM platform_settings WHERE key = $1`, settings.AccessMode)
	})
	if got := settings.Get(settings.AccessMode); got != settings.AccessPrivate {
		t.Fatalf("the platform is %q", got)
	}

	slug := fmt.Sprintf("invited-%d", time.Now().UnixNano())
	email := slug + "@example.mn"
	created, err := service.CreateTenant(ctx, sess, NewTenant{
		Name:       "Invited Organisation",
		Slug:       slug,
		AdminEmail: email,
		Reason:     "a customer signed a contract",
	})
	if err != nil {
		t.Fatalf("a private platform refused an invited organisation: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1::uuid`, created.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	// The account exists and is an administrator of the new organisation.
	var admin bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (
		    SELECT 1 FROM users u
		      JOIN memberships m ON m.user_id = u.id AND m.tenant_id = $2::uuid
		      JOIN membership_roles mr ON mr.membership_id = m.id
		      JOIN roles r ON r.id = mr.role_id AND r.code = 'admin'
		     WHERE u.email = $1)`, email, created.ID).Scan(&admin); err != nil {
		t.Fatalf("look for the administrator: %v", err)
	}
	if !admin {
		t.Fatal("the invited administrator was not created")
	}
	// Nothing was sent, because no mail is configured in a test — and the
	// console says so rather than pretending.
	if created.Invited {
		t.Fatal("an invitation was reported as sent with no mail configured")
	}
}

// A setting changes, is recorded, and can be put back — and the rollback is
// itself a change rather than an erasure.
func TestASettingChangesAndRollsBack(t *testing.T) {
	pool := openPool(t)
	service := configService(t, pool)
	operator, _ := newOperator(t, pool, RoleOperator)
	sess := sessionFor(operator)
	ctx := context.Background()

	// The history is append-only in spirit and cumulative in fact, so a
	// previous run's rows are cleared before counting this one's.
	clear := func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM platform_settings WHERE key = $1`, settings.SessionIdleTimeout)
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM platform_settings_history WHERE key = $1`, settings.SessionIdleTimeout)
	}
	clear()
	t.Cleanup(clear)

	if err := service.SetSetting(ctx, sess, settings.SessionIdleTimeout, "20m",
		"a customer asked for tighter sessions"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := settings.Duration(settings.SessionIdleTimeout); got != 20*time.Minute {
		t.Fatalf("the platform reads %v", got)
	}

	// A value the registry refuses never reaches the table.
	if err := service.SetSetting(ctx, sess, settings.SessionIdleTimeout, "soon", "typo"); err == nil {
		t.Fatal("a nonsense duration was accepted")
	}
	if got := settings.Duration(settings.SessionIdleTimeout); got != 20*time.Minute {
		t.Fatalf("a refused value changed the setting to %v", got)
	}

	changes, err := service.SettingHistory(ctx, settings.SessionIdleTimeout)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("the history has %d rows, want 1", len(changes))
	}

	if err := service.RollbackSetting(ctx, sess, changes[0].ID, "it was worse"); err != nil {
		t.Fatalf("roll back: %v", err)
	}
	// Back to the default, because that change was the first: there was no
	// previous value to return to.
	if got := settings.Duration(settings.SessionIdleTimeout); got != 90*time.Minute {
		t.Fatalf("after the rollback the platform reads %v", got)
	}
	// And the rollback is in the history rather than having removed anything.
	changes, err = service.SettingHistory(ctx, settings.SessionIdleTimeout)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("the history has %d rows after a rollback, want 2", len(changes))
	}
}

// A kill switch, aimed at one organisation and then at everybody.
func TestAFlagCanBeAimedAtOneOrganisation(t *testing.T) {
	pool := openPool(t)
	service := configService(t, pool)
	operator, _ := newOperator(t, pool, RoleOperator)
	sess := sessionFor(operator)
	tenantID, _ := newTenant(t, pool)
	ctx := context.Background()

	key := fmt.Sprintf("module.test-%d.disabled", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM feature_flags WHERE key = $1`, key)
	})

	if err := service.SaveFlag(ctx, sess, FlagInput{
		Key: key, Kind: flags.KindKillSwitch, Owner: "platform",
		Enabled: false, Rollout: 100, Reason: "prepared during an incident",
	}); err != nil {
		t.Fatalf("save the flag: %v", err)
	}

	on := true
	if err := service.SetFlagOverride(ctx, sess, key, tenantID, &on,
		"this organisation is the one seeing the fault"); err != nil {
		t.Fatalf("set the override: %v", err)
	}

	// Read through the same store the platform reads: the override is on for
	// this organisation and the flag is still off for everybody else.
	flags.UseStore(service.flags)
	t.Cleanup(func() { flags.UseStore(nil) })
	if !flags.Enabled(tenantContext(tenantID), key) {
		t.Fatal("the override did not reach the evaluation")
	}
	if flags.Enabled(tenantContext("00000000-0000-0000-0000-000000000000"), key) {
		t.Fatal("a flag switched on for one organisation was on for another")
	}

	// Removing the override puts them back with everybody else.
	if err := service.SetFlagOverride(ctx, sess, key, tenantID, nil, "the fault is fixed"); err != nil {
		t.Fatalf("remove the override: %v", err)
	}
	if flags.Enabled(tenantContext(tenantID), key) {
		t.Fatal("the override survived being removed")
	}

	if err := service.DeleteFlag(ctx, sess, key, "no longer needed"); err != nil {
		t.Fatalf("delete the flag: %v", err)
	}
	list, err := service.ListFlags(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, flag := range list {
		if flag.Key == key {
			t.Fatal("the flag survived being deleted")
		}
	}
}

// tenantContext is what the request middleware would have built.
func tenantContext(tenantID string) context.Context {
	return nexus.WithTenantID(context.Background(), tenantID)
}
