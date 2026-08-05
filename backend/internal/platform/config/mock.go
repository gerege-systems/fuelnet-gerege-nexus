// Package config centralises how the platform reads environment-driven
// runtime switches.
package config

import (
	"log/slog"
	"os"
	"strings"
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
