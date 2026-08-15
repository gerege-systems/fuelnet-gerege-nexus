/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// LoadFile reads catalog/apps.json and the manifests beside it.
//
// This is the deployment shape every self-hosted instance runs and the last
// fallback for every other one. It used to live in the platform's server
// constructor; it moved here when a second source of catalogues appeared, so
// both are validated by exactly the same code.
//
// A manifest that failed to load used to be replaced by a silent stub with no
// dependencies, permissions or menus. Three shipped manifests were in fact
// malformed (object instead of array for "dependencies", plain strings instead
// of objects for "permissions") and nobody noticed: the apps installed with an
// empty dependency graph and never contributed a menu entry. Catalog integrity
// is a startup error.
func LoadFile(path string, platformVersion string) ([]catalog.CatalogApp, error) {
	// #nosec G304 -- APP_CATALOG_PATH is deployment configuration read once at
	// startup. No request reaches this.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawCatalog []catalog.CatalogApp
	if err := json.Unmarshal(data, &rawCatalog); err != nil {
		return nil, err
	}
	// Before the manifest paths are built from the slugs, so a catalogue file
	// written under the old names still finds manifests/organisation.json.
	//
	// DEPRECATED: remove in vNEXT — see alias.go.
	rawCatalog = applyRenames(rawCatalog)

	catalogDir := filepath.Dir(path)
	apps := make([]catalog.CatalogApp, 0, len(rawCatalog))
	for _, app := range rawCatalog {
		if !security.IsValidSlug(app.Slug) {
			return nil, fmt.Errorf("catalog app %q has an invalid slug %q", app.ID, app.Slug)
		}
		manifestPath := filepath.Join(catalogDir, "manifests", app.Slug+".json")
		manifest, err := LoadManifestFile(manifestPath, platformVersion)
		if err != nil {
			return nil, fmt.Errorf("load manifest for %s: %w", app.ID, err)
		}
		// The chronicle entry for the version being shipped is folded into the
		// manifest here, so nothing downstream has to know the chronicle exists:
		// the store card, the history drawer and the registry all read
		// catalog.Manifest.ReleaseNotes. Written in two places by hand, the two would
		// disagree the first time somebody edited one of them.
		notes, err := releaseNotesFor(catalogDir, app.Slug, manifest.Version)
		if err != nil {
			return nil, fmt.Errorf("load chronicle for %s: %w", app.ID, err)
		}
		if notes != nil {
			manifest.ReleaseNotes = notes
		}
		app.Manifest = manifest
		// One resolved answer for who may be offered this app, so nothing
		// downstream has to read two fields and decide which it believes. The
		// entry and the manifest are written by hand in two files; a private
		// declaration in either is what survives (see CatalogApp.IsPrivate),
		// and both halves are set to it so the API, the apps table and the
		// store card cannot disagree about the same app.
		if app.IsPrivate() {
			app.Visibility, app.Manifest.Visibility = catalog.VisibilityPrivate, catalog.VisibilityPrivate
		} else if app.Visibility == "" {
			app.Visibility = catalog.VisibilityPublic
		}
		apps = append(apps, app)
	}

	if err := catalog.ValidateCatalog(apps, platformVersion); err != nil {
		return nil, err
	}
	return apps, nil
}

// releaseNotesFor reads the chronicle entry for one version, or nil when the
// app keeps no chronicle or has not recorded this version.
//
// Neither absence is an error here. The obligation to write an entry is
// enforced by the CI guard in chronicle_test.go, where it can be stated once
// against the whole catalogue and where failing it stops a release rather than
// a running instance: a deployment whose bundled file predates the chronicle
// must still boot.
func releaseNotesFor(catalogDir, slug, version string) (*catalog.ReleaseNote, error) {
	chronicle, kept, err := LoadChronicle(catalogDir, slug)
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
