/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package reports is the app that serves every other module's reports.
 */

// The reports module knows nothing about billing, inventory or e-signatures.
//
// It serves whatever is in the reporting registry, gated by which apps the
// caller's tenant has installed. That is the Odoo shape: a report ships with
// the module that understands the data, and the reporting layer is generic.
// The alternative — this module importing every other one to know their
// reports — is exactly the coupling the compile-time module registry was built
// to avoid.
package reports

import (
	"context"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

// ID is the app id, in the catalogue and in every installation row.
const ID = "io.gerege.nexus.reports"

// Permissions. Two, and the split matters: viewing a report is an ordinary act
// for anybody who works with the data, while a schedule sends this
// organisation's numbers to an address list on a timer, which is an
// administrative decision.
const (
	PermissionView     = "reports.view"
	PermissionSchedule = "reports.schedule"
	// PermissionShare covers both directions of a sharing agreement: asking
	// another organisation to show you a report, and agreeing to show them
	// yours. Separate from the other two because it is the only one that
	// crosses the tenant boundary at all.
	PermissionShare = "reports.share"
)

// InstalledApps answers which apps a tenant has, which is how the module
// decides which reports exist for them. Supplied by the platform rather than
// queried here, because the platform already caches that answer and this module
// must not become a second, differently-stale source of it.
type InstalledApps func(ctx context.Context, tenantID string) (map[string]bool, error)

// Module is the reports app.
type Module struct {
	db        nexus.DB
	engine    *reporting.Engine
	scheduler *reporting.Scheduler
	perms     nexus.PermissionStore
	installed InstalledApps
}

// New builds the module and registers it.
//
// installedApps is the platform's own gate, handed in: a tenant sees the
// reports of the apps it has installed and no others, and "which apps" has
// exactly one answer on this deployment.
func New(p nexus.Platform, installedApps InstalledApps) *Module {
	db := p.DB()
	engine := reporting.NewEngine(db)
	m := &Module{
		db:        db,
		engine:    engine,
		scheduler: reporting.NewScheduler(engine, reporting.NewSMTPDeliverer()),
		perms:     p.Permissions(),
		installed: installedApps,
	}
	nexus.Register(m)
	return m
}

func (m *Module) ID() string      { return ID }
func (m *Module) Name() string    { return "Reports" }
func (m *Module) Version() string { return "1.0.0" }

// Dependencies is empty on purpose. Reports is useful with any combination of
// the other apps and useless with none of them, and declaring a dependency on
// billing would mean a tenant that only runs inventory could not install it.
func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: PermissionView, Name: "View Reports", Description: "Run and export the reports of the apps this organisation has installed"},
		{Code: PermissionSchedule, Name: "Schedule Reports", Description: "Create and remove scheduled reports that are mailed out automatically"},
		{Code: PermissionShare, Name: "Share Reports Across Organisations", Description: "Ask another organisation to share a report, and agree to share this organisation's own"},
	}
}

func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{
			ID: "reports", ParentID: "operations", Label: "Reports", Path: "/reports",
			Icon: "bar-chart-3", Order: 90,
			Labels: map[string]string{
				"mn": "Тайлан", "ar": "التقارير", "zh": "报表",
				"fr": "Rapports", "ru": "Отчёты", "es": "Informes",
			},
		},
	}
}

// StartHousekeeping runs the schedule sweep. The platform calls it for every
// module that has one, so scheduled reports need no process of their own.
func (m *Module) StartHousekeeping(ctx context.Context) {
	m.scheduler.Start(ctx)
}

// RegisterRoutes mounts the API behind the app gate.
//
// Every route is inside gateMiddleware: there is no public reporting endpoint
// and there must never be one — a report is an aggregate of a tenant's data,
// which is the thing this platform is most careful about.
func (m *Module) RegisterRoutes(r chi.Router, gateMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/reports", func(rr chi.Router) {
		rr.Use(gateMiddleware)

		// Reading. `reports.view` plus, per report, the app it belongs to —
		// checked in the handler rather than here, because which app depends
		// on which report was asked for.
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Get("/", m.handleList)
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Get("/schedules", m.handleListSchedules)
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Get("/{key}", m.handleMetadata)
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Post("/{key}/run", m.handleRun)
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Post("/{key}/export", m.handleExport)

		// The consolidated view. Reading, from this organisation's side: the
		// permission that matters on the other side is the grant, and it was
		// given by the organisation that owns the data rather than by anything
		// here.
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Post("/{key}/run-consolidated", m.handleRunConsolidated)
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Get("/grants", m.handleListGrants)
		// Who has read our data. A read, and one this organisation is entitled
		// to — see handleAccessHistory.
		rr.With(nexus.RequirePermission(m.perms, PermissionView)).Get("/grants/history", m.handleAccessHistory)

		// Scheduling. Administrative: it sends this organisation's numbers to
		// an address list, on a timer, without anybody present.
		rr.With(nexus.RequirePermission(m.perms, PermissionSchedule)).Post("/schedules", m.handleCreateSchedule)
		rr.With(nexus.RequirePermission(m.perms, PermissionSchedule)).Put("/schedules/{id}", m.handleUpdateSchedule)
		rr.With(nexus.RequirePermission(m.perms, PermissionSchedule)).Delete("/schedules/{id}", m.handleDeleteSchedule)

		// Sharing. Under reports.share, not reports.schedule: asking another
		// organisation for their numbers, and agreeing to hand over your own,
		// is a different decision from mailing your own out — and the second
		// one is the one this platform is most careful about.
		rr.With(nexus.RequirePermission(m.perms, PermissionShare)).Post("/grants", m.handleRequestGrant)
		rr.With(nexus.RequirePermission(m.perms, PermissionShare)).Post("/grants/{id}/accept", m.handleAcceptGrant)
		rr.With(nexus.RequirePermission(m.perms, PermissionShare)).Post("/grants/{id}/revoke", m.handleRevokeGrant)
	})
}
