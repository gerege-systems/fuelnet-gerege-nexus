/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appcatalog

import (
	"path/filepath"

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
	entries, err := catalog.LoadEntries(path)
	if err != nil {
		return nil, err
	}
	// Before the manifest paths are built from the slugs, so a catalogue file
	// written under the old names still finds manifests/organisation.json.
	// This step is the only reason the platform does not simply call
	// catalog.LoadFile: the renames are a deprecation this repository is
	// carrying, and not something every distribution should inherit.
	//
	// DEPRECATED: remove in vNEXT — see alias.go.
	entries = applyRenames(entries)

	return catalog.Assemble(filepath.Dir(path), entries, platformVersion)
}
