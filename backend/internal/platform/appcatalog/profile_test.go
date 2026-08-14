package appcatalog_test

import (
	"path/filepath"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// An instance profile is a catalogue file, and nothing else.
//
// Which apps an instance offers is already decided by the catalogue it loads,
// so making one instance an App Store and another a government office needs no
// new concept in the core — no "profile" type, no registry of deployments, no
// flag. It needs a second file. One binary, two catalogues.
//
// catalog/profiles/appstore is that second file: the instance that *is* the
// App Store. It runs core — an organisation, its people, its roles — and the
// three modules that make it a store, and none of the business apps, because a
// registry has no invoices.
//
// This holds it to the same standard as the bundled catalogue. A profile that
// only loaded on the day it was written would be discovered by an operator
// pointing a deployment at it.
func TestTheAppStoreProfileLoadsLikeAnyOtherCatalogue(t *testing.T) {
	const profile = "../../../../catalog/profiles/appstore"

	apps, err := appcatalog.LoadFile(filepath.Join(profile, "apps.json"), "1.0.0")
	if err != nil {
		t.Fatalf("the App Store profile does not load: %v", err)
	}

	want := map[string]bool{
		"io.gerege.nexus.organisation":      true,
		"io.gerege.nexus.egov":              true,
		"io.gerege.nexus.appstore_registry": true,
		"io.gerege.nexus.publisher_studio":  true,
		"io.gerege.nexus.store_review":      true,
	}
	got := map[string]bool{}
	for _, app := range apps {
		got[app.ID] = true

		// Every claim the bundled catalogue makes, this one makes too: a
		// manifest that agrees with its entry, and a chronicle entry for the
		// version being offered — folded into the manifest on the way in.
		if app.Manifest.ID != app.ID {
			t.Errorf("%s: the manifest declares %q", app.ID, app.Manifest.ID)
		}
		if app.Manifest.Version != app.Version {
			t.Errorf("%s: entry is %s and manifest is %s", app.ID, app.Version, app.Manifest.Version)
		}
		notes := app.Manifest.ReleaseNotes
		if notes == nil {
			t.Errorf("%s carries no chronicle entry into its manifest", app.ID)
			continue
		}
		if notes.Summary["mn"] == "" || notes.Summary["en"] == "" {
			t.Errorf("%s: the chronicle entry is missing a source language", app.ID)
		}
	}

	for id := range want {
		if !got[id] {
			t.Errorf("the App Store profile does not carry %s", id)
		}
	}
	// And nothing else. An App Store instance offering the billing module would
	// be an instance somebody installed billing on by accident.
	for id := range got {
		if !want[id] {
			t.Errorf("the App Store profile carries %s, which is not part of a store", id)
		}
	}
}

// The two catalogues describe the same core app, and they have to agree about
// it: an instance loading either one compiles the same module, and a version
// that differed would be caught at boot by verifyCatalogVersions — on whichever
// deployment happened to load the stale one.
func TestBothCataloguesAgreeAboutCore(t *testing.T) {
	bundled, err := appcatalog.LoadFile(filepath.FromSlash("../../../../catalog/apps.json"), "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := appcatalog.LoadFile(
		filepath.FromSlash("../../../../catalog/profiles/appstore/apps.json"), "1.0.0")
	if err != nil {
		t.Fatal(err)
	}

	find := func(apps []catalog.CatalogApp, id string) (catalog.CatalogApp, bool) {
		for _, app := range apps {
			if app.ID == id {
				return app, true
			}
		}
		return catalog.CatalogApp{}, false
	}

	here, ok := find(bundled, "io.gerege.nexus.organisation")
	if !ok {
		t.Fatal("the bundled catalogue does not carry core")
	}
	there, ok := find(profile, "io.gerege.nexus.organisation")
	if !ok {
		t.Fatal("the App Store profile does not carry core")
	}
	if here.Version != there.Version {
		t.Errorf("core is %s in the bundled catalogue and %s in the App Store profile",
			here.Version, there.Version)
	}
}
