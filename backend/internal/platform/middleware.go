/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Package platform provides the core HTTP Server orchestrator, routing table,
 * authentication middleware, and app installer wiring.
 */

package platform

import (
	"log/slog"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/flags"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := auth.TokenFromRequest(r)
		if token == "" {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized: missing session token")
			return
		}

		claims, err := s.sessions.Resolve(r.Context(), token)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized: invalid or expired session")
			return
		}

		// A suspended organisation is one nobody may act in, including the
		// people already signed in to it. Suspending revokes their sessions in
		// the same transaction, so this is the belt to that braces: a client
		// holding a token issued a moment before, or a replica whose cache is
		// a few seconds behind, is refused here.
		if s.refuseIfSuspended(w, r, claims.TenantID) {
			return
		}

		// Maintenance is checked after suspension and before anything else,
		// and only for writes: the point of a maintenance window is that
		// people can still see what they need.
		if s.refuseIfReadOnly(w, r, claims.TenantID) {
			return
		}

		ctx := auth.WithUserContext(r.Context(), claims)
		if claims.Impersonated {
			// Everything this request records is marked as ours. It is done
			// here, once, rather than in the handlers that write audit rows:
			// there are dozens of them, in every module, and a mark that each
			// of them has to remember is a mark that is missing from whichever
			// one somebody writes next.
			ctx = audit.MarkImpersonated(ctx, claims.ImpersonatedBy)
		}
		ctx = tenant.WithTenantID(ctx, claims.TenantID)
		// The organisations this session reads across, straight from the
		// session row. dbguard turns it into the policy's array; almost every
		// session carries none and behaves exactly as it always has.
		ctx = tenant.WithAllowed(ctx, claims.AllowedTenantIDs)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdmin gates tenant-administrative endpoints. It must be layered after
// authMiddleware.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := auth.UserFromContext(r.Context())
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !claims.IsAdmin {
			httpx.Error(w, http.StatusForbidden, "forbidden: tenant administrator role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) appGateMiddleware(appID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := tenant.Require(w, r)
			if !ok {
				return
			}

			// Whether a tenant has this app — see appInstalled, which the
			// authorization endpoint asks the same question of on behalf of
			// apps that run outside this binary. A database that cannot answer
			// refuses here rather than admitting: this is the check that keeps
			// one tenant out of another tenant's application.
			// A module's kill switch, before the installation check: an app
			// being switched off platform-wide is an operator's decision
			// during an incident, and it should not depend on what any
			// organisation has installed.
			if flags.Enabled(r.Context(), flags.ModuleKillSwitch(appID)) {
				httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{
					"error": "энэ модулийг түр хугацаанд унтраасан байна",
				})
				return
			}

			enabled, err := s.appInstalled(r.Context(), tenantID, appID)
			if err != nil {
				slog.Error("could not check the app installation", "error", err,
					"app_id", appID, "tenant_id", tenantID)
			}

			if !enabled {
				httpx.Error(w, http.StatusForbidden, "forbidden: app module "+appID+" is not installed or enabled for this tenant")
				return
			}

			// Model-level access rights are additive across all assigned roles,
			// matching Odoo's ir.model.access behaviour. Government workflow has
			// its own action- and unit-aware permission checks.
			if permission := appRequestPermission(appID, r.Method, r.URL.Path); permission != "" {
				rbac.RequirePermission(s.permissions, permission)(next).ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

// appRequestPermission asks the module which permission this request needs.
//
// The mapping used to be a literal in this function listing five apps by ID.
// That worked while every module was compiled from this repository and broke
// quietly the moment one was not: an extracted module simply fell out of the
// map, and falling out of the map means no permission is required. See
// nexus.AccessPolicy for why the answer now travels with the module.
func appRequestPermission(appID, method, path string) string {
	mod, found := lookupModule(appID)
	if !found {
		return ""
	}
	prefix := nexus.RoutePermissionPrefixOf(mod)
	if prefix == "" {
		return ""
	}
	if method == http.MethodGet || method == http.MethodHead {
		return prefix + ".read"
	}
	return prefix + ".manage"
}
