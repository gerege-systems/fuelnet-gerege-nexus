/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Reading a manifest off this deployment's disk.
 *
 * The schema, its validation and now the reading itself are in
 * `backend/pkg/catalog`, where a distribution can reach them. What is left here
 * is the name this repository's callers already use.
 */

package appcatalog

import (
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// LoadManifestFile loads and validates a manifest file.
//
// A thin name over catalog.LoadManifest, for the same reason LoadChronicle is
// one: the reading moved to the contract package so a distribution could reach
// it, and the callers in this repository did not have to move with it.
func LoadManifestFile(path string, platformVersion string) (catalog.Manifest, error) {
	return catalog.LoadManifest(path, platformVersion)
}
