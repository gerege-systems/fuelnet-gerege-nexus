package platform

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appregistry"
	"github.com/go-chi/chi/v5"
)

// stubModule is a module that exists only to be found in the registry.
type stubModule struct {
	id      string
	version string
}

func (m stubModule) ID() string                                                 { return m.id }
func (m stubModule) Name() string                                               { return "Stub" }
func (m stubModule) Version() string                                            { return m.version }
func (m stubModule) Dependencies() []internal.Dependency                        { return nil }
func (m stubModule) Permissions() []internal.PermissionDefinition               { return nil }
func (m stubModule) Menus() []internal.MenuDefinition                           { return nil }
func (m stubModule) RegisterRoutes(chi.Router, func(http.Handler) http.Handler) {}

func appregistryRegisterStub(t *testing.T, id, version string) {
	t.Helper()
	appregistry.Register(stubModule{id: id, version: version})
}

// The three places an app's version is written — the compiled module, the
// catalogue entry and the manifest — drifted apart for two shipped apps because
// nothing compared them. This is the half of the comparison that needs the
// module registry; the catalogue-against-manifest half lives in
// appcatalog.ValidateCatalog, which every catalogue source goes through.
func TestACatalogVersionMustMatchTheCompiledModule(t *testing.T) {
	// The registry is filled by apps.Bootstrap, which a unit test does not run,
	// so a real app id would still find no module here. Registering a stub is
	// what makes the comparison happen at all.
	appregistryRegisterStub(t, "io.example.stub", "1.0.0")

	catalog := []appcatalog.CatalogApp{{
		ID:      "io.example.stub",
		Slug:    "stub",
		Version: "1.1.0",
		Manifest: appcatalog.Manifest{
			ID: "io.example.stub", Name: "Stub", Version: "1.1.0",
		},
	}}

	err := verifyCatalogVersions(catalog)
	if err == nil {
		t.Fatal("expected a catalog entry ahead of its compiled module to be refused")
	}
	if !strings.Contains(err.Error(), "io.example.stub") {
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
