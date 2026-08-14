/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package catalog

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
)

// The chronicle — «Шастир» — is the record of what changed in an app, kept
// beside the code that changed it.
//
// The problem it solves is that a version number moves and nothing says why.
// `app_versions.manifest` on this side and `store_app_versions.manifest` on the
// registry's have both been recording every published version for a while, so
// the archive already existed; what was missing was the sentence a person could
// read, and any obligation to write one.
//
// It travels inside the manifest rather than on a channel of its own. A
// manifest is already signed, already cached, already replicated to every
// instance — a second transport for the same trip would be a second thing to
// keep in step. So the entry for the version being published is copied into
// Manifest.ReleaseNotes as the catalogue is assembled, and everything
// downstream — the store card, the history drawer, the storefront — reads it
// from there without knowing this file exists.
//
// The source of truth is the repository, because that is where the change is
// made. A third party publishing through the console types its notes into the
// submission form instead, and they land in the same manifest field.

// ChronicleKind classifies a change. It is a closed set: the store card colours
// by it and a reader scanning a list is looking for "did anything break".
const (
	KindFeature  = "feature"
	KindFix      = "fix"
	KindSecurity = "security"
	KindBreaking = "breaking"
	KindDocs     = "docs"
)

// chronicleKinds is the set ValidateChronicle admits.
var chronicleKinds = map[string]bool{
	KindFeature: true, KindFix: true, KindSecurity: true,
	KindBreaking: true, KindDocs: true,
}

// ReleaseNote is one version's entry: what changed, in the languages the
// platform speaks, and who is answerable for it.
//
// Summary is required and short — it is what fits on the update button's line.
// Details is optional and is where the reasoning goes, for the reader who
// clicked through because the summary was not enough.
type ReleaseNote struct {
	// Version is the release this note describes. It is absent from the copy
	// embedded in a manifest, where the manifest's own version already says it.
	Version string `json:"version,omitempty"`
	// ReleasedAt is a plain date (2006-01-02). A change belongs to a day, not to
	// a moment: the day is what an operator matches against their own records,
	// and a timestamp would invite a precision the publishing flow does not have.
	ReleasedAt string `json:"released_at,omitempty"`
	Kind       string `json:"kind"`
	// Summary and Details are keyed by ISO 639-1 code, the same shape as
	// CatalogAppText. mn and en are required in Summary; the other five
	// platform languages are welcome and never demanded — the same policy the
	// rest of the catalogue's text follows.
	Summary map[string]string `json:"summary"`
	Details map[string]string `json:"details,omitempty"`
	Authors []string          `json:"authors,omitempty"`
	// Refs are issue or merge-request identifiers, free-form because the two
	// repositories that publish here use different trackers.
	Refs []string `json:"refs,omitempty"`
}

// Chronicle is one app's whole history, newest first once loaded.
type Chronicle struct {
	AppID   string        `json:"app_id"`
	Entries []ReleaseNote `json:"entries"`
}

// Person is a named human attached to an app: who wrote it, who keeps it.
//
// GeregeSub is the subject claim from this platform's own OIDC issuer, present
// when the person has an account here. It is what lets a storefront say "this
// is the same publisher you already know" rather than matching on an e-mail
// address somebody could put in a manifest.
type Person struct {
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	GeregeSub string `json:"gerege_sub,omitempty"`
}

// Find returns the entry for a version.
func (c Chronicle) Find(version string) (ReleaseNote, bool) {
	for _, entry := range c.Entries {
		if entry.Version == version {
			return entry, true
		}
	}
	return ReleaseNote{}, false
}

// ValidateChronicle holds a chronicle to the rules the CI guard enforces.
//
// It is deliberately strict about very little. A version that parses, a kind
// from the closed set, a summary in both source languages, and no version
// claimed twice — everything else is the author's business. The obligation this
// subsystem creates is to write one sentence, and a validator that demanded
// more would be an argument for not writing anything.
func ValidateChronicle(c Chronicle) error {
	if c.AppID == "" {
		return fmt.Errorf("chronicle has no app_id")
	}
	seen := make(map[string]bool, len(c.Entries))
	for i, entry := range c.Entries {
		where := fmt.Sprintf("%s entry %d", c.AppID, i)
		if entry.Version == "" {
			return fmt.Errorf("%s has no version", where)
		}
		if _, err := semver.NewVersion(entry.Version); err != nil {
			return fmt.Errorf("%s has a version that is not semver: %w", where, err)
		}
		if seen[entry.Version] {
			return fmt.Errorf("%s: version %s is recorded twice", c.AppID, entry.Version)
		}
		seen[entry.Version] = true

		if entry.ReleasedAt != "" {
			if _, err := time.Parse(time.DateOnly, entry.ReleasedAt); err != nil {
				return fmt.Errorf("%s: released_at %q is not a %s date",
					where, entry.ReleasedAt, time.DateOnly)
			}
		}
		if err := validateNote(where, entry); err != nil {
			return err
		}
	}
	return nil
}

// validateNote is the part shared with a note that arrived inside a manifest,
// where there is no version of its own and no surrounding chronicle.
func validateNote(where string, note ReleaseNote) error {
	if !chronicleKinds[note.Kind] {
		return fmt.Errorf("%s has an unknown kind %q (expected one of feature, fix, security, breaking, docs)",
			where, note.Kind)
	}
	// mn is the platform's source language and en is what every other
	// translation is made from, so a note in neither is a note nobody can
	// route. The remaining five are optional exactly as they are elsewhere.
	for _, lang := range [...]string{"mn", "en"} {
		if note.Summary[lang] == "" {
			return fmt.Errorf("%s needs a summary in %q", where, lang)
		}
	}
	return nil
}
