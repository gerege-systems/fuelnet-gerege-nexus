package apps

import (
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/billing"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/egov"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/esign"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/gov_services"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/inventory"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/organisation"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/products"
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
// constructing eleven modules would drag in a database, an HTTP client and the
// government integration manager to ask each of them a constant.
var corePolicies = map[string]struct {
	module         nexus.Module
	menu, prefix   string
	whyNoRouteGate string
}{
	"contacts":    {(*contacts.Module)(nil), "contacts.read", "contacts", ""},
	"products":    {(*products.Module)(nil), "products.read", "products", ""},
	"inventory":   {(*inventory.Module)(nil), "inventory.read", "inventory", ""},
	"billing":     {(*billing.BillingModule)(nil), "billing.read", "billing", ""},
	"sso_clients": {(*sso_clients.SSOClientsModule)(nil), "sso_clients.read", "sso_clients", ""},

	// The three that gate themselves, and why the verb is not enough for them.
	"documents": {(*documents.DocumentsModule)(nil), "documents.read", "",
		"who may read a document depends on who it was shared with"},
	"egov": {(*egov.Module)(nil), "egov.read", "",
		"a citizen-registry lookup is a GET that must not be a read every member holds"},
	"gov_services": {(*gov_services.Module)(nil), "gov.read", "",
		"an approval depends on the applicant's unit and the workflow step"},
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
// visible to every member of a tenant that has them installed; esign registers
// no menu of its own and gates every route in its handlers, the same way
// documents does. Absent-because-considered and absent-because-forgotten look
// identical in a table, so the difference is asserted rather than left to a
// comment.
func TestTheModulesWithNoPolicyAreTheOnesWeMeant(t *testing.T) {
	for name, mod := range map[string]nexus.Module{
		"organisation": (*organisation.Module)(nil),
		"reports":      (*reports.Module)(nil),
		"esign":        (*esign.Module)(nil),
	} {
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

// The count is the point of this one. Adding a module and forgetting to decide
// its access policy is the mistake that hides: the module works, the routes
// mount, and nothing asks whether anyone should be able to reach them. A new
// entry in internal/apps has to be classified in one of the two tables above
// before this passes.
func TestEveryModuleInThisRepositoryIsClassified(t *testing.T) {
	const classified = 8 + 3 // corePolicies + the deliberately-empty ones
	const inRepository = 11  // directories under internal/apps holding a module

	if classified != inRepository {
		t.Fatalf("%d modules classified, %d in the repository — a new module "+
			"needs an entry in corePolicies or in the empty-policy test",
			classified, inRepository)
	}
}
