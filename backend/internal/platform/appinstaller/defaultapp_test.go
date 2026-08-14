package appinstaller_test

import (
	"context"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/organisation"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appinstaller"
)

// A default app is what a new organisation starts with, and nothing more than
// that: the sweep installs it for a tenant that has no record of it, and an
// administrator can then remove it. Both halves are database behaviour — a
// sweep over tenants and a flag on an installation row — so they are tested
// against a real schema.
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/platform/appinstaller/...
func organisationCatalogApp() appcatalog.CatalogApp {
	manifest := appcatalog.Manifest{
		ID: organisation.ID, Name: "Organisation & People", Version: "1.0.0", Platform: ">=1.0.0",
	}
	return appcatalog.CatalogApp{
		ID: organisation.ID, Slug: "organisation", Name: manifest.Name,
		Version: "1.0.0", Visibility: "public", Manifest: manifest,
	}
}

// newSweptTenant makes a tenant, runs the catalogue and the default-app sweep
// over it, and returns the tenant and the installer both tests then work on.
func newSweptTenant(t *testing.T) (*appinstaller.AppInstaller, string) {
	t.Helper()
	pool := testPool(t)
	ctx := context.Background()

	// The module has to be registered for a module-type app to install at all;
	// in the running server this is apps.Bootstrap.
	organisation.New(nexus.NewPlatform(pool, rbac.NewSQLPermissionStore(pool)))

	var tenantID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('org-' || substr(gen_random_uuid()::text, 1, 8), 'Sweep')
		 RETURNING id::text`).Scan(&tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID) })

	installer := appinstaller.NewAppInstaller(pool, []appcatalog.CatalogApp{organisationCatalogApp()}, "1.0.0")
	// In the same order as the server: the catalogue reaches the apps table
	// first, and an installation row references it.
	if err := installer.SyncCatalog(ctx); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := installer.EnsureDefaultApps(ctx); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	return installer, tenantID
}

func TestEveryTenantGetsTheDefaultAppWithoutAnybodyInstallingIt(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	installer, tenantID := newSweptTenant(t)

	var installed string
	if err := pool.QueryRow(ctx,
		`SELECT installed_version FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
		tenantID, organisation.ID).Scan(&installed); err != nil {
		t.Fatalf("the default app was not installed for a tenant that lacked it: %v", err)
	}
	if installed != "1.0.0" {
		t.Fatalf("installed version %q", installed)
	}

	// And running again is a no-op rather than a second row or an error: this
	// runs at every boot and after every catalogue sync.
	if err := installer.EnsureDefaultApps(ctx); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
		tenantID, organisation.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one installation, got %d", count)
	}
}

// Removing the organisation app is allowed, survives the sweep that installed
// it, and takes nothing with it.
//
// The three claims are one test because they are one behaviour: "uninstall" on
// this platform means the gate closes, and it has to still mean that after the
// next catalogue refresh — otherwise a tenant that removed an app would find it
// back within the hour, which is indistinguishable from the removal having
// silently failed.
func TestTheDefaultAppCanBeRemovedAndStaysRemovedWithoutLosingData(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	installer, tenantID := newSweptTenant(t)

	// Something in the app's own tables, so "the data survived" is a row and
	// not an assumption.
	if _, err := pool.Exec(ctx,
		`INSERT INTO departments (tenant_id, code, name) VALUES ($1, 'hq', 'Төв оффис')`,
		tenantID); err != nil {
		t.Fatalf("department: %v", err)
	}

	if err := installer.DisableApp(ctx, tenantID, "organisation", "someone"); err != nil {
		t.Fatalf("the default app refused to be disabled: %v", err)
	}

	var enabled bool
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT enabled, status FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
		tenantID, organisation.ID).Scan(&enabled, &status); err != nil {
		t.Fatalf("the installation row was deleted rather than disabled: %v", err)
	}
	if enabled || status != "disabled" {
		t.Fatalf("after disabling: enabled=%v status=%q", enabled, status)
	}

	// The sweep runs at every boot and after every catalogue sync. It must not
	// undo somebody's decision.
	if err := installer.EnsureDefaultApps(ctx); err != nil {
		t.Fatalf("sweep after removal: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT enabled FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
		tenantID, organisation.ID).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("the sweep put back an app the tenant had removed")
	}

	// Nothing was dropped: turning it back on finds the department still there.
	if err := installer.EnableApp(ctx, tenantID, "organisation", "someone"); err != nil {
		t.Fatalf("re-enable: %v", err)
	}
	var name string
	if err := pool.QueryRow(ctx,
		`SELECT name FROM departments WHERE tenant_id = $1 AND code = 'hq'`, tenantID).Scan(&name); err != nil {
		t.Fatalf("the department did not survive the removal: %v", err)
	}
	if name != "Төв оффис" {
		t.Fatalf("department came back as %q", name)
	}
}
