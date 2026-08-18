package domain_test

import (
	"errors"
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// What `domain/` is, asserted rather than argued.
//
// The apps left the platform's import graph one paragraph at a time, and
// internal/apps/boundaries_test.go exists because paragraphs were the only
// thing holding that line. This is the same argument one layer down. A domain
// package that reaches for the platform compiles perfectly well; nothing falls
// over; it just stops being a domain package, and whoever tries to lift the app
// out next pays for it.
//
// Three properties, and each one is a different way of losing the same thing.

const modulePrefix = "github.com/gerege-systems/open-gerege-nexus/backend"

// platformExceptions are domain packages allowed to name the platform, with the
// reason. It is empty, and an entry here is a decision — adding one should feel
// like making it, which is why it is a map with a sentence in it rather than a
// list of paths.
var platformExceptions = map[string]string{}

// driverExceptions are root domain packages allowed to name a database driver.
//
// Also empty. The store subpackages — domain/<app>/postgres — are not
// exceptions: they are where the driver is meant to be, and this test does not
// look at them.
var driverExceptions = map[string]string{}

// TestTheDomainDoesNotReachIntoThePlatform is the property that makes any of
// this worth doing.
//
// internal/ is this deployment: its auth, its RBAC, its tenants, its wiring. A
// domain package that imports one of them cannot be read, tested or moved
// without the whole of it, which is the state the rules were in before they
// came out of the HTTP handlers — reachable only through a migrated database
// and a request.
func TestTheDomainDoesNotReachIntoThePlatform(t *testing.T) {
	for _, pkg := range domainPackages(t) {
		for _, imported := range directImports(t, pkg) {
			if !strings.HasPrefix(imported, modulePrefix+"/internal/") {
				continue
			}
			if why, allowed := platformExceptions[short(pkg)]; allowed {
				t.Logf("%s imports %s — %s", short(pkg), short(imported), why)
				continue
			}
			t.Errorf("%s imports %s.\n"+
				"internal/ is this deployment — its auth, its tenants, its wiring. A domain "+
				"package that needs it can only be read with the whole platform around it, "+
				"which is what these rules were before they came out of the handlers.",
				short(pkg), short(imported))
		}
	}
}

// TestTheDomainDoesNotNameTheSDK is the subtler half, and the one that would
// have gone unnoticed.
//
// pkg/nexus is a fine dependency — every app has it, it is published, it is
// semver'd. That is exactly the problem: a domain that speaks nexus.DB,
// nexus.Params and nexus.Module is already in the platform's shape, and the
// only thing separating it from the app it came out of is a directory name. It
// would still need a Platform to construct, still need the SDK's idea of a
// tenant, and still be untestable without one.
func TestTheDomainDoesNotNameTheSDK(t *testing.T) {
	for _, pkg := range domainPackages(t) {
		for _, imported := range directImports(t, pkg) {
			if imported != modulePrefix+"/pkg/nexus" && !strings.HasPrefix(imported, modulePrefix+"/pkg/nexus/") {
				continue
			}
			if why, allowed := platformExceptions[short(pkg)]; allowed {
				t.Logf("%s imports the SDK — %s", short(pkg), why)
				continue
			}
			t.Errorf("%s imports %s.\n"+
				"The SDK is the platform's contract. A domain that speaks it is a domain "+
				"already wearing the platform's shape: it needs a Platform to build, the "+
				"SDK's idea of a tenant to run, and neither can be had in a unit test. "+
				"What the domain needs from storage it should declare as its own port.",
				short(pkg), short(imported))
		}
	}
}

// TestADomainRootDoesNotNameADriver keeps the store where it can be replaced.
//
// pgx in domain/<app> would mean the rules and the SQL are in one package
// again — which is where they were, in the handlers, and is why the five rules
// this app is could only be run against a migrated PostgreSQL. The driver lives
// in domain/<app>/postgres, and domain/<app>/memory is what lets the rules be
// checked in a second on a laptop.
func TestADomainRootDoesNotNameADriver(t *testing.T) {
	const driver = "github.com/jackc/pgx"
	for _, pkg := range domainPackages(t) {
		if strings.Count(strings.TrimPrefix(pkg, modulePrefix+"/domain/"), "/") > 0 {
			continue // a subpackage; the store is meant to be one of these
		}
		for _, imported := range directImports(t, pkg) {
			if !strings.HasPrefix(imported, driver) {
				continue
			}
			if why, allowed := driverExceptions[short(pkg)]; allowed {
				t.Logf("%s imports %s — %s", short(pkg), imported, why)
				continue
			}
			t.Errorf("%s imports %s.\n"+
				"The rules and the SQL in one package is where this app started, and it is "+
				"why none of its rules could be run without a migrated database. Put the "+
				"queries in %s/postgres and let the root keep asking through its port.",
				short(pkg), imported, short(pkg))
		}
	}
}

// domainPackages walks domain/ by reading the tree rather than a list, because
// a list is one more thing to keep in step and what it would fall out of step
// with is exactly what this file is about.
func domainPackages(t *testing.T) []string {
	t.Helper()
	var pkgs []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || path == "." || strings.Contains(path, "testdata") {
			return nil
		}
		pkgs = append(pkgs, modulePrefix+"/domain/"+filepath.ToSlash(path))
		return nil
	})
	if err != nil {
		t.Fatalf("walk domain: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no domain packages found; this test is not looking where it thinks it is")
	}
	return pkgs
}

// directImports is what a package imports itself, test files included: a test
// that reaches across the same boundary is the same coupling, and the split
// still breaks when somebody moves the package and the tests will not build.
func directImports(t *testing.T, importPath string) []string {
	t.Helper()
	pkg, err := build.Import(importPath, ".", 0)
	if err != nil {
		// A directory with no buildable Go files is not a package, which is a
		// normal state rather than what this is looking for.
		var noGo *build.NoGoError
		if errors.As(err, &noGo) {
			return nil
		}
		t.Fatalf("resolve %s: %v", importPath, err)
	}
	return append(append([]string{}, pkg.Imports...), append(pkg.TestImports, pkg.XTestImports...)...)
}

func short(importPath string) string { return strings.TrimPrefix(importPath, modulePrefix+"/") }
