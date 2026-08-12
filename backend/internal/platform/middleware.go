/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package platform provides the core HTTP Server orchestrator, routing table,
 * authentication middleware, and app installer wiring.
 */

package platform

import (
	"log/slog"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
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

		ctx := auth.WithUserContext(r.Context(), claims)
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

func appRequestPermission(appID, method, path string) string {
	prefixes := map[string]string{
		"io.gerege.nexus.contacts": "contacts", "io.gerege.nexus.products": "products",
		"io.gerege.nexus.inventory": "inventory", "io.gerege.nexus.billing": "billing",
		"io.gerege.nexus.developer_portal": "developer",
	}
	prefix := prefixes[appID]
	if prefix == "" {
		return ""
	}
	if method == http.MethodGet || method == http.MethodHead {
		return prefix + ".read"
	}
	return prefix + ".manage"
}
