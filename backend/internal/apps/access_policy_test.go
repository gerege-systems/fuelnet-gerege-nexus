package apps

import (
	"os"
	"testing"

	appai "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/ai"
	appintegrations "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/integrations"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	appstaffpin "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/staffpin"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/urtuu"
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
	"sso_clients": {(*sso_clients.SSOClientsModule)(nil), "sso_clients.read", "sso_clients", ""},

	// documents was the other one that gates itself — who may read a document
	// depends on who it was shared with — and it is in client-gerege-nexus now.
	// Its claim went with it: the assertion belongs beside the module, not in
	// the repository the module used to be in.
	//
	// Өртөө gates itself for the same shape of reason, but a sharper one:
	// accepting a task and sending a task are both POSTs and are different
	// authorities held by different people — urtuu.process answers
	// for work this organisation has been given, urtuu.manage commits somebody
	// else's time. A prefix rule keyed on the verb would collapse them into one.
	"urtuu": {(*urtuu.Module)(nil), "urtuu.read", "",
		"accepting work and commissioning work are both POSTs and are not the same authority"},
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

// The modules that deliberately declare nothing. reports is visible to every
// member of a tenant that has it installed.
// Absent-because-considered and absent-because-forgotten look identical in a
// table, so the difference is asserted rather than left to a comment.
//
// organisation was the other one and left with documents.
//
// ai is the second: every one of its routes names its own permission —
// ai.read to ask, ai.manage to write the prompt every member of the
// organisation then talks to — and it has no menu to gate, because the shell
// reaches the assistant from the chat affordance rather than from the sidebar.
// A prefix rule would have to collapse those two into one.
var policylessModules = map[string]nexus.Module{
	"reports": (*reports.Module)(nil),
	"ai":      (*appai.Module)(nil),
	// integrations is the third, and its single permission is administrative:
	// there is nothing to read that is not also the power to change it, so a
	// menu gate and a route prefix would both name integrations.manage and say
	// nothing the routes do not already say for themselves.
	"integrations": (*appintegrations.Module)(nil),
	// staffpin is the fourth. Its one route is administrative and names its own
	// permission; the route it exists for is the platform's device sign-in,
	// which no module gates because no module answers it.
	"staffpin": (*appstaffpin.Module)(nil),
}

// Directories under internal/apps that hold no module at all.
//
// There is one entry's worth of history and no entries. esign was here: the PDF
// rails of the documents app, registering nothing with nexus, listed rather than
// deleted because the directory was still under internal/apps and the count
// below reads directories. It is internal/platform/esign now — a package that
// answers none of nexus.Module's methods does not belong in the tree of things
// that do — so the classification has nothing to say about it.
//
// The map stays for the next package that ends up in the same position, because
// absent-because-considered and absent-because-forgotten look identical in a
// table.
var nonModulePackages = map[string]string{}

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
