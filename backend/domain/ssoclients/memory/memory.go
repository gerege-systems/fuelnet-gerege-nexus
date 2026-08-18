/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * NOT FOR PRODUCTION. A vocabulary and an identifier source a test can fix, so
 * the registration rules can be run without the authorization server and
 * without a coin toss. Nothing else may import it.
 */
package memory

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

// Vocabulary is what a deployment supports, stated rather than discovered.
type Vocabulary struct {
	Grants []string
	Scopes []string
	// Hosts is the operator's https allowlist. Empty allows nothing, which is
	// the real default's behaviour for any host the operator has not named.
	Hosts []string
}

func (v Vocabulary) SupportedGrantTypes() []string { return v.Grants }

func (v Vocabulary) IsSupportedScope(scope string) bool { return slices.Contains(v.Scopes, scope) }

// AllowedRedirect answers in the provider's words, because those are the words
// the caller has always read.
func (v Vocabulary) AllowedRedirect(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid redirect URI")
	}
	for _, host := range v.Hosts {
		if strings.EqualFold(host, parsed.Hostname()) {
			return nil
		}
	}
	return errors.New("redirect URI host is not allowed")
}

// Identifiers hands out the same string every time, which is what makes an
// assertion about a client_id possible at all.
type Identifiers struct{ Value string }

func (i Identifiers) New(int) string {
	if i.Value == "" {
		return "fixed"
	}
	return i.Value
}
