package platform

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appinstaller"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
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
func (m stubModule) Dependencies() []nexus.Dependency                           { return nil }
func (m stubModule) Permissions() []nexus.PermissionDefinition                  { return nil }
func (m stubModule) Menus() []nexus.MenuDefinition                              { return nil }
func (m stubModule) RegisterRoutes(chi.Router, func(http.Handler) http.Handler) {}

func appregistryRegisterStub(t *testing.T, id, version string) {
	t.Helper()
	nexus.Register(stubModule{id: id, version: version})
}

// The three places an app's version is written — the compiled module, the
// catalogue entry and the manifest — drifted apart for two shipped apps because
// nothing compared them. This is the half of the comparison that needs the
// module registry; the catalogue-against-manifest half lives in
// catalog.ValidateCatalog, which every catalogue source goes through.
func TestACatalogVersionMustMatchTheCompiledModule(t *testing.T) {
	// The registry is filled by apps.Bootstrap, which a unit test does not run,
	// so a real app id would still find no module here. Registering a stub is
	// what makes the comparison happen at all.
	appregistryRegisterStub(t, "io.gerege.nexus.stub", "1.0.0")

	available := []catalog.CatalogApp{{
		ID:      "io.gerege.nexus.stub",
		Slug:    "stub",
		Version: "1.1.0",
		Manifest: catalog.Manifest{
			ID: "io.gerege.nexus.stub", Name: "Stub", Version: "1.1.0",
		},
	}}

	err := verifyCatalogVersions(available)
	if err == nil {
		t.Fatal("expected a catalog entry ahead of its compiled module to be refused")
	}
	if !strings.Contains(err.Error(), "io.gerege.nexus.stub") {
		t.Fatalf("the error should name the app; got %v", err)
	}
}

func TestAnAppWithNoCompiledModuleIsAccepted(t *testing.T) {
	// An external app has no Go module by definition, so a missing registry
	// entry is not drift.
	available := []catalog.CatalogApp{{
		ID:      "mn.example.hrms",
		Slug:    "hrms",
		Version: "2026.8.0",
		Manifest: catalog.Manifest{
			ID: "mn.example.hrms", Name: "HRMS", Version: "2026.8.0",
		},
	}}

	if err := verifyCatalogVersions(withDefaultApp(available...)); err != nil {
		t.Fatalf("expected an app with no compiled module to pass; got %v", err)
	}
}

// And the refusal itself: a catalogue with no default app is one this build must
// not run on, whatever else it carries.
func TestACatalogWithoutThePlatformsOwnAppIsRefused(t *testing.T) {
	err := verifyCatalogVersions([]catalog.CatalogApp{{
		ID: "mn.example.hrms", Slug: "hrms", Version: "2026.8.0",
		Manifest: catalog.Manifest{ID: "mn.example.hrms", Name: "HRMS", Version: "2026.8.0"},
	}})
	if err == nil {
		t.Fatal("expected a catalogue without the default app to be refused")
	}
	if !strings.Contains(err.Error(), appinstaller.DefaultApps[0]) {
		t.Fatalf("the refusal should name the app it is missing; got %v", err)
	}
}

// withDefaultApp puts the platform's own apps beside a fixture.
//
// Every real catalogue carries them, and a catalogue without one is now refused
// as predating this build — so a fixture testing anything else has to carry
// them as well, or it is testing that refusal by accident.
//
// Built from DefaultApps rather than from a literal list, so adding a default
// app does not silently fail every catalogue fixture in the package.
func withDefaultApp(apps ...catalog.CatalogApp) []catalog.CatalogApp {
	full := make([]catalog.CatalogApp, 0, len(appinstaller.DefaultApps)+len(apps))
	for _, id := range appinstaller.DefaultApps {
		slug := id[strings.LastIndex(id, ".")+1:]
		full = append(full, catalog.CatalogApp{
			ID: id, Slug: slug, Version: "1.0.0",
			Manifest: catalog.Manifest{ID: id, Name: slug, Version: "1.0.0"},
		})
	}
	return append(full, apps...)
}
