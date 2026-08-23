// Package apps assembles the business-module runtime. The platform owns HTTP,
// sessions and infrastructure; it does not need to import every module or know
// its route table.
package apps

import (
	"context"
	"log/slog"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	appai "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/ai"
	appintegrations "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/integrations"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	appstaffpin "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/staffpin"
	appurtuu "github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/urtuu"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/integration"
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
// of them before it — before a distribution's modules too, which are built in
// between. So it is reported and the zero value is used, which leaves one
// module degraded instead of refusing to start a deployment over a dependency
// most of it does not touch. Anything whose zero value is a nil that something
// would later call is handed over as a parameter instead; see the rails below.
func Bootstrap(p nexus.Platform) Runtime {
	sso := required[*ssoprovider.SSOProvider]()
	link := required[nexus.Link]()
	installedApps := required[InstalledApps]()

	// Three apps were constructed here and are not any more. All three moved to
	// client-gerege-nexus on 2026-08-23, in the order their contracts were
	// published: the e-Government link once the state's registers and the audit
	// trail were nexus.StateRegistry and nexus.AuditReader; documents once the
	// identity rails and the PDF signing rail were nexus.EIDSigner,
	// nexus.DANAuthenticator and nexus.SigningRails; the organisation once who
	// belongs to it was nexus.Directory and its two columns had a table of
	// their own (migration 00076).
	//
	// None of them was removed for being bad. They were removed for being apps:
	// an app that ships inside the platform is one every deployment carries
	// whether it has a use for it or not, and a queue kiosk has no use for a
	// staff directory.
	//
	// The PDF signing rails left too, upward rather than sideways: server.go
	// builds them before any module, because a module that signs a PDF asks for
	// nexus.SigningRails in its constructor and a distribution's modules are
	// built before this function is called. Their housekeeping is appended to
	// this runtime's list there, where the value is in scope — not asked back
	// out of the capability registry, which would key a housekeeping loop on an
	// internal/ type no distribution can name and turn a deleted Provide into a
	// nil that only panics five minutes after a clean boot.
	sso_clients.New(sso)
	// The assistant. An app since 2026-08-23 rather than ten routes in
	// server.go: it asks the platform for the deployment's rate limit and the
	// organisation's monthly allowance (nexus.RateLimit, nexus.QuotaGate) and
	// keeps the rest — the prompt, the knowledge, the model traffic — to
	// itself, which is what makes it removable.
	appai.New(p)
	// The connector administration. The manager it edits is the platform's —
	// one dispatch loop, one encryption key — so it is asked for rather than
	// built, the same way every other rail this repository's apps reach for is.
	appintegrations.New(p, required[*integration.Manager]())
	// The till's staff PIN. It publishes nexus.StaffCredential, which the
	// platform's device sign-in route asks for — the credential is a product's
	// and the session it opens is the platform's, and this is the seam between
	// them. Handed the same installed-apps gate the reports app takes, because
	// the route that consumes the credential carries a device token and no
	// session, so the app gate cannot stand in front of it.
	appstaffpin.New(p, appstaffpin.InstalledApps(installedApps))
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
	return Runtime{Background: []BackgroundModule{reportsModule}}
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
