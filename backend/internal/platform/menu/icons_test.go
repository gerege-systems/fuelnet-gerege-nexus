package menu

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// An icon named in Go and missing from the frontend's map renders nothing.
//
// Not a broken image, not a fallback glyph — an empty square where a sidebar
// entry's icon should be, on a screen nobody looks at twice. That is how three
// of these shipped before, and how core's `building-2` and `network` would have
// shipped: the menu is declared in Go and drawn from a Record in TypeScript,
// and no compiler sees both ends.
//
// The check reads the frontend the way TestBlueprintEntriesHaveRealPages does,
// for the same reason: this drift is invisible to any single-language test.
var (
	iconMapKeys = regexp.MustCompile(`[\s,{]"?([a-z0-9-]+)"?\s*:\s*<`)
	goMenuIcons = regexp.MustCompile(`Icon:\s*"([a-z0-9-]+)"`)
)

func TestEveryMenuIconIsDrawnByTheFrontend(t *testing.T) {
	root := repoRoot(t)

	layout, err := os.ReadFile(filepath.Join(root, "frontend", "components", "Layout.tsx"))
	if err != nil {
		t.Skipf("frontend Layout.tsx not readable: %v", err)
	}
	// Only the map, not the rest of the file: every other `x: <Foo/>` in a
	// component would otherwise count as a drawable icon.
	start := strings.Index(string(layout), "const iconMap")
	if start < 0 {
		t.Skip("Layout.tsx no longer declares iconMap; this check needs rewriting")
	}
	end := strings.Index(string(layout[start:]), "\n};")
	if end < 0 {
		t.Fatal("could not find the end of iconMap in Layout.tsx")
	}
	drawn := map[string]bool{}
	for _, match := range iconMapKeys.FindAllStringSubmatch(string(layout[start:start+end]), -1) {
		drawn[match[1]] = true
	}
	if len(drawn) == 0 {
		t.Fatal("read no icons out of iconMap; the check is not measuring anything")
	}

	// Two sources, because menus come from two places: the blueprint of screens
	// still to be built, and the menus each module registers itself.
	named := map[string]string{}
	for appID, bp := range blueprints {
		for _, item := range append(append([]futureMenu{}, bp.Modules...), bp.Settings...) {
			named[item.Icon] = appID + "/" + item.ID
		}
	}
	appsDir := filepath.Join(root, "backend", "internal", "apps")
	err = filepath.WalkDir(appsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err //nolint:wrapcheck // walk errors are reported as they are
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err //nolint:wrapcheck
		}
		for _, match := range goMenuIcons.FindAllStringSubmatch(string(source), -1) {
			rel, _ := filepath.Rel(root, path)
			named[match[1]] = rel
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan the modules: %v", err)
	}

	for icon, where := range named {
		if icon == "" {
			continue // an entry with no icon is a choice, not a gap
		}
		if !drawn[icon] {
			t.Errorf("%s names the icon %q, which frontend/components/Layout.tsx does not draw;"+
				" add it to iconMap or use one that is there", where, icon)
		}
	}
}
