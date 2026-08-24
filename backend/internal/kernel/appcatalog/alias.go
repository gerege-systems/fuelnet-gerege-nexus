/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Applying the renames as the catalogue is parsed.
 *
 * The names themselves are in internal/kernel/appid, because a tenant
 * installing an app by the name it used to have asks the same question from
 * the other plane. This half — rewriting a parsed catalogue before anything
 * downstream compares it against the compiled modules — is the catalogue's own
 * and stays here.
 *
 * DEPRECATED: remove in vNEXT, when every registry has republished.
 */

package appcatalog

import (
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/appid"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// applyRenames rewrites a catalogue in place of the names it was published
// under.
//
// Every id in the document is rewritten, not just the app's own: a dependency
// naming the old id would resolve to nothing, and the copy inside the manifest
// is what an upgrade compares an installation against.
func applyRenames(apps []catalog.CatalogApp) []catalog.CatalogApp {
	for i := range apps {
		apps[i].ID = appid.ResolveAppID(apps[i].ID)
		apps[i].Slug = appid.ResolveAppSlug(apps[i].Slug)
		apps[i].Manifest.ID = appid.ResolveAppID(apps[i].Manifest.ID)
		for d := range apps[i].Manifest.Dependencies {
			apps[i].Manifest.Dependencies[d].ID = appid.ResolveAppID(apps[i].Manifest.Dependencies[d].ID)
		}
	}
	return apps
}
