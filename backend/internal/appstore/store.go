/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package appstore is the registry that serves appstore.gerege.mn: the
 * published catalogue every Nexus instance pulls, and the publishing side that
 * fills it.
 *
 * It shares the platform's appcatalog types deliberately. The wire contract
 * between this service and a Nexus instance is defined by the client in
 * internal/platform/appcatalog/source.go, and the cheapest way to keep two
 * programs agreeing about a format is to have them use the same structs and the
 * same validator rather than two descriptions of one thing.
 */

package appstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Version statuses. A version is submitted for review, then published; a
// published version can be yanked, which stops it being offered without
// pretending it never existed.
const (
	StatusDraft     = "draft"
	StatusInReview  = "in_review"
	StatusPublished = "published"
	StatusRejected  = "rejected"
	StatusYanked    = "yanked"
)

// ErrNotFound is returned when an app, publisher or version does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when something already exists that may not be
// replaced — a version number that has been used, a slug somebody else holds.
var ErrConflict = errors.New("already exists")

// Store is the registry's persistence.
type Store struct{ db *pgxpool.Pool }

func NewStore(db *pgxpool.Pool) *Store { return &Store{db: db} }

// DB exposes the pool for tests that assert on rows this package writes but no
// caller reads — the snapshot cache in particular, whose whole contract is that
// it stays small.
func (s *Store) DB() *pgxpool.Pool { return s.db }

// Publisher is an organisation that publishes apps.
type Publisher struct {
	ID              string    `json:"id"`
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	ContactEmail    string    `json:"contact_email"`
	Verified        bool      `json:"verified"`
	OwnerSub        string    `json:"-"`
	OwnerEmail      string    `json:"owner_email"`
	OwnerTenantID   string    `json:"-"`
	OwnerTenantSlug string    `json:"owner_tenant_slug"`
	CreatedAt       time.Time `json:"created_at"`
}

// App is a catalogue entry, without its versions.
type App struct {
	ID          string    `json:"id"`
	PublisherID string    `json:"publisher_id"`
	Slug        string    `json:"slug"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IconURL     string    `json:"icon_url"`
	Category    string    `json:"category"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// PublisherName is filled by the queries the storefront uses: a catalogue
	// entry without a publisher is an app nobody stands behind.
	PublisherName string `json:"publisher_name,omitempty"`
	// LatestVersion is the published version on the stable channel, when there
	// is one. Empty means the app exists but has nothing published.
	LatestVersion string `json:"latest_version,omitempty"`
}

// Version is one published (or pending) release of an app.
type Version struct {
	ID          string                `json:"id"`
	AppID       string                `json:"app_id"`
	Version     string                `json:"version"`
	Channel     string                `json:"channel"`
	MinPlatform string                `json:"min_platform"`
	Manifest    appcatalog.Manifest   `json:"manifest"`
	Status      string                `json:"status"`
	SubmittedBy string                `json:"submitted_by,omitempty"`
	ReviewNote  string                `json:"review_note,omitempty"`
	PublishedAt *time.Time            `json:"published_at,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	App         *App                  `json:"app,omitempty"`
	External    *ExternalRegistration `json:"external,omitempty"`
}

// ExternalRegistration is the queryable form of a manifest's external section.
type ExternalRegistration struct {
	AppID       string   `json:"app_id"`
	LaunchURL   string   `json:"launch_url"`
	SSOClientID string   `json:"sso_client_id"`
	Scopes      []string `json:"scopes"`
	Embed       string   `json:"embed"`
	HealthURL   string   `json:"health_url"`
	WebhookURL  string   `json:"webhook_url,omitempty"`
}

// Revision returns the number that changes whenever the published catalogue
// does. A snapshot built under an older one is stale.
func (s *Store) Revision(ctx context.Context) (int64, error) {
	var revision int64
	err := s.db.QueryRow(ctx, `SELECT revision FROM registry_state WHERE id`).Scan(&revision)
	return revision, err
}

// bumpRevision is called inside every transaction that changes what the
// catalogue would say.
func bumpRevision(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `UPDATE registry_state SET revision = revision + 1 WHERE id`)
	return err
}

// --- publishers -------------------------------------------------------------

// PublisherByOwner returns the publisher a signed-in developer acts for.
func (s *Store) PublisherByOwner(ctx context.Context, sub string) (*Publisher, error) {
	return s.scanPublisher(s.db.QueryRow(ctx, publisherColumns+` WHERE owner_sub = $1`, sub))
}

func (s *Store) PublisherByID(ctx context.Context, id string) (*Publisher, error) {
	return s.scanPublisher(s.db.QueryRow(ctx, publisherColumns+` WHERE id = $1`, id))
}

const publisherColumns = `SELECT id::text, slug, name, contact_email, verified, owner_sub,
	owner_email, owner_tenant_id, owner_tenant_slug, created_at FROM publishers`

func (s *Store) scanPublisher(row pgx.Row) (*Publisher, error) {
	var p Publisher
	err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.ContactEmail, &p.Verified, &p.OwnerSub,
		&p.OwnerEmail, &p.OwnerTenantID, &p.OwnerTenantSlug, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePublisher registers a developer's organisation.
func (s *Store) CreatePublisher(ctx context.Context, p *Publisher) (*Publisher, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`INSERT INTO publishers (slug, name, contact_email, owner_sub, owner_email,
		                         owner_tenant_id, owner_tenant_slug)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		p.Slug, p.Name, p.ContactEmail, p.OwnerSub, p.OwnerEmail,
		p.OwnerTenantID, p.OwnerTenantSlug).Scan(&id)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return s.PublisherByID(ctx, id)
}

// ListPublishers is an administrative read.
func (s *Store) ListPublishers(ctx context.Context) ([]Publisher, error) {
	rows, err := s.db.Query(ctx, publisherColumns+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Publisher, 0)
	for rows.Next() {
		var p Publisher
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.ContactEmail, &p.Verified, &p.OwnerSub,
			&p.OwnerEmail, &p.OwnerTenantID, &p.OwnerTenantSlug, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// SetPublisherVerified is the administrator's decision that a publisher is who
// they claim to be.
func (s *Store) SetPublisherVerified(ctx context.Context, id string, verified bool) error {
	tag, err := s.db.Exec(ctx, `UPDATE publishers SET verified = $1 WHERE id = $2`, verified, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- apps -------------------------------------------------------------------

const appColumns = `SELECT a.id, a.publisher_id::text, a.slug, a.type, a.name, a.description,
	a.icon_url, a.category, a.visibility, a.created_at, a.updated_at, p.name,
	COALESCE((SELECT v.version FROM store_app_versions v
	           WHERE v.app_id = a.id AND v.status = 'published' AND v.channel = 'stable'
	           ORDER BY v.published_at DESC LIMIT 1), '')
	FROM store_apps a JOIN publishers p ON p.id = a.publisher_id`

func scanApp(row pgx.Row) (*App, error) {
	var a App
	err := row.Scan(&a.ID, &a.PublisherID, &a.Slug, &a.Type, &a.Name, &a.Description,
		&a.IconURL, &a.Category, &a.Visibility, &a.CreatedAt, &a.UpdatedAt,
		&a.PublisherName, &a.LatestVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) AppByID(ctx context.Context, id string) (*App, error) {
	return scanApp(s.db.QueryRow(ctx, appColumns+` WHERE a.id = $1`, id))
}

func (s *Store) AppBySlug(ctx context.Context, slug string) (*App, error) {
	return scanApp(s.db.QueryRow(ctx, appColumns+` WHERE a.slug = $1`, slug))
}

// ListApps returns catalogue entries. publisherID filters to one publisher's
// own apps (the developer console); empty returns everything public.
func (s *Store) ListApps(ctx context.Context, publisherID string) ([]App, error) {
	query := appColumns + ` WHERE a.visibility = 'public' ORDER BY a.name`
	args := []any{}
	if publisherID != "" {
		query = appColumns + ` WHERE a.publisher_id = $1 ORDER BY a.name`
		args = append(args, publisherID)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]App, 0)
	for rows.Next() {
		var a App
		if err := rows.Scan(&a.ID, &a.PublisherID, &a.Slug, &a.Type, &a.Name, &a.Description,
			&a.IconURL, &a.Category, &a.Visibility, &a.CreatedAt, &a.UpdatedAt,
			&a.PublisherName, &a.LatestVersion); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// UpsertApp creates or updates a catalogue entry. The publisher may not be
// changed by an update: an app belongs to whoever registered it.
func (s *Store) UpsertApp(ctx context.Context, a *App, texts map[string]appcatalog.CatalogAppText) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ownerID string
	err = tx.QueryRow(ctx, `SELECT publisher_id::text FROM store_apps WHERE id = $1`, a.ID).Scan(&ownerID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// New app.
	case err != nil:
		return err
	case ownerID != a.PublisherID:
		return ErrConflict
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO store_apps (id, publisher_id, slug, type, name, description, icon_url,
		                         category, visibility, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		 ON CONFLICT (id) DO UPDATE SET
		     slug = EXCLUDED.slug, type = EXCLUDED.type, name = EXCLUDED.name,
		     description = EXCLUDED.description, icon_url = EXCLUDED.icon_url,
		     category = EXCLUDED.category, visibility = EXCLUDED.visibility,
		     updated_at = NOW()`,
		a.ID, a.PublisherID, a.Slug, a.Type, a.Name, a.Description, a.IconURL,
		a.Category, a.Visibility); err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}

	for locale, text := range texts {
		if _, err := tx.Exec(ctx,
			`INSERT INTO store_app_texts (app_id, locale, name, description, category)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (app_id, locale) DO UPDATE SET
			     name = EXCLUDED.name, description = EXCLUDED.description,
			     category = EXCLUDED.category`,
			a.ID, locale, text.Name, text.Description, text.Category); err != nil {
			return err
		}
	}

	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AppTexts returns every translation held for an app.
func (s *Store) AppTexts(ctx context.Context, appID string) (map[string]appcatalog.CatalogAppText, error) {
	rows, err := s.db.Query(ctx,
		`SELECT locale, name, description, category FROM store_app_texts WHERE app_id = $1`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	texts := make(map[string]appcatalog.CatalogAppText)
	for rows.Next() {
		var locale string
		var text appcatalog.CatalogAppText
		if err := rows.Scan(&locale, &text.Name, &text.Description, &text.Category); err != nil {
			return nil, err
		}
		texts[locale] = text
	}
	return texts, rows.Err()
}

// --- versions ---------------------------------------------------------------

const versionColumns = `SELECT id::text, app_id, version, channel, min_platform, manifest,
	status, submitted_by, review_note, published_at, created_at FROM store_app_versions`

func scanVersion(row pgx.Row) (*Version, error) {
	var v Version
	var manifest []byte
	err := row.Scan(&v.ID, &v.AppID, &v.Version, &v.Channel, &v.MinPlatform, &manifest,
		&v.Status, &v.SubmittedBy, &v.ReviewNote, &v.PublishedAt, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(manifest, &v.Manifest); err != nil {
		return nil, fmt.Errorf("decode manifest of %s %s: %w", v.AppID, v.Version, err)
	}
	return &v, nil
}

func (s *Store) VersionByID(ctx context.Context, id string) (*Version, error) {
	return scanVersion(s.db.QueryRow(ctx, versionColumns+` WHERE id = $1`, id))
}

// ListVersions returns an app's releases, newest first.
func (s *Store) ListVersions(ctx context.Context, appID string, publishedOnly bool) ([]Version, error) {
	query := versionColumns + ` WHERE app_id = $1 ORDER BY created_at DESC`
	if publishedOnly {
		query = versionColumns + ` WHERE app_id = $1 AND status = 'published' ORDER BY published_at DESC`
	}
	return s.queryVersions(ctx, query, appID)
}

// ListReviewQueue returns everything waiting for a decision, oldest first —
// a queue is answered in the order people joined it.
func (s *Store) ListReviewQueue(ctx context.Context) ([]Version, error) {
	versions, err := s.queryVersions(ctx,
		versionColumns+` WHERE status = 'in_review' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		if app, err := s.AppByID(ctx, versions[i].AppID); err == nil {
			versions[i].App = app
		}
	}
	return versions, nil
}

func (s *Store) queryVersions(ctx context.Context, query string, args ...any) ([]Version, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Version, 0)
	for rows.Next() {
		var v Version
		var manifest []byte
		if err := rows.Scan(&v.ID, &v.AppID, &v.Version, &v.Channel, &v.MinPlatform, &manifest,
			&v.Status, &v.SubmittedBy, &v.ReviewNote, &v.PublishedAt, &v.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(manifest, &v.Manifest); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// SubmitVersion records a new release for review.
//
// A version number is used once. Re-submitting under a number that already
// exists is refused rather than overwritten: instances cache by version, and a
// manifest that changes under a version they already hold is a change they
// would never see.
func (s *Store) SubmitVersion(ctx context.Context, v *Version, external *ExternalRegistration) (*Version, error) {
	manifest, err := json.Marshal(v.Manifest)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO store_app_versions (app_id, version, channel, min_platform, manifest,
		                                 status, submitted_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		v.AppID, v.Version, v.Channel, v.MinPlatform, manifest, v.Status, v.SubmittedBy).Scan(&id)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}

	if external != nil {
		if _, err := tx.Exec(ctx,
			`INSERT INTO external_registrations (app_id, launch_url, sso_client_id, scopes, embed, health_url)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (app_id) DO UPDATE SET
			     launch_url = EXCLUDED.launch_url, sso_client_id = EXCLUDED.sso_client_id,
			     scopes = EXCLUDED.scopes, embed = EXCLUDED.embed, health_url = EXCLUDED.health_url`,
			v.AppID, external.LaunchURL, external.SSOClientID, external.Scopes,
			external.Embed, external.HealthURL); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO review_events (version_id, actor, action, note) VALUES ($1, $2, 'submitted', '')`,
		id, v.SubmittedBy); err != nil {
		return nil, err
	}

	// A submission changes nothing a Nexus instance can see, so the revision is
	// deliberately not bumped here — only a decision does that.
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.VersionByID(ctx, id)
}

// DecideVersion publishes, rejects or yanks a version.
//
// Publishing is the only moment the catalogue every instance pulls actually
// changes, which is why the revision is bumped inside the same transaction: a
// snapshot built a microsecond earlier must not be served afterwards.
func (s *Store) DecideVersion(ctx context.Context, versionID, action, actor, note string) error {
	var status string
	switch action {
	case "publish":
		status = StatusPublished
	case "reject":
		status = StatusRejected
	case "yank":
		status = StatusYanked
	default:
		return fmt.Errorf("unknown review action %q", action)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	publishedAt := "published_at"
	if status == StatusPublished {
		publishedAt = "NOW()"
	}
	tag, err := tx.Exec(ctx,
		`UPDATE store_app_versions
		    SET status = $1, review_note = $2, published_at = `+publishedAt+`
		  WHERE id = $3`, status, note, versionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO review_events (version_id, actor, action, note) VALUES ($1, $2, $3, $4)`,
		versionID, actor, action, note); err != nil {
		return err
	}
	if err := bumpRevision(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReviewHistory returns what has happened to a version.
func (s *Store) ReviewHistory(ctx context.Context, versionID string) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx,
		`SELECT actor, action, note, created_at FROM review_events
		  WHERE version_id = $1 ORDER BY created_at`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]map[string]any, 0)
	for rows.Next() {
		var actor, action, note string
		var at time.Time
		if err := rows.Scan(&actor, &action, &note, &at); err != nil {
			return nil, err
		}
		history = append(history, map[string]any{
			"actor": actor, "action": action, "note": note, "created_at": at,
		})
	}
	return history, rows.Err()
}

// ExternalRegistration returns an external app's registration, if it has one.
func (s *Store) ExternalRegistration(ctx context.Context, appID string) (*ExternalRegistration, error) {
	var e ExternalRegistration
	err := s.db.QueryRow(ctx,
		`SELECT app_id, launch_url, sso_client_id, scopes, embed, health_url, webhook_url
		   FROM external_registrations WHERE app_id = $1`, appID).
		Scan(&e.AppID, &e.LaunchURL, &e.SSOClientID, &e.Scopes, &e.Embed, &e.HealthURL, &e.WebhookURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// isUniqueViolation reports whether the database refused a duplicate. The
// driver's error is a string away from being useful; 23505 is the whole answer.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
