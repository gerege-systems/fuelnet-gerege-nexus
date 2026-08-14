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
