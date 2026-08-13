package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	userContextKey contextKey = "authenticated_user"
)

var ErrUnauthorized = errors.New("unauthorized request")

type UserClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
	// AllowedTenantIDs is every organisation this session reads across, and
	// TenantID is always among them. Empty means only TenantID, which is what
	// every session is until somebody asks for more.
	//
	// IsAdmin is deliberately not widened with it: being an administrator is a
	// role held in one organisation, and holding it in the parent says nothing
	// about the subsidiary.
	AllowedTenantIDs []string `json:"allowed_tenant_ids,omitempty"`
	// Impersonated says this session belongs to a platform operator acting as
	// this person rather than to the person themselves.
	//
	// It travels with the claims rather than being looked up where it is
	// needed, because three separate things depend on it — the banner the
	// shell shows, the mark on every audit row, and the fact that /me reports
	// it — and a fact three consumers each fetch for themselves is a fact they
	// can disagree about.
	Impersonated bool `json:"impersonated,omitempty"`
	// ImpersonatedBy is the operator's id. Never their address: this reaches
	// the browser of somebody whose organisation is being looked at, and who
	// is looking is answered by the organisation's own audit trail, in full,
	// rather than by a claim in a JSON body.
	ImpersonatedBy string `json:"-"`
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func WithUserContext(ctx context.Context, claims UserClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

func UserFromContext(ctx context.Context) (UserClaims, error) {
	claims, ok := ctx.Value(userContextKey).(UserClaims)
	if !ok || claims.UserID == "" {
		return UserClaims{}, ErrUnauthorized
	}
	return claims, nil
}
