package menu

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appregistry"
)

type InstalledAppStore interface {
	GetEnabledAppIDsForTenant(context.Context, string) ([]string, error)
	// GetCatalog is what external apps are read from. They have no compiled
	// module to ask for menus, so their manifest is the only place their
	// navigation exists.
	GetCatalog() []appcatalog.CatalogApp
}

func GetTenantMenus(ctx context.Context, store InstalledAppStore, tenantID, locale string) ([]internal.MenuDefinition, error) {
	enabledIDs, err := store.GetEnabledAppIDsForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	for _, id := range enabledIDs {
		enabled[id] = true
	}
	menus := make([]internal.MenuDefinition, 0)
	for _, mod := range appregistry.List() {
		if !enabled[mod.ID()] {
			continue
		}
		// A module with no blueprint still has screens: the ones it registers
		// itself. Skipping it outright is how an app ships with three working
		// pages and nothing in the sidebar pointing at them, which is exactly
		// what happened to core — a blueprint lists the entries still to be
		// built, so having none of those is an ordinary state, not a reason to
		// go unlisted.
		bp, ok := blueprints[mod.ID()]
		if !ok {
			bp = blueprint{Slug: routeSlug(mod.ID())}
		}
		modulesID, settingsID := bp.Slug+"_modules", bp.Slug+"_settings"
		menus = append(menus,
			localized(internal.MenuDefinition{ID: modulesID, AppID: mod.ID(), AppName: mod.Name(), Label: "Modules", Icon: "boxes", Order: 10, Labels: groupModules}, locale),
			localized(internal.MenuDefinition{ID: settingsID, AppID: mod.ID(), AppName: mod.Name(), Label: "Settings", Icon: "settings", Order: 20, Labels: groupSettings}, locale),
		)
		for _, item := range mod.Menus() {
			item.AppID, item.AppName, item.ParentID, item.Order = mod.ID(), mod.Name(), modulesID, 10
			menus = append(menus, localized(item, locale))
		}
		for i, item := range bp.Modules {
			menus = append(menus, futureDefinition(mod.ID(), mod.Name(), modulesID, bp.Slug, item, 20+i*10, locale))
		}
		for i, item := range bp.Settings {
			menus = append(menus, futureDefinition(mod.ID(), mod.Name(), settingsID, bp.Slug, item, 10+i*10, locale))
		}
	}
	// External apps: a third-party service the tenant has installed. There is no
	// Go module behind them and no blueprint of screens still to be built, so
	// what they contribute is exactly what their manifest declares — usually one
	// entry pointing out of this platform altogether.
	for _, app := range store.GetCatalog() {
		if !app.Manifest.IsExternal() || !enabled[app.ID] {
			continue
		}
		modulesID := app.Slug + "_modules"
		menus = append(menus,
			localized(internal.MenuDefinition{ID: modulesID, AppID: app.ID, AppName: app.Name, Label: "Modules", Icon: "boxes", Order: 10, Labels: groupModules}, locale))
		for _, item := range app.Manifest.Menus {
			item.AppID, item.AppName, item.ParentID, item.Order = app.ID, app.Name, modulesID, 10
			menus = append(menus, localized(item, locale))
		}
	}

	sort.Slice(menus, func(i, j int) bool {
		if menus[i].AppID != menus[j].AppID {
			return menus[i].AppID < menus[j].AppID
		}
		if menus[i].ParentID != menus[j].ParentID {
			return menus[i].ParentID < menus[j].ParentID
		}
		return menus[i].Order < menus[j].Order
	})
	return menus, nil
}

// routeSlug is the last segment of an app id — io.gerege.nexus.core -> core — which
// is the convention every blueprint slug already follows.
func routeSlug(appID string) string {
	slug := appID
	if idx := strings.LastIndex(appID, "."); idx >= 0 {
		slug = appID[idx+1:]
	}
	return strings.ReplaceAll(slug, "_", "-")
}

func localized(item internal.MenuDefinition, locale string) internal.MenuDefinition {
	item.Label = item.LocalizedLabel(locale)
	return item
}
func futureDefinition(appID, appName, parent, slug string, item futureMenu, order int, locale string) internal.MenuDefinition {
	// Resolving through LocalizedLabel rather than an if/else on "mn" is what
	// lets a blueprint entry answer in all seven languages: an unknown locale
	// falls back to EN instead of silently returning Mongolian.
	return localized(internal.MenuDefinition{
		ID:       fmt.Sprintf("%s_%s", slug, item.ID),
		AppID:    appID,
		AppName:  appName,
		ParentID: parent,
		Label:    item.EN,
		Path:     fmt.Sprintf("/module/%s/%s", slug, item.ID),
		Icon:     item.Icon,
		Order:    order,
		Labels:   item.Labels,
	}, locale)
}
