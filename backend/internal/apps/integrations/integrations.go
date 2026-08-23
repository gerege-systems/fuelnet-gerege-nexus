/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Package integrations is the connector manager: the screens an administrator
// registers a Zoom, Teams, Google or Dropbox account on, and the delivery log
// that says what this deployment sent where.
//
// The screens were the platform's until 2026-08-23 and the split was already
// written down (docs/CORE_BOUNDARY_PLAN.md §4.2): the *rail* is the platform's
// and the *administration of it* is an app. A deployment that books no meetings
// and files nothing outside itself has no use for either, and until now it
// served both anyway.
//
// What did not move is internal/platform/integration — the manager, the
// provider registry, the OAuth exchange, the credential encryption and the
// dispatch loop. Two things depend on it that are not this app: the PDF signing
// rails file finished documents through it, and nexus.MeetingBooker is its
// adapter. A rail two other things hold is not an app's to take away.
package integrations

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/integration"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// ID is the app id, the same one the catalogue and the store carry.
const ID = "io.gerege.nexus.integrations"

// PermissionManage is the only permission this app declares.
//
// One, and administrative: there is nothing here to read that is not also the
// power to change it. A connector's target URL makes this server issue
// outbound requests to an address somebody typed, and beginning an OAuth flow
// binds an account outside the platform to an organisation inside it. Both were
// behind requireAdmin before this app existed and neither should be reachable
// by a suffix rule — hence AdminOnly, which withholds it from the default
// manager and user roles rather than granting it by the shape of its name.
const PermissionManage = "integrations.manage"

// Module is the app.
type Module struct {
	mgr   *integration.Manager
	perms nexus.PermissionStore
}

// New builds the module and registers it.
//
// The manager is handed in rather than built: it is the deployment's, one per
// process, with a dispatch loop and an encryption key behind it. A second one
// would be a second loop over the same table.
func New(p nexus.Platform, mgr *integration.Manager) *Module {
	m := &Module{mgr: mgr, perms: p.Permissions()}
	nexus.Register(m)
	return m
}

func (m *Module) ID() string      { return ID }
func (m *Module) Name() string    { return "Integrations" }
func (m *Module) Version() string { return "1.0.0" }

// Dependencies is empty: a connector is useful on its own — a webhook target, a
// meeting account — and the apps that use one work without it.
func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: PermissionManage, Name: "Manage Integrations",
			Description: "Register the accounts and endpoints this organisation connects to, and read what has been sent to them",
			AdminOnly:   true},
	}
}

// Menus is empty: the screen is /settings/integrations, which the shell draws
// in its own settings group rather than as an app of its own — see the manifest,
// which marks this app chrome for that reason.
func (m *Module) Menus() []nexus.MenuDefinition { return nil }

// RegisterRoutes mounts the connector administration, and one route that is not
// administration at all.
//
// The OAuth callback is deliberately outside the gate and outside the
// permission. It is where a provider sends the administrator's browser back to,
// and that request carries no session of this platform — the flow is identified
// by the state row, which is what CompleteConnect reads the tenant and the
// actor from. Putting the app gate in front of it would fail every connect at
// the last step, and requiring a permission would fail the ones that arrive in
// a fresh browser tab.
func (m *Module) RegisterRoutes(r chi.Router, gateMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/integrations", func(ir chi.Router) {
		ir.Get("/oauth/callback", m.handleIntegrationOAuthCallback)

		ir.Group(func(ar chi.Router) {
			ar.Use(gateMiddleware)
			ar.Use(nexus.RequirePermission(m.perms, PermissionManage))

			ar.Get("/", m.handleListIntegrations)
			ar.Post("/", m.handleRegisterIntegration)
			ar.Get("/providers", m.handleIntegrationProviders)
			ar.Get("/deliveries", m.handleIntegrationDeliveries)
			ar.Put("/{id}", m.handleUpdateIntegration)
			ar.Delete("/{id}", m.handleDeleteIntegration)
			ar.Post("/{id}/connect", m.handleConnectIntegration)
			ar.Post("/{id}/disconnect", m.handleDisconnectIntegration)
		})
	})
}
