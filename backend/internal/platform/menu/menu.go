package menu

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// defaultMenuOrder is where an entry sits when its module expresses no
// preference: after anything that asked to come first, before the blueprint
// entries, which start at 20.
const defaultMenuOrder = 10

type InstalledAppStore interface {
	GetEnabledAppIDsForTenant(context.Context, string) ([]string, error)
	// GetCatalog is what external apps are read from. They have no compiled
	// module to ask for menus, so their manifest is the only place their
	// navigation exists.
	GetCatalog() []appcatalog.CatalogApp
}

func GetTenantMenus(ctx context.Context, store InstalledAppStore, tenantID, locale string) ([]nexus.MenuDefinition, error) {
	enabledIDs, err := store.GetEnabledAppIDsForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	for _, id := range enabledIDs {
		enabled[id] = true
	}
	menus := make([]nexus.MenuDefinition, 0)
	for _, mod := range nexus.List() {
		if !enabled[mod.ID()] {
			continue
		}
		// A module with no blueprint still has screens: the ones it registers
		// itself. Skipping it outright is how an app ships with three working
		// pages and nothing in the sidebar pointing at them, which is exactly
		// what happened to the organisation app — a blueprint lists the entries
		// still to be built, so having none of those is an ordinary state, not
		// a reason to go unlisted.
		bp, ok := blueprints[mod.ID()]
		if !ok {
			bp = blueprint{Slug: routeSlug(mod.ID())}
		}
		modulesID, settingsID := bp.Slug+"_modules", bp.Slug+"_settings"
		menus = append(menus,
			localized(nexus.MenuDefinition{ID: modulesID, AppID: mod.ID(), AppName: mod.Name(), Label: "Modules", Icon: "boxes", Order: 10, Labels: groupModules}, locale),
			localized(nexus.MenuDefinition{ID: settingsID, AppID: mod.ID(), AppName: mod.Name(), Label: "Settings", Icon: "settings", Order: 20, Labels: groupSettings}, locale),
		)
		for _, item := range mod.Menus() {
			// The parent is the platform's to decide; the order is the
			// module's. It used to be overwritten with 10 for every entry,
			// which left the organisation app's screens — departments and
			// people — sorting equal and coming out in whatever order the sort
			// happened to leave them, changing between builds.
			item.AppID, item.AppName, item.ParentID = mod.ID(), mod.Name(), modulesID
			if item.Order == 0 {
				item.Order = defaultMenuOrder
			}
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
			localized(nexus.MenuDefinition{ID: modulesID, AppID: app.ID, AppName: app.Name, Label: "Modules", Icon: "boxes", Order: 10, Labels: groupModules}, locale))
		for _, item := range app.Manifest.Menus {
			item.AppID, item.AppName, item.ParentID = app.ID, app.Name, modulesID
			if item.Order == 0 {
				item.Order = defaultMenuOrder
			}
			menus = append(menus, localized(item, locale))
		}
	}

	// Stable, so entries that share an order keep the order their module
	// declared them in rather than one the sort invented.
	sort.SliceStable(menus, func(i, j int) bool {
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

// routeSlug is the last segment of an app id — io.gerege.nexus.organisation -> organisation — which
// is the convention every blueprint slug already follows.
func routeSlug(appID string) string {
	slug := appID
	if idx := strings.LastIndex(appID, "."); idx >= 0 {
		slug = appID[idx+1:]
	}
	return strings.ReplaceAll(slug, "_", "-")
}

func localized(item nexus.MenuDefinition, locale string) nexus.MenuDefinition {
	item.Label = item.LocalizedLabel(locale)
	return item
}
func futureDefinition(appID, appName, parent, slug string, item futureMenu, order int, locale string) nexus.MenuDefinition {
	// Resolving through LocalizedLabel rather than an if/else on "mn" is what
	// lets a blueprint entry answer in all seven languages: an unknown locale
	// falls back to EN instead of silently returning Mongolian.
	return localized(nexus.MenuDefinition{
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
