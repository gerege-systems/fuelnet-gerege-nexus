package auth

import (
	"context"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"golang.org/x/crypto/bcrypt"
)

// ErrUnauthorized is returned when a context carries no caller.
var ErrUnauthorized = nexus.ErrUnauthenticated

// UserClaims is who the platform decided the caller is.
//
// The type and its context key moved to `backend/pkg/nexus`: a module in
// another repository names the caller on nearly every handler, and it cannot
// import a package under internal/ to do it. This is an alias rather than a
// second struct, so the value the session middleware writes is the value a
// module reads — two identically shaped types would not have been.
type UserClaims = nexus.UserClaims

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// WithUserContext injects the caller's claims into a context.
func WithUserContext(ctx context.Context, claims UserClaims) context.Context {
	return nexus.WithUser(ctx, claims)
}

// UserFromContext returns who the platform decided the caller is.
func UserFromContext(ctx context.Context) (UserClaims, error) {
	return nexus.UserFromContext(ctx)
}
