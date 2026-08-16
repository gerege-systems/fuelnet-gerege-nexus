/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Which organisation a request is acting for.
 *
 * The context key and the accessors moved to `backend/pkg/nexus`: a module in
 * another repository reads the tenant on every handler, and it cannot import a
 * package under internal/ to do it. These forward rather than reimplement,
 * which is the point — two packages each holding their own context key would
 * write and read different values and the second one would always be empty.
 */

package tenant

import (
	"context"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// ErrTenantMissing is returned when a context carries no acting organisation.
var ErrTenantMissing = nexus.ErrTenantMissing

// WithTenantID injects tenant_id into context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return nexus.WithTenantID(ctx, tenantID)
}

// WithAllowed records the organisations this request may read across.
func WithAllowed(ctx context.Context, tenantIDs []string) context.Context {
	return nexus.WithAllowedTenants(ctx, tenantIDs)
}

// Allowed returns the organisations this request may read across.
func Allowed(ctx context.Context) []string { return nexus.AllowedTenants(ctx) }

// Without strips the tenant from a context, putting the caller back on the
// platform path — the login role, outside the row-level policies (see
// internal/platform/dbguard).
func Without(ctx context.Context) context.Context { return nexus.WithoutTenant(ctx) }

// FromContext extracts tenant_id from context.
func FromContext(ctx context.Context) (string, error) { return nexus.TenantID(ctx) }

// Require resolves the caller's tenant, or answers 401 and reports false. A
// handler that gets false has already had its response written and must return.
func Require(w http.ResponseWriter, r *http.Request) (string, bool) {
	return nexus.RequireTenant(w, r)
}
