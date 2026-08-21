// Package config centralises how the platform reads environment-driven
// runtime switches.
package config

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// MockEnabled reports whether a national-integration connector (E-ID, DAN,
// XYP) should serve canned identities instead of calling the live gateway.
//
// The previous rule was `os.Getenv(name) != "false"`, i.e. mock mode was ON by
// default. In production that turns the E-ID and DAN login endpoints into an
// unauthenticated door: any registration number returns a "verified" citizen.
// Mock mode is therefore never implicit in production — it must be requested.
func MockEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "true", "1", "yes":
		if IsProduction() {
			slog.Warn("mock integration mode explicitly enabled in production", "flag", name)
		}
		return true
	case "false", "0", "no":
		return false
	}
	// Unset: convenient for local development, refused in production.
	if IsProduction() {
		slog.Info("mock integration mode disabled (production default)", "flag", name)
		return false
	}
	return true
}

// IsProduction reports whether ENVIRONMENT=production.
func IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ENVIRONMENT")), "production")
}

// SupportedLocales and LocaleFromRequest moved to pkg/nexus.
//
// They had to: a module cannot import internal/platform/config, and a module
// that renders a menu label or a report title per locale has no other way to
// learn which locale was asked for. internal/apps/reports imported this package
// for exactly one function, which was the last reason it imported the platform
// at all.
//
// Kept as aliases so the platform's own callers did not all have to change in
// the same commit.
var SupportedLocales = nexus.SupportedLocales

// LocaleFromRequest is the language this caller asked for. See
// nexus.LocaleFromRequest.
func LocaleFromRequest(r *http.Request) string { return nexus.LocaleFromRequest(r) }

// SeedingEnabled reports whether the documented demo account should be created.
//
// It lives here rather than beside the seeder because two packages need the
// same answer: the seeder, which acts on it, and the platform, which warns when
// it disagrees with the access mode — a seeder that creates accounts on a
// platform configured to create no accounts is somebody's intent getting lost
// between two files.
//
// The account has a published password, so it is never seeded into production
// unless an operator asks for it in so many words.
func SeedingEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SEED_DEMO_DATA"))) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return !IsProduction()
}
