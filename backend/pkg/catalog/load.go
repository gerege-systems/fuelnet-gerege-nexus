/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Reading a catalogue off disk: apps.json, the manifests beside it, and the
// chronicle entry for the version each one ships.
//
// This lives in the contract package rather than in the platform's internals
// because a distribution has a catalogue too, and had no way to read its own.
// The App Store hit it first: a test that wanted to check its bundled
// catalogue against the modules compiled beside it could not, because the
// loader was in `internal/` — the rule that makes distributions possible,
// working exactly as designed and against the product it was designed for. So
// it is here, and the platform's own loader calls it.
//
// One implementation, not two. A second copy of "how a catalogue is read" is a
// second answer to whether a manifest is valid, and the two would agree until
// the day a deployment and the store it publishes to disagreed about an app
// nobody had touched.

package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadFile reads a catalogue directory: `path` is apps.json, and the manifests
// and chronicles are read from `manifests/` and `chronicle/` beside it.
//
// platformVersion is what each manifest's `"platform"` constraint is checked
// against; empty skips that check, which is what a tool that is not a running
// platform wants.
//
// A manifest that fails to load is an error rather than a stub. Three shipped
// manifests were once malformed — an object where an array belonged, strings
// where objects did — and nobody noticed, because the apps installed with an
// empty dependency graph and never contributed a menu. Catalogue integrity is a
// startup error.
func LoadFile(path string, platformVersion string) ([]CatalogApp, error) {
	entries, err := LoadEntries(path)
	if err != nil {
		return nil, err
	}
	return Assemble(filepath.Dir(path), entries, platformVersion)
}

// LoadEntries reads apps.json and nothing else.
//
// Separate from Assemble for one caller: the platform rewrites app ids it has
// renamed before the manifest paths are built from their slugs, so it needs the
// raw entries in its hands between the two steps.
func LoadEntries(path string) ([]CatalogApp, error) {
	// #nosec G304 -- the catalogue path is deployment configuration read at
	// startup or by a build tool. No request reaches this.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []CatalogApp
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Assemble folds each entry's manifest and release note in, resolves what the
// entry and the manifest say about visibility, and validates the result.
func Assemble(catalogDir string, entries []CatalogApp, platformVersion string) ([]CatalogApp, error) {
	apps := make([]CatalogApp, 0, len(entries))
	for _, app := range entries {
		if !IsValidSlug(app.Slug) {
			return nil, fmt.Errorf("catalog app %q has an invalid slug %q", app.ID, app.Slug)
		}
		manifest, err := LoadManifest(filepath.Join(catalogDir, "manifests", app.Slug+".json"), platformVersion)
		if err != nil {
			return nil, fmt.Errorf("load manifest for %s: %w", app.ID, err)
		}
		// The chronicle entry for the version being shipped is folded into the
		// manifest here, so nothing downstream has to know the chronicle
		// exists: the store card, the history drawer and the registry all read
		// Manifest.ReleaseNotes. Written by hand in two places, the two would
		// disagree the first time somebody edited one of them.
		notes, err := ReleaseNotesFor(catalogDir, app.Slug, manifest.Version)
		if err != nil {
			return nil, fmt.Errorf("load chronicle for %s: %w", app.ID, err)
		}
		if notes != nil {
			manifest.ReleaseNotes = notes
		}
		app.Manifest = manifest

		// One resolved answer for who may be offered this app, so nothing
		// downstream has to read two fields and decide which it believes. A
		// private declaration in either half wins (see CatalogApp.IsPrivate),
		// and both are set to it so the API, the apps table and the store card
		// cannot disagree about the same app.
		if app.IsPrivate() {
			app.Visibility, app.Manifest.Visibility = VisibilityPrivate, VisibilityPrivate
		} else if app.Visibility == "" {
			app.Visibility = VisibilityPublic
		}
		apps = append(apps, app)
	}

	if err := ValidateCatalog(apps, platformVersion); err != nil {
		return nil, err
	}
	return apps, nil
}

// LoadManifest reads and validates one manifest file.
func LoadManifest(path string, platformVersion string) (Manifest, error) {
	// #nosec G304 -- built from a slug that has been checked to contain neither
	// a separator nor a dot, under a directory the deployment configured.
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("unmarshal manifest %s: %w", path, err)
	}
	if err := ValidateManifest(manifest, platformVersion); err != nil {
		return Manifest{}, fmt.Errorf("validate manifest %s: %w", path, err)
	}
	return manifest, nil
}

// LoadChronicleFile reads one app's chronicle from a catalogue directory.
//
// A missing file is not an error and not an empty chronicle either — it is
// "this app keeps no chronicle", which is the state every third-party app
// published through the console is in. The bool is what tells them apart, and a
// CI guard needs the difference to say anything useful: "no file" and "a file
// with no entry for this version" ask for different things to be written.
func LoadChronicleFile(catalogDir, slug string) (Chronicle, bool, error) {
	if !IsValidSlug(slug) {
		return Chronicle{}, false, fmt.Errorf("chronicle requested for an invalid slug %q", slug)
	}
	path := filepath.Join(catalogDir, "chronicle", slug+".json")
	// #nosec G304 -- the slug is checked above and admits neither a separator
	// nor a dot; this is read at startup from deployment-owned files.
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Chronicle{}, false, nil
	}
	if err != nil {
		return Chronicle{}, false, fmt.Errorf("read chronicle %s: %w", path, err)
	}
	var c Chronicle
	if err := json.Unmarshal(data, &c); err != nil {
		return Chronicle{}, false, fmt.Errorf("unmarshal chronicle %s: %w", path, err)
	}
	if err := ValidateChronicle(c); err != nil {
		return Chronicle{}, false, fmt.Errorf("validate chronicle %s: %w", path, err)
	}
	return c, true, nil
}

// ReleaseNotesFor is the chronicle entry for one version, or nil when the app
// keeps no chronicle or has not recorded that version.
//
// Neither absence is an error. Whether an entry had to be written is a question
// for a CI guard, where it can be asked once of a whole catalogue and where
// failing it stops a release rather than a running instance: a deployment whose
// bundled file predates the chronicle must still boot.
func ReleaseNotesFor(catalogDir, slug, version string) (*ReleaseNote, error) {
	chronicle, kept, err := LoadChronicleFile(catalogDir, slug)
	if err != nil {
		return nil, err
	}
	if !kept {
		return nil, nil
	}
	entry, found := chronicle.Find(version)
	if !found {
		return nil, nil
	}
	// The version is dropped: the manifest it is about to live in already
	// declares it, and validateProvenance refuses the two disagreeing.
	entry.Version = ""
	return &entry, nil
}
