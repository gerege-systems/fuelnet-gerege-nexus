package appinstaller_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/core"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appinstaller"
)

// A core app is the floor the rest of the platform stands on: every tenant has
// it whether or not anybody installed it, and nobody can take it away. Both
// halves are database behaviour — a sweep over tenants and a refusal on an
// installation row — so they are tested against a real schema.
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/platform/appinstaller/...
func coreCatalogApp() appcatalog.CatalogApp {
	manifest := appcatalog.Manifest{
		ID: core.ID, Name: "Organisation & People", Version: "1.0.0", Platform: ">=1.0.0",
	}
	return appcatalog.CatalogApp{
		ID: core.ID, Slug: "core", Name: manifest.Name,
		Version: "1.0.0", Visibility: "public", Manifest: manifest,
	}
}

func TestEveryTenantGetsTheCoreAppWithoutAnybodyInstallingIt(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	// The module has to be registered for a module-type app to install at all;
	// in the running server this is apps.Bootstrap.
	core.New(pool)

	var tenantID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('core-' || substr(gen_random_uuid()::text, 1, 8), 'Core sweep')
		 RETURNING id::text`).Scan(&tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID) })

	installer := appinstaller.NewAppInstaller(pool, []appcatalog.CatalogApp{coreCatalogApp()}, "1.0.0")
	// In the same order as the server: the catalogue reaches the apps table
	// first, and an installation row references it.
	if err := installer.SyncCatalog(ctx); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := installer.EnsureCoreApps(ctx); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	var installed string
	if err := pool.QueryRow(ctx,
		`SELECT installed_version FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
		tenantID, core.ID).Scan(&installed); err != nil {
		t.Fatalf("the core app was not installed for a tenant that lacked it: %v", err)
	}
	if installed != "1.0.0" {
		t.Fatalf("installed version %q", installed)
	}

	// And running again is a no-op rather than a second row or an error: this
	// runs at every boot and after every catalogue sync.
	if err := installer.EnsureCoreApps(ctx); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
		tenantID, core.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one installation, got %d", count)
	}

	// Removing it would leave the tenant with no organisation screen and no way
	// to install one — the store is behind the app that would be missing.
	err := installer.DisableApp(ctx, tenantID, "core", "someone")
	if !errors.Is(err, appinstaller.ErrCoreApp) {
		t.Fatalf("expected the core app to refuse being disabled, got %v", err)
	}
}
