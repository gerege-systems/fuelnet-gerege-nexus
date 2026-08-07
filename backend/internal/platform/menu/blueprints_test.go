package menu

import (
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
)

func TestEveryAppBlueprintHasTwoPopulatedGroups(t *testing.T) {
	for appID, bp := range blueprints {
		// The working module screen is contributed by mod.Menus(), so two future
		// module entries make the group total at least three.
		if len(bp.Modules) < 2 {
			t.Errorf("%s module group has %d future entries; want at least 2", appID, len(bp.Modules))
		}
		if len(bp.Settings) < 3 {
			t.Errorf("%s settings group has %d entries; want at least 3", appID, len(bp.Settings))
		}
		if bp.Slug == "" {
			t.Errorf("%s has an empty route slug", appID)
		}
	}
}

// A menu label missing a locale does not fail, it falls back to English — which
// is why the gap went unnoticed until a screen showed three languages at once.
// Coverage is therefore asserted rather than left to be noticed.
func TestEveryMenuLabelCoversEverySupportedLocale(t *testing.T) {
	check := func(name, en string, labels map[string]string) {
		t.Helper()
		if en == "" {
			t.Errorf("%s: empty English label", name)
		}
		for _, locale := range config.SupportedLocales {
			if locale == "en" {
				continue // en is the fallback and lives in the Label field
			}
			if labels[locale] == "" {
				t.Errorf("%s: no %s translation", name, locale)
			}
		}
	}

	for appID, bp := range blueprints {
		for _, item := range append(append([]futureMenu{}, bp.Modules...), bp.Settings...) {
			check(appID+"/"+item.ID, item.EN, item.Labels)
		}
	}
	check("group/modules", "Modules", groupModules)
	check("group/settings", "Settings", groupSettings)
}
