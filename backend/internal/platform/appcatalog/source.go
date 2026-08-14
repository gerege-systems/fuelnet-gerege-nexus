/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appcatalog

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// Where the catalogue comes from.
//
// The file is the source of truth for a self-hosted deployment and the last
// fallback for every other one, so it never stops being read: turning the
// registry on adds two sources in front of it rather than replacing it.
//
//	remote registry  → signed, fetched with an ETag, cached to disk
//	disk cache       → the last signed answer this instance accepted
//	bundled file     → catalog/apps.json, shipped with the release
//
// With APP_CATALOG_URL empty the first two do not exist and this package
// behaves exactly as it did when the file was all there was.
type Config struct {
	// FilePath is APP_CATALOG_PATH — the bundled catalogue. Manifests are read
	// from a "manifests" directory beside it.
	FilePath string

	// URL is APP_CATALOG_URL, the registry's base address. Empty means file
	// mode, which is the default and the only mode a self-hosted deployment
	// needs.
	URL string

	// PublicKey verifies the registry's signature. A remote catalogue is never
	// used without one: an unverifiable answer is discarded and the fallback
	// runs, because a registry that has been compromised or impersonated would
	// otherwise be able to hand every instance any manifest it liked.
	PublicKey ed25519.PublicKey

	// CachePath is where the last accepted signed document is kept, so a
	// restart while the registry is down still gets the catalogue it was
	// serving rather than the one the release shipped with.
	CachePath string

	// Channel is the publication channel asked for; "stable" unless set.
	Channel string

	// PlatformVersion is sent to the registry so it can answer with what this
	// binary can actually run, and is what manifests are validated against.
	PlatformVersion string

	// SyncInterval is how often the background refresh runs.
	SyncInterval time.Duration

	// Verify is an additional check a candidate catalogue must pass before it
	// is accepted. The platform uses it to hold the catalogue against the
	// modules compiled into this binary. A candidate that fails is discarded
	// and the next source is tried, so a bad publish costs an instance its
	// updates rather than its store.
	Verify func([]catalog.CatalogApp) error

	// HTTPClient is the client used for the registry. Nil gets a bounded
	// default; a test passes its own.
	HTTPClient *http.Client
}

const (
	// DefaultCachePath is where a packaged deployment can keep state.
	DefaultCachePath = "/var/lib/nexus/catalog.cache.json"
	// DefaultSyncInterval is deliberately unhurried: a catalogue change is a
	// publisher's act, and every instance polling harder buys nothing.
	DefaultSyncInterval = time.Hour
	// defaultChannel is what an instance follows unless it opts into another.
	defaultChannel = "stable"
	// maxCatalogBytes bounds what is read from the registry. The whole
	// catalogue with every manifest is tens of kilobytes; this is the ceiling
	// that keeps a hostile or broken endpoint from being an out-of-memory.
	maxCatalogBytes = 8 << 20
	// fetchTimeout bounds one attempt. Boot waits on the first one.
	fetchTimeout = 15 * time.Second
)

// ConfigFromEnv reads the deployment's catalogue configuration.
//
// Every failure here — an unparseable key, a nonsense interval — leaves the
// instance in file mode with a warning rather than refusing to start. The
// registry is an optional source of updates; it is not what makes this platform
// able to serve the apps a tenant has already installed.
func ConfigFromEnv(filePath, platformVersion string) Config {
	cfg := Config{
		FilePath:        filePath,
		URL:             strings.TrimSuffix(strings.TrimSpace(os.Getenv("APP_CATALOG_URL")), "/"),
		CachePath:       strings.TrimSpace(os.Getenv("CATALOG_CACHE_PATH")),
		Channel:         strings.TrimSpace(os.Getenv("APP_CATALOG_CHANNEL")),
		PlatformVersion: platformVersion,
		SyncInterval:    DefaultSyncInterval,
	}
	if cfg.CachePath == "" {
		cfg.CachePath = DefaultCachePath
	}
	if cfg.Channel == "" {
		cfg.Channel = defaultChannel
	}

	if raw := strings.TrimSpace(os.Getenv("CATALOG_SYNC_INTERVAL")); raw != "" {
		interval, err := time.ParseDuration(raw)
		switch {
		case err != nil:
			slog.Warn("catalog: CATALOG_SYNC_INTERVAL is not a duration; using the default",
				"value", raw, "default", DefaultSyncInterval)
		case interval < time.Minute:
			// A minute is already far more often than a catalogue changes.
			// Below that this is a load generator pointed at the registry.
			slog.Warn("catalog: CATALOG_SYNC_INTERVAL is below one minute; using one minute", "value", raw)
			cfg.SyncInterval = time.Minute
		default:
			cfg.SyncInterval = interval
		}
	}

	if cfg.URL == "" {
		return cfg
	}

	key := strings.TrimSpace(os.Getenv("APPSTORE_PUBLIC_KEY"))
	if key == "" {
		slog.Error("catalog: APP_CATALOG_URL is set but APPSTORE_PUBLIC_KEY is not; " +
			"an unsigned catalogue is never accepted, so this instance stays on its bundled catalog file")
		cfg.URL = ""
		return cfg
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		slog.Error("catalog: APPSTORE_PUBLIC_KEY is not a base64 Ed25519 public key; "+
			"this instance stays on its bundled catalog file", "length", len(decoded), "error", err)
		cfg.URL = ""
		return cfg
	}
	cfg.PublicKey = ed25519.PublicKey(decoded)
	return cfg
}

// Provider resolves the catalogue from whichever source can answer.
//
// It satisfies CatalogRepository, the boundary this package declared for a
// remote registry before there was one to talk to.
type Provider struct {
	cfg    Config
	client *http.Client

	mu      sync.RWMutex
	etag    string
	current []catalog.CatalogApp
}

// NewProvider builds a provider. It performs no I/O; Load does.
func NewProvider(cfg Config) *Provider {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}
	return &Provider{cfg: cfg, client: client}
}

// Remote reports whether a registry is configured and usable.
func (p *Provider) Remote() bool { return p.cfg.URL != "" && p.cfg.PublicKey != nil }

// SyncInterval is how often Refresh should be called.
func (p *Provider) SyncInterval() time.Duration { return p.cfg.SyncInterval }

// Load resolves the catalogue at boot, trying each source in turn.
//
// It fails only when the bundled file fails, which is the same startup error
// this platform has always had. A registry that is unreachable, slow, lying or
// gone costs a warning and nothing else — the same rule the cold-database path
// follows, and for the same reason: an instance that cannot boot serves nobody,
// including the tenants whose apps are already installed and have not changed.
func (p *Provider) Load(ctx context.Context) ([]catalog.CatalogApp, error) {
	if p.Remote() {
		apps, _, err := p.Refresh(ctx)
		if err == nil && apps != nil {
			return apps, nil
		}
		if err != nil {
			slog.Warn("catalog: could not load from the registry; falling back",
				"url", p.cfg.URL, "error", err)
		}

		if cached, err := p.loadCache(); err != nil {
			// Absent on a first boot, which is not worth an error line.
			if !errors.Is(err, os.ErrNotExist) {
				slog.Warn("catalog: the disk cache could not be used; falling back to the bundled file",
					"path", p.cfg.CachePath, "error", err)
			}
		} else if err := p.check(cached); err != nil {
			// The cache is the only source that was accepted without being held
			// against the binary that is about to serve it, and it is the one
			// source that can be older than that binary: it was written by a
			// previous build, from a registry that has since moved on. A cache
			// naming apps this build no longer knows is worse than no cache —
			// the bundled file always matches the code.
			slog.Warn("catalog: the disk cache does not match this build; falling back to the bundled file",
				"path", p.cfg.CachePath, "error", err)
		} else {
			slog.Warn("catalog: serving the last catalogue accepted from the registry",
				"path", p.cfg.CachePath, "apps", len(cached))
			p.remember(cached)
			return cached, nil
		}
	}

	apps, err := LoadFile(p.cfg.FilePath, p.cfg.PlatformVersion)
	if err != nil {
		return nil, err
	}
	if err := p.check(apps); err != nil {
		return nil, err
	}
	p.remember(apps)
	return apps, nil
}

// check holds a candidate catalogue against the binary about to serve it.
func (p *Provider) check(apps []catalog.CatalogApp) error {
	if p.cfg.Verify == nil {
		return nil
	}
	return p.cfg.Verify(apps)
}

// Refresh asks the registry for a catalogue newer than the one held.
//
// The second return value is false when the registry answered 304, which is the
// common case and the reason the ETag is kept: a sync that changes nothing must
// not churn the database or drop every tenant's app-gate cache.
func (p *Provider) Refresh(ctx context.Context) ([]catalog.CatalogApp, bool, error) {
	if !p.Remote() {
		return nil, false, errors.New("no app catalog registry is configured")
	}

	p.mu.RLock()
	etag := p.etag
	p.mu.RUnlock()

	body, newETag, changed, err := p.fetch(ctx, etag)
	if err != nil || !changed {
		return nil, false, err
	}

	apps, err := p.parseSigned(body)
	if err != nil {
		return nil, false, err
	}

	// Cached after acceptance, not before: the cache is the last catalogue this
	// instance was willing to run, so writing a document that failed
	// verification would poison the fallback it exists to be.
	if err := p.writeCache(body); err != nil {
		slog.Warn("catalog: could not write the disk cache; a restart will fall back to the bundled file",
			"path", p.cfg.CachePath, "error", err)
	}

	p.mu.Lock()
	p.etag, p.current = newETag, apps
	p.mu.Unlock()
	return apps, true, nil
}

// fetch performs one conditional GET. changed is false on 304.
func (p *Provider) fetch(ctx context.Context, etag string) (body []byte, newETag string, changed bool, err error) {
	target := fmt.Sprintf("%s/catalog?platform=%s&channel=%s",
		p.cfg.URL, urlValue(p.cfg.PlatformVersion), urlValue(p.cfg.Channel))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Accept", "application/json")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	res, err := p.client.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusNotModified:
		return nil, etag, false, nil
	case http.StatusOK:
	default:
		return nil, "", false, fmt.Errorf("registry answered %s", res.Status)
	}

	body, err = io.ReadAll(io.LimitReader(res.Body, maxCatalogBytes+1))
	if err != nil {
		return nil, "", false, err
	}
	if len(body) > maxCatalogBytes {
		return nil, "", false, fmt.Errorf("registry catalog exceeds %d bytes", maxCatalogBytes)
	}
	return body, res.Header.Get("ETag"), true, nil
}

// signedCatalog is the registry's answer.
//
// apps is held as raw JSON because that is what the signature covers. Decoding
// and re-encoding it to verify would mean trusting this program's field order,
// number formatting and escaping to reproduce the publisher's bytes exactly —
// which is how signature checks quietly stop checking anything.
type signedCatalog struct {
	GeneratedAt string          `json:"generated_at"`
	KeyID       string          `json:"key_id"`
	Apps        json.RawMessage `json:"apps"`
	Signature   string          `json:"signature"`
}

// signedMessage is what the publisher signs and this verifies:
//
//	generated_at "\n" <the apps array, verbatim>
//
// generated_at is inside the signature so a captured older catalogue cannot be
// replayed at an instance as if it were current.
func signedMessage(doc signedCatalog) []byte {
	message := make([]byte, 0, len(doc.GeneratedAt)+1+len(doc.Apps))
	message = append(message, doc.GeneratedAt...)
	message = append(message, '\n')
	return append(message, doc.Apps...)
}

// parseSigned verifies a registry document and validates what it carries.
//
// A signature that does not check out is not a reason to read the document more
// carefully — it is a reason to stop reading it. Nothing from an unverified
// document reaches the rest of the platform, not even a field.
func (p *Provider) parseSigned(body []byte) ([]catalog.CatalogApp, error) {
	var doc signedCatalog
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal registry catalog: %w", err)
	}
	if len(doc.Apps) == 0 {
		return nil, errors.New("registry catalog carries no apps")
	}

	signature, err := base64.StdEncoding.DecodeString(doc.Signature)
	if err != nil {
		return nil, fmt.Errorf("decode catalog signature: %w", err)
	}
	if !ed25519.Verify(p.cfg.PublicKey, signedMessage(doc), signature) {
		return nil, fmt.Errorf("catalog signature does not verify (key_id %q)", doc.KeyID)
	}

	var apps []catalog.CatalogApp
	if err := json.Unmarshal(doc.Apps, &apps); err != nil {
		return nil, fmt.Errorf("unmarshal catalog apps: %w", err)
	}
	// Before validation, not after: a registry that has not republished since a
	// platform app was renamed is signing a document that is correct about
	// everything except the name, and validation is where the old name would
	// otherwise be met as an app this build does not have.
	//
	// DEPRECATED: remove in vNEXT — see alias.go.
	apps = applyRenames(apps)
	if err := catalog.ValidateCatalog(apps, p.cfg.PlatformVersion); err != nil {
		return nil, err
	}
	if err := p.check(apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// loadCache reads the last accepted document back, signature and all.
//
// It is verified again rather than trusted for having been verified once: the
// file sits on disk between two boots, and anything that could edit it could
// also have written it in the first place.
func (p *Provider) loadCache() ([]catalog.CatalogApp, error) {
	// #nosec G304 -- CATALOG_CACHE_PATH is deployment configuration, and what
	// is read from it is only used if it carries the registry's signature.
	body, err := os.ReadFile(p.cfg.CachePath)
	if err != nil {
		return nil, err
	}
	return p.parseSigned(body)
}

func (p *Provider) writeCache(body []byte) error {
	if p.cfg.CachePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p.cfg.CachePath), 0o750); err != nil {
		return err
	}
	// Written beside and renamed: a half-written cache is a fallback that fails
	// exactly when the registry is already down, which is the one moment it has
	// a job to do.
	temp := p.cfg.CachePath + ".tmp"
	if err := os.WriteFile(temp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, p.cfg.CachePath)
}

func (p *Provider) remember(apps []catalog.CatalogApp) {
	p.mu.Lock()
	p.current = apps
	p.mu.Unlock()
}

// ListApps returns the catalogue currently held. It does not fetch: the held
// copy is what the rest of the platform is serving, and a read path that could
// block on somebody else's HTTP server is not one.
func (p *Provider) ListApps(_ context.Context) ([]catalog.CatalogApp, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.current == nil {
		return nil, errors.New("the app catalog has not been loaded yet")
	}
	return p.current, nil
}

// GetAppBySlug returns one catalogue entry.
func (p *Provider) GetAppBySlug(ctx context.Context, slug string) (catalog.CatalogApp, error) {
	apps, err := p.ListApps(ctx)
	if err != nil {
		return catalog.CatalogApp{}, err
	}
	for _, app := range apps {
		if app.Slug == slug {
			return app, nil
		}
	}
	return catalog.CatalogApp{}, fmt.Errorf("app %q is not in the catalog", slug)
}

// urlValue keeps a configured value from changing the shape of the query.
func urlValue(value string) string {
	return strings.NewReplacer("&", "", "?", "", " ", "", "#", "").Replace(value)
}
