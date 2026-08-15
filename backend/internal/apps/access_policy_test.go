package apps

import (
	"os"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/egov"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/organisation"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// What every module compiled into this repository declares to nexus.AccessPolicy.
//
// This table is the platform's two switch statements, turned inside out. They
// used to be code: the platform looked up an app ID and decided. That could not
// survive a module moving to another repository, so the answer moved to the
// modules — and the table stayed, as an assertion. The difference matters. As
// code it was load-bearing and could be edited without anyone noticing what had
// changed; as a test it is a claim that fails loudly when a module's gate moves.
//
// A permission that quietly disappears is the failure this guards against. It
// does not look like a bug from the outside: the app still works, the pages
// still load, and more people can reach them than should. Nothing turns red on
// its own. So it is written down twice on purpose — once where the module is
// defined, once here — and the two have to agree.
//
// The modules are nil pointers. None of these methods touches a field, and
// constructing ten modules would drag in a database, an HTTP client and the
// government integration manager to ask each of them a constant.
var corePolicies = map[string]struct {
	module         nexus.Module
	menu, prefix   string
	whyNoRouteGate string
}{
	"contacts":    {(*contacts.Module)(nil), "contacts.read", "contacts", ""},
	"sso_clients": {(*sso_clients.SSOClientsModule)(nil), "sso_clients.read", "sso_clients", ""},

	// The two that gate themselves, and why the verb is not enough for them.
	"documents": {(*documents.DocumentsModule)(nil), "documents.read", "",
		"who may read a document depends on who it was shared with"},
	"egov": {(*egov.Module)(nil), "egov.read", "",
		"a citizen-registry lookup is a GET that must not be a read every member holds"},
}

func TestEveryCoreModuleDeclaresTheAccessPolicyWeThinkItDoes(t *testing.T) {
	for name, want := range corePolicies {
		t.Run(name, func(t *testing.T) {
			if got := nexus.MenuPermissionOf(want.module); got != want.menu {
				t.Errorf("menu permission: got %q, want %q", got, want.menu)
			}
			if got := nexus.RoutePermissionPrefixOf(want.module); got != want.prefix {
				t.Errorf("route prefix: got %q, want %q", got, want.prefix)
			}
			if want.prefix == "" && want.whyNoRouteGate == "" {
				t.Errorf("a module that declines route gating needs a reason recorded here")
			}
		})
	}
}

// The modules that deliberately declare nothing. organisation and reports are
// visible to every member of a tenant that has them installed.
// Absent-because-considered and absent-because-forgotten look identical in a
// table, so the difference is asserted rather than left to a comment.
var policylessModules = map[string]nexus.Module{
	"organisation": (*organisation.Module)(nil),
	"reports":      (*reports.Module)(nil),
}

// Directories under internal/apps that hold no module at all.
//
// esign is one since the merge: it is the PDF rails of the documents app, built
// by documents.New and mounted by its RegisterRoutes, and it registers nothing
// with nexus. It appears here rather than being deleted from the classification
// because the directory is still there, and the count below reads directories —
// a package that stops being a module must say so somewhere, or the next person
// reads its absence as an oversight.
//
// Its routes are gated: every handler asserts one of the documents permissions
// through Module.require, which is the same way documents gates its own.
var nonModulePackages = map[string]string{
	"esign": "the documents app's PDF rails; documents.New builds it and mounts its routes",
}

func TestTheModulesWithNoPolicyAreTheOnesWeMeant(t *testing.T) {
	for name, mod := range policylessModules {
		t.Run(name, func(t *testing.T) {
			if got := nexus.MenuPermissionOf(mod); got != "" {
				t.Errorf("expected no menu permission, got %q", got)
			}
			if got := nexus.RoutePermissionPrefixOf(mod); got != "" {
				t.Errorf("expected no route prefix, got %q", got)
			}
		})
	}
}

// The count is the point of this one, and it reads the directories rather than
// trusting a number somebody typed.
//
// The first version compared two hand-written constants. It would have passed
// happily while three modules were deleted from the tree, because both
// constants get edited in the same breath — a test that can only catch somebody
// editing one of its own numbers and not the other is not watching anything.
//
// What it is for: adding a module and never deciding its access policy. The
// module works, its routes mount, and nothing asks whether anyone should be
// able to reach them. That mistake leaves no other trace.
func TestEveryModuleInThisRepositoryIsClassified(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read internal/apps: %v", err)
	}

	unseen := map[string]bool{}
	for name := range corePolicies {
		unseen[name] = true
	}
	for name := range policylessModules {
		unseen[name] = true
	}
	for name := range nonModulePackages {
		unseen[name] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !unseen[entry.Name()] {
			t.Errorf("internal/apps/%s is not classified — add it to corePolicies, "+
				"to policylessModules or to nonModulePackages, having decided "+
				"what gates it", entry.Name())
		}
		delete(unseen, entry.Name())
	}
	for name := range unseen {
		t.Errorf("%s is classified but no longer exists; drop it from the table", name)
	}
}
