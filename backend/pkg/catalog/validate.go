/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package catalog

import "fmt"

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
		if !IsValidSlug(app.Slug) {
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

// IsValidSlug reports whether a slug may name an app.
//
// Lowercase alphanumerics, hyphens and underscores, at most 64. It is a
// path-traversal guard first — a slug becomes a store URL segment and a
// manifest filename — and a naming rule second.
//
// Underscores are permitted because catalogue slugs used them. None of the
// platform's own carries one any more, but a slug is a third party's to choose
// and narrowing this would make an app in somebody's registry uninstallable
// without warning.
func IsValidSlug(slug string) bool {
	if slug == "" || len(slug) > 64 {
		return false
	}
	for _, ch := range slug {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}
