package appinstaller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
)

type AppInstaller struct {
	db              *pgxpool.Pool
	catalog         []appcatalog.CatalogApp
	platformVersion string
}

func NewAppInstaller(db *pgxpool.Pool, catalog []appcatalog.CatalogApp, platformVersion string) *AppInstaller {
	return &AppInstaller{
		db:              db,
		catalog:         catalog,
		platformVersion: platformVersion,
	}
}

func (ai *AppInstaller) GetCatalog() []appcatalog.CatalogApp {
	return ai.catalog
}

func (ai *AppInstaller) GetAppBySlug(slug string) (appcatalog.CatalogApp, bool) {
	for _, app := range ai.catalog {
		if app.Slug == slug {
			return app, true
		}
	}
	return appcatalog.CatalogApp{}, false
}

func (ai *AppInstaller) GetAppByID(id string) (appcatalog.CatalogApp, bool) {
	for _, app := range ai.catalog {
		if app.ID == id {
			return app, true
		}
	}
	return appcatalog.CatalogApp{}, false
}

// InstallApp handles recursive dependency resolution and tenant installation.
func (ai *AppInstaller) InstallApp(ctx context.Context, tenantID, appSlug, userID string) error {
	targetApp, ok := ai.GetAppBySlug(appSlug)
	if !ok {
		return fmt.Errorf("app with slug %q not found in store catalog", appSlug)
	}

	// Build dependency graph from catalog
	manifests := make([]appcatalog.Manifest, 0, len(ai.catalog))
	for _, app := range ai.catalog {
		manifests = append(manifests, app.Manifest)
	}
	graph := NewDependencyGraph(manifests)

	installOrderIDs, err := graph.ResolveInstallOrder(targetApp.ID)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Verify all modules in install order are compiled into binary
	for _, appID := range installOrderIDs {
		if err := appregistry.VerifyModuleExists(appID); err != nil {
			return fmt.Errorf("compile-time module missing for %s: %w", appID, err)
		}
	}

	tx, err := ai.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Install apps in topological order (dependencies first)
	for _, appID := range installOrderIDs {
		app, _ := ai.GetAppByID(appID)

		// Check if already installed
		var existingID string
		err := tx.QueryRow(ctx,
			`SELECT id FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
			tenantID, app.ID).Scan(&existingID)

		now := time.Now()
		var installID string

		if err != nil {
			// Not installed yet — insert installation
			installID = uuid.New().String()
			_, err = tx.Exec(ctx,
				`INSERT INTO app_installations (id, tenant_id, app_id, installed_version, status, enabled, installed_at, updated_at)
				 VALUES ($1, $2, $3, $4, 'installed', TRUE, $5, $6)`,
				installID, tenantID, app.ID, app.Version, now, now)
			if err != nil {
				return fmt.Errorf("insert app installation for %s: %w", app.ID, err)
			}
		} else {
			installID = existingID
			// Enable and update status
			_, err = tx.Exec(ctx,
				`UPDATE app_installations SET status = 'installed', enabled = TRUE, updated_at = $1
				 WHERE id = $2`, now, installID)
			if err != nil {
				return fmt.Errorf("update app installation for %s: %w", app.ID, err)
			}
		}

		// Register app permissions for tenant
		mod, _ := appregistry.Get(app.ID)
		for _, perm := range mod.Permissions() {
			permID := uuid.New().String()
			_, _ = tx.Exec(ctx,
				`INSERT INTO permissions (id, code, name, description)
				 VALUES ($1, $2, $3, $4) ON CONFLICT (code) DO NOTHING`,
				permID, perm.Code, perm.Name, perm.Description)

			// Grant to tenant admin role
			var adminRoleID string
			err := tx.QueryRow(ctx,
				`SELECT id FROM roles WHERE tenant_id = $1 AND code = 'admin'`, tenantID).Scan(&adminRoleID)
			if err == nil {
				var pID string
				_ = tx.QueryRow(ctx, `SELECT id FROM permissions WHERE code = $1`, perm.Code).Scan(&pID)
				if pID != "" {
					_, _ = tx.Exec(ctx,
						`INSERT INTO role_permissions (role_id, permission_id)
						 VALUES ($1, $2) ON CONFLICT DO NOTHING`, adminRoleID, pID)
				}
			}
		}

		// Log installation event
		evtDetails, _ := json.Marshal(map[string]string{"version": app.Version, "user_id": userID})
		evtID := uuid.New().String()
		_, _ = tx.Exec(ctx,
			`INSERT INTO installation_events (id, installation_id, event_type, details, created_at)
			 VALUES ($1, $2, 'installed', $3, $4)`,
			evtID, installID, evtDetails, now)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit install transaction: %w", err)
	}

	audit.Record(ctx, tenantID, userID, "app.install", targetApp.ID, map[string]any{
		"app_slug": targetApp.Slug,
		"version":  targetApp.Version,
	})

	return nil
}

func (ai *AppInstaller) DisableApp(ctx context.Context, tenantID, appSlug, userID string) error {
	targetApp, ok := ai.GetAppBySlug(appSlug)
	if !ok {
		return fmt.Errorf("app with slug %q not found", appSlug)
	}

	now := time.Now()
	res, err := ai.db.Exec(ctx,
		`UPDATE app_installations SET enabled = FALSE, status = 'disabled', updated_at = $1
		 WHERE tenant_id = $2 AND app_id = $3`,
		now, tenantID, targetApp.ID)
	if err != nil || res.RowsAffected() == 0 {
		return fmt.Errorf("app %s is not installed for tenant", targetApp.Slug)
	}

	audit.Record(ctx, tenantID, userID, "app.disable", targetApp.ID, map[string]any{
		"app_slug": targetApp.Slug,
	})

	return nil
}

func (ai *AppInstaller) EnableApp(ctx context.Context, tenantID, appSlug, userID string) error {
	targetApp, ok := ai.GetAppBySlug(appSlug)
	if !ok {
		return fmt.Errorf("app with slug %q not found", appSlug)
	}

	now := time.Now()
	res, err := ai.db.Exec(ctx,
		`UPDATE app_installations SET enabled = TRUE, status = 'installed', updated_at = $1
		 WHERE tenant_id = $2 AND app_id = $3`,
		now, tenantID, targetApp.ID)
	if err != nil || res.RowsAffected() == 0 {
		return fmt.Errorf("app %s is not installed for tenant", targetApp.Slug)
	}

	audit.Record(ctx, tenantID, userID, "app.enable", targetApp.ID, map[string]any{
		"app_slug": targetApp.Slug,
	})

	return nil
}

func (ai *AppInstaller) GetEnabledAppIDsForTenant(ctx context.Context, tenantID string) ([]string, error) {
	rows, err := ai.db.Query(ctx,
		`SELECT app_id FROM app_installations WHERE tenant_id = $1 AND enabled = TRUE`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
