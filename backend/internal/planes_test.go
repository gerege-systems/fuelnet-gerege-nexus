package internal_test

import (
	"errors"
	"go/build"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The line between the two planes, written before the code moves across it.
//
// internal/apps/boundaries_test.go stops an app reaching into another app, and
// it was written the same way round: the property first, the tree afterwards.
// The reason to do it in that order is that a move is cheap and a rule is not.
// While the packages are still where they are, the rule below costs one file;
// once internal/tenant and internal/platform both exist and the compiler is
// happy with whatever imports they happen to have, the same rule costs an
// argument about which of the imports were meant.
//
// So this test is green today and green the day the tree appears. What changes
// is what it is measuring: nothing yet, then the boundary, without anybody
// having to remember to switch it on. What it does say today is how far there
// is to go — TestCountTodaysCrossPlaneImports prints the number Үе C has to
// bring to zero.
//
// Three rules, from docs/TWO_PLANES_PROPOSAL.md §2.3 and §2.8:
//
//	internal/tenant   must not import internal/platform
//	internal/platform must not import internal/tenant
//	internal/kernel   must import neither; both may import it, and pkg/…
//
// The third is the one that keeps kernel a floor rather than a third plane. A
// kernel package that imports a plane has picked a side, and everything under
// it inherits the choice — which is how internal/platform came to mean "the
// rest of the code" in the first place.

const modulePrefix = "github.com/gerege-systems/open-gerege-nexus/backend"

// crossPlaneExceptions are the imports across the planes that are meant.
//
// There are none, and there should not be one. Where the planes genuinely have
// to meet, the meeting is a contract rather than an import: five tables the
// platform writes and a tenant reads (ownership_test.go), and a small interface
// in kernel where a token minted by one plane is verified by the other
// (§2.5). Both of those are things a reviewer can look at.
//
// The map stays for the same reason internal/apps has one: the next argument
// of that kind should have to be written down, and adding an entry here should
// feel like making a decision rather than fixing a test.
var crossPlaneExceptions = map[string]map[string]string{}

// Where each file still sitting in a plane's root package is going.
//
// internal/platform's 81 packages and files have been dealt with: what is left
// there is one service.go, which is the plane composing its own subpackages
// rather than a place to put a handler. The tenant plane is one step behind —
// its 43 handler files arrived together and are being taken apart by domain —
// so the same list now describes that, and the same test refuses a file with
// nowhere to go.
//
// It is a map and not prose because a file added to a plane's root while the
// split is in progress lands in no list, and TestEveryRootFileHasAPlannedHome
// says so on the run after it appears. A move that quietly gains files is how
// the last root directory ended up with 46 of them.
//
// The values are destinations. The plane is the first segment; the ones with no
// slash are the composition root itself.
var plannedTenantPackages = map[string]string{
	// ------------------------------------------------------------ packages

	// --------------------------------------------- files in the root package
	// The platform decides whether strangers may sign up and whether the
	// deployment is read-only; this applies the answer on a tenant's request,
	// through the settings store rather than a query. That is §2.9's rule
	// working, so the file follows the request path it serves.
	"access_mode.go":      "tenant/access",
	"access_mode_test.go": "tenant/access",
	// "The tenant-facing half of two things the control plane starts", says
	// its own header. It reads credential_grants, which is a platform table
	// and not one of the five a tenant may read — see the PR.
	"app_gate_test.go": "tenant/appinstall",
	"appgate_test.go":  "tenant/appinstall",
	"appgate.go":       "tenant/appinstall",
	// The tenant's own store screens, not the platform's catalogue: every one
	// of these handlers reads claims.TenantID and installs for that
	// organisation. The catalogue they read is the deployment's and is parsed
	// in kernel/appcatalog.
	"catalog_handlers.go":      "tenant/appinstall",
	"catalog_history.go":       "tenant/appinstall",
	"catalog_history_test.go":  "tenant/appinstall",
	"catalog_overview.go":      "tenant/appinstall",
	"catalog_runnable_test.go": "tenant/appinstall",
	"catalog_versions_test.go": "tenant/appinstall",
	// Suspension and the user quota are read on the request path, not written
	// by the console: "a cached read on the request path, like the app gate
	// beside it", says the file. The console's half of an organisation's
	// lifecycle is in platform/controlplane.
	"suspension_test.go":      "tenant/access",
	"tenant_lifecycle.go":     "tenant/access + tenant/quota",
	"auth_handlers.go":        "tenant/auth",
	"capabilities_test.go":    "tenant/appinstall",
	"eid_linking_test.go":     "tenant/identity",
	"emailverify_handlers.go": "tenant/emailverify",
	"external_app_test.go":    "tenant/appinstall",
	"external_apps.go":        "tenant/appinstall",
	"extra_modules_test.go":   "tenant/appinstall",
	"login_lockout_test.go":   "tenant/auth",
	"middleware.go":           "tenant/auth", // appGateMiddleware follows external_apps.go
	"module_platform.go":      "tenant/appinstall",
	"signing.go":              "tenant/signing",
	"signing_test.go":         "tenant/signing",
}

// Nothing. internal/platform's root holds one file — service.go, the plane
// composing its own subpackages — and it is named in plannedSplitOrRemoved
// below, once, for both planes: each has exactly one and they are the same
// decision.
var plannedPlatformPackages = map[string]string{}

// The floor. These own no table and answer to neither plane, which is what
// makes them safe for both to import — and why the third rule below matters
// more than it looks: internal/kernel/security already imports
// internal/tenant/auth, and auth is the tenant plane's.
var plannedKernelPackages = map[string]string{}

// service.go, which did not move whole.
//
// server.go was the seam: it built both planes, mounted both route tables and
// owned the router. It is three files now — internal/tenant/service.go,
// internal/platform/service.go and pkg/platform/server.go — and the last of
// those is where the two planes become one process. The route tests and the
// golden route table went with it, because the surface they describe is the
// assembled one.
var plannedSplitOrRemoved = map[string]string{
	"service.go": "already three files: internal/tenant, internal/platform and pkg/platform, which is the only one that names both planes",
}

func TestTenantDoesNotImportPlatform(t *testing.T) {
	assertNoImportsAcross(t, "tenant", "platform",
		"A tenant plane that imports the platform's packages cannot be reasoned about "+
			"separately, deployed separately or reviewed separately, which is the whole "+
			"of what the split buys. Where the planes must meet, they meet through the "+
			"five boundary tables or a contract in kernel — see ownership_test.go.")
}

func TestPlatformDoesNotImportTenant(t *testing.T) {
	assertNoImportsAcross(t, "platform", "tenant",
		"This is the direction that rots quietly: the console needs one thing a tenant "+
			"handler already does, imports it, and now the operator's code runs a query "+
			"written for somebody acting inside one organisation. If the platform needs "+
			"the answer, the platform asks the database for it.")
}

// The kernel is a floor, not a third plane.
//
// It is the rule with something already against it: internal/kernel/security
// imports internal/tenant/auth today, and auth is the tenant plane's. That
// import is what this test exists to make visible before the directories are
// named — after the move it would be a kernel package that quietly belongs to
// one plane, and everything built on it would inherit the choice.
func TestKernelImportsNeitherPlane(t *testing.T) {
	pkgs := planePackages(t, "kernel")
	if len(pkgs) == 0 {
		return
	}
	for _, pkg := range pkgs {
		for _, imported := range directImports(t, pkg) {
			for _, plane := range []string{"tenant", "platform"} {
				if !strings.HasPrefix(imported, modulePrefix+"/internal/"+plane+"/") {
					continue
				}
				t.Errorf("%s imports %s.\n"+
					"kernel is what both planes stand on; a kernel package that imports one "+
					"of them has picked a side, and every package built on it inherits the "+
					"choice without being asked. If the plane needs this, it belongs in the "+
					"plane; if both do, what they share is smaller than this import.",
					short(pkg), short(imported))
			}
		}
	}
}

// Every directory and file under internal/platform has somewhere to go.
//
// The lists above are Үе C's work order, and this is what keeps them being
// that. A file added to internal/platform while the move is under way is a
// file nobody decided about, and the failure names it on the first run after
// it appears rather than on the day the directory is supposed to be empty.
func TestEveryRootFileHasAPlannedHome(t *testing.T) {
	var entries []os.DirEntry
	for _, plane := range []string{"tenant", "platform"} {
		found, err := os.ReadDir(plane)
		if err != nil {
			t.Fatalf("read internal/%s: %v", plane, err)
		}
		entries = append(entries, found...)
	}

	planned := map[string]string{}
	for _, list := range []map[string]string{
		plannedTenantPackages, plannedPlatformPackages,
		plannedKernelPackages, plannedSplitOrRemoved,
	} {
		for name, home := range list {
			if other, dup := planned[name]; dup {
				t.Errorf("%s is planned twice: %q and %q", name, other, home)
			}
			planned[name] = home
		}
	}

	var unplanned, gone []string
	present := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if name == "testdata" {
			continue
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue // a subpackage is where a file is going, not a thing to place
		}
		present[name] = true
		if _, ok := planned[name]; !ok {
			unplanned = append(unplanned, name)
		}
	}
	for name := range planned {
		if !present[name] {
			gone = append(gone, name)
		}
	}

	sort.Strings(unplanned)
	if len(unplanned) > 0 {
		t.Errorf(`a plane's root package holds %d file(s) with no planned home:

	%s

A plane's root package composes its subpackages; it is not where a handler
lives. Say which subpackage this one belongs in — or add it to
plannedSplitOrRemoved, if the honest answer is that it does not go anywhere
whole.`, len(unplanned), strings.Join(unplanned, "\n\t"))
	}

	// The other direction is the progress bar: a name that has already moved
	// is a name to strike off, and a list that keeps naming things which are
	// not there stops being a work order.
	sort.Strings(gone)
	if len(gone) > 0 {
		t.Logf("already in a subpackage: %s", strings.Join(gone, ", "))
	}
}

// How far there is to go, printed rather than asserted.
//
// Everything here is one package tree today, so none of these imports is
// wrong yet — they are wrong the moment the directories are named, which is
// exactly the work Үе C is. The number is the size of that work, and it should
// read zero on the day the move lands.
//
// It counts at package granularity. The 46 files in internal/platform's root
// are one Go package spanning both planes, so nothing can be attributed to a
// file until server.go splits; those imports are logged separately.
func TestCountTodaysCrossPlaneImports(t *testing.T) {
	plane := map[string]string{}
	for name := range plannedTenantPackages {
		plane[name] = "tenant"
	}
	for name := range plannedPlatformPackages {
		plane[name] = "platform"
	}
	for name := range plannedKernelPackages {
		plane[name] = "kernel"
	}

	var crossings []string
	for name, from := range plane {
		if strings.HasSuffix(name, ".go") {
			continue // a file in the root package, counted below
		}
		if !holdsGoFiles(filepath.Join("platform", name)) {
			continue // already moved out
		}
		seen := map[string]bool{}
		for _, imported := range directImports(t, modulePrefix+"/internal/platform/"+name) {
			dep, ok := strings.CutPrefix(imported, modulePrefix+"/internal/platform/")
			if !ok {
				continue
			}
			dep = strings.SplitN(dep, "/", 2)[0]
			to, known := plane[dep]
			if !known || to == from || to == "kernel" && from != "kernel" || seen[dep] {
				continue // kernel is what both planes are allowed to import
			}
			seen[dep] = true
			crossings = append(crossings, from+"/"+name+" imports "+to+"/"+dep)
		}
	}

	sort.Strings(crossings)
	for _, crossing := range crossings {
		t.Log(crossing)
	}
	t.Logf("cross-plane imports between internal/platform's subpackages: %d", len(crossings))

	// The root package is both planes at once until server.go is split, so its
	// imports are the work rather than a violation of it.
	root := map[string]int{}
	for _, imported := range directImports(t, modulePrefix+"/internal/platform") {
		if dep, ok := strings.CutPrefix(imported, modulePrefix+"/internal/platform/"); ok {
			root[plane[strings.SplitN(dep, "/", 2)[0]]]++
		}
	}
	t.Logf("internal/platform (the root package, both planes) imports: %d tenant, %d platform, %d kernel",
		root["tenant"], root["platform"], root["kernel"])
}

// assertNoImportsAcross is both directions of the first two rules.
func assertNoImportsAcross(t *testing.T, from, to, why string) {
	t.Helper()
	pkgs := planePackages(t, from)
	if len(pkgs) == 0 || len(planePackages(t, to)) == 0 {
		return
	}
	for _, pkg := range pkgs {
		for _, imported := range directImports(t, pkg) {
			if !strings.HasPrefix(imported, modulePrefix+"/internal/"+to+"/") {
				continue
			}
			fromPkg, toPkg := short(pkg), short(imported)
			if reason, allowed := crossPlaneExceptions[fromPkg][toPkg]; allowed {
				t.Logf("%s imports %s — %s", fromPkg, toPkg, reason)
				continue
			}
			t.Errorf("%s imports %s.\n%s", fromPkg, toPkg, why)
		}
	}
}

// planePackages walks internal/<plane> and every package under it, the same
// way internal/apps/boundaries_test.go walks internal/platform.
//
// The tree does not exist yet, and that is a state to report rather than to
// skip: a skipped test reads as "not applicable here" long after it has become
// applicable, and this one becomes applicable on a day nobody will think to
// come back and check.
func planePackages(t *testing.T, plane string) []string {
	t.Helper()
	root := filepath.Join(plane)
	if _, err := os.Stat(root); err != nil {
		t.Logf("internal/%s does not exist yet; this rule starts measuring the day Үе C creates it", plane)
		return nil
	}
	var pkgs []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err //nolint:wrapcheck // walk errors are reported as they are
		}
		if !entry.IsDir() || strings.Contains(filepath.ToSlash(path), "/testdata") || !holdsGoFiles(path) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err //nolint:wrapcheck
		}
		importPath := modulePrefix + "/internal/" + plane
		if rel != "." {
			importPath += "/" + filepath.ToSlash(rel)
		}
		pkgs = append(pkgs, importPath)
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/%s: %v", plane, err)
	}
	return pkgs
}

// directImports is what a package imports itself, not what it reaches
// transitively — the same choice, and for the same reason, as
// internal/apps/boundaries_test.go makes: the error should name the package
// that made the decision, and a transitive walk names every package above it.
//
// Test files are included. A test that reaches across the boundary is the same
// coupling; the code compiles without it and the move still breaks on it.
func directImports(t *testing.T, importPath string) []string {
	t.Helper()
	pkg, err := build.Import(importPath, ".", 0)
	if err != nil {
		var noGo *build.NoGoError
		if errors.As(err, &noGo) {
			return nil
		}
		t.Fatalf("resolve %s: %v", importPath, err)
	}
	return append(append([]string{}, pkg.Imports...), append(pkg.TestImports, pkg.XTestImports...)...)
}

// holdsGoFiles reports whether a directory is a package at all. A directory
// that has been emptied by the move — or one that only ever held others — is
// not something to resolve an import path for.
func holdsGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func short(importPath string) string { return strings.TrimPrefix(importPath, modulePrefix+"/") }
