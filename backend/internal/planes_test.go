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

// Where each of internal/platform's 81 packages and files is going.
//
// This is the §2.3 tree read backwards — destination per thing that exists
// now, rather than contents per directory that does not — because that is the
// form Үе C has to work in and the form a reviewer can disagree with one line
// of. It is also why it is a map and not prose: a file added to
// internal/platform while the move is in progress lands in neither list, and
// TestEveryPlatformThingHasAPlannedHome says so on the run after it appears.
// A move that quietly gains files is how the last one ended up with 46 in the
// root directory.
//
// The values are destinations, not planes; the plane is the first segment.
var plannedTenantPackages = map[string]string{
	// ------------------------------------------------------------ packages
	"ai":           "tenant/ai",
	"appinstaller": "tenant/appinstall",
	"audit":        "tenant/audit", // audit_events; operator_audit is the platform's, in controlplane
	"auth":         "tenant/auth",
	"dan":          "tenant/identity",
	"directory":    "tenant/directory",
	"eid":          "tenant/identity",
	"eidmongolia":  "tenant/identity",
	"emailverify":  "tenant/emailverify",
	"esign":        "tenant/signing",
	"gerege":       "tenant/identity", // §1.8's fifth citizen-verification package
	"integration":  "tenant/integration",
	"memo":         "tenant/memo",
	"menu":         "tenant/menu",
	"quota":        "tenant/quota",
	"rbac":         "tenant/access",
	"reporting":    "tenant/reporting",
	"ssoclient":    "tenant/ssoclient",
	"ssoprovider":  "tenant/ssoprovider",
	"staffpin":     "tenant/devices",
	"urtuu":        "tenant/urtuu",

	// --------------------------------------------- files in the root package
	"access_control.go":      "tenant/access",
	"access_control_test.go": "tenant/access",
	// The platform decides whether strangers may sign up and whether the
	// deployment is read-only; this applies the answer on a tenant's request,
	// through the settings store rather than a query. That is §2.9's rule
	// working, so the file follows the request path it serves.
	"access_mode.go":      "tenant/access",
	"access_mode_test.go": "tenant/access",
	// "The tenant-facing half of two things the control plane starts", says
	// its own header. It reads credential_grants, which is a platform table
	// and not one of the five a tenant may read — see the PR.
	"access_recovery.go":                 "tenant/access",
	"app_gate_test.go":                   "tenant/appinstall",
	"auth_handlers.go":                   "tenant/auth",
	"capabilities_test.go":               "tenant/appinstall",
	"device_handlers.go":                 "tenant/devices",
	"device_handlers_test.go":            "tenant/devices",
	"eid_linking_test.go":                "tenant/identity",
	"eid_poll_limit_test.go":             "tenant/identity",
	"emailverify_handlers.go":            "tenant/emailverify",
	"external_app_test.go":               "tenant/appinstall",
	"external_apps.go":                   "tenant/appinstall",
	"extra_modules_test.go":              "tenant/appinstall",
	"google_binding_e2e_test.go":         "tenant/identity",
	"google_link_e2e_test.go":            "tenant/identity",
	"google_login_handlers.go":           "tenant/identity",
	"google_login_test.go":               "tenant/identity",
	"identity_binding.go":                "tenant/identity",
	"identity_handlers.go":               "tenant/identity",
	"login_lockout_test.go":              "tenant/auth",
	"middleware.go":                      "tenant/auth", // appGateMiddleware follows external_apps.go
	"module_platform.go":                 "tenant/appinstall",
	"native_operations_handlers.go":      "tenant/devices",
	"native_operations_handlers_test.go": "tenant/devices",
	"profile_handlers.go":                "tenant/profile",
	"profile_unlink_test.go":             "tenant/profile",
	"signing.go":                         "tenant/signing",
	"signing_test.go":                    "tenant/signing",
	"sso_client_handlers.go":             "tenant/ssoclient",
	"sso_client_test.go":                 "tenant/ssoclient",
	"tenant_profile_handlers.go":         "tenant/profile",
	"tenant_profile_test.go":             "tenant/profile",
}

var plannedPlatformPackages = map[string]string{
	// ------------------------------------------------------------ packages
	"appcatalog": "platform/catalog",
	// 20 files and 5867 lines that Үе C splits by domain: operator, tenants,
	// approvals, settings, flags, catalog, metering, backup, announce,
	// support, observability, audit. It is one entry here because it moves as
	// one decision, not because it lands in one place.
	"controlplane": "platform/* (split by domain)",
	"flags":        "platform/flags",
	"metering":     "platform/metering",
	"settings":     "platform/settings",

	// --------------------------------------------- files in the root package
	"catalog_handlers.go":      "platform/catalog",
	"catalog_history.go":       "platform/catalog",
	"catalog_history_test.go":  "platform/catalog",
	"catalog_overview.go":      "platform/catalog",
	"catalog_runnable_test.go": "platform/catalog",
	"catalog_versions_test.go": "platform/catalog",
	"suspension_test.go":       "platform/tenants",
	"tenant_lifecycle.go":      "platform/tenants",
}

// The floor. These own no table and answer to neither plane, which is what
// makes them safe for both to import — and why the third rule below matters
// more than it looks: internal/platform/security already imports
// internal/platform/auth, and auth is the tenant plane's.
var plannedKernelPackages = map[string]string{
	"async":      "kernel/async",
	"cache":      "kernel/cache",
	"config":     "kernel/config",
	"dbguard":    "kernel/dbguard",
	"httpx":      "kernel/httpx",
	"resilience": "kernel/resilience",
	"security":   "kernel/security",
}

// The things that do not move whole.
//
// Four of the five are the seam itself, and naming them is the point: a plan
// that lists only the clean moves is a plan that discovers these on the day it
// runs out of time for them.
var plannedSplitOrRemoved = map[string]string{
	"observability":         "kernel/telemetry (collection) + platform/observability (the operator's view) — §2.3 note 3",
	"tenant":                "deleted; its 29 callers use pkg/nexus directly — §2.3 note 1, and the only real name collision in the move",
	"server.go":             "tenant.Service + platform.Service, each with its own Routes — Үе C step 4",
	"route_policy_test.go":  "both planes: which routes a stranger may reach is a question about the whole table",
	"routes_golden_test.go": "both planes: the golden file is every route there is",
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
// It is the rule with something already against it: internal/platform/security
// imports internal/platform/auth today, and auth is the tenant plane's. That
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
func TestEveryPlatformThingHasAPlannedHome(t *testing.T) {
	root := filepath.Join("platform")
	entries, err := os.ReadDir(root)
	if err != nil {
		// internal/platform is gone, which is the end state, not a failure.
		t.Log("internal/platform no longer exists; the move is done")
		return
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
		if !entry.IsDir() && !strings.HasSuffix(name, ".go") {
			continue
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
		t.Errorf(`internal/platform holds %d thing(s) with no planned home:

	%s

Every package and file here is going to the tenant plane, the platform plane or
kernel, and which one is a decision somebody makes rather than a thing that
falls out of the move. Add it to the list it belongs in — or to
plannedSplitOrRemoved, if the honest answer is that it does not go anywhere
whole.`, len(unplanned), strings.Join(unplanned, "\n\t"))
	}

	// The other direction is the progress bar: a name that has already moved
	// is a name to strike off, and a list that keeps naming things which are
	// not there stops being a work order.
	sort.Strings(gone)
	if len(gone) > 0 {
		t.Logf("moved out of internal/platform already: %s", strings.Join(gone, ", "))
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
		if !entry.IsDir() || strings.Contains(filepath.ToSlash(path), "/testdata") {
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

func short(importPath string) string { return strings.TrimPrefix(importPath, modulePrefix+"/") }
