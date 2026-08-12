/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package config

import (
	"os"
	"strings"
)

// The two origins a deployment has, and why there are two.
//
// In production they are the same string: the API and the browser app are
// served from one host, which is what PUBLIC_ORIGIN names and what every
// callback URL in this codebase is already built from. In development they are
// not — the API is on :8080 and Next.js is on :3000 — and a redirect that
// confuses them lands the person on a JSON endpoint instead of a page.
//
// Both are read from the environment rather than from the incoming request. A
// Host header is attacker-controlled, and an origin derived from one is an
// origin somebody else chose.

// SelfOrigin is where this platform's own API answers. It is what the OAuth2
// provider half already calls its issuer, so the two cannot drift.
func SelfOrigin() string {
	if origin := trimOrigin(os.Getenv("SSO_ISSUER")); origin != "" {
		return origin
	}
	if origin := trimOrigin(os.Getenv("PUBLIC_ORIGIN")); origin != "" {
		return origin
	}
	return "http://localhost:8080"
}

// WebOrigin is where the browser app lives — where to send somebody after a
// redirect that has to land on a page rather than on an endpoint.
//
// PUBLIC_ORIGIN first, because in a real deployment that is the answer. Failing
// that, the first entry of ALLOWED_ORIGINS: the list of origins allowed to call
// this API with credentials is, in development, exactly the frontend.
func WebOrigin() string {
	if origin := trimOrigin(os.Getenv("PUBLIC_ORIGIN")); origin != "" {
		return origin
	}
	for _, candidate := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if origin := trimOrigin(candidate); origin != "" {
			return origin
		}
	}
	return "http://localhost:3000"
}

func trimOrigin(raw string) string { return strings.TrimRight(strings.TrimSpace(raw), "/") }
