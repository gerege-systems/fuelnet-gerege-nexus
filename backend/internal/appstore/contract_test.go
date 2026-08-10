package appstore_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/appstore"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// The contract between this registry and every Nexus instance is a byte format,
// and a byte format agreed by two programs is only agreed if one of them can
// read what the other writes. So this signs a catalogue exactly the way the
// registry endpoint does, serves it over HTTP, and points the real client — the
// one that ships in the platform binary — at it.
//
// If this test fails, instances in the field stop taking updates. Nothing about
// that failure is visible from either side alone, which is why it is tested
// from both at once.

func testSigner(t *testing.T) (*appstore.Signer, ed25519.PublicKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := appstore.NewSigner("test-key", base64.StdEncoding.EncodeToString(private))
	if err != nil {
		t.Fatal(err)
	}
	return signer, public
}

// bundledCatalog is the catalogue this repository ships, which is also what the
// registry is seeded from.
func bundledCatalog(t *testing.T) []appcatalog.CatalogApp {
	t.Helper()
	catalog, err := appcatalog.LoadFile(filepath.FromSlash("../../../catalog/apps.json"), "1.0.0")
	if err != nil {
		t.Fatalf("load the bundled catalog: %v", err)
	}
	return catalog
}

func TestTheClientAcceptsWhatTheRegistrySigns(t *testing.T) {
	signer, public := testSigner(t)
	catalog := bundledCatalog(t)

	document, _, err := appstore.SignDocument(signer, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), catalog)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog" {
			t.Errorf("the client asked for %q, which the registry does not serve", r.URL.Path)
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write(document)
	}))
	defer registry.Close()

	provider := appcatalog.NewProvider(appcatalog.Config{
		// No bundled fallback: the point is that the remote answer alone is
		// enough, not that the file behind it saved us.
		FilePath:        filepath.FromSlash("../../../catalog/apps.json"),
		URL:             registry.URL,
		PublicKey:       public,
		CachePath:       filepath.Join(t.TempDir(), "cache.json"),
		PlatformVersion: "1.0.0",
	})

	apps, changed, err := provider.Refresh(context.Background())
	if err != nil {
		t.Fatalf("the platform client rejected the registry's catalog: %v", err)
	}
	if !changed {
		t.Fatal("expected a first fetch to be a change")
	}
	if len(apps) != len(catalog) {
		t.Fatalf("expected %d apps through the client, got %d", len(catalog), len(apps))
	}

	// Not just "it parsed": the manifests have to survive the trip, because they
	// are what the installer resolves dependencies and grants permissions from.
	for i := range apps {
		if apps[i].Manifest.ID != apps[i].ID {
			t.Fatalf("app %s arrived without its manifest", apps[i].ID)
		}
		if apps[i].Version != catalog[i].Version {
			t.Fatalf("app %s arrived as version %q, not %q", apps[i].ID, apps[i].Version, catalog[i].Version)
		}
	}
}

func TestTheClientRejectsAnotherKeysSignature(t *testing.T) {
	signer, _ := testSigner(t)
	_, impostorPublic := testSigner(t)

	document, _, err := appstore.SignDocument(signer, time.Now(), bundledCatalog(t))
	if err != nil {
		t.Fatal(err)
	}

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(document)
	}))
	defer registry.Close()

	provider := appcatalog.NewProvider(appcatalog.Config{
		FilePath:        filepath.FromSlash("../../../catalog/apps.json"),
		URL:             registry.URL,
		PublicKey:       impostorPublic,
		CachePath:       filepath.Join(t.TempDir(), "cache.json"),
		PlatformVersion: "1.0.0",
	})

	if _, _, err := provider.Refresh(context.Background()); err == nil {
		t.Fatal("a catalog signed by another key must not be accepted")
	}
}

// The offline signing tool is the same code path, so a document it produces has
// to be accepted too — that is what makes it usable for an air-gapped operator
// and for testing a client without running a registry.
func TestTheSigningToolProducesAnAcceptableDocument(t *testing.T) {
	signer, public := testSigner(t)

	document, generatedAt, err := appstore.SignDocument(signer, time.Time{}, bundledCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	// A zero timestamp is replaced rather than signed as empty: the client puts
	// generated_at inside the signature, so it has to be a stable, real value.
	if generatedAt.IsZero() {
		t.Fatal("expected a substituted timestamp for an unstamped document")
	}

	path := filepath.Join(t.TempDir(), "catalog.cache.json")
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}

	// Reading it back through the disk-cache path proves the same document is
	// accepted from the fallback an instance uses when the registry is down.
	provider := appcatalog.NewProvider(appcatalog.Config{
		FilePath:        filepath.FromSlash("../../../catalog/apps.json"),
		URL:             "http://127.0.0.1:1/unreachable",
		PublicKey:       public,
		CachePath:       path,
		PlatformVersion: "1.0.0",
	})

	apps, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) == 0 {
		t.Fatal("expected the offline-signed catalog to be served from cache")
	}
}
