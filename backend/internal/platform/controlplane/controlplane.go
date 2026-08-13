/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

// Package controlplane is the operator console that runs beside the platform
// rather than inside it.
//
// The distinction the design rests on (docs/CONTROL_PLANE_PLAN.md §1) is the
// one every multi-tenant platform eventually draws: the data plane is where
// organisations do their work, the control plane is where the platform is
// operated. They share a binary here, because a second Go process would double
// the deployment and the monitoring for a console two people use — but they
// share nothing else:
//
//	tenant user   → users / sessions      → gerege_nexus_app, one organisation
//	operator      → operator_accounts /
//	                operator_sessions     → gerege_nexus_operator, read-only
//
// Separate accounts, separate sessions, separate cookie, separate database
// role, separate audit table. A tenant administrator's account being taken does
// not reach anything in this package, and an operator's account being taken
// reaches only what migration 00049 named — which is a list of SELECTs.
//
// Three rules hold everywhere in here, and each has a home:
//
//   - The console answers on one hostname only. hostGate, and nginx in front
//     of it with an address allowlist.
//   - Every write is audited, in the same transaction as the write. Do, and
//     requireAudit above it — a write whose audit row did not land does not
//     reach the caller as a success.
//   - Nothing runs as the login role. Every query is made with a context
//     marked by dbguard.AsOperator.
package controlplane

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/dbguard"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service holds what every control-plane request needs.
type Service struct {
	db       *pgxpool.Pool
	sessions *SessionStore
	// host is the only hostname the console answers on, from
	// CONTROL_PLANE_HOST. Empty has a meaning that depends on the environment —
	// see hostGate.
	host string
}

// New builds the console. It performs no I/O: a deployment without the
// migrations still constructs, and its routes refuse at the door.
func New(db *pgxpool.Pool) *Service {
	return &Service{
		db:       db,
		sessions: NewSessionStore(db),
		host:     normaliseHost(os.Getenv("CONTROL_PLANE_HOST")),
	}
}

// StartHousekeeping purges operator sessions that can no longer be used.
func (s *Service) StartHousekeeping(ctx context.Context) { s.sessions.StartHousekeeping(ctx) }

// scoped puts a context on the operator's database role.
//
// Every query in this package goes through it, including the ones made before
// anybody is signed in — resolving a session, checking a password. Those could
// have run on the platform path like the tenant side's do, and that is exactly
// why this exists: the platform path is the login role, which is outside the
// row-level policies and can write anywhere. A console that reads other
// people's organisations for a living should never be one forgotten WHERE
// clause away from that, so it does not have the ability at all.
func scoped(ctx context.Context) context.Context { return dbguard.AsOperator(ctx) }

// normaliseHost lowercases a hostname and drops any port, so that a value
// written as "cp.nexus.gerege.mn:443" in an environment file compares equal to
// what a browser sends.
func normaliseHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if colon := strings.LastIndex(host, ":"); colon > -1 && !strings.Contains(host[colon:], "]") {
		host = host[:colon]
	}
	return strings.TrimSuffix(host, ".")
}

// requestHost is the hostname a request was addressed to.
//
// r.Host is what nginx forwarded with `proxy_set_header Host $host`, which is
// the header the browser sent. X-Forwarded-Host is deliberately not consulted:
// it is a header any client can set, and consulting it would let a request
// arriving at the public hostname claim to be a control-plane one. The
// TRUST_PROXY_HEADERS convention exists for the client address, where there is
// no alternative; here there is one.
func requestHost(r *http.Request) string { return normaliseHost(r.Host) }

// productionEnv reports whether this process runs as a deployment rather than
// on somebody's machine.
func productionEnv() bool { return os.Getenv("ENVIRONMENT") == "production" }

// Timings the console is built around.
const (
	// SessionTTL bounds one operator sign-in end to end. Shorter than the
	// tenant side's twelve hours: a console session that outlives the working
	// day is one an unattended machine keeps open overnight.
	SessionTTL = 8 * time.Hour

	// SessionIdleTimeout ends a console session nobody is using. §2.1 of the
	// plan asks for 30 minutes, against the platform's 90, and the reason is
	// the difference between a screen showing an invoice and a screen that can
	// suspend an organisation.
	SessionIdleTimeout = 30 * time.Minute

	// StepUpWindow is how long a re-confirmed second factor stays good for.
	// Long enough to complete the action it was asked for and anything
	// immediately following it; short enough that walking away ends it.
	StepUpWindow = 5 * time.Minute
)
