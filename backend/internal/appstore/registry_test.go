package appstore_test

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/appstore"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// The registry against a real PostgreSQL, because what it guarantees lives
// partly in SQL: one row per (app, version), the revision that says the
// catalogue changed, and the snapshot that must be served byte-for-byte as it
// was signed.
//
//	APPSTORE_TEST_DATABASE_URL=postgres://... go test ./internal/appstore/...
//
// Without one these skip, so `go test ./...` stays green on a machine with no
// database.
func testRegistry(t *testing.T) (*appstore.Server, *appstore.Store, ed25519.PublicKey) {
	t.Helper()
	dsn := os.Getenv("APPSTORE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set APPSTORE_TEST_DATABASE_URL to a throwaway database to run the registry tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	goose.SetBaseFS(appstore.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	// Down then up: each run starts from an empty registry, so seeding and
	// publishing are exercised rather than skipped as "already there".
	_ = goose.DownTo(db, "migrations", 0)
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = db.Close()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := appstore.NewSigner("test-key", base64.StdEncoding.EncodeToString(private))
	if err != nil {
		t.Fatal(err)
	}

	server := appstore.NewServer(pool, signer, appstore.Config{
		Origin: "https://appstore.test", Issuer: "https://nexus.test",
		ConsoleAudience: "console",
	})
	return server, appstore.NewStore(pool), public
}

func seedRegistry(t *testing.T, store *appstore.Store) {
	t.Helper()
	err := store.SeedFromCatalogFile(context.Background(),
		filepath.FromSlash("../../../catalog/apps.json"), "gerege", "Gerege Systems", "seed:test")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func get(t *testing.T, server *appstore.Server, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	return rec
}

func TestTheSeededRegistryServesACatalogTheClientAccepts(t *testing.T) {
	server, store, public := testRegistry(t)
	seedRegistry(t, store)

	res := get(t, server, "/api/v1/registry/catalog?platform=1.0.0&channel=stable", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("catalog answered %d: %s", res.Code, res.Body.String())
	}
	etag := res.Header().Get("ETag")
	if etag == "" {
		t.Fatal("a catalog with no ETag makes every instance re-download it for ever")
	}
	document := res.Body.Bytes()

	// The real client, against this service's real output.
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", etag)
		_, _ = w.Write(document)
	}))
	defer registry.Close()

	provider := appcatalog.NewProvider(appcatalog.Config{
		FilePath:        filepath.FromSlash("../../../catalog/apps.json"),
		URL:             registry.URL,
		PublicKey:       public,
		CachePath:       filepath.Join(t.TempDir(), "cache.json"),
		PlatformVersion: "1.0.0",
	})
	apps, _, err := provider.Refresh(context.Background())
	if err != nil {
		t.Fatalf("the platform client rejected the served catalog: %v", err)
	}
	if len(apps) == 0 {
		t.Fatal("the seeded registry served an empty catalog")
	}

	// Asking again with the ETag must cost nothing. This is the difference
	// between every instance in the field pulling the whole catalogue every
	// hour and pulling it when it changes.
	again := get(t, server, "/api/v1/registry/catalog?platform=1.0.0&channel=stable",
		map[string]string{"If-None-Match": etag})
	if again.Code != http.StatusNotModified {
		t.Fatalf("expected 304 for a known ETag, got %d", again.Code)
	}

	// And the bytes are stable: a second build of the same revision must be the
	// same document, or the signature the client verified stops matching.
	repeat := get(t, server, "/api/v1/registry/catalog?platform=1.0.0&channel=stable", nil)
	if repeat.Header().Get("ETag") != etag {
		t.Fatal("the same catalog produced two different ETags")
	}
}

func TestPublishingAVersionChangesTheCatalog(t *testing.T) {
	server, store, _ := testRegistry(t)
	seedRegistry(t, store)

	before := get(t, server, "/api/v1/registry/catalog?platform=1.0.0", nil).Header().Get("ETag")

	app, err := store.AppBySlug(context.Background(), "contacts")
	if err != nil {
		t.Fatalf("seeded app: %v", err)
	}
	manifest := appcatalog.Manifest{
		ID: app.ID, Name: "Contacts", Version: "1.1.0", Platform: ">=1.0.0",
	}
	version, err := store.SubmitVersion(context.Background(), &appstore.Version{
		AppID: app.ID, Version: "1.1.0", Channel: "stable", MinPlatform: ">=1.0.0",
		Manifest: manifest, Status: appstore.StatusInReview, SubmittedBy: "test",
	}, nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// A submission is not a publication: until somebody decides, instances see
	// what they saw before.
	pending := get(t, server, "/api/v1/registry/catalog?platform=1.0.0", nil).Header().Get("ETag")
	if pending != before {
		t.Fatal("a version awaiting review must not change the published catalog")
	}

	if err := store.DecideVersion(context.Background(), version.ID, "publish", "tester", "looks good"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	after := get(t, server, "/api/v1/registry/catalog?platform=1.0.0", nil)
	if after.Header().Get("ETag") == before {
		t.Fatal("publishing a version did not change the catalog")
	}
	if !strings.Contains(after.Body.String(), `"1.1.0"`) {
		t.Fatal("the published version is missing from the catalog")
	}
}

func TestAVersionIsHiddenFromAPlatformThatCannotRunIt(t *testing.T) {
	server, store, _ := testRegistry(t)
	seedRegistry(t, store)

	app, err := store.AppBySlug(context.Background(), "contacts")
	if err != nil {
		t.Fatal(err)
	}
	version, err := store.SubmitVersion(context.Background(), &appstore.Version{
		AppID: app.ID, Version: "2.0.0", Channel: "stable", MinPlatform: ">=2.0.0",
		Manifest: appcatalog.Manifest{ID: app.ID, Name: "Contacts", Version: "2.0.0", Platform: ">=2.0.0"},
		Status:   appstore.StatusInReview, SubmittedBy: "test",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DecideVersion(context.Background(), version.ID, "publish", "tester", ""); err != nil {
		t.Fatal(err)
	}

	// An instance that cannot run it is not told about it — and, just as
	// importantly, still sees the version it can. An app that vanishes from the
	// catalogue because somebody published a release for a newer platform is an
	// app the tenant already has installed and can no longer find.
	if version := catalogVersionOf(t, server, "1.0.0", app.ID); version != "1.0.0" {
		t.Fatalf("a 1.0.0 platform should still be offered 1.0.0, was offered %q", version)
	}
	if version := catalogVersionOf(t, server, "2.1.0", app.ID); version != "2.0.0" {
		t.Fatalf("a 2.1.0 platform should see 2.0.0, saw %q", version)
	}
}

// catalogVersionOf reads one app's version out of a served catalog. Asserting
// on the whole body would match any app that happens to share a version number
// — two of the shipped ones are on 2.0.0.
func catalogVersionOf(t *testing.T, server *appstore.Server, platform, appID string) string {
	t.Helper()
	res := get(t, server, "/api/v1/registry/catalog?platform="+platform, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("catalog answered %d", res.Code)
	}
	var document struct {
		Apps []appcatalog.CatalogApp `json:"apps"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	for _, app := range document.Apps {
		if app.ID == appID {
			return app.Version
		}
	}
	return ""
}

func TestTheDeveloperApiRefusesAnyoneWithoutAToken(t *testing.T) {
	server, _, _ := testRegistry(t)

	for _, path := range []string{"/api/v1/dev/me", "/api/v1/dev/apps", "/api/v1/admin/review"} {
		if res := get(t, server, path, nil); res.Code != http.StatusUnauthorized {
			t.Fatalf("%s answered %d for an anonymous caller; expected 401", path, res.Code)
		}
	}
}

func TestThePublicApiDescribesAnApp(t *testing.T) {
	server, store, _ := testRegistry(t)
	seedRegistry(t, store)

	res := get(t, server, "/api/v1/registry/apps/esign?locale=mn", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("app detail answered %d", res.Code)
	}
	var view map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	// The storefront is public and multilingual, so the API resolves the
	// language rather than making every client carry the catalogue's
	// translations.
	if name, _ := view["name"].(string); name != "PDF цахим гарын үсэг" {
		t.Fatalf("expected the Mongolian name, got %q", name)
	}
	if versions, _ := view["versions"].([]any); len(versions) == 0 {
		t.Fatal("an app detail with no versions cannot be installed from")
	}
}
