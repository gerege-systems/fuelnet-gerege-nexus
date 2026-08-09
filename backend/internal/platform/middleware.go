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
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := auth.TokenFromRequest(r)
		if token == "" {
			http.Error(w, `{"error":"unauthorized: missing session token"}`, http.StatusUnauthorized)
			return
		}

		claims, err := s.sessions.Resolve(r.Context(), token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized: invalid or expired session"}`, http.StatusUnauthorized)
			return
		}

		ctx := auth.WithUserContext(r.Context(), claims)
		ctx = tenant.WithTenantID(ctx, claims.TenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdmin gates tenant-administrative endpoints. It must be layered after
// authMiddleware.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := auth.UserFromContext(r.Context())
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if !claims.IsAdmin {
			http.Error(w, `{"error":"forbidden: tenant administrator role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) appGateMiddleware(appID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, err := tenant.FromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if app is installed and enabled for this tenant
			var enabled bool
			err = s.db.QueryRow(r.Context(),
				`SELECT enabled FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
				tenantID, appID).Scan(&enabled)

			if err != nil || !enabled {
				http.Error(w, `{"error":"forbidden: app module `+appID+` is not installed or enabled for this tenant"}`, http.StatusForbidden)
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
		"io.example.contacts": "contacts", "io.example.products": "products",
		"io.example.inventory": "inventory", "io.example.billing": "billing",
		"io.example.developer_portal": "developer",
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
