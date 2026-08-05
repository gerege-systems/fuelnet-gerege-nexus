package menu

import (
	"context"
	"sort"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
)

type InstalledAppStore interface {
	GetEnabledAppIDsForTenant(ctx context.Context, tenantID string) ([]string, error)
}

func GetTenantMenus(ctx context.Context, store InstalledAppStore, tenantID string) ([]internal.MenuDefinition, error) {
	enabledAppIDs, err := store.GetEnabledAppIDsForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	enabledMap := make(map[string]bool)
	for _, id := range enabledAppIDs {
		enabledMap[id] = true
	}

	var menus []internal.MenuDefinition
	for _, mod := range appregistry.List() {
		if enabledMap[mod.ID()] {
			menus = append(menus, mod.Menus()...)
		}
	}

	sort.Slice(menus, func(i, j int) bool {
		return menus[i].Order < menus[j].Order
	})

	return menus, nil
}
