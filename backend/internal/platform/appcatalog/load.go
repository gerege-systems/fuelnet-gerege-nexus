/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Reading a manifest off this deployment's disk.
 *
 * The schema and its validation are in `backend/pkg/catalog`; where a file sits
 * is this deployment's business and stays here.
 */

package appcatalog

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// LoadManifestFile loads and validates a manifest file.
func LoadManifestFile(path string, platformVersion string) (catalog.Manifest, error) {
	// #nosec G304 -- the only caller builds this from the catalogue directory
	// and a slug already checked by security.IsValidSlug, which admits neither
	// a separator nor a dot. It is read once at startup, never per request.
	data, err := os.ReadFile(path)
	if err != nil {
		return catalog.Manifest{}, fmt.Errorf("read manifest file %s: %w", path, err)
	}
	var m catalog.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return catalog.Manifest{}, fmt.Errorf("unmarshal manifest JSON: %w", err)
	}
	if err := catalog.ValidateManifest(m, platformVersion); err != nil {
		return catalog.Manifest{}, fmt.Errorf("validate manifest %s: %w", path, err)
	}
	return m, nil
}
