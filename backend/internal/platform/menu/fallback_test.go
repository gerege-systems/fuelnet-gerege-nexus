package menu_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/menu"
	"github.com/go-chi/chi/v5"
)

// A module used to have to appear in blueprints.go before any of its menus were
// shown — including the ones it declares itself. That is a silent failure: the
// app installs, its screens work, and nothing in the sidebar points at them.
// This is the module that has no blueprint, standing in for the next one.
type blueprintlessModule struct{}

func (blueprintlessModule) ID() string                          { return "io.example.noblueprint" }
func (blueprintlessModule) Name() string                        { return "No Blueprint" }
func (blueprintlessModule) Version() string                     { return "1.0.0" }
func (blueprintlessModule) Dependencies() []internal.Dependency { return nil }
func (blueprintlessModule) Permissions() []internal.PermissionDefinition {
	return nil
}
func (blueprintlessModule) Menus() []internal.MenuDefinition {
	return []internal.MenuDefinition{
		{ID: "noblueprint_home", Label: "Home", Path: "/noblueprint", Icon: "box", Order: 5},
	}
}
func (blueprintlessModule) RegisterRoutes(chi.Router, func(http.Handler) http.Handler) {}

type enabledStore struct{ ids []string }

func (s enabledStore) GetEnabledAppIDsForTenant(context.Context, string) ([]string, error) {
	return s.ids, nil
}
func (s enabledStore) GetCatalog() []appcatalog.CatalogApp { return nil }

func TestAModuleWithoutABlueprintStillContributesItsOwnScreens(t *testing.T) {
	mod := blueprintlessModule{}
	appregistry.Register(mod)

	menus, err := menu.GetTenantMenus(context.Background(),
		enabledStore{ids: []string{mod.ID()}}, "tenant", "en")
	if err != nil {
		t.Fatalf("menus: %v", err)
	}

	var found *internal.MenuDefinition
	for i := range menus {
		if menus[i].ID == "noblueprint_home" {
			found = &menus[i]
		}
	}
	if found == nil {
		t.Fatalf("the module's own screen is missing from the sidebar: %+v", menus)
	}
	// Hung under the app's Modules group, with the slug taken from the app id,
	// exactly as a module that does have a blueprint would be.
	if found.ParentID != "noblueprint_modules" {
		t.Fatalf("expected the entry under the app's Modules group, got parent %q", found.ParentID)
	}
	if found.AppName != mod.Name() {
		t.Fatalf("expected the entry to name its app, got %q", found.AppName)
	}
}
