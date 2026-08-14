/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The two things this package has to be true about itself.
 *
 * It is an external test package (`nexus_test`) on purpose: it can only reach
 * what is exported, which is the same view a distribution repository has.
 */

package nexus_test

import (
	"go/build"
	"net/http"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

// probeModule is what a module in another repository looks like.
//
// It is written against this package and nothing else — no import of anything
// under internal/ appears in this file — so if it compiles and registers, a
// third party's module can too. That is the claim the SDK exists to make, and
// before this package the claim was false: every module had to name
// internal.Module, which no other repository may import.
type probeModule struct{ id string }

func (p probeModule) ID() string      { return p.id }
func (p probeModule) Name() string    { return "Probe" }
func (p probeModule) Version() string { return "1.0.0" }

func (p probeModule) Dependencies() []nexus.Dependency {
	return []nexus.Dependency{{ID: "io.gerege.nexus.organisation", VersionConstraint: "^1.0.0"}}
}

func (p probeModule) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: "probe.read", Name: "Read"},
		{Code: "probe.secret", Name: "Secret", AdminOnly: true},
	}
}

func (p probeModule) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{{
		ID: "probe_home", Label: "Probe", Path: "/probe", Icon: "boxes", Order: 10,
		Labels: map[string]string{"mn": "Туршилт"},
	}}
}

func (p probeModule) RegisterRoutes(r chi.Router, gate func(http.Handler) http.Handler) {
	r.Route("/api/v1/probe", func(pr chi.Router) {
		pr.Use(gate)
		pr.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	})
}

func TestAModuleDefinedOutsideThePlatformCanRegisterItself(t *testing.T) {
	const id = "mn.example.probe"
	nexus.Register(probeModule{id: id})

	got, ok := nexus.Get(id)
	if !ok {
		t.Fatalf("a registered module was not in the registry")
	}
	if got.Name() != "Probe" {
		t.Fatalf("the registry returned something else: %s", got.Name())
	}
	if err := nexus.VerifyModuleExists(id); err != nil {
		t.Fatalf("VerifyModuleExists disagrees with Get: %v", err)
	}
	if err := nexus.VerifyModuleExists("mn.example.absent"); err == nil {
		t.Fatal("an id nothing registered was reported as present")
	}

	var listed bool
	for _, m := range nexus.List() {
		if m.ID() == id {
			listed = true
		}
	}
	if !listed {
		t.Fatal("the module is in the registry but not in List()")
	}

	// The routes really mount. A module that registers and then cannot be
	// served is the failure this would otherwise only show at boot.
	router := chi.NewRouter()
	got.RegisterRoutes(router, func(next http.Handler) http.Handler { return next })
	if !router.Match(chi.NewRouteContext(), http.MethodGet, "/api/v1/probe/") {
		t.Fatal("the module's route did not mount")
	}
}

func TestALabelFallsBackToTheDefault(t *testing.T) {
	item := nexus.MenuDefinition{Label: "People", Labels: map[string]string{"mn": "Ажилтнууд", "fr": ""}}
	for _, c := range []struct{ locale, want string }{
		{"mn", "Ажилтнууд"},
		{"en", "People"},
		// An empty translation is a missing one. Left to win, it renders as a
		// blank menu entry — which is how a locale nobody finished shipping
		// produces a sidebar of nameless rows.
		{"fr", "People"},
		{"zh", "People"},
	} {
		if got := item.LocalizedLabel(c.locale); got != c.want {
			t.Errorf("%s: got %q, want %q", c.locale, got, c.want)
		}
	}
}

// The contract may not depend on the implementation.
//
// `pkg/nexus` is the one package in this repository that other repositories
// compile against. An import of anything under `internal/` — direct or through
// another of our packages — would make it uncompilable outside this module, and
// the error a third party would get names a package they have never heard of.
// Nothing else in the build would notice, because inside this module the import
// is perfectly legal.
func TestTheSDKDoesNotDependOnInternal(t *testing.T) {
	const modulePrefix = "github.com/gerege-systems/open-gerege-nexus/backend"

	seen := map[string]bool{}
	var walk func(importPath, via string)
	walk = func(importPath, via string) {
		if seen[importPath] {
			return
		}
		seen[importPath] = true

		if strings.HasPrefix(importPath, modulePrefix+"/internal") {
			t.Errorf("pkg/nexus reaches %s (via %s); the SDK cannot import internal/", importPath, via)
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
	walk(modulePrefix+"/pkg/nexus", "the test")
}
