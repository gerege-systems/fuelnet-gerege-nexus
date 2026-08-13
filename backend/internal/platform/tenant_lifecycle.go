/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * What the control plane's decisions mean on this side of the platform: a
 * suspended organisation cannot be used, and a full one cannot grow.
 */

package platform

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/memo"
	"github.com/jackc/pgx/v5"
)

// suspendedTTL bounds how long a replica keeps believing an organisation is
// running after another replica has suspended it.
//
// Thirty seconds, matching the app gate, and it is not the primary control:
// suspending revokes every live session in the same transaction, so the people
// already signed in are stopped immediately. This is what stops the *next*
// request from a client that had not noticed yet.
const suspendedTTL = 30 * time.Second

// suspendedCacheName is what the invalidation bus knows the cache as, so that
// resuming an organisation takes effect on every replica at once rather than
// after the slowest of them has waited out its own copy.
const suspendedCacheName = "suspended"

// forgetSuspension drops one organisation's cached state, here and everywhere.
func (s *Server) forgetSuspension(tenantID string) {
	s.bus.Invalidate(suspendedCacheName, memo.Key(tenantID, ""))
}

// ErrTenantSuspended is what every path refuses with.
var ErrTenantSuspended = errors.New("this organisation has been suspended")

// tenantSuspended reports whether an organisation is closed.
//
// A cached read on the request path, like the app gate beside it. The query
// runs on the platform path deliberately: the caller may not have a tenant
// context yet — this is asked during sign-in, before any session exists — and
// `tenants` carries no tenant_id, so no policy applies to it either way.
func (s *Server) tenantSuspended(ctx context.Context, tenantID string) (bool, string) {
	key := memo.Key(tenantID, "")
	if suspended, cached := s.suspended.Get(key); cached {
		return suspended, s.suspensionReason(ctx, tenantID, suspended)
	}

	var suspended bool
	var reason string
	err := s.db.QueryRow(ctx,
		`SELECT suspended_at IS NOT NULL OR deletion_scheduled_at IS NOT NULL,
		        suspension_reason
		   FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&suspended, &reason)
	if errors.Is(err, pgx.ErrNoRows) {
		// An organisation that is not there is not one anybody may act in.
		// This is the state a session held across a completed deletion is in.
		return true, "this organisation no longer exists"
	}
	if err != nil {
		// Fail open, and say so loudly. The alternative — refusing every
		// request on this replica because one query failed — turns a database
		// hiccup into an outage for organisations that are perfectly fine.
		slog.Error("could not check whether the organisation is suspended",
			"tenant_id", tenantID, "error", err)
		return false, ""
	}

	s.suspended.Put(key, suspended)
	return suspended, reason
}

// suspensionReason fetches the sentence to show, only when there is one to
// show. The cache holds the boolean rather than the text so that a reason
// edited in the console is never served from memory.
func (s *Server) suspensionReason(ctx context.Context, tenantID string, suspended bool) string {
	if !suspended {
		return ""
	}
	var reason string
	_ = s.db.QueryRow(ctx, `SELECT suspension_reason FROM tenants WHERE id = $1::uuid`,
		tenantID).Scan(&reason)
	return reason
}

// refuseIfSuspended answers 403 and reports true when the organisation is
// closed. Layered into authMiddleware, so every authenticated route on the
// platform is covered by one check rather than by each handler remembering.
func (s *Server) refuseIfSuspended(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	suspended, reason := s.tenantSuspended(r.Context(), tenantID)
	if !suspended {
		return false
	}
	message := ErrTenantSuspended.Error()
	if reason != "" {
		message += ": " + reason
	}
	// 403 rather than 401: the session is valid and signing in again will not
	// help, and a client that treats 401 as "log in again" would otherwise put
	// somebody in a loop through a login screen that also refuses them.
	httpx.Error(w, http.StatusForbidden, message)
	return true
}

// Quotas.
//
// One limit is enforced today — the number of people — because it is the only
// one this platform can currently count. Storage and AI calls are recorded by
// the console and shown as not-yet-enforced; CP-5's usage_events is what gives
// them numbers, and wiring them then is a change to this file rather than to
// every handler.

// ErrQuotaExceeded is a hard limit refusing to be crossed.
var ErrQuotaExceeded = errors.New("this organisation has reached its limit")

// checkUserQuota decides whether one more person may join an organisation.
//
// Soft mode logs and allows; hard mode refuses. The count and the limit are
// read in one statement, so two people joining at the same moment cannot both
// see the last free place — the check is still not a lock, and it does not
// need to be: a limit exceeded by one because of a race is a warning on a
// screen, not a breach.
func (s *Server) checkUserQuota(ctx context.Context, tenantID string) error {
	var limit, current int
	var enforcement string
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(q.max_users, -1), COALESCE(q.enforcement, 'soft'),
		        (SELECT count(*) FROM memberships m WHERE m.tenant_id = $1::uuid)
		   FROM tenants t
		   LEFT JOIN tenant_quotas q ON q.tenant_id = t.id
		  WHERE t.id = $1::uuid`, tenantID).Scan(&limit, &enforcement, &current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		// Same reasoning as the suspension check: a failure here must not stop
		// people joining organisations that are nowhere near their limit.
		slog.Error("could not check the organisation's limits", "tenant_id", tenantID, "error", err)
		return nil
	}
	if limit < 0 || current < limit {
		return nil
	}

	if enforcement != "hard" {
		slog.Warn("an organisation is over its user limit",
			"tenant_id", tenantID, "limit", limit, "users", current)
		return nil
	}
	return ErrQuotaExceeded
}
