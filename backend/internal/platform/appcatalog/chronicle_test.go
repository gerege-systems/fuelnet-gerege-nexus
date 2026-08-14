package appcatalog_test

import (
	"path/filepath"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// catalogDir is the bundled catalogue, from this package's directory.
const catalogDir = "../../../../catalog"

func bundledCatalog(t *testing.T) []catalog.CatalogApp {
	t.Helper()
	apps, err := appcatalog.LoadFile(filepath.Join(catalogDir, "apps.json"), "1.0.0")
	if err != nil {
		t.Fatalf("the bundled catalogue does not load: %v", err)
	}
	return apps
}

// This is the guard the chronicle exists for: a version that moved without
// anybody saying why.
//
// It runs against the bundled catalogue rather than against a fixture, so
// adding an app or raising its version is what triggers it — no separate list
// to keep in step. Failing it costs one sentence in
// catalog/chronicle/<slug>.json, which is the whole point: the record is
// written while the change is still in somebody's head, not reconstructed from
// a diff six months later.
func TestEveryCatalogVersionHasAChronicleEntry(t *testing.T) {
	for _, app := range bundledCatalog(t) {
		chronicle, kept, err := appcatalog.LoadChronicle(catalogDir, app.Slug)
		if err != nil {
			t.Errorf("%s: %v", app.ID, err)
			continue
		}
		if !kept {
			t.Errorf("%s (%s) keeps no chronicle: create catalog/chronicle/%s.json with an entry for version %s",
				app.ID, app.Slug, app.Slug, app.Version)
			continue
		}
		if _, found := chronicle.Find(app.Version); !found {
			t.Errorf("%s is at version %s and the chronicle does not record it: add an entry to catalog/chronicle/%s.json",
				app.ID, app.Version, app.Slug)
		}
	}
}

// The entry has to reach the manifest, because that is the only way it reaches
// anything else — the store card, the history drawer and the registry all read
// catalog.Manifest.ReleaseNotes and none of them opens a chronicle file.
func TestTheChronicleEntryTravelsInTheManifest(t *testing.T) {
	for _, app := range bundledCatalog(t) {
		notes := app.Manifest.ReleaseNotes
		if notes == nil {
			t.Errorf("%s carries no release notes into its manifest", app.ID)
			continue
		}
		if notes.Summary["mn"] == "" || notes.Summary["en"] == "" {
			t.Errorf("%s: release notes reached the manifest without both summaries", app.ID)
		}
		// The version is deliberately dropped on the way in: the manifest
		// already declares it, and two copies are two things that can disagree.
		if notes.Version != "" {
			t.Errorf("%s: the embedded note repeats the version %q", app.ID, notes.Version)
		}
	}
}

// Every chronicle file is valid on its own terms, including entries for
// versions the catalogue has moved past. A history that stops being loadable
// once it is no longer the current release is not a history.
func TestEveryChronicleFileIsValid(t *testing.T) {
	for _, app := range bundledCatalog(t) {
		chronicle, kept, err := appcatalog.LoadChronicle(catalogDir, app.Slug)
		if err != nil {
			t.Errorf("%s: %v", app.ID, err)
			continue
		}
		if !kept {
			continue
		}
		if chronicle.AppID != app.ID {
			t.Errorf("chronicle for %s declares app_id %q", app.Slug, chronicle.AppID)
		}
		if err := catalog.ValidateChronicle(chronicle); err != nil {
			t.Errorf("%s: %v", app.ID, err)
		}
	}
}

func TestValidateChronicleRefusesWhatItShould(t *testing.T) {
	good := catalog.ReleaseNote{
		Version: "1.0.0", Kind: catalog.KindFeature,
		Summary: map[string]string{"mn": "Шинэ", "en": "New"},
	}
	note := func(mutate func(*catalog.ReleaseNote)) catalog.Chronicle {
		entry := good
		entry.Summary = map[string]string{"mn": "Шинэ", "en": "New"}
		mutate(&entry)
		return catalog.Chronicle{AppID: "io.example.app", Entries: []catalog.ReleaseNote{entry}}
	}

	for name, c := range map[string]catalog.Chronicle{
		"no app_id":          {Entries: []catalog.ReleaseNote{good}},
		"no version":         note(func(e *catalog.ReleaseNote) { e.Version = "" }),
		"version not semver": note(func(e *catalog.ReleaseNote) { e.Version = "one" }),
		"unknown kind":       note(func(e *catalog.ReleaseNote) { e.Kind = "refactor" }),
		"no mongolian":       note(func(e *catalog.ReleaseNote) { delete(e.Summary, "mn") }),
		"no english":         note(func(e *catalog.ReleaseNote) { delete(e.Summary, "en") }),
		"bad date":           note(func(e *catalog.ReleaseNote) { e.ReleasedAt = "11-08-2026" }),
	} {
		if err := catalog.ValidateChronicle(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	// The same version twice is a history that cannot be read in order.
	twice := catalog.Chronicle{AppID: "io.example.app", Entries: []catalog.ReleaseNote{good, good}}
	if err := catalog.ValidateChronicle(twice); err == nil {
		t.Error("a version recorded twice was accepted")
	}

	if err := catalog.ValidateChronicle(note(func(*catalog.ReleaseNote) {})); err != nil {
		t.Errorf("a well-formed chronicle was refused: %v", err)
	}
}
