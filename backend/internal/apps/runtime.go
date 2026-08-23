// Package apps assembles the business-module runtime. The platform owns HTTP,
// sessions and infrastructure; it does not need to import every module or know
// its route table.
package apps

import (
	"context"
	"log/slog"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/organisation"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	appurtuu "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/urtuu"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/esign"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
)

type BackgroundModule interface {
	StartHousekeeping(context.Context)
}

type Runtime struct {
	Background []BackgroundModule
}

// InstalledApps is how the platform tells the reports module which apps a
// tenant has. Passed in rather than queried there, so there is one answer to
// that question on this deployment and one place it is cached.
type InstalledApps = reports.InstalledApps

// Bootstrap builds every module this binary carries, in the order they need.
//
// One parameter, and it stays one. Everything else is asked of the capability
// registry, so the platform lending its modules something new is a Provide call
// where this line used to be — see the note on this signature's history in
// runtime_test.go.
//
// A missing capability is a platform bug rather than a distribution's mistake:
// nothing outside this repository calls Bootstrap, and server.go provides all
// of them immediately before it. So it is reported and the zero value is used,
// which leaves one module degraded instead of refusing to start a deployment
// over a dependency most of it does not touch.
func Bootstrap(p nexus.Platform) Runtime {
	sso := required[*ssoprovider.SSOProvider]()
	link := required[nexus.Link]()
	signer := required[nexus.Signer]()
	installedApps := required[InstalledApps]()

	// First, and not merely in order: organisation is what the others assume. It
	// is the organisation, the people in it and how it is arranged — the module
	// Odoo calls base.
	//
	// The contact register was briefly part of it and is not: it went to
	// commerce-gerege-nexus, because everybody has departments and only a
	// business has customers.
	organisation.New(p)
	// The e-Government link was constructed here. It moved to
	// client-gerege-nexus on 2026-08-23: the state's registers and the audit
	// trail are contracts now (nexus.StateRegistry, nexus.AuditReader), so the
	// module had nothing left that only this repository could give it.
	// The PDF signing rails. Constructed in server.go now, not here: a module
	// that signs a PDF asks for nexus.SigningRails in its constructor, and a
	// distribution's modules are built before this function is called. Asked
	// for rather than rebuilt — a second Rails would be a second housekeeping
	// loop over the same tables.
	esignModule := required[*esign.Rails]()
	// The signing rail, as the SDK publishes it. A document that carries a file
	// is signed over that file's digest through this; one that carries nothing
	// is approved on the sign-in rail as before. See ADR 0003.
	documents.New(p, signer)
	sso_clients.New(sso)
	// Өртөө: the task board over the platform's channel to other installations.
	// Constructed whether or not this deployment has a signing key — the module
	// registers the readers for the task envelopes, and a deployment given a key
	// later must not need a second restart before its backlog is read.
	appurtuu.New(p, link)
	// The App Store's three modules used to be constructed here. They are a
	// product of their own now — github.com/gerege-systems/appstore-gerege-nexus
	// — and reach this list through platform.Options.Modules, the same way any
	// distribution's do. Every other deployment stopped carrying them as dead
	// weight the day they left.
	// Last, and after every module that registers a report: the reports app
	// serves the registry, and a module constructed after it would have its
	// reports missing from the first listing until something else rebuilt it.
	reportsModule := reports.New(p, installedApps)
	return Runtime{Background: []BackgroundModule{esignModule, reportsModule}}
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
