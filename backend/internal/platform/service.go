/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Package platform is the plane that acts for the whole deployment.
//
// One organisation is never what a request here is about. An operator suspends
// a tenant, reads the audit trail of every tenant, changes a setting that every
// tenant is served under, counts what all of them used — work that is done on
// behalf of the deployment rather than on anybody's behalf inside it. The other
// plane, internal/tenant, is the opposite of that, and neither imports the
// other.
//
// What is here is the console (docs/CONTROL_PLANE.md) and the daily meter. Both
// run on the operator's database role, whose grants are a written list rather
// than a switch that turns tenant isolation off — see internal/kernel/dbguard.
//
// The two planes are assembled into one process by pkg/platform, which is also
// what owns the router they are both mounted on.
package platform

import (
	"context"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/controlplane"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/metering"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConsoleDeps are the platform's own services the console borrows: the
// installer, so a new organisation's apps are installed by the same code path
// the store uses, and the mail rail, so its first administrator can be invited.
//
// An alias rather than a second struct, so the seam that fills it in names one
// thing and this package does not become a layer that only copies fields.
type ConsoleDeps = controlplane.Deps

// Service is this plane, as one thing to mount and one thing to start.
type Service struct {
	db *pgxpool.Pool
	cp *controlplane.Service
}

// New builds the plane. It performs no I/O.
func New(db *pgxpool.Pool, console ConsoleDeps) *Service {
	return &Service{db: db, cp: controlplane.New(db, console)}
}

// Routes mounts the operator console.
//
// Mounted unconditionally, and closed by its own first middleware rather than
// by leaving the routes off: a route table that changes shape with the
// environment is one where "is the console reachable" has a different answer in
// production from the one the tests exercise. HostGate answers 404 for every
// request that did not arrive on the console's hostname, which on this
// deployment is every request that is not an operator's.
func (s *Service) Routes(r chi.Router) {
	r.Route("/cp/api", s.cp.Routes)
}

// StartBackgroundJobs launches this plane's periodic work. It returns
// immediately; every job runs until ctx is cancelled at shutdown.
func (s *Service) StartBackgroundJobs(ctx context.Context) {
	// Yesterday's usage, every night: what the console charts and what the AI
	// limit is enforced against.
	metering.New(s.db).Start(ctx)
	// The console's sessions are a separate table with the same problem every
	// session table has, and the organisations whose grace period has ended are
	// removed by the same call.
	s.cp.StartHousekeeping(ctx)
}
