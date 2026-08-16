/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package config

import (
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// The platform's clock, kept in pkg/nexus so a module in another repository can
// read it too — a register number carries a year and a report covers days, and
// both are calendar decisions a module makes on its own.
//
// These three names stay because the platform's own packages already call them
// and because this is where somebody looks for a configuration value. They
// delegate: one clock, not two that agree until somebody edits one.

// DefaultTimezone is the wall clock this platform keeps.
const DefaultTimezone = nexus.DefaultTimezone

// Location is the platform's clock, for every Go-side calendar decision.
func Location() *time.Location { return nexus.Location() }

// TimezoneName is the same answer as an IANA name, for the places that need to
// tell somebody else — chiefly the database, which is given it as a connection
// parameter so that `::date`, `CURRENT_DATE` and `date_trunc` all agree with
// the Go side rather than with whatever the server was configured with. See
// dbguard.Install.
func TimezoneName() string { return nexus.TimezoneName() }

// Now is the current instant on the platform's clock.
func Now() time.Time { return nexus.Now() }

// Today is the current date on the platform's clock, at midnight.
func Today() time.Time { return nexus.Today() }
