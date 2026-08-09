package tenant

import (
	"context"
	"errors"
)

type contextKey string

const tenantIDKey contextKey = "tenant_id"

var ErrTenantMissing = errors.New("tenant context is missing")

// WithTenantID injects tenant_id into context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// FromContext extracts tenant_id from context.
//
// There is deliberately no RequireTenantMiddleware here. One existed and was
// never mounted on a single route, which made it look like a guard the platform
// relies on. The real guard is authMiddleware, which is the only thing that
// puts a tenant into the context at all, and appGateMiddleware, which refuses
// the request when this returns an error.
func FromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", ErrTenantMissing
	}
	return tenantID, nil
}
