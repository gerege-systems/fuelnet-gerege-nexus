package apps_test

import (
	"errors"
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shape of the core, asserted rather than argued.
//
// Four apps have left this repository — the App Store, State Services,
// Commerce, and the contact register — and every one of those decisions was
// made in prose: a paragraph in docs/ECOSYSTEM_GIT_STRATEGY.md explaining why
// the code could go. The paragraphs are right and they are also the only thing
// holding the line. Nothing failed when an app reached into another app's
// package, or when the platform reached into an app's; the split just got
// quietly more expensive, and the price was paid by whoever tried it next.
//
// Both properties below are true today. Written down here, they stay true
// without anybody re-making the argument, and the day one of them stops being
// true it says so in a place somebody is already looking.

const modulePrefix = "github.com/gerege-systems/open-gerege-nexus/backend"

// crossAppExceptions are the imports between app packages that are meant.
//
// One entry, and it is the reason this is a map rather than a flat refusal:
// documents absorbed the PDF signing rails when the two store cards became one,
// so it builds and mounts esign. That is a module composing another module's
// routes inside one app, which is a different thing from an app reaching into a
// neighbour for a helper — the shape this test exists to catch.
//
// A new entry here is a decision, and adding one should feel like making it.
var crossAppExceptions = map[string]map[string]string{
	"documents": {"esign": "the PDF rails moved inside the documents app; documents.New builds them and RegisterRoutes mounts them"},
}

// TestNoAppReachesIntoAnother is the property that makes a distribution split
// possible at all.
//
// An app that imports another app cannot leave without dragging it along, and
// nothing says so until somebody tries. Commerce found this out the good way:
// the plan named a contacts↔products dependency as the obstacle, and measuring
// found there was none — no Go import between any two app modules. That measure
// was taken by hand, once. This is the same measure, taken on every run.
//
// What an app should reach for instead is pkg/nexus. If two apps genuinely need
// the same thing, that thing is a platform capability and belongs behind the
// SDK — which is how nexus.DocumentFiler and nexus.MeetingBooker came to exist.
func TestNoAppReachesIntoAnother(t *testing.T) {
	for _, from := range appDirs(t) {
		imports := directImports(t, modulePrefix+"/internal/apps/"+from)
		for _, imported := range imports {
			to, ok := strings.CutPrefix(imported, modulePrefix+"/internal/apps/")
			if !ok {
				continue
			}
			to = strings.SplitN(to, "/", 2)[0]
			if to == from {
				continue
			}
			if why, allowed := crossAppExceptions[from][to]; allowed {
				t.Logf("%s imports %s — %s", from, to, why)
				continue
			}
			t.Errorf("internal/apps/%s imports internal/apps/%s.\n"+
				"An app that imports another app cannot leave this repository without it. "+
				"If both need the same thing, put it behind pkg/nexus; if this really is "+
				"one app composing another, say so in crossAppExceptions.", from, to)
		}
	}
}

// TestThePlatformDoesNotReachIntoAnApp is the same line from the other side.
//
// The platform is what every deployment runs and every distribution imports; an
// app is what one product ships. A platform package that imports an app pins
// that app into the core — the app cannot leave, and a distribution that does
// not want it compiles it anyway.
//
// This one has a history. The platform used to hold two switch statements
// keyed by app id, deciding which permission gated each app's menu and routes.
// They were not imports, so nothing here would have caught them, and they broke
// in the worst way available: when the App Store's modules moved out, the
// switches simply stopped matching and every route in that product became
// reachable by any member of a tenant. The answers moved to the modules
// (nexus.AccessPolicy) and the switches went. An import is the same mistake
// with a compiler error attached, which is the version worth having.
func TestThePlatformDoesNotReachIntoAnApp(t *testing.T) {
	for _, pkg := range platformPackages(t) {
		for _, imported := range directImports(t, pkg) {
			if strings.HasPrefix(imported, modulePrefix+"/internal/apps/") {
				t.Errorf("%s imports %s.\n"+
					"The platform is what every deployment runs; an app is what one product "+
					"ships. Importing one into the other pins it into the core, and a "+
					"distribution that does not want that app compiles it anyway.",
					strings.TrimPrefix(pkg, modulePrefix+"/"), strings.TrimPrefix(imported, modulePrefix+"/"))
			}
		}
	}
}

// appDirs lists the app packages by reading the tree rather than a list. A list
// here would be one more thing to keep in step, and the thing it would fall out
// of step with is exactly what these tests are about.
func appDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read internal/apps: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("no app packages found; this test is not looking where it thinks it is")
	}
	return names
}

// platformPackages walks internal/platform and every package under it.
func platformPackages(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "platform")
	var pkgs []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || strings.Contains(path, "/testdata") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		importPath := modulePrefix + "/internal/platform"
		if rel != "." {
			importPath += "/" + filepath.ToSlash(rel)
		}
		pkgs = append(pkgs, importPath)
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/platform: %v", err)
	}
	return pkgs
}

// directImports is what a package imports itself, not what it reaches
// transitively.
//
// Direct is the right depth for both properties here: an app importing a
// platform package that happens to import another app is the platform's fault
// and is caught by the second test, at the package that made the choice. A
// transitive walk would report it against every app in the tree and name none
// of them usefully.
//
// Test files are included. A test that reaches across the same boundary is the
// same coupling — the code compiles without it, but the split still breaks when
// somebody moves the package and the tests will not build.
func directImports(t *testing.T, importPath string) []string {
	t.Helper()
	pkg, err := build.Import(importPath, ".", 0)
	if err != nil {
		// A directory with no buildable Go files is not a package. That is a
		// normal state — a package whose files are all behind a build tag, a
		// directory holding only testdata — and not what this is looking for.
		var noGo *build.NoGoError
		if errors.As(err, &noGo) {
			return nil
		}
		t.Fatalf("resolve %s: %v", importPath, err)
	}
	return append(append([]string{}, pkg.Imports...), append(pkg.TestImports, pkg.XTestImports...)...)
}
