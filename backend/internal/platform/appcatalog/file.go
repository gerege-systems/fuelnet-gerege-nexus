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
func LoadFile(path string, platformVersion string) ([]CatalogApp, error) {
	// #nosec G304 -- APP_CATALOG_PATH is deployment configuration read once at
	// startup. No request reaches this.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawCatalog []CatalogApp
	if err := json.Unmarshal(data, &rawCatalog); err != nil {
		return nil, err
	}

	catalogDir := filepath.Dir(path)
	catalog := make([]CatalogApp, 0, len(rawCatalog))
	for _, app := range rawCatalog {
		if !security.IsValidSlug(app.Slug) {
			return nil, fmt.Errorf("catalog app %q has an invalid slug %q", app.ID, app.Slug)
		}
		manifestPath := filepath.Join(catalogDir, "manifests", app.Slug+".json")
		manifest, err := LoadManifestFile(manifestPath, platformVersion)
		if err != nil {
			return nil, fmt.Errorf("load manifest for %s: %w", app.ID, err)
		}
		app.Manifest = manifest
		catalog = append(catalog, app)
	}

	if err := ValidateCatalog(catalog, platformVersion); err != nil {
		return nil, err
	}
	return catalog, nil
}

// ValidateCatalog is what a catalogue has to satisfy whatever it arrived on.
//
// The file path validates each manifest as it reads it, so for that source this
// mostly re-checks; for a registry answer, where the manifests arrive inline, it
// is the only check there is. Both go through it so a remote catalogue can never
// be held to a weaker standard than the file one.
func ValidateCatalog(catalog []CatalogApp, platformVersion string) error {
	seenSlugs := make(map[string]string, len(catalog))
	seenIDs := make(map[string]bool, len(catalog))
	for _, app := range catalog {
		if app.ID == "" {
			return fmt.Errorf("catalog entry %q has no id", app.Slug)
		}
		if !security.IsValidSlug(app.Slug) {
			return fmt.Errorf("catalog app %q has an invalid slug %q", app.ID, app.Slug)
		}
		// A slug is a URL path segment in the store API and the key the shell
		// installs by, so two apps claiming one is not a cosmetic conflict:
		// whichever is listed first would answer for both.
		if other, taken := seenSlugs[app.Slug]; taken {
			return fmt.Errorf("catalog apps %s and %s both claim the slug %q", other, app.ID, app.Slug)
		}
		if seenIDs[app.ID] {
			return fmt.Errorf("catalog lists %s twice", app.ID)
		}
		seenSlugs[app.Slug], seenIDs[app.ID] = app.ID, true

		if app.Manifest.ID != app.ID {
			return fmt.Errorf("manifest for %s declares id %q but the catalog entry is %q",
				app.Slug, app.Manifest.ID, app.ID)
		}
		if err := ValidateManifest(app.Manifest, platformVersion); err != nil {
			return fmt.Errorf("validate manifest for %s: %w", app.ID, err)
		}
		if app.Manifest.Version != app.Version {
			return fmt.Errorf("catalog entry %s is version %q but its manifest declares %q",
				app.ID, app.Version, app.Manifest.Version)
		}
	}
	return nil
}
