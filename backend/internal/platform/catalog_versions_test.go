package platform

import (
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// The three places an app's version is written — the compiled module, the
// catalogue entry and the manifest — drifted apart for two shipped apps because
// nothing compared them. These are the comparisons that now refuse to boot.
func TestCatalogAndManifestVersionsMustAgree(t *testing.T) {
	catalog := []appcatalog.CatalogApp{{
		ID:      "io.example.contacts",
		Slug:    "contacts",
		Version: "1.1.0",
		Manifest: appcatalog.Manifest{
			ID: "io.example.contacts", Name: "Contacts", Version: "1.0.0",
		},
	}}

	err := verifyCatalogVersions(catalog)
	if err == nil {
		t.Fatal("expected a catalog entry that disagrees with its manifest to be refused")
	}
	if !strings.Contains(err.Error(), "io.example.contacts") {
		t.Fatalf("the error should name the app; got %v", err)
	}
}

func TestAnAppWithNoCompiledModuleIsAccepted(t *testing.T) {
	// An external app has no Go module by definition, so a missing registry
	// entry is not drift.
	catalog := []appcatalog.CatalogApp{{
		ID:      "mn.example.hrms",
		Slug:    "hrms",
		Version: "2026.8.0",
		Manifest: appcatalog.Manifest{
			ID: "mn.example.hrms", Name: "HRMS", Version: "2026.8.0",
		},
	}}

	if err := verifyCatalogVersions(catalog); err != nil {
		t.Fatalf("expected an app with no compiled module to pass; got %v", err)
	}
}
