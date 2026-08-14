/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Reading an app's chronicle off this deployment's disk. The chronicle itself —
 * what a release note is and what makes one valid — is in `backend/pkg/catalog`.
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

// LoadChronicle reads one app's chronicle from the catalogue directory.
//
// A missing file is not an error and not an empty chronicle either — it is
// "this app keeps no chronicle", which is the state every third-party app
// published through the console is in. The caller distinguishes them by the
// bool, because the CI guard has to tell "no file" from "a file with no entry
// for this version" to say anything useful.
func LoadChronicle(catalogDir, slug string) (catalog.Chronicle, bool, error) {
	if !security.IsValidSlug(slug) {
		return catalog.Chronicle{}, false, fmt.Errorf("chronicle requested for an invalid slug %q", slug)
	}
	path := filepath.Join(catalogDir, "chronicle", slug+".json")
	// #nosec G304 -- the slug is checked above and admits neither a separator
	// nor a dot; this is read at startup from deployment-owned files.
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return catalog.Chronicle{}, false, nil
	}
	if err != nil {
		return catalog.Chronicle{}, false, fmt.Errorf("read chronicle %s: %w", path, err)
	}
	var c catalog.Chronicle
	if err := json.Unmarshal(data, &c); err != nil {
		return catalog.Chronicle{}, false, fmt.Errorf("unmarshal chronicle %s: %w", path, err)
	}
	if err := catalog.ValidateChronicle(c); err != nil {
		return catalog.Chronicle{}, false, fmt.Errorf("validate chronicle %s: %w", path, err)
	}
	return c, true, nil
}
