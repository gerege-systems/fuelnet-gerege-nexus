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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eid"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eidmongolia"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/jackc/pgx/v5"
	"golang.org/x/time/rate"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"invalid login credentials"}`, http.StatusBadRequest)
		return
	}

	// A user can hold memberships in several tenants. LIMIT 1 without an ORDER
	// BY let Postgres pick, so the same credentials could land in a different
	// tenant from one sign-in to the next — and the session, its audit trail
	// and every subsequent read are scoped to whichever it picked. Oldest
	// membership first makes it the same tenant every time.
	var userID, passwordHash, tenantID, name string
	var isAdmin bool
	err := s.db.QueryRow(r.Context(),
		`SELECT u.id, u.password_hash, u.name,
		        EXISTS (
		            SELECT 1 FROM membership_roles mr
		            JOIN roles r ON r.id=mr.role_id
		            WHERE mr.membership_id=m.id AND r.tenant_id=m.tenant_id
		              AND r.code='admin' AND r.active
		        ) AS is_admin,
		        m.tenant_id
		 FROM users u
		 JOIN memberships m ON m.user_id = u.id
		 WHERE u.email = $1
		 ORDER BY m.created_at, m.tenant_id
		 LIMIT 1`, req.Email).Scan(&userID, &passwordHash, &name, &isAdmin, &tenantID)

	if err != nil || !auth.CheckPasswordHash(req.Password, passwordHash) {
		audit.Record(r.Context(), "unknown", "anonymous", "auth.login_failed", "user", map[string]any{"email": req.Email})
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	token, expiresAt, err := s.issueSession(r, userID, tenantID, "password")
	if err != nil {
		http.Error(w, `{"error":"failed to establish session"}`, http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, token, expiresAt)

	audit.Record(r.Context(), tenantID, userID, "auth.login_success", "user", map[string]any{"email": req.Email})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"expires_at": expiresAt,
		"user": map[string]any{
			"id":        userID,
			"tenant_id": tenantID,
			"name":      name,
			"email":     req.Email,
			"is_admin":  isAdmin,
		},
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Logout previously only cleared the cookie; the token stayed valid
	// forever for anyone who had captured it.
	if token := auth.TokenFromRequest(r); token != "" {
		if err := s.sessions.Revoke(r.Context(), token); err != nil {
			slog.Error("failed to revoke session", "error", err)
		}
	}

	auth.ClearSessionCookie(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

// Sign-in and eID polling are budgeted separately, because they are different
// things happening at different rates.
//
// Signing in is the endpoint worth guessing against, and starting an eID
// session pushes a notification to a real person's phone, so it stays tight.
//
// Polling is neither: it needs a session ID the relying party only ever handed
// to whoever started that session, and it cannot be turned into an attempt at a
// second one. What it does do is repeat — once every eid.PollWindow for as long
// as a citizen takes to reach their phone. Sharing the sign-in budget meant a
// few citizens waiting behind one office or NAT address spent it between them,
// and the next person there could not sign in at all. So this is budgeted by
// how many of them may plausibly be waiting behind a single address at once.
const (
	loginRatePerMinute = 5
	loginBurst         = 5
	pollRatePerMinute  = 60
	pollBurst          = 15
)

func newLoginLimiter() *security.IPRateLimiter {
	return security.NewIPRateLimiter(rate.Limit(float64(loginRatePerMinute)/60.0), loginBurst)
}

func newPollLimiter() *security.IPRateLimiter {
	return security.NewIPRateLimiter(rate.Limit(float64(pollRatePerMinute)/60.0), pollBurst)
}

// issueSession creates a persisted session bound to the caller's IP and agent.
func (s *Server) issueSession(r *http.Request, userID, tenantID, method string) (string, time.Time, error) {
	return s.sessions.Create(r.Context(), userID, tenantID, method,
		r.UserAgent(), security.ClientIP(r))
}

// signInError carries a reason that is meant for the person signing in. Account
// linking also fails for reasons that are ours alone — a missing key, a broken
// query, a rejected hash — and those are logged, never rendered: the citizen
// once saw "bcrypt: password length exceeds 72 bytes" in the eID card.
type signInError struct{ msg string }

func (e signInError) Error() string { return e.msg }

// reportSignInFailure answers with the reason when it is the caller's to act
// on, and with a stable message otherwise.
func reportSignInFailure(w http.ResponseWriter, err error) {
	var visible signInError
	if errors.As(err, &visible) {
		writeJSONError(w, http.StatusForbidden, visible.Error())
		return
	}
	slog.Error("failed to link verified national identity", "error", err)
	writeJSONError(w, http.StatusInternalServerError, "Баталгаажсан eID хэрэглэгчийг Gerege Nexus бүртгэлтэй холбож чадсангүй")
}

// resolveNationalIdentityUser maps a verified national identity (E-ID / DAN)
// onto a local platform user.
//
// The previous implementation ran `SELECT id FROM users LIMIT 1` and granted
// is_admin unconditionally, i.e. any successful gateway response logged the
// caller in as an arbitrary — in practice the seeded admin — account.
func (s *Server) resolveNationalIdentityUser(ctx context.Context, email, regNumber string) (userID, tenantID string, err error) {
	if email != "" {
		err = s.db.QueryRow(ctx,
			`SELECT u.id::text, m.tenant_id::text
			   FROM users u
			   JOIN memberships m ON m.user_id = u.id
			  WHERE lower(u.email) = lower($1)
			  ORDER BY m.created_at, m.tenant_id
			  LIMIT 1`, email).Scan(&userID, &tenantID)
		if err == nil {
			return userID, tenantID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", "", err
		}
	}

	if config.IsProduction() {
		return "", "", signInError{fmt.Sprintf("no Gerege Nexus user is linked to national identity %s", regNumber)}
	}

	// Development convenience only: fall back to the seeded demo account so
	// the documented mock login flow keeps working locally.
	err = s.db.QueryRow(ctx,
		`SELECT u.id::text, m.tenant_id::text
		   FROM users u
		   JOIN memberships m ON m.user_id = u.id
		  ORDER BY u.created_at
		  LIMIT 1`).Scan(&userID, &tenantID)
	if err != nil {
		return "", "", fmt.Errorf("no platform user available for national identity login: %w", err)
	}
	slog.Warn("national identity login fell back to the demo account",
		"reg_number", regNumber, "email", email)
	return userID, tenantID, nil
}

// eidLinkingDigest derives the stable, non-PII handle for an eID subject. It
// doubles as the synthetic account's password preimage, so its length is not
// cosmetic: bcrypt rejects anything over 72 bytes outright, and a suffix that
// pushed it to 73 once failed every first-time eID sign-in.
func eidLinkingDigest(linkingKey, subject string) string {
	mac := hmac.New(sha256.New, []byte(linkingKey))
	_, _ = mac.Write([]byte("eid-mn:" + subject))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// linkEIDIdentity records who a signed-in user is to eID Mongolia.
//
// Qualified remote signing addresses the citizen by their ETSI semantics
// identifier, and until this row exists nothing on the platform knows how to
// reach the phone of the person who just authenticated. Without it every
// signature would make the citizen retype the registration number they had
// just proved — and a typo there would push a PIN2 prompt at somebody else.
//
// It is best effort on purpose. Sign-in has already succeeded by this point,
// and failing the login because a convenience row could not be written would
// trade a working session for a missing one.
func (s *Server) linkEIDIdentity(ctx context.Context, userID string, identity *eid.EIDIdentity) {
	if identity == nil {
		return
	}
	subject := strings.TrimSpace(identity.CivilID)
	if subject == "" {
		subject = strings.TrimSpace(identity.RegNumber)
	}
	if subject == "" {
		return
	}
	personEtsi := eidmongolia.PersonEtsi(subject)

	// The conflict target is person_etsi as well as user_id: one eID citizen
	// resolves to one platform account, and a second account claiming the same
	// identifier would silently split that person's signing history in two.
	if _, err := s.db.Exec(ctx,
		`INSERT INTO user_eid_identities
		     (user_id, civil_id, reg_number, person_etsi, given_name, surname, last_seen_at)
		 VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4, NULLIF($5,''), NULLIF($6,''), NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		     civil_id     = COALESCE(EXCLUDED.civil_id, user_eid_identities.civil_id),
		     reg_number   = COALESCE(EXCLUDED.reg_number, user_eid_identities.reg_number),
		     person_etsi  = EXCLUDED.person_etsi,
		     given_name   = COALESCE(EXCLUDED.given_name, user_eid_identities.given_name),
		     surname      = COALESCE(EXCLUDED.surname, user_eid_identities.surname),
		     last_seen_at = NOW()`,
		userID, identity.CivilID, identity.RegNumber, personEtsi,
		identity.FirstName, identity.LastName); err != nil {
		slog.Warn("could not link the eID identity to the platform account",
			"user_id", userID, "error", err)
	}
}

// resolveOrProvisionEIDUser links an eID subject to a stable, non-PII local
// identifier. JIT provisioning is opt-in per tenant and always receives the
// standard user role through the membership_default_role database trigger.
func (s *Server) resolveOrProvisionEIDUser(ctx context.Context, identity *eid.EIDIdentity) (userID, tenantID string, err error) {
	if identity.Email != "" {
		if userID, tenantID, err = s.resolveNationalIdentityUser(ctx, identity.Email, identity.RegNumber); err == nil {
			return userID, tenantID, nil
		}
	}
	subject := strings.TrimSpace(identity.CivilID)
	if subject == "" {
		subject = strings.TrimSpace(identity.RegNumber)
	}
	if subject == "" {
		return "", "", errors.New("eID identity carries neither a civil ID nor a registration number")
	}
	linkingKey := os.Getenv("EID_RP_SECRET")
	if linkingKey == "" {
		return "", "", errors.New("EID_RP_SECRET is unset, so no account-linking key is available")
	}
	digest := eidLinkingDigest(linkingKey, subject)
	syntheticEmail := "eid+" + digest[:32] + "@identity.invalid"
	if err = s.db.QueryRow(ctx,
		`SELECT u.id::text, m.tenant_id::text FROM users u JOIN memberships m ON m.user_id=u.id WHERE u.email=$1
		 ORDER BY m.created_at, m.tenant_id LIMIT 1`,
		syntheticEmail).Scan(&userID, &tenantID); err == nil {
		return userID, tenantID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}

	tenantSlug := strings.TrimSpace(os.Getenv("EID_JIT_TENANT_SLUG"))
	if tenantSlug == "" {
		return "", "", signInError{"eID identity is verified but account provisioning is disabled"}
	}
	if err = s.db.QueryRow(ctx, `SELECT id::text FROM tenants WHERE slug=$1`, tenantSlug).Scan(&tenantID); err != nil {
		return "", "", fmt.Errorf("eID provisioning tenant %q is unavailable: %w", tenantSlug, err)
	}
	name := strings.TrimSpace(identity.LastName + " " + identity.FirstName)
	if name == "" {
		name = "eID Mongolia хэрэглэгч"
	}
	// The synthetic account has no password login path. Keep the random-looking
	// preimage within bcrypt's strict 72-byte input limit.
	passwordHash, err := auth.HashPassword(digest)
	if err != nil {
		return "", "", err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = tx.QueryRow(ctx,
		`INSERT INTO users(email,password_hash,name,is_admin) VALUES($1,$2,$3,FALSE)
		 ON CONFLICT(email) DO UPDATE SET name=EXCLUDED.name RETURNING id::text`,
		syntheticEmail, passwordHash, name).Scan(&userID); err != nil {
		return "", "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO memberships(tenant_id,user_id) VALUES($1,$2) ON CONFLICT(tenant_id,user_id) DO NOTHING`, tenantID, userID); err != nil {
		return "", "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return userID, tenantID, nil
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var name, email string
	var tenantName string
	_ = s.db.QueryRow(r.Context(), `SELECT name, email FROM users WHERE id = $1`, claims.UserID).Scan(&name, &email)
	_ = s.db.QueryRow(r.Context(), `SELECT name FROM tenants WHERE id = $1`, claims.TenantID).Scan(&tenantName)

	// The effective grant of every role the member holds, so a screen can hide
	// what the caller may not do. Administrators bypass the check, so their
	// list stays empty rather than enumerating the whole catalog.
	granted := make([]string, 0)
	if !claims.IsAdmin {
		if permissions, permErr := s.permissions.GetUserPermissions(r.Context(), claims.TenantID, claims.UserID); permErr == nil {
			for code := range permissions {
				granted = append(granted, code)
			}
			sort.Strings(granted)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          claims.UserID,
		"tenant_id":   claims.TenantID,
		"tenant_name": tenantName,
		"name":        name,
		"email":       email,
		"is_admin":    claims.IsAdmin,
		"permissions": granted,
	})
}
