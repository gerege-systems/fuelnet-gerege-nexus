/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * publish-catalog submits this build's manifests to the App Store registry.
 */
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// What this is for.
//
// A first-party module's version is decided by a merge. Asking somebody to
// retype it into the publishing console afterwards is how a catalogue falls
// behind the code it describes — and the drift is silent, because a stale
// catalogue looks exactly like a current one from every screen.
//
// So the release pipeline submits, carrying the manifests and the chronicle
// entries that are already in this repository. The registry puts them
// in_review like every other submission; publishing stays a human decision.
//
// It submits rather than publishes on purpose. A pipeline that could publish
// its own submissions would be a review queue nobody reads.
//
// Usage:
//
//	publish-catalog -registry https://appstore.gerege.mn -token "$APPSTORE_PUBLISH_TOKEN"
//	publish-catalog -dry-run          # print what would be sent, send nothing
//	publish-catalog -only esign,core  # a subset, by slug
func main() {
	var (
		registry = flag.String("registry", envOr("APP_CATALOG_REGISTRY", "https://appstore.gerege.mn"),
			"registry base URL")
		token = flag.String("token", os.Getenv("APPSTORE_PUBLISH_TOKEN"),
			"the submit-only publish token")
		catalogPath = flag.String("catalog", envOr("APP_CATALOG_PATH", "catalog/apps.json"),
			"path to the bundled catalogue")
		only    = flag.String("only", "", "comma-separated slugs; default is every app in the catalogue")
		channel = flag.String("channel", "stable", `"stable" or "beta"`)
		dryRun  = flag.Bool("dry-run", false, "print what would be submitted and exit")
		timeout = flag.Duration("timeout", 30*time.Second, "per-request timeout")
	)
	flag.Parse()

	// The catalogue is loaded through the same code the server uses, so a
	// manifest this would submit is one an instance would accept — including
	// the chronicle entry, which LoadFile folds into the manifest. A pipeline
	// that assembled its own payload could submit something the platform
	// itself would reject on arrival.
	apps, err := appcatalog.LoadFile(*catalogPath, platform.PlatformVersion)
	if err != nil {
		fail("could not load %s: %v", *catalogPath, err)
	}

	wanted := map[string]bool{}
	for _, slug := range strings.Split(*only, ",") {
		if slug = strings.TrimSpace(slug); slug != "" {
			wanted[slug] = true
		}
	}

	if !*dryRun && strings.TrimSpace(*token) == "" {
		fail("no publish token: set APPSTORE_PUBLISH_TOKEN or pass -token (or use -dry-run)")
	}

	client := &http.Client{Timeout: *timeout}
	submitted, skipped, already := 0, 0, 0
	for _, app := range apps {
		if len(wanted) > 0 && !wanted[app.Slug] {
			continue
		}
		// Only what this repository publishes. A catalogue can carry a third
		// party's app — the example external one does — and submitting it under
		// the platform's own publisher would be claiming somebody else's work.
		if app.Manifest.Publisher != "gerege" {
			skipped++
			continue
		}

		body, err := json.Marshal(map[string]any{"channel": *channel, "manifest": app.Manifest})
		if err != nil {
			fail("%s: could not encode the manifest: %v", app.ID, err)
		}
		if *dryRun {
			fmt.Printf("would submit %s %s (%s)\n", app.ID, app.Version, *channel)
			submitted++
			continue
		}

		status, reply, err := submit(client, *registry, app.Slug, *token, body)
		switch {
		case err != nil:
			fail("%s: %v", app.ID, err)
		case status == http.StatusCreated:
			fmt.Printf("submitted %s %s\n", app.ID, app.Version)
			submitted++
		case status == http.StatusOK:
			// The registry answers 200 for a version it already holds. A re-run
			// of a pipeline is not a failure.
			fmt.Printf("already submitted %s %s\n", app.ID, app.Version)
			already++
		default:
			fail("%s: the registry answered %d: %s", app.ID, status, strings.TrimSpace(reply))
		}
	}

	fmt.Printf("\n%d submitted, %d already there, %d not ours\n", submitted, already, skipped)
	if submitted == 0 && already == 0 {
		// Nothing reached the registry. Silence here would be a green pipeline
		// that published nothing, which is the failure this command exists to
		// prevent in the first place.
		fail("no manifest was submitted; check -only and the catalogue's publisher fields")
	}
}

func submit(client *http.Client, registry, slug, token string, body []byte) (int, string, error) {
	endpoint := strings.TrimSuffix(registry, "/") + "/api/v1/ci/apps/" + slug + "/versions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = res.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
	return res.StatusCode, string(reply), nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

// fail prints to stderr and exits non-zero, so a pipeline stops rather than
// reporting success over a catalogue that never left the runner.
func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "publish-catalog: "+format+"\n", args...)
	os.Exit(1)
}
