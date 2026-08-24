package appcatalog_test

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/appcatalog"
)

// testdata/golden_catalog.json is the contract with the registry, written down.
//
// The registry lives in another repository now
// (gitlab.gerege.mn/gerege-line/gerege-core/appstore-gerege-mn), so nothing at
// compile time stops the two drifting apart. This is what does: a document
// signed by a fixed key over fixed input, which that repository must still
// reproduce byte for byte and this one must still accept.
//
// If it fails here, this client stopped accepting what the registry publishes —
// which in the field means an instance silently stops receiving updates, with
// the catalogue it already has still being served and nothing looking wrong.
// The same file and the same key are in the registry's repository; changing the
// format means changing both.
//
// The key is a test key, seeded 0,1,2… It signs nothing real.
const goldenPublicKey = "A6EHv/POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg="

func TestTheClientStillAcceptsTheGoldenCatalog(t *testing.T) {
	document, err := os.ReadFile(filepath.Join("testdata", "golden_catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	public, err := base64.StdEncoding.DecodeString(goldenPublicKey)
	if err != nil {
		t.Fatal(err)
	}

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"golden"`)
		_, _ = w.Write(document)
	}))
	defer registry.Close()

	provider := appcatalog.NewProvider(appcatalog.Config{
		FilePath:        filepath.FromSlash("../../../../catalog/apps.json"),
		URL:             registry.URL,
		PublicKey:       ed25519.PublicKey(public),
		CachePath:       filepath.Join(t.TempDir(), "cache.json"),
		PlatformVersion: "1.0.0",
	})

	apps, changed, err := provider.Refresh(context.Background())
	if err != nil {
		t.Fatalf("this client no longer accepts what the registry publishes: %v", err)
	}
	if !changed || len(apps) != 2 {
		t.Fatalf("expected two apps from the golden document, got %d (changed=%v)", len(apps), changed)
	}

	// The fields that carry meaning across the boundary, one of each kind: a
	// module app's manifest, an external app's launch URL and its menu entry.
	if apps[0].Manifest.ID != apps[0].ID {
		t.Fatal("a manifest did not survive the trip")
	}
	external := apps[1]
	if !external.Manifest.IsExternal() {
		t.Fatal("the external app arrived as a module")
	}
	if external.Manifest.External == nil || external.Manifest.External.LaunchURL == "" {
		t.Fatal("the external section did not survive the trip")
	}
	if len(external.Manifest.Menus) == 0 || external.Manifest.Menus[0].ExternalURL == "" {
		t.Fatal("the external menu entry did not survive the trip")
	}
	if external.Translations["mn"].Name == "" && apps[0].Translations["mn"].Name == "" {
		t.Fatal("translations did not survive the trip")
	}
}
