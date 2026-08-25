/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The platform's own binary.
 *
 * Everything it used to do lives in `backend/pkg/platform` now, because a
 * distribution repository has to do exactly the same thing and could not reach
 * a package under internal/. This file is what is left, and being three lines
 * is the assertion: the platform's binary is built the way every distribution's
 * is, so a boot path that works here works there.
 */

package main

import (
	"log/slog"
	"os"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/fuel"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/platform"
)

func main() {
	if err := platform.Run(platform.Options{
		// Every organisation on this deployment gets the fuel network.
		//
		// It is not an app somebody chooses. A citizen signing in with eID is
		// provisioned into the citizens organisation and expects to draw their
		// ration immediately; asking them to wait for an administrator to visit
		// a store and install something would be asking the wrong person to do
		// the wrong thing. Operators need it for the same reason from the other
		// side — a fuel company on a fuel platform has no other purpose here.
		//
		// This is a distribution's decision rather than the platform's, which is
		// why it is stated in this file and not inside the installer.
		DefaultApps: []string{fuel.ID},
	}); err != nil {
		slog.Error("the platform stopped", "error", err)
		os.Exit(1)
	}
}
