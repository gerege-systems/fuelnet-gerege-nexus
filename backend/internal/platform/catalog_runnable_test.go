package platform

import (
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// The catalogue outlives the split, and the store keeps advertising what left.
//
// This is not a hypothetical. State Services moved to its own repository on
// 2026-08-15 and nexus.gerege.mn went on carrying `io.gerege.nexus.gov_services`
// in its apps table the same afternoon, because that row comes from a signed
// catalogue served to every deployment in the field — not from this
// repository's manifests. Republishing the catalogue is a deliberate act with
// its own consequences, so a deployment has to be able to tell the truth about
// itself in the meantime.
func TestTheStoreDoesNotOfferAnAppThisBinaryCannotRun(t *testing.T) {
	withModules(t, gatedModule{})

	compiled := catalog.CatalogApp{ID: "io.gerege.test.gated"}
	if !runnableHere(compiled) {
		t.Error("an app with a compiled module must be offered")
	}

	// The app that left. Its manifest is still in the catalogue, there is no
	// module behind it, and the installer would refuse with a message about
	// binary registries that means nothing to whoever pressed Install.
	departed := catalog.CatalogApp{ID: "io.gerege.nexus.gov_services"}
	if runnableHere(departed) {
		t.Error("an app with no module in this binary must not be offered")
	}
}

// External apps have no Go module by definition — they are somebody else's
// running service, reached over OIDC. The rule above would hide the entire
// category if it did not except them, which would be a worse bug than the one
// it fixes: third-party apps are the whole point of a public catalogue.
func TestAnExternalAppIsOfferedWithoutAModule(t *testing.T) {
	withModules(t)

	external := catalog.CatalogApp{ID: "com.example.hrms"}
	external.Manifest.Type = "external"
	if !external.Manifest.IsExternal() {
		t.Fatalf("fixture is wrong: the manifest does not read as external")
	}
	if !runnableHere(external) {
		t.Error("an external app must be offered even though nothing compiled it")
	}
}

var _ nexus.Module = gatedModule{}
