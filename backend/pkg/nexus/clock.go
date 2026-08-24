/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * What clock this platform keeps.
 *
 * In the SDK rather than in internal/kernel/config, where it was, because a
 * module needs it and a module may live in another repository: a register
 * number carries a year, a report covers a range of days, a schedule fires at
 * an hour. Every one of those is a calendar decision, and a module that reached
 * for time.Now() in UTC would put a Mongolian office's Monday morning on
 * Sunday. internal/kernel/config still answers to the same names — it
 * delegates here, so there is one clock and not two.
 */

package nexus

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	// The zone database, compiled in.
	//
	// Without it `time.LoadLocation` reads the operating system's copy, which
	// the runtime image happens to carry today (deploy/Dockerfile installs
	// tzdata) and a leaner base image tomorrow would not. The failure that
	// would cause is the worst shape available: LoadLocation returns an error,
	// the fallback is UTC, and every daily figure quietly shifts by eight hours
	// with nothing in the logs anybody would connect to it. Four hundred
	// kilobytes buys the guarantee that the platform's clock does not depend on
	// which base image somebody chose.
	_ "time/tzdata"
)

// DefaultTimezone is the wall clock this platform keeps.
//
// Ulaanbaatar, because that is the day the people using it work in. It matters
// wherever an instant is reduced to a *calendar*: which day a usage figure
// belongs to, which month a quota is counted against, which date a report's
// range covers, which year a register number carries, and what hour a schedule
// fires at. A platform that quietly used UTC would give a Mongolian office a
// day that ends at eight in the morning.
//
// Storage is unaffected and always was: every timestamp column is `timestamptz`
// and holds an instant. This decides only how an instant is *read*.
const DefaultTimezone = "Asia/Ulaanbaatar"

// timezoneEnv overrides it. A deployment outside Mongolia — and this is an open
// platform, so there is one — sets its own rather than forking a constant.
const timezoneEnv = "PLATFORM_TIMEZONE"

var (
	timezoneOnce sync.Once
	location     *time.Location
	locationName string
)

func resolveTimezone() {
	name := strings.TrimSpace(os.Getenv(timezoneEnv))
	if name == "" {
		name = DefaultTimezone
	}
	loaded, err := time.LoadLocation(name)
	if err != nil {
		// Named but unloadable. UTC rather than a refusal to start — a wrong
		// clock is a wrong figure, and a platform that will not boot is every
		// figure — but said loudly, because the numbers on the console are
		// about to be in a different day from the ones the operator expects.
		slog.Error("nexus: "+timezoneEnv+" names a zone this build cannot load; keeping UTC",
			"timezone", name, "error", err)
		location, locationName = time.UTC, "UTC"
		return
	}
	location, locationName = loaded, name
}

// Location is the platform's clock, for every Go-side calendar decision.
func Location() *time.Location {
	timezoneOnce.Do(resolveTimezone)
	return location
}

// TimezoneName is the same answer as an IANA name, for the places that need to
// tell somebody else — chiefly the database, which is given it as a connection
// parameter so that `::date`, `CURRENT_DATE` and `date_trunc` all agree with
// the Go side rather than with whatever the server was configured with. See
// dbguard.Install.
func TimezoneName() string {
	timezoneOnce.Do(resolveTimezone)
	return locationName
}

// Today is the platform's current date, and Now is the current instant read on
// the platform's clock.
//
// Two helpers rather than `time.Now()` at each call site, because `time.Now()`
// carries the *process's* zone — which in a container is whatever the image
// decided, usually UTC — and a calendar taken from it is a calendar nobody
// chose.
func Now() time.Time { return time.Now().In(Location()) }

// Today is the current date on the platform's clock, at midnight.
func Today() time.Time {
	now := Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
