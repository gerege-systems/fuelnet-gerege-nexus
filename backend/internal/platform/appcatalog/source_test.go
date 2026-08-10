package appcatalog_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// A registry answer is a document plus a signature over it. These tests build
// both, because what is being checked is that an instance refuses everything it
// cannot verify — that is the whole security argument for letting a catalogue
// come from another machine.

const (
	bundledApp = "io.example.bundled"
	remoteApp  = "io.example.remote"
)

// writeCatalogDir lays out the catalog/apps.json + manifests/ shape on disk.
func writeCatalogDir(t *testing.T, id, slug, version string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "manifests"), 0o750); err != nil {
		t.Fatal(err)
	}
	apps := fmt.Sprintf(`[{"id":%q,"slug":%q,"name":"Bundled","version":%q}]`, id, slug, version)
	if err := os.WriteFile(filepath.Join(dir, "apps.json"), []byte(apps), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"id":%q,"name":"Bundled","version":%q,"platform":">=0.1.0"}`, id, version)
	if err := os.WriteFile(filepath.Join(dir, "manifests", slug+".json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "apps.json")
}

// signedDocument builds what the registry would serve.
func signedDocument(t *testing.T, key ed25519.PrivateKey, generatedAt, apps string) []byte {
	t.Helper()
	message := append([]byte(generatedAt+"\n"), apps...)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(key, message))
	return []byte(fmt.Sprintf(`{"generated_at":%q,"key_id":"test","apps":%s,"signature":%q}`,
		generatedAt, apps, signature))
}

func remoteApps(version string) string {
	return fmt.Sprintf(`[{"id":%q,"slug":"remote","name":"Remote","version":%q,
		"manifest":{"id":%q,"name":"Remote","version":%q,"platform":">=0.1.0"}}]`,
		remoteApp, version, remoteApp, version)
}

func newConfig(t *testing.T, public ed25519.PublicKey, url string) appcatalog.Config {
	t.Helper()
	return appcatalog.Config{
		FilePath:        writeCatalogDir(t, bundledApp, "bundled", "1.0.0"),
		URL:             url,
		PublicKey:       public,
		CachePath:       filepath.Join(t.TempDir(), "catalog.cache.json"),
		Channel:         "stable",
		PlatformVersion: "1.0.0",
	}
}

func TestFileModeIsUnchangedByTheRegistryCode(t *testing.T) {
	provider := appcatalog.NewProvider(appcatalog.Config{
		FilePath:        writeCatalogDir(t, bundledApp, "bundled", "1.0.0"),
		PlatformVersion: "1.0.0",
	})
	if provider.Remote() {
		t.Fatal("a provider with no URL must not consider itself remote")
	}

	apps, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) != 1 || apps[0].ID != bundledApp {
		t.Fatalf("expected the bundled catalog; got %+v", apps)
	}
	// The manifest beside the file has to be attached, or the dependency graph
	// and every permission grant would be built from an empty one.
	if apps[0].Manifest.ID != bundledApp {
		t.Fatalf("expected the manifest to be loaded; got %+v", apps[0].Manifest)
	}
}

func TestASignedCatalogIsAcceptedAndCached(t *testing.T) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	var served int
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		// The platform version is what tells a registry which apps this binary
		// can run, so it has to be on the query.
		if got := r.URL.Query().Get("platform"); got != "1.0.0" {
			t.Errorf("expected the platform version on the query; got %q", got)
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write(signedDocument(t, private, "2026-08-10T00:00:00Z", remoteApps("2.0.0")))
	}))
	defer registry.Close()

	cfg := newConfig(t, public, registry.URL)
	provider := appcatalog.NewProvider(cfg)

	apps, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) != 1 || apps[0].ID != remoteApp {
		t.Fatalf("expected the registry catalog; got %+v", apps)
	}

	// The document is kept so a restart while the registry is down still gets
	// the catalogue this instance was serving.
	if _, err := os.Stat(cfg.CachePath); err != nil {
		t.Fatalf("expected the accepted catalog to be cached: %v", err)
	}

	// The ETag is what keeps a sync that changes nothing from churning the
	// database and dropping every tenant's app gate.
	_, changed, err := provider.Refresh(context.Background())
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if changed {
		t.Fatal("expected a 304 to be reported as unchanged")
	}
	if served != 2 {
		t.Fatalf("expected two requests; got %d", served)
	}
}

func TestATamperedCatalogIsDiscardedAndTheBundledFileIsUsed(t *testing.T) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Signed correctly, then edited afterwards — the shape an attacker who
		// can rewrite the response but not sign it would produce.
		document := signedDocument(t, private, "2026-08-10T00:00:00Z", remoteApps("2.0.0"))
		var parsed map[string]json.RawMessage
		if err := json.Unmarshal(document, &parsed); err != nil {
			t.Error(err)
			return
		}
		parsed["apps"] = json.RawMessage(remoteApps("9.9.9"))
		tampered, err := json.Marshal(parsed)
		if err != nil {
			t.Error(err)
			return
		}
		_, _ = w.Write(tampered)
	}))
	defer registry.Close()

	provider := appcatalog.NewProvider(newConfig(t, public, registry.URL))

	if _, _, err := provider.Refresh(context.Background()); err == nil {
		t.Fatal("expected a tampered catalog to be refused")
	}

	// And boot carries on, on the catalogue that shipped with the release.
	apps, err := provider.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) != 1 || apps[0].ID != bundledApp {
		t.Fatalf("expected the bundled catalog after a refused registry answer; got %+v", apps)
	}
}

func TestAnUnreachableRegistryFallsBackToTheDiskCache(t *testing.T) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg := newConfig(t, public, "http://127.0.0.1:1/registry")
	if err := os.WriteFile(cfg.CachePath,
		signedDocument(t, private, "2026-08-09T00:00:00Z", remoteApps("2.0.0")), 0o600); err != nil {
		t.Fatal(err)
	}

	apps, err := appcatalog.NewProvider(cfg).Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) != 1 || apps[0].ID != remoteApp {
		t.Fatalf("expected the cached registry catalog; got %+v", apps)
	}
}

func TestACacheWrittenByAnyoneElseIsNotTrusted(t *testing.T) {
	public, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	_, otherKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg := newConfig(t, public, "http://127.0.0.1:1/registry")
	// Signed — just not by the registry this instance trusts.
	if err := os.WriteFile(cfg.CachePath,
		signedDocument(t, otherKey, "2026-08-09T00:00:00Z", remoteApps("2.0.0")), 0o600); err != nil {
		t.Fatal(err)
	}

	apps, err := appcatalog.NewProvider(cfg).Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) != 1 || apps[0].ID != bundledApp {
		t.Fatalf("expected a foreign-signed cache to be ignored; got %+v", apps)
	}
}

func TestACatalogueThatFailsTheExtraCheckIsRejected(t *testing.T) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(signedDocument(t, private, "2026-08-10T00:00:00Z", remoteApps("2.0.0")))
	}))
	defer registry.Close()

	cfg := newConfig(t, public, registry.URL)
	// This is where the platform holds a catalogue against the modules compiled
	// into the binary. A signature proves who sent a catalogue, not that this
	// build can run it.
	cfg.Verify = func(apps []appcatalog.CatalogApp) error {
		for _, app := range apps {
			if app.ID == remoteApp {
				return fmt.Errorf("%s is not compiled into this binary", app.ID)
			}
		}
		return nil
	}

	apps, err := appcatalog.NewProvider(cfg).Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(apps) != 1 || apps[0].ID != bundledApp {
		t.Fatalf("expected the registry answer to be rejected; got %+v", apps)
	}
}
