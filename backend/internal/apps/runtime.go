// Package apps assembles the business-module runtime. The platform owns HTTP,
// sessions and infrastructure; it does not need to import every module or know
// its route table.
package apps

import (
	"context"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/egov"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/esign"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/organisation"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eidmongolia"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/gerege"
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

func Bootstrap(p nexus.Platform, integrations *integration.Manager, eidMN *eidmongolia.Service,
	sso *ssoprovider.SSOProvider, xyp *gerege.GeregeService, rails egov.Rails,
	installedApps InstalledApps) Runtime {
	// First, and not merely in order: organisation is what the others assume. It is the
	// organisation, the people in it and how it is arranged — the module Odoo
	// calls base and never lets you uninstall.
	organisation.New(p)
	// The state's systems, as an app rather than as two handlers in the
	// platform's route table. The low-level clients stay where they are; this
	// is their app-facing surface, and the thing contacts reaches through.
	egov.New(p, xyp, rails)
	contacts.New(p)
	documents.New(p)
	sso_clients.New(sso)
	// The App Store's three modules used to be constructed here. They are a
	// product of their own now — github.com/gerege-systems/appstore-gerege-nexus
	// — and reach this list through platform.Options.Modules, the same way any
	// distribution's do. Every other deployment stopped carrying them as dead
	// weight the day they left.
	esignModule := esign.New(p, gerege.NewEsignService(), eidMN, integrations)
	// Last, and after every module that registers a report: the reports app
	// serves the registry, and a module constructed after it would have its
	// reports missing from the first listing until something else rebuilt it.
	reportsModule := reports.New(p, installedApps)
	return Runtime{Background: []BackgroundModule{esignModule, reportsModule}}
}
