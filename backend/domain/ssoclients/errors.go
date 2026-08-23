/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package ssoclients

// Every refusal here is answered as a 400 and reads as a sentence an integrator
// can act on, which is why several of them name the offending value.
var (
	ErrNameRequired      = rule("client_name is required")
	ErrNameTooLong       = rule("client_name is too long")
	ErrClientType        = rule("client_type must be confidential or public")
	ErrNoRedirect        = rule("authorization_code requires at least one redirect_uri")
	ErrPublicCredentials = rule("a public client cannot use client_credentials: it has no secret to prove with")
	ErrAbsoluteURIs      = rule("client_uri and logo_uri must be absolute URLs")
	ErrPublicHasNoSecret = rule("a public client has no secret to rotate")
)

// The refusals that name the value they are about. An integrator with twenty
// redirect URIs registered needs to know which one.
func unsupportedGrant(grant string) error { return rule("unsupported grant type: " + grant) }

func unknownScope(scope string) error { return rule("unknown scope: " + scope) }

func notAbsolute(raw string) error { return rule("redirect_uri must be an absolute URL: " + raw) }

func hasFragment(raw string) error { return rule("redirect_uri must not contain a fragment: " + raw) }

func hasWildcard(raw string) error {
	return rule("wildcards are not allowed in a redirect_uri: " + raw)
}

func hasUserinfo(raw string) error { return rule("redirect_uri must not carry userinfo: " + raw) }

func insecure(raw string) error { return rule("redirect_uri must use https outside localhost: " + raw) }

func notWeb(raw string) error {
	return rule("a confidential client's redirect_uri must be http(s): " + raw)
}

// Refused carries a refusal whose words came from the deployment's own
// allowlist. The operator wrote that rule; restating it here would be a second
// opinion about their configuration.
func Refused(err error) error { return rule(err.Error()) }

type ruleError struct{ message string }

func (e *ruleError) Error() string { return e.message }

func rule(message string) *ruleError { return &ruleError{message: message} }
