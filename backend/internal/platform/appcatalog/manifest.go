package appcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// App types. An app is a Go module compiled into this binary unless it says
// otherwise, which is why the empty string means TypeModule: every manifest
// written before external apps existed is a module manifest.
const (
	TypeModule   = "module"
	TypeExternal = "external"
)

type Manifest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// Type is "module" (default) or "external". An external app is a service
	// that runs somewhere else entirely: the platform holds its registration,
	// its permissions and its menu entry, and hands its users over by OIDC.
	// Nothing of it is compiled in.
	Type         string                       `json:"type,omitempty"`
	External     *ExternalSpec                `json:"external,omitempty"`
	Platform     string                       `json:"platform"`
	Dependencies []nexus.Dependency           `json:"dependencies"`
	Permissions  []nexus.PermissionDefinition `json:"permissions"`
	Menus        []nexus.MenuDefinition       `json:"menus"`

	// Everything below is manifest v2.1: who stands behind this app and what
	// changed in this version of it. Every field is optional and every one is
	// omitempty, so a manifest written before any of this existed marshals to
	// exactly the bytes it did before — which is what keeps the signed
	// catalogue byte-reproducible across the two repositories that build it.
	Publisher   string   `json:"publisher,omitempty"`
	Authors     []Person `json:"authors,omitempty"`
	Maintainers []Person `json:"maintainers,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	// License is an SPDX identifier. It is shape-checked rather than looked up:
	// a closed list would have to be maintained here and would reject a licence
	// that is perfectly real and merely newer than this build.
	License string `json:"license,omitempty"`
	// ReleaseNotes is this version's chronicle entry, copied in as the
	// catalogue is assembled. It is not authored here — see chronicle.go.
	ReleaseNotes *ReleaseNote `json:"release_notes,omitempty"`
}

// ExternalSpec describes how to reach a third-party platform and how it signs
// its users in.
type ExternalSpec struct {
	// LaunchURL is where a user is sent. It is the third party's own entry
	// point, typically the one that starts the OIDC dance back at this issuer.
	LaunchURL string `json:"launch_url"`
	// SSOClientID ties the app to an OAuth2 client registered here. It is what
	// makes "has this tenant installed the app" answerable at the authorization
	// endpoint — see the install gate in ssoprovider.
	SSOClientID string   `json:"sso_client_id,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	// Embed is "new_tab" (default) or "iframe". new_tab is the default because
	// this platform sends X-Frame-Options: DENY and a Content-Security-Policy
	// to match: framing somebody else's product inside this one is a decision
	// both sides have to make, not a default.
	Embed string `json:"embed,omitempty"`
	// HealthURL is an address an operator can check. Nothing polls it yet.
	HealthURL string `json:"health_url,omitempty"`
}

// IsExternal reports whether this app runs outside the platform binary.
func (m Manifest) IsExternal() bool { return m.Type == TypeExternal }

type CatalogApp struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IconURL     string   `json:"icon_url"`
	Category    string   `json:"category"`
	Visibility  string   `json:"visibility"`
	Version     string   `json:"version"`
	Manifest    Manifest `json:"manifest"`

	// Translations holds per-locale overrides keyed by ISO 639-1 code. The
	// store API resolves them before responding, so clients never have to
	// translate catalog content themselves.
	Translations map[string]CatalogAppText `json:"translations,omitempty"`
}

// CatalogAppText is the translatable part of a catalog entry.
type CatalogAppText struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

// Localized returns a copy with any translation for the locale applied. Fields
// missing from the translation keep their default value.
func (a CatalogApp) Localized(locale string) CatalogApp {
	text, ok := a.Translations[locale]
	if !ok {
		return a
	}
	if text.Name != "" {
		a.Name = text.Name
	}
	if text.Description != "" {
		a.Description = text.Description
	}
	if text.Category != "" {
		a.Category = text.Category
	}
	return a
}

// ValidateManifest validates semver and manifest rules.
func ValidateManifest(m Manifest, platformVersion string) error {
	if m.ID == "" || m.Name == "" || m.Version == "" {
		return fmt.Errorf("invalid manifest: id, name, and version are required")
	}

	_, err := semver.NewVersion(m.Version)
	if err != nil {
		return fmt.Errorf("invalid app version semver %q: %w", m.Version, err)
	}

	if m.Platform != "" && platformVersion != "" {
		constraint, err := semver.NewConstraint(m.Platform)
		if err != nil {
			return fmt.Errorf("invalid platform constraint %q: %w", m.Platform, err)
		}
		platVer, err := semver.NewVersion(platformVersion)
		if err != nil {
			return fmt.Errorf("invalid platform version %q: %w", platformVersion, err)
		}
		if !constraint.Check(platVer) {
			return fmt.Errorf("app %s version %s requires platform %s, current is %s", m.ID, m.Version, m.Platform, platformVersion)
		}
	}

	switch m.Type {
	case "", TypeModule:
		if m.External != nil {
			return fmt.Errorf("app %s is a module but carries an external section", m.ID)
		}
	case TypeExternal:
		if m.External == nil || m.External.LaunchURL == "" {
			return fmt.Errorf("external app %s must declare external.launch_url", m.ID)
		}
		// Absolute and HTTPS, checked here rather than at the point of use.
		// This URL is put in front of a user as a link this platform vouches
		// for: a relative one would resolve against the platform's own origin
		// and a plain-HTTP one would carry a signed-in person out of TLS on the
		// way to a service that is about to receive their identity.
		launch, err := url.Parse(m.External.LaunchURL)
		if err != nil {
			return fmt.Errorf("external app %s has an unparseable launch_url: %w", m.ID, err)
		}
		if launch.Scheme != "https" || launch.Host == "" {
			return fmt.Errorf("external app %s must have an absolute https launch_url, got %q",
				m.ID, m.External.LaunchURL)
		}
		if embed := m.External.Embed; embed != "" && embed != "new_tab" && embed != "iframe" {
			return fmt.Errorf("external app %s has an unknown embed mode %q", m.ID, embed)
		}
	default:
		return fmt.Errorf("app %s has an unknown type %q", m.ID, m.Type)
	}

	if err := validateProvenance(m); err != nil {
		return err
	}

	for _, dep := range m.Dependencies {
		if dep.ID == "" {
			return fmt.Errorf("dependency ID cannot be empty in app %s", m.ID)
		}
		if dep.VersionConstraint != "" {
			if _, err := semver.NewConstraint(dep.VersionConstraint); err != nil {
				return fmt.Errorf("invalid dependency constraint %q for dep %s in app %s: %w", dep.VersionConstraint, dep.ID, m.ID, err)
			}
		}
	}
	return nil
}

// spdxLicense is the shape of an SPDX identifier, not a list of them: letters,
// digits, dots and dashes, optionally joined by WITH/OR/AND. It catches a
// sentence typed into the field and lets "Apache-2.0", "MIT" and
// "GPL-3.0-or-later WITH Classpath-exception-2.0" through alike.
var spdxLicense = regexp.MustCompile(`^[A-Za-z0-9.+-]+( (WITH|OR|AND) [A-Za-z0-9.+-]+)*$`)

// validateProvenance checks the manifest v2.1 fields. Each is optional; each
// that is present has to mean something.
//
// The rule worth stating is the one about people: an entry with no name is not
// a weaker claim about who wrote the app, it is an empty line rendered in a
// storefront credit list. An app may name nobody. It may not name a blank.
func validateProvenance(m Manifest) error {
	for _, group := range [...]struct {
		field  string
		people []Person
	}{{"authors", m.Authors}, {"maintainers", m.Maintainers}} {
		for i, person := range group.people {
			if strings.TrimSpace(person.Name) == "" {
				return fmt.Errorf("app %s: %s[%d] has no name", m.ID, group.field, i)
			}
		}
	}

	for _, link := range [...]struct {
		field string
		raw   string
	}{{"repository", m.Repository}, {"homepage", m.Homepage}} {
		if link.raw == "" {
			continue
		}
		// Same reasoning as external.launch_url: these are put in front of a
		// user as links this platform vouches for.
		parsed, err := url.Parse(link.raw)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("app %s: %s must be an absolute https URL, got %q",
				m.ID, link.field, link.raw)
		}
	}

	if m.License != "" && !spdxLicense.MatchString(m.License) {
		return fmt.Errorf("app %s: license %q is not an SPDX identifier", m.ID, m.License)
	}

	if m.ReleaseNotes != nil {
		if err := validateNote("app "+m.ID+" release_notes", *m.ReleaseNotes); err != nil {
			return err
		}
		// The note travels inside a manifest that already carries the version,
		// so a second copy is a second thing that can disagree with it.
		if v := m.ReleaseNotes.Version; v != "" && v != m.Version {
			return fmt.Errorf("app %s: release_notes are for version %q but the manifest declares %q",
				m.ID, v, m.Version)
		}
	}
	return nil
}

// IsNewerVersion reports whether candidate is a later release than installed.
//
// It is the one place the store decides that an update exists, so both the API
// that offers the button and the installer that refuses a pointless upgrade
// answer the same question the same way. Semver, not string comparison: "1.10.0"
// sorts before "1.9.0" as text and after it as a version.
//
// A version either side cannot parse falls back to "different means newer".
// Manifest versions are validated as semver on the way in, so this only covers
// a catalogue that reached this instance by some other route.
func IsNewerVersion(candidate, installed string) bool {
	if candidate == "" {
		return false
	}
	newer, candidateErr := semver.NewVersion(candidate)
	held, installedErr := semver.NewVersion(installed)
	if candidateErr != nil || installedErr != nil {
		return candidate != installed
	}
	return newer.GreaterThan(held)
}

// LoadManifestFile loads and validates a manifest file.
func LoadManifestFile(path string, platformVersion string) (Manifest, error) {
	// #nosec G304 -- the only caller builds this from the catalogue directory
	// and a slug already checked by security.IsValidSlug, which admits neither
	// a separator nor a dot. It is read once at startup, never per request.
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest file %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("unmarshal manifest JSON: %w", err)
	}
	if err := ValidateManifest(m, platformVersion); err != nil {
		return Manifest{}, fmt.Errorf("validate manifest %s: %w", path, err)
	}
	return m, nil
}

// Interfaces for future marketplace / remote registry boundary:
type CatalogRepository interface {
	GetAppBySlug(ctx context.Context, slug string) (CatalogApp, error)
	ListApps(ctx context.Context) ([]CatalogApp, error)
}

type PackageStorage interface {
	FetchPackage(ctx context.Context, packageURL string) ([]byte, error)
}

type PackageVerifier interface {
	VerifyChecksum(data []byte, expectedSHA256 string) error
	VerifySignature(data []byte, signature string) error
}

type Installer interface {
	InstallApp(ctx context.Context, tenantID, appSlug string) error
	DisableApp(ctx context.Context, tenantID, appSlug string) error
	EnableApp(ctx context.Context, tenantID, appSlug string) error
}
