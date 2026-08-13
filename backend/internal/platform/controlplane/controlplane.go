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
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/emailverify"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/flags"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/settings"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Installer puts an app into an organisation, the way the store does.
//
// An interface rather than the concrete installer because the console must not
// be able to reach the rest of it: this is the whole of what CP-2 asks the
// platform to do on its behalf, and a narrower dependency is a narrower blast
// radius when somebody adds a handler here in a hurry.
type Installer interface {
	InstallAppForTenant(ctx context.Context, tenantID, appSlug, userID string) error
}

// Mailer is the platform's one rail for sending somebody a link. It is
// satisfied by *emailverify.Service.
type Mailer interface {
	Send(ctx context.Context, tenantID string, req emailverify.Request) (*emailverify.Verification, error)
}

// Deps are the platform's own services the console borrows. All may be nil:
// a deployment with no mail configured still runs a console, it just cannot
// invite anybody, and it says so on the screen rather than in the log.
type Deps struct {
	Installer Installer
	Mail      Mailer
	// Settings and Flags are what CP-3 edits. Held as the platform's own
	// stores rather than rebuilt here, so a change the console makes is felt
	// by the running platform rather than by a second copy of it.
	Settings *settings.Store
	Flags    *flags.Store
	// TenantChanged is called after the console changes an organisation's
	// lifecycle, so the platform can drop what it has cached about it — on
	// every replica, through the invalidation bus, rather than after each of
	// them has waited out its own copy.
	//
	// A callback rather than the console reaching into the platform's caches:
	// this package must not know that those caches exist, and the platform
	// must not have to expose them.
	TenantChanged func(tenantID string)
	// Warnings are the platform's own complaints about its configuration — a
	// demo seeder on a private deployment, and whatever CP-4 adds. A callback
	// because the answer lives in the platform package, which imports this one:
	// the console asks rather than reaching.
	Warnings func() []string
}

// Service holds what every control-plane request needs.
type Service struct {
	db            *pgxpool.Pool
	sessions      *SessionStore
	installer     Installer
	mail          Mailer
	settings      *settings.Store
	flags         *flags.Store
	tenantChanged func(tenantID string)
	warningsFrom  func() []string
	// host is the only hostname the console answers on, from
	// CONTROL_PLANE_HOST. Empty has a meaning that depends on the environment —
	// see hostGate.
	host string
}

// New builds the console. It performs no I/O: a deployment without the
// migrations still constructs, and its routes refuse at the door.
func New(db *pgxpool.Pool, deps Deps) *Service {
	return &Service{
		db:            db,
		sessions:      NewSessionStore(db),
		installer:     deps.Installer,
		mail:          deps.Mail,
		settings:      deps.Settings,
		flags:         deps.Flags,
		tenantChanged: deps.TenantChanged,
		warningsFrom:  deps.Warnings,
		host:          normaliseHost(os.Getenv("CONTROL_PLANE_HOST")),
	}
}

// StartHousekeeping runs the console's background work: purging operator
// sessions that can no longer be used, and removing the organisations whose
// grace period has run out.
func (s *Service) StartHousekeeping(ctx context.Context) {
	s.sessions.StartHousekeeping(ctx)
	s.StartDeletionSweep(ctx)
}

// changed tells the platform that an organisation's lifecycle moved.
func (s *Service) changed(tenantID string) {
	if s.tenantChanged != nil {
		s.tenantChanged(tenantID)
	}
}

// warnings is what the configuration screen shows above the fields: the
// platform's own complaints, plus the feature flags that have outlived the date
// somebody gave them.
//
// Flag debt is only ever paid when somebody is reminded, and the moment they
// are looking at the configuration is the moment to remind them.
func (s *Service) warnings() []string {
	warnings := make([]string, 0, 2)
	if s.warningsFrom != nil {
		warnings = append(warnings, s.warningsFrom()...)
	}
	if s.flags != nil {
		if _, expired := s.flags.Snapshot(time.Now()); len(expired) > 0 {
			warnings = append(warnings,
				"Хугацаа нь дууссан feature flag: "+strings.Join(expired, ", ")+
					". Кодоос нь салгаад flag-ийг устгах цаг болсон.")
		}
	}
	return warnings
}

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
