package appcatalog_test

import (
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

func TestValidateManifest(t *testing.T) {
	validManifest := catalog.Manifest{
		ID:       "io.gerege.nexus.test",
		Name:     "Test App",
		Version:  "1.0.0",
		Platform: ">=0.1.0 <2.0.0",
	}

	t.Run("Valid manifest passes", func(t *testing.T) {
		err := catalog.ValidateManifest(validManifest, "1.0.0")
		if err != nil {
			t.Fatalf("expected valid manifest to pass, got: %v", err)
		}
	})

	t.Run("Invalid semver fails", func(t *testing.T) {
		invalid := validManifest
		invalid.Version = "invalid-semver"
		err := catalog.ValidateManifest(invalid, "1.0.0")
		if err == nil {
			t.Fatal("expected invalid semver to fail validation")
		}
	})

	t.Run("Incompatible platform constraint fails", func(t *testing.T) {
		incompatible := validManifest
		incompatible.Platform = ">=2.0.0"
		err := catalog.ValidateManifest(incompatible, "1.0.0")
		if err == nil {
			t.Fatal("expected platform constraint incompatibility to fail validation")
		}
	})

	// Visibility decides which platforms are offered the app at all, so an
	// unrecognised value is refused rather than read as one of the two. Read as
	// public it would publish an app on a typo; read as private it would hide
	// one for a reason nobody could see. Both are only noticed by the person
	// they should not have reached.
	t.Run("Visibility is public, private, or absent", func(t *testing.T) {
		for _, ok := range []string{"", catalog.VisibilityPublic, catalog.VisibilityPrivate} {
			m := validManifest
			m.Visibility = ok
			if err := catalog.ValidateManifest(m, "1.0.0"); err != nil {
				t.Errorf("visibility %q should be accepted: %v", ok, err)
			}
		}
		for _, bad := range []string{"Private", "internal", "restricted", "PUBLIC", "hidden"} {
			m := validManifest
			m.Visibility = bad
			if err := catalog.ValidateManifest(m, "1.0.0"); err == nil {
				t.Errorf("visibility %q should be refused, not guessed at", bad)
			}
		}
	})

	// An unset visibility is public, and the two halves of a catalogue entry
	// are read together: a private declaration in either wins, because an app
	// hidden by mistake is a support question and an app published by mistake
	// is not recallable.
	t.Run("Private is read from either half of a catalogue entry", func(t *testing.T) {
		cases := []struct {
			name          string
			entry, mnfest string
			private       bool
		}{
			{"both silent", "", "", false},
			{"both public", catalog.VisibilityPublic, catalog.VisibilityPublic, false},
			{"manifest private", "", catalog.VisibilityPrivate, true},
			{"entry private", catalog.VisibilityPrivate, "", true},
			{"entry private, manifest public", catalog.VisibilityPrivate, catalog.VisibilityPublic, true},
			{"both private", catalog.VisibilityPrivate, catalog.VisibilityPrivate, true},
		}
		for _, c := range cases {
			app := catalog.CatalogApp{Visibility: c.entry, Manifest: catalog.Manifest{Visibility: c.mnfest}}
			if got := app.IsPrivate(); got != c.private {
				t.Errorf("%s: IsPrivate() = %v, want %v", c.name, got, c.private)
			}
		}
	})
}

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		installed string
		want      bool
	}{
		// The case string comparison gets wrong, and the reason this is semver.
		{name: "a tenth minor is newer than a ninth", candidate: "1.10.0", installed: "1.9.0", want: true},
		{name: "the same version is not an update", candidate: "1.0.0", installed: "1.0.0", want: false},
		{name: "an older catalog is not an update", candidate: "1.0.0", installed: "1.1.0", want: false},
		{name: "a prerelease loses to its release", candidate: "2.0.0-rc.1", installed: "2.0.0", want: false},
		// A catalogue that reached this instance some other way than manifest
		// validation may carry anything; different is then the best answer left.
		{name: "unparseable versions fall back to difference", candidate: "2026.8", installed: "2026.7", want: true},
		{name: "no candidate is never an update", candidate: "", installed: "1.0.0", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := catalog.IsNewerVersion(tc.candidate, tc.installed); got != tc.want {
				t.Fatalf("catalog.IsNewerVersion(%q, %q) = %v, want %v", tc.candidate, tc.installed, got, tc.want)
			}
		})
	}
}
