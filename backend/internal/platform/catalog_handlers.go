/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package platform provides the core HTTP Server orchestrator, routing table,
 * authentication middleware, and app installer wiring.
 */

package platform

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appinstaller"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/menu"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleMenus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	menus, err := menu.GetTenantMenus(r.Context(), s.installer, tenantID, config.LocaleFromRequest(r))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to fetch menus")
		return
	}
	claims, _ := auth.UserFromContext(r.Context())
	if !claims.IsAdmin {
		permissions, permissionErr := s.permissions.GetUserPermissions(r.Context(), tenantID, claims.UserID)
		if permissionErr != nil {
			httpx.Error(w, http.StatusInternalServerError, "failed to resolve menu access")
			return
		}
		visible := menus[:0]
		for _, item := range menus {
			permission := s.appReadPermission(item.AppID)
			if permission == "" || permissions[permission] {
				visible = append(visible, item)
			}
		}
		menus = visible
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(menus)
}

// appReadPermission decides which permission a menu entry is hidden behind.
//
// An external app has no Go module to ask — the whole point is that it arrives
// from a registry rather than from a compiler — so its manifest answers for it.
// A tenant that installs somebody else's HRMS is not thereby putting a link to
// it in front of every member of the organisation.
func (s *Server) appReadPermission(appID string) string {
	if app, found := s.installer.GetAppByID(appID); found && app.Manifest.IsExternal() {
		for _, permission := range app.Manifest.Permissions {
			if strings.HasSuffix(permission.Code, ".read") {
				return permission.Code
			}
		}
		// A manifest that asks for nothing is visible to the tenant that
		// installed it — the same answer a module gives when it declares no
		// menu permission.
		return ""
	}
	return appReadPermission(appID)
}

// appReadPermission asks the module. It used to be a switch listing every app
// by name, which meant a module in another repository could not answer for
// itself — and losing the entry was silent, so an extracted app would keep
// appearing in everyone's sidebar. See nexus.AccessPolicy.
func appReadPermission(appID string) string {
	mod, found := nexus.Get(appID)
	if !found {
		return ""
	}
	return nexus.MenuPermissionOf(mod)
}

func (s *Server) handleListStoreApps(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenant.FromContext(r.Context())
	available := s.installer.GetCatalog()

	// "installed" and "enabled" are distinct states: an app can be installed
	// and then disabled. Deriving both from the enabled-only query reported
	// disabled apps as never installed, so the UI offered "Install" again.
	installedStates, err := s.installer.GetInstallationsForTenant(r.Context(), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load installed apps")
		return
	}

	type StoreAppResponse struct {
		catalog.CatalogApp
		Installed bool `json:"installed"`
		Enabled   bool `json:"enabled"`
		// What this tenant is running, and what the catalogue carries. They
		// were the same number for the life of the platform because an
		// installation's version never moved; now that it does, the store is
		// where the difference has to show.
		InstalledVersion string `json:"installed_version,omitempty"`
		LatestVersion    string `json:"latest_version"`
		UpdateAvailable  bool   `json:"update_available"`
	}

	locale := config.LocaleFromRequest(r)
	res := make([]StoreAppResponse, 0, len(available))
	for _, app := range available {
		held, installed := installedStates[app.ID]
		res = append(res, StoreAppResponse{
			CatalogApp:       app.Localized(locale),
			Installed:        installed,
			Enabled:          held.Enabled,
			InstalledVersion: held.Version,
			LatestVersion:    app.Version,
			UpdateAvailable:  installed && catalog.IsNewerVersion(app.Version, held.Version),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleGetStoreApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid app slug format")
		return
	}

	app, ok := s.installer.GetAppBySlug(slug)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "app not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(app.Localized(config.LocaleFromRequest(r)))
}

func (s *Server) handleListInstalledApps(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	rows, err := s.db.Query(r.Context(),
		`SELECT ai.id, ai.app_id, a.slug, a.name, ai.installed_version, ai.status, ai.enabled,
		        ai.installed_at, ai.auto_update, COALESCE(ai.pinned_version, ''),
		        COALESCE((SELECT e.details ->> 'added' FROM installation_events e
		                   WHERE e.installation_id = ai.id AND e.event_type = 'held'
		                   ORDER BY e.created_at DESC LIMIT 1), ''),
		        COALESCE((SELECT e.details ->> 'reason' FROM installation_events e
		                   WHERE e.installation_id = ai.id AND e.event_type = 'held'
		                   ORDER BY e.created_at DESC LIMIT 1), '')
		 FROM app_installations ai
		 JOIN apps a ON a.id = ai.app_id
		 WHERE ai.tenant_id = $1`, tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type InstalledApp struct {
		ID               string    `json:"id"`
		AppID            string    `json:"app_id"`
		Slug             string    `json:"slug"`
		Name             string    `json:"name"`
		InstalledVersion string    `json:"installed_version"`
		Status           string    `json:"status"`
		Enabled          bool      `json:"enabled"`
		InstalledAt      time.Time `json:"installed_at"`
		// What this app does about new versions, and what is waiting.
		AutoUpdate      bool   `json:"auto_update"`
		PinnedVersion   string `json:"pinned_version,omitempty"`
		LatestVersion   string `json:"latest_version,omitempty"`
		UpdateAvailable bool   `json:"update_available"`
		// HeldFor lists what a waiting version asks for that the installed one
		// did not, and HeldReason says why it is waiting at all. Either being
		// set means an administrator has a decision to make rather than a
		// button to press — see appinstaller.AutoUpdate.
		HeldFor    []string `json:"held_for,omitempty"`
		HeldReason string   `json:"held_reason,omitempty"`
	}

	locale := config.LocaleFromRequest(r)
	list := make([]InstalledApp, 0)
	for rows.Next() {
		var item InstalledApp
		// Skipping unreadable rows reported a tenant's app as not installed,
		// and the store then offered to install it again over the top.
		var heldFor, heldReason string
		if err := rows.Scan(&item.ID, &item.AppID, &item.Slug, &item.Name, &item.InstalledVersion,
			&item.Status, &item.Enabled, &item.InstalledAt, &item.AutoUpdate,
			&item.PinnedVersion, &heldFor, &heldReason); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "failed to read installed apps")
			return
		}
		// apps.name is the manifest's English name, and this was the one catalogue
		// surface that answered with it: the store and the sidebar both resolve
		// through the caller's locale. So an installed app was called "Digital
		// Documents & Signatures" here and "Баримт ба цахим гарын үсэг" in the menu
		// beside it — the same app under two names, which reads as an app that is
		// installed and yet missing from the menu.
		if catalogApp, ok := s.installer.GetAppBySlug(item.Slug); ok {
			if localized := catalogApp.Localized(locale).Name; localized != "" {
				item.Name = localized
			}
			item.LatestVersion = catalogApp.Version
			item.UpdateAvailable = catalog.IsNewerVersion(catalogApp.Version, item.InstalledVersion)
			// Only report a hold that is still true: once the pin is at the
			// version being offered, or the offer is gone, the recorded reason
			// is history rather than a decision.
			if item.UpdateAvailable && item.PinnedVersion == item.InstalledVersion {
				item.HeldFor = parseHeldFor(heldFor)
				item.HeldReason = heldReason
			}
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to read installed apps")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid app slug format")
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := s.installer.InstallApp(r.Context(), claims.TenantID, slug, claims.UserID); err != nil {
		// The failure used to be handed to the browser verbatim. That answered
		// a database outage with "bad request" and described the inside of the
		// server — constraint names, the module registry, the dependency graph
		// — to anyone who could press Install. Only the caller's own mistake is
		// reported as such; the rest goes to the log, where an operator can act
		// on it.
		if errors.Is(err, appinstaller.ErrAppNotFound) {
			httpx.Error(w, http.StatusNotFound, "app not found")
			return
		}
		slog.Error("app installation failed", "error", err, "app_slug", slug, "tenant_id", claims.TenantID)
		httpx.Error(w, http.StatusInternalServerError,
			"could not install this app; the failure has been logged for your administrator")
		return
	}
	// The app gate reads a cached copy of this row, so the screen that just
	// pressed the button has to stop being told the old answer.
	s.forgetAppGate(claims.TenantID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "installed", "app": slug})
}

// handleUpgradeApp moves this tenant to the version the catalogue carries.
//
// Separate from install rather than folded into it: pressing "Install" on an
// app you already have is a mistake worth ignoring, while pressing "Update"
// when there is nothing to update is a question worth answering — and the
// answer, 409, is what stops a store screen offering the button for ever.
func (s *Server) handleUpgradeApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid app slug format")
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	from, to, err := s.installer.UpgradeApp(r.Context(), claims.TenantID, slug, claims.UserID)
	switch {
	case errors.Is(err, appinstaller.ErrAppNotFound):
		httpx.Error(w, http.StatusNotFound, "app not found")
		return
	case errors.Is(err, appinstaller.ErrNotInstalled):
		httpx.Error(w, http.StatusNotFound, "this app is not installed for your organisation")
		return
	case errors.Is(err, appinstaller.ErrAlreadyCurrent):
		httpx.JSON(w, http.StatusConflict, map[string]string{
			"error":             "this app is already on the latest version",
			"installed_version": from,
			"latest_version":    to,
		})
		return
	case err != nil:
		slog.Error("app upgrade failed", "error", err, "app_slug", slug, "tenant_id", claims.TenantID)
		httpx.Error(w, http.StatusInternalServerError,
			"could not update this app; the failure has been logged for your administrator")
		return
	}

	// The app gate reads a cached copy of this row, so the screen that just
	// pressed the button has to stop being told the old answer.
	s.forgetAppGate(claims.TenantID)

	httpx.JSON(w, http.StatusOK, map[string]string{
		"status": "upgraded", "app": slug, "from": from, "to": to,
	})
}

// handleSyncCatalog is the "check for updates" button.
//
// The background sync runs on its own clock, which is the right cadence for a
// catalogue and the wrong one for an administrator who has just been told a new
// version exists. It answers what happened rather than always "ok": an
// administrator who presses it needs to know whether anything moved.
func (s *Server) handleSyncCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</cp/api/catalog/sync>; rel="successor-version"`)
	w.Header().Set("Sunset", "Sat, 01 Mar 2026 00:00:00 GMT")

	if !s.catalogSource.Remote() {
		httpx.Error(w, http.StatusNotImplemented,
			"this deployment reads its app catalog from a file; there is no registry to sync with")
		return
	}

	changed, err := s.syncCatalogFromRegistry(r.Context())
	if err != nil {
		slog.Error("catalog: manual registry sync failed", "error", err)
		httpx.Error(w, http.StatusBadGateway, "could not reach the app registry; the current catalog is unchanged")
		return
	}

	status := "unchanged"
	if changed {
		status = "updated"
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": status,
		"apps":   len(s.installer.GetCatalog()),
	})
}

// parseHeldFor reads back what holdForApproval recorded.
//
// It was written with fmt.Sprint over a slice — "[a b c]" — because the details
// column is a flat string map. Read here rather than at the point of storage so
// the event keeps the shape everything else in that table has.
func parseHeldFor(recorded string) []string {
	trimmed := strings.Trim(recorded, "[]")
	if trimmed == "" {
		return nil
	}
	return strings.Fields(trimmed)
}

// handleSetAutoUpdate records whether an app should follow the catalogue.
//
// Turning it on also clears any pin: an administrator saying "keep this current"
// and one saying "hold this version" are the same decision from either end, and
// leaving the pin would make the switch look broken.
func (s *Server) handleSetAutoUpdate(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid app slug format")
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}

	switch err := s.installer.SetAutoUpdate(r.Context(), claims.TenantID, slug, body.Enabled); {
	case errors.Is(err, appinstaller.ErrAppNotFound):
		httpx.Error(w, http.StatusNotFound, "app not found")
		return
	case errors.Is(err, appinstaller.ErrNotInstalled):
		httpx.Error(w, http.StatusNotFound, "this app is not installed for your organisation")
		return
	case err != nil:
		slog.Error("could not set auto-update", "error", err, "app_slug", slug, "tenant_id", claims.TenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not save that preference")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"app": slug, "auto_update": body.Enabled})
}

// handleCatalogStatus reports where the catalogue comes from and how the last
// attempt to refresh it went.
//
// The manual sync button answers for itself; the hourly one leaves a log line
// on a server nobody is watching, so a registry that has been unreachable for a
// week looks exactly like one that has published nothing. This is the screen
// that tells them apart — and it is also where an app that is held back stops
// being a mystery, because the reason is on the installed-apps list beside it.
func (s *Server) handleCatalogStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</cp/api/catalog/status>; rel="successor-version"`)
	w.Header().Set("Sunset", "Sat, 01 Feb 2027 00:00:00 GMT")

	s.syncMu.RLock()
	at, ok, failure := s.lastSyncAt, s.lastSyncOK, s.lastSyncErr
	s.syncMu.RUnlock()

	status := map[string]any{
		"source":        "file",
		"apps":          len(s.installer.GetCatalog()),
		"sync_interval": s.catalogSource.SyncInterval().String(),
	}
	if s.catalogSource.Remote() {
		status["source"] = "registry"
	}
	if !at.IsZero() {
		status["last_sync_at"] = at
		status["last_sync_ok"] = ok
		// The registry's own words, not a redaction: this is a tenant
		// administrator being told why their store is not moving, and "an error
		// occurred" is not something anybody can act on. It says nothing about
		// this deployment that the catalogue URL does not.
		if failure != "" {
			status["last_sync_error"] = failure
		}
	}
	httpx.JSON(w, http.StatusOK, status)
}

func (s *Server) handleDisableApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid app slug format")
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := s.installer.DisableApp(r.Context(), claims.TenantID, slug, claims.UserID); err != nil {
		if errors.Is(err, appinstaller.ErrAppNotFound) {
			httpx.Error(w, http.StatusNotFound, "app not found")
			return
		}
		slog.Error("could not disable an app", "error", err, "app_slug", slug, "tenant_id", claims.TenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not disable this app")
		return
	}
	// The app gate reads a cached copy of this row, so the screen that just
	// pressed the button has to stop being told the old answer.
	s.forgetAppGate(claims.TenantID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "disabled", "app": slug})
}

func (s *Server) handleEnableApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		httpx.Error(w, http.StatusBadRequest, "invalid app slug format")
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := s.installer.EnableApp(r.Context(), claims.TenantID, slug, claims.UserID); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	// The app gate reads a cached copy of this row, so the screen that just
	// pressed the button has to stop being told the old answer.
	s.forgetAppGate(claims.TenantID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "enabled", "app": slug})
}
