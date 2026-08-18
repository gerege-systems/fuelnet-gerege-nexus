/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package ssoclients

// Vocabulary is what this deployment's authorization server supports, and what
// its operator allows.
//
// All three answers belong to the provider rather than to this app: the scope
// list is what the consent screen renders, the grant list is what the token
// endpoint implements, and the redirect allowlist is the operator's
// deployment-wide floor. A second opinion about any of them here would be a
// registration the provider then refuses, or one it accepts and this screen
// would not.
type Vocabulary interface {
	SupportedGrantTypes() []string
	IsSupportedScope(scope string) bool
	// AllowedRedirect is the operator's host allowlist, applied on top of this
	// package's own rules and only to https addresses.
	AllowedRedirect(raw string) error
}

// Identifiers is where a client_id's random half and a client secret come from.
//
// A port because the randomness is the platform's — one source, seeded once,
// audited once — and because a test that could not fix it would be asserting
// against a coin toss.
type Identifiers interface {
	New(length int) string
}
