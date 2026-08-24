/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package config

// PlatformVersion is the semver the app-store manifests are validated against.
//
// It is a var rather than a const so a release build can stamp the version it
// actually is:
//
//	go build -ldflags "-X github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/config.PlatformVersion=1.2.0"
//
// A manifest names the platform it needs (`"platform": ">=1.1.0"`), and a store
// that separates from this binary has to be told which platform is asking. A
// constant would have every deployment claim 1.0.0 for ever, so every app would
// look compatible with every instance. The default stays 1.0.0, which is what an
// unstamped build has always reported; whatever is injected must be valid
// semver, because manifest validation parses it.
var PlatformVersion = "1.1.0"
