package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionCookieName is the cookie carrying the opaque session token.
const SessionCookieName = "session_token"

// DefaultSessionTTL bounds how long a single login stays valid.
const DefaultSessionTTL = 12 * time.Hour

// DefaultIdleTimeout ends a session nobody has used.
//
// It answers a different question from DefaultSessionTTL: not "how long may
// one sign-in last" but "how long may an unattended machine keep somebody
// signed in". Eight hours of work should never be interrupted, and a screen
// left in a shared office should not still be signed in after lunch.
//
// The desktop clients are why this is a duration rather than something
// tighter: a native client sitting open on somebody's second monitor is doing
// nothing for long stretches and must not be signed out for it. Deployments
// that want a stricter rule set SESSION_IDLE_TIMEOUT; setting it to 0 turns
// the timeout off and leaves only the absolute lifetime.
const DefaultIdleTimeout = 90 * time.Minute

// touchInterval is how stale last_seen_at may get before it is written again.
// Without it every authenticated read would carry a write.
const touchInterval = time.Minute

// IdleTimeoutFromEnv reads SESSION_IDLE_TIMEOUT as a Go duration ("45m", "2h").
// An unset or unreadable value gives the default; "0" turns it off.
func IdleTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("SESSION_IDLE_TIMEOUT"))
	if raw == "" {
		return DefaultIdleTimeout
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < 0 {
		slog.Warn("SESSION_IDLE_TIMEOUT is not a duration; using the default",
			"value", raw, "default", DefaultIdleTimeout)
		return DefaultIdleTimeout
	}
	return parsed
}

var ErrSessionInvalid = errors.New("session is invalid or expired")

// SessionStore issues and resolves opaque server-side session tokens.
//
// Tokens are 256 bits of crypto/rand output; only their SHA-256 digest is
// persisted, so a database leak cannot be replayed as a login.
type SessionStore struct {
	db   *pgxpool.Pool
	ttl  time.Duration
	idle time.Duration
}

func NewSessionStore(db *pgxpool.Pool, ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &SessionStore{db: db, ttl: ttl, idle: IdleTimeoutFromEnv()}
}

// TTL returns the configured session lifetime.
func (s *SessionStore) TTL() time.Duration { return s.ttl }

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// Create issues a new session and returns the plaintext token. The token is
// shown to the caller exactly once — only its digest is stored.
func (s *SessionStore) Create(ctx context.Context, userID, tenantID, authMethod, userAgent, ip string) (string, time.Time, error) {
	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().Add(s.ttl)
	if authMethod == "" {
		authMethod = "password"
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, tenant_id, auth_method, user_agent, ip_address, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		hashToken(token), userID, tenantID, authMethod, userAgent, ip, expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("persist session: %w", err)
	}

	return token, expiresAt, nil
}

// Resolve validates a token and returns the claims bound to it.
func (s *SessionStore) Resolve(ctx context.Context, token string) (UserClaims, error) {
	if token == "" {
		return UserClaims{}, ErrSessionInvalid
	}

	// Validating and touching in one statement, because this runs in front of
	// every authenticated request and a second round trip here is a second
	// round trip everywhere. The UPDATE writes at most once a minute per
	// session; the rest of the time its WHERE matches nothing and the CTE costs
	// an index lookup.
	//
	// idleCutoff is passed rather than computed in SQL so that turning the
	// timeout off is a value this query does not have to branch on.
	idleCutoff := time.Time{}
	if s.idle > 0 {
		idleCutoff = time.Now().Add(-s.idle)
	}

	var claims UserClaims
	err := s.db.QueryRow(ctx,
		`WITH live AS (
		    SELECT id FROM sessions
		     WHERE token_hash = $1
		       AND revoked_at IS NULL
		       AND expires_at > NOW()
		       AND ($2::timestamptz IS NULL OR last_seen_at > $2)
		), touched AS (
		    UPDATE sessions SET last_seen_at = NOW()
		     WHERE id IN (SELECT id FROM live)
		       AND last_seen_at < NOW() - $3::interval
		)
		SELECT s.user_id::text, s.tenant_id::text, u.email,
		        EXISTS (
		            SELECT 1 FROM memberships m
		            JOIN membership_roles mr ON mr.membership_id=m.id
		            JOIN roles r ON r.id=mr.role_id
		            WHERE m.tenant_id=s.tenant_id AND m.user_id=s.user_id
		              AND r.tenant_id=s.tenant_id AND r.code='admin' AND r.active
		        ) AS is_admin
		   FROM sessions s
		   JOIN live ON live.id = s.id
		   JOIN users u ON u.id = s.user_id
		   JOIN memberships sm ON sm.tenant_id=s.tenant_id AND sm.user_id=s.user_id`,
		hashToken(token), nullableTime(idleCutoff), touchInterval.String()).
		Scan(&claims.UserID, &claims.TenantID, &claims.Email, &claims.IsAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserClaims{}, ErrSessionInvalid
		}
		return UserClaims{}, fmt.Errorf("resolve session: %w", err)
	}

	return claims, nil
}

// Revoke invalidates a single session. Revoking an unknown token is a no-op so
// that logout never leaks whether a token existed.
func (s *SessionStore) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.db.Exec(ctx,
		`UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`,
		hashToken(token))
	return err
}

// RevokeAllForUser ends every session a person holds, on every device.
//
// It is what somebody needs after leaving a session open on a machine they no
// longer have, and what an operator needs when an account is suspected. The
// caller's own session goes with the rest: "sign out everywhere" that leaves
// the browser it was pressed in signed in is not what it says.
func (s *SessionStore) RevokeAllForUser(ctx context.Context, userID string) (int64, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE sessions SET revoked_at = NOW()
		  WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`, userID)
	if err != nil {
		return 0, fmt.Errorf("revoke every session: %w", err)
	}
	return tag.RowsAffected(), nil
}

// nullableTime turns the zero time into a SQL NULL, which is how the resolve
// query says "no idle timeout" without a second version of itself.
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// DeleteExpired purges sessions that can no longer be used.
func (s *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM sessions WHERE expires_at < NOW() - INTERVAL '7 days'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// TokenFromRequest extracts a session token from the cookie or the
// Authorization header, in that order.
func TokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	const prefix = "Bearer "
	if header := r.Header.Get("Authorization"); len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return ""
}

// SetSessionCookie writes the session cookie with hardened attributes.
func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   IsProduction(),
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearSessionCookie expires the session cookie on the client.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   IsProduction(),
		SameSite: http.SameSiteStrictMode,
	})
}

// IsProduction reports whether the process runs with ENVIRONMENT=production.
func IsProduction() bool {
	return os.Getenv("ENVIRONMENT") == "production"
}
