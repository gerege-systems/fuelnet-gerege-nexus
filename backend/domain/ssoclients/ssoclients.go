/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Package ssoclients is what an organisation may register as a system that
 * signs people in through this platform.
 *
 * Not the OAuth2 provider — that is internal/platform/ssoprovider, and it stays
 * there: issuing a code, exchanging a token, publishing a JWKS and holding the
 * clients are one authorization server the whole deployment shares. What is
 * here is the half that is this app's: what a registration has to look like
 * before the provider is allowed to hold it.
 *
 * That half is worth having on its own, because it decides where this platform
 * will hand a person back after signing them in. A fragment the server never
 * sees, a wildcard nothing can match exactly, credentials in a callback URL,
 * plain HTTP off the loopback — every one of those is a registration the
 * authorization endpoint would later refuse, or worse, accept.
 */
package ssoclients

// Registration is a client application as somebody has just described it, and —
// once it has been through the rules — as it is stored.
//
// The JSON tags are the request shape the screen posts, unchanged.
type Registration struct {
	ClientName   string   `json:"client_name"`
	ClientURI    string   `json:"client_uri"`
	LogoURI      string   `json:"logo_uri"`
	ClientType   string   `json:"client_type"`
	RedirectURIs []string `json:"redirect_uris"`
	// PostLogoutRedirectURIs is where the platform may return somebody after
	// this application signs them out of it. Optional: an application that
	// never ends a session here needs none, and one that does gets its return
	// address matched exactly.
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris"`
	GrantTypes             []string `json:"grant_types"`
	Scopes                 []string `json:"scopes"`
	Disabled               bool     `json:"disabled"`
}

// The two kinds of client, and the difference that matters: a public client
// authenticates with PKCE alone, because a secret embedded in a mobile app or
// an SPA is readable by anyone who downloads it.
const (
	TypeConfidential = "confidential"
	TypePublic       = "public"
)

// IsPublic reports whether this registration is one that gets no secret.
func (r Registration) IsPublic() bool { return r.ClientType == TypePublic }

// The defaults a half-filled form is completed with.
var (
	DefaultGrantTypes = []string{"authorization_code", "refresh_token"}
	DefaultScopes     = []string{"openid", "profile", "erp.read"}
)
