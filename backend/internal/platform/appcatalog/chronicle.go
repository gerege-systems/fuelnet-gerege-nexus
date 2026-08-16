/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Reading an app's chronicle off this deployment's disk. The chronicle itself —
 * what a release note is and what makes one valid — is in `backend/pkg/catalog`.
 */

package appcatalog

import (
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// LoadChronicle reads one app's chronicle from the catalogue directory.
//
// A thin name over catalog.LoadChronicleFile, kept because the CI guard and the
// history drawer in this repository both call it and because the package that
// reads catalogues on disk is where somebody looks for it. The reading itself
// is in the contract package, where a distribution can reach it too.
func LoadChronicle(catalogDir, slug string) (catalog.Chronicle, bool, error) {
	return catalog.LoadChronicleFile(catalogDir, slug)
}
