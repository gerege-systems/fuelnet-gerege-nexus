package tenant

import (
	"context"
	"errors"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
)

type contextKey string

const tenantIDKey contextKey = "tenant_id"

// allowedKey carries every organisation the caller's session is active in.
//
// tenantIDKey stays the one they are acting in — the organisation a new row is
// written into. This is the wider set they may read across, and it is empty for
// almost every session, which is what keeps the behaviour identical to the day
// before it existed.
const allowedKey contextKey = "allowed_tenant_ids"

var ErrTenantMissing = errors.New("tenant context is missing")

// WithTenantID injects tenant_id into context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// WithAllowed records the organisations this request may read across.
//
// The caller is expected to have taken the list from the session row, which is
// the only place it is written and only after the same membership check that
// decides the acting tenant. Never build it from something a request said.
func WithAllowed(ctx context.Context, tenantIDs []string) context.Context {
	if len(tenantIDs) == 0 {
		return ctx
	}
	return context.WithValue(ctx, allowedKey, tenantIDs)
}

// Allowed returns the organisations this request may read across, which is
// always at least the one it is acting in.
//
// Handlers that mean "this organisation" should keep using Require. This is for
// the few lists that are deliberately a group view.
func Allowed(ctx context.Context) []string {
	current, _ := ctx.Value(tenantIDKey).(string)
	set, _ := ctx.Value(allowedKey).([]string)
	if len(set) == 0 {
		if current == "" {
			return nil
		}
		return []string{current}
	}
	return set
}

// Without strips the tenant from a context, putting the caller back on the
// platform path — the login role, outside the row-level policies (see
// internal/platform/dbguard).
//
// It is for the handful of questions that are genuinely about a person rather
// than about a tenant: which tenants somebody may act for, and moving their
// session to one of them. Both read `memberships`, which carries a tenant_id
// and is therefore filtered to the current tenant for everybody else — the
// answer would be "the tenant you are already in", every time.
//
// Reach for this only where crossing tenants is the point, and never with an
// id that arrived from a request without a membership check behind it.
func Without(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, allowedKey, []string(nil))
	return context.WithValue(ctx, tenantIDKey, "")
}

// FromContext extracts tenant_id from context.
//
// Handlers should reach for Require instead. This is for callers that have a
// context but no ResponseWriter to answer on.
func FromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", ErrTenantMissing
	}
	return tenantID, nil
}

// Require resolves the caller's tenant, or answers 401 and reports false. A
// handler that gets false has already had its response written and must return.
//
// Fifty-seven handlers opened with the same four lines, and they had drifted:
// most said "unauthorized", three passed a bare 401 instead of the constant,
// and two answered "unauthorized tenant context" — which tells whoever is
// holding a bad session something about this platform's internals rather than
// about their request. One sentence, said once.
//
// This is not middleware, and there is deliberately none. A RequireTenantMiddleware
// existed here once and was mounted on no route at all, which made it read as a
// guard the platform relies on. What actually guards a request is authMiddleware,
// the only thing that puts a tenant into the context, and appGateMiddleware,
// which refuses the request when this fails.
func Require(w http.ResponseWriter, r *http.Request) (string, bool) {
	tenantID, err := FromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return tenantID, true
}
