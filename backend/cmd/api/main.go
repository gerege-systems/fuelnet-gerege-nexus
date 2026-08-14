/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
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

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/platform"
)

func main() {
	if err := platform.Run(platform.Options{}); err != nil {
		slog.Error("the platform stopped", "error", err)
		os.Exit(1)
	}
}
