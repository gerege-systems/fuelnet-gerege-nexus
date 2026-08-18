/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package ssoclients

import (
	"net/url"
	"slices"
	"strings"
)

// maxNameLength is what fits on a consent screen and in a log line. A name
// longer than this is a paste, not a name.
const maxNameLength = 200

type Service struct {
	vocabulary  Vocabulary
	identifiers Identifiers
}

func NewService(vocabulary Vocabulary, identifiers Identifiers) *Service {
	return &Service{vocabulary: vocabulary, identifiers: identifiers}
}

// Normalise validates and cleans a registration.
//
// It answers with the value that should be stored rather than editing the one
// it was given: half-normalised input reaching a database is how a redirect
// list ends up holding an address nobody registered.
func (s *Service) Normalise(request Registration) (Registration, error) {
	out := Registration{
		ClientName: strings.TrimSpace(request.ClientName),
		ClientURI:  strings.TrimSpace(request.ClientURI),
		LogoURI:    strings.TrimSpace(request.LogoURI),
		ClientType: strings.TrimSpace(request.ClientType),
		Disabled:   request.Disabled,
	}

	if out.ClientName == "" {
		return Registration{}, ErrNameRequired
	}
	if len(out.ClientName) > maxNameLength {
		return Registration{}, ErrNameTooLong
	}

	if out.ClientType == "" {
		out.ClientType = TypeConfidential
	}
	if out.ClientType != TypeConfidential && out.ClientType != TypePublic {
		return Registration{}, ErrClientType
	}

	if out.GrantTypes = dedupe(request.GrantTypes); len(out.GrantTypes) == 0 {
		out.GrantTypes = slices.Clone(DefaultGrantTypes)
	}
	for _, grant := range out.GrantTypes {
		if !slices.Contains(s.vocabulary.SupportedGrantTypes(), grant) {
			return Registration{}, unsupportedGrant(grant)
		}
		if grant == "client_credentials" && out.IsPublic() {
			return Registration{}, ErrPublicCredentials
		}
	}

	if out.Scopes = dedupe(request.Scopes); len(out.Scopes) == 0 {
		out.Scopes = slices.Clone(DefaultScopes)
	}
	for _, scope := range out.Scopes {
		if !s.vocabulary.IsSupportedScope(scope) {
			return Registration{}, unknownScope(scope)
		}
	}

	out.RedirectURIs = dedupe(request.RedirectURIs)
	if slices.Contains(out.GrantTypes, "authorization_code") && len(out.RedirectURIs) == 0 {
		return Registration{}, ErrNoRedirect
	}
	for _, raw := range out.RedirectURIs {
		if err := s.checkRedirect(raw, out.ClientType); err != nil {
			return Registration{}, err
		}
	}
	// The same rules: a post-logout address is matched exactly by the logout
	// endpoint, so a wildcard or a fragment registered here would be a target
	// that can never match, and a plain-HTTP one off the loopback would be a
	// person handed back over an unprotected hop.
	out.PostLogoutRedirectURIs = dedupe(request.PostLogoutRedirectURIs)
	for _, raw := range out.PostLogoutRedirectURIs {
		if err := s.checkRedirect(raw, out.ClientType); err != nil {
			return Registration{}, err
		}
	}

	for _, raw := range []string{out.ClientURI, out.LogoURI} {
		if raw == "" {
			continue
		}
		if parsed, err := url.Parse(raw); err != nil || !parsed.IsAbs() {
			return Registration{}, ErrAbsoluteURIs
		}
	}

	return out, nil
}

// checkRedirect enforces what the authorization endpoint will later match
// against.
func (s *Service) checkRedirect(raw, clientType string) error {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() {
		return notAbsolute(raw)
	}
	// A fragment is never sent to the server and cannot be matched, so a
	// registration carrying one is a mistake worth naming now.
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return hasFragment(raw)
	}
	if strings.Contains(raw, "*") {
		return hasWildcard(raw)
	}
	// Credentials in the URI are never part of a callback anyone means to
	// register, and they would be handed to whoever reads the browser's history.
	if parsed.User != nil {
		return hasUserinfo(raw)
	}

	switch parsed.Scheme {
	case "https":
		// The operator's host allowlist, when they have set one.
		if err := s.vocabulary.AllowedRedirect(raw); err != nil {
			return Refused(err)
		}
		return nil
	case "http":
		// Plain HTTP is only ever safe on the loopback interface, which is how
		// native apps and local development receive the redirect.
		if isLoopback(parsed.Hostname()) {
			return nil
		}
		return insecure(raw)
	default:
		// Custom schemes (com.example.app:/callback) are how a mobile app
		// receives a redirect, and are meaningful only for public clients.
		if clientType == TypePublic {
			return nil
		}
		return notWeb(raw)
	}
}

// Credentials decides what a newly registered client is given.
//
// A public client is issued no secret at all: PKCE stands in for it. The empty
// string is the answer rather than an error, because "this client has no
// secret" is the correct state for a mobile app rather than a failure to give
// it one.
func (s *Service) Credentials(registration Registration) (clientID, secret string) {
	clientID = "app_" + slugify(registration.ClientName) + "_" + s.identifiers.New(8)
	if registration.IsPublic() {
		return clientID, ""
	}
	return clientID, s.NewSecret()
}

// RotateSecret issues a fresh secret for a client that has one.
//
// A public client is refused rather than quietly given one: it authenticates
// with PKCE, nothing would ever present the secret, and a screen that showed it
// would be showing a credential that means nothing.
func (s *Service) RotateSecret(clientType string) (string, error) {
	if clientType == TypePublic {
		return "", ErrPublicHasNoSecret
	}
	return s.NewSecret(), nil
}

// NewSecret is the shape of a client secret: a prefix that says what it is when
// somebody finds one in a log, and 48 characters of randomness.
func (s *Service) NewSecret() string { return "sec_" + s.identifiers.New(48) }

// FixedType is the rule that the kind of client cannot change after
// registration: flipping a public client to confidential would leave it with no
// secret, and the other direction would leave a secret in a binary that cannot
// keep one.
func FixedType(request Registration, existingType string) Registration {
	request.ClientType = existingType
	return request
}

func dedupe(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}
	return out
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// slugify turns an application name into the readable half of its client_id.
func slugify(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteRune('-')
			}
		}
		if b.Len() >= 24 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}
