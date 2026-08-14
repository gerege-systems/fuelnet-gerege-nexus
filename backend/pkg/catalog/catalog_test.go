/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * An external test package, so it sees only what a registry or a publisher sees.
 */

package catalog_test

import (
	"go/build"
	"strings"
	"testing"
)

// The contract may not depend on the implementation.
//
// `pkg/catalog` is compiled against by whoever runs a registry or publishes
// to one. An import of anything under `internal/` — direct or through
// another of our packages — would make it uncompilable outside this module, and
// the error a third party would get names a package they have never heard of.
// Nothing else in the build would notice, because inside this module the import
// is perfectly legal.
func TestTheCatalogContractDoesNotDependOnInternal(t *testing.T) {
	const modulePrefix = "github.com/gerege-systems/open-gerege-nexus/backend"

	seen := map[string]bool{}
	var walk func(importPath, via string)
	walk = func(importPath, via string) {
		if seen[importPath] {
			return
		}
		seen[importPath] = true

		if strings.HasPrefix(importPath, modulePrefix+"/internal") {
			t.Errorf("pkg/catalog reaches %s (via %s); the catalogue contract cannot import internal/", importPath, via)
			return
		}
		// Only our own packages are walked. A third-party dependency cannot
		// import this module's internal packages — Go forbids it — so following
		// them would cost time and find nothing.
		if !strings.HasPrefix(importPath, modulePrefix) {
			return
		}

		pkg, err := build.Import(importPath, "", 0)
		if err != nil {
			t.Fatalf("resolve %s: %v", importPath, err)
		}
		for _, next := range pkg.Imports {
			walk(next, importPath)
		}
	}
	walk(modulePrefix+"/pkg/catalog", "the test")
}
