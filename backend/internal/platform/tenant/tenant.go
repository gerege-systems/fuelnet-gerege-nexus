package tenant

import (
	"context"
	"errors"
	"net/http"
)

type contextKey string

const tenantIDKey contextKey = "tenant_id"

var ErrTenantMissing = errors.New("tenant context is missing")

// WithTenantID injects tenant_id into context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// FromContext extracts tenant_id from context.
func FromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", ErrTenantMissing
	}
	return tenantID, nil
}

// RequireTenantMiddleware ensures a valid tenant ID exists in request context.
func RequireTenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := FromContext(r.Context())
		if err != nil || tenantID == "" {
			http.Error(w, `{"error":"unauthorized: missing tenant context"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
