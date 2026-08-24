// Package apps assembles the business-module runtime. The platform owns HTTP,
// sessions and infrastructure; it does not need to import every module or know
// its route table.
package apps

import (
	"context"
	"log/slog"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/tenant/ssoprovider"
)

type BackgroundModule interface {
	StartHousekeeping(context.Context)
}

type Runtime struct {
	Background []BackgroundModule
}

// InstalledApps is how the platform tells a module which apps a tenant has.
type InstalledApps = nexus.InstalledApps

// Bootstrap builds every module this binary carries, in the order they need.
func Bootstrap(p nexus.Platform) Runtime {
	sso := required[*ssoprovider.SSOProvider]()

	sso_clients.New(sso)
	// Өртөө's task board was constructed here until 2026-08-23. It is
	// client-gerege-nexus's now, and it reaches the channel the way any
	// distribution's module does: nexus.Link to send, nexus.PeerDirectory to
	// read. The channel itself did not move — a link an administrator
	// established keeps carrying what is in flight over it whatever apps come
	// and go.
	// The App Store's three modules used to be constructed here. They are a
	// product of their own now — github.com/gerege-systems/appstore-gerege-nexus
	// — and reach this list through platform.Options.Modules, the same way any
	// distribution's do. Every other deployment stopped carrying them as dead
	// weight the day they left.
	// Last, and after every module that registers a report: the reports app
	// serves the registry, and a module constructed after it would have its
	// reports missing from the first listing until something else rebuilt it.
	// Reports was constructed here until 2026-08-23 and is client-gerege-nexus's
	// now. What stayed is the engine underneath it — the SQL, the export, the
	// sweep that mails a schedule at three in the morning, the check that lets
	// one organisation's report read another's rows — published as
	// nexus.ReportEngine, ReportSchedules and ReportGrants.

	return Runtime{}
}

// required fetches a capability the platform is expected to have provided.
func required[T any]() T {
	value, err := nexus.Capability[T]()
	if err != nil {
		slog.Error("a module is being built without something the platform should have provided",
			"error", err)
	}
	return value
}
