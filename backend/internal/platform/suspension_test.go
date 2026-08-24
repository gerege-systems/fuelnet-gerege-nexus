package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/memo"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/tenant/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The tenant-facing half of CP-2, against a real schema: a suspended
// organisation is refused everywhere, an invitation can be redeemed once, and
// an impersonation produces a session that says what it is.
//
// These run through the actual middleware and the actual handlers rather than
// against the queries, because what is being tested is that the checks are
// *wired in* — a suspension the console records and the platform never reads
// would pass every query-level test there is.

func suspensionServer(t *testing.T) (*Server, *pgxpool.Pool) {
	t.Helper()
	pool := lockoutPool(t)
	return &Server{
		db:        pool,
		sessions:  auth.NewSessionStore(pool, time.Hour),
		suspended: memo.New[bool](suspendedTTL),
	}, pool
}

// tenantWithMember creates an organisation, somebody in it, and a live session.
func tenantWithMember(t *testing.T, pool *pgxpool.Pool) (tenantID, userID, token string) {
	t.Helper()
	ctx := context.Background()
	slug := fmt.Sprintf("susp-%d", time.Now().UnixNano())

	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ($1, $1) RETURNING id::text`, slug).Scan(&tenantID); err != nil {
		t.Fatalf("create the organisation: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, 'x', 'Member') RETURNING id::text`,
		slug+"@identity.invalid").Scan(&userID); err != nil {
		t.Fatalf("create the person: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1::uuid`, userID) })

	if _, err := pool.Exec(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1::uuid, $2::uuid)`, tenantID, userID); err != nil {
		t.Fatalf("add the membership: %v", err)
	}

	token, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("mint a token: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, tenant_id, expires_at)
		 VALUES ($1, $2::uuid, $3::uuid, NOW() + INTERVAL '1 hour')`,
		auth.HashSessionToken(token), userID, tenantID); err != nil {
		t.Fatalf("create the session: %v", err)
	}
	return tenantID, userID, token
}

// A live session in a suspended organisation is refused by the middleware
// every other authenticated route sits behind.
func TestASuspendedOrganisationIsRefusedByEveryRoute(t *testing.T) {
	server, pool := suspensionServer(t)
	tenantID, _, token := tenantWithMember(t, pool)

	reached := false
	guarded := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	recorder := httptest.NewRecorder()
	guarded.ServeHTTP(recorder, request)
	if !reached || recorder.Code != http.StatusOK {
		t.Fatalf("a healthy organisation was refused: %d", recorder.Code)
	}

	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET suspended_at = NOW(), suspension_reason = 'unpaid' WHERE id = $1::uuid`,
		tenantID); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	server.suspended = memo.New[bool](suspendedTTL) // what the invalidation bus does across replicas

	reached = false
	request = httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
	recorder = httptest.NewRecorder()
	guarded.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("a suspended organisation answered %d, want 403", recorder.Code)
	}
	if reached {
		t.Fatal("the request reached the handler")
	}
	// The reason is shown: somebody being refused should know whether to call
	// their account manager or their administrator.
	if !strings.Contains(recorder.Body.String(), "unpaid") {
		t.Fatalf("the refusal did not carry the reason: %s", recorder.Body.String())
	}
}

// Signing in is refused at the one place every method of signing in passes
// through, so no route needs to remember.
func TestSigningInToASuspendedOrganisationIsRefused(t *testing.T) {
	server, pool := suspensionServer(t)
	tenantID, userID, _ := tenantWithMember(t, pool)

	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET suspended_at = NOW() WHERE id = $1::uuid`, tenantID); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	if _, _, err := server.issueSession(request, userID, tenantID, "password"); err == nil {
		t.Fatal("a session was issued for a suspended organisation")
	}
}

// The invitation and reset link: one use, and the account's sessions go with
// the password change.
func TestACredentialLinkWorksOnceAndEndsTheSessions(t *testing.T) {
	server, pool := suspensionServer(t)
	_, userID, token := tenantWithMember(t, pool)
	ctx := context.Background()

	link, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("mint a link: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO credential_grants (user_id, purpose, token_hash, expires_at)
		 VALUES ($1::uuid, 'invite', $2, NOW() + INTERVAL '1 hour')`,
		userID, hashRecoveryToken(link)); err != nil {
		t.Fatalf("issue the grant: %v", err)
	}

	redeem := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"token": link, "password": "a good long password"})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/credential/redeem",
			strings.NewReader(string(body)))
		recorder := httptest.NewRecorder()
		server.handleCredentialRedeem(recorder, request)
		return recorder
	}

	if recorder := redeem(); recorder.Code != http.StatusOK {
		t.Fatalf("redeeming answered %d: %s", recorder.Code, recorder.Body.String())
	}
	// The session the account already had is gone: a password given to
	// somebody who was locked out is usually a password somebody else knew.
	if _, err := server.sessions.Resolve(ctx, token); err == nil {
		t.Fatal("a session survived the password being set")
	}
	// And the link is spent.
	if recorder := redeem(); recorder.Code != http.StatusGone {
		t.Fatalf("reusing the link answered %d, want 410", recorder.Code)
	}
}

// The impersonation handover: one use, a session that ends when the visit
// does, and claims that say whose it really is.
func TestAnImpersonationHandoverProducesAMarkedSession(t *testing.T) {
	server, pool := suspensionServer(t)
	tenantID, userID, _ := tenantWithMember(t, pool)
	ctx := context.Background()

	var operatorID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO operator_accounts (email, name, role, password_hash)
		 VALUES ($1, 'Operator', 'support', 'x') RETURNING id::text`,
		fmt.Sprintf("imp-%d@controlplane.test", time.Now().UnixNano())).Scan(&operatorID); err != nil {
		t.Fatalf("create the operator: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM operator_accounts WHERE id=$1::uuid`, operatorID)
	})

	handover, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("mint a handover: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO operator_impersonations
		     (operator_id, operator_email, tenant_id, user_id, reason,
		      handover_hash, handover_expires_at, ends_at)
		 VALUES ($1::uuid, 'operator@example.mn', $2::uuid, $3::uuid, 'a support call',
		         $4, NOW() + INTERVAL '1 minute', NOW() + INTERVAL '30 minutes')`,
		operatorID, tenantID, userID, hashRecoveryToken(handover)); err != nil {
		t.Fatalf("record the impersonation: %v", err)
	}

	redeem := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"token": handover})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/impersonation/redeem",
			strings.NewReader(string(body)))
		recorder := httptest.NewRecorder()
		server.handleImpersonationRedeem(recorder, request)
		return recorder
	}

	recorder := redeem()
	if recorder.Code != http.StatusOK {
		t.Fatalf("the handover answered %d: %s", recorder.Code, recorder.Body.String())
	}

	var sessionToken string
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == auth.SessionCookieName {
			sessionToken = cookie.Value
		}
	}
	if sessionToken == "" {
		t.Fatal("no session cookie was set")
	}

	claims, err := server.sessions.Resolve(ctx, sessionToken)
	if err != nil {
		t.Fatalf("the borrowed session does not resolve: %v", err)
	}
	// The flag the banner is drawn from, and the mark every audit row written
	// from this session will carry.
	if !claims.Impersonated || claims.ImpersonatedBy != operatorID {
		t.Fatalf("the session does not know it is an impersonation: %+v", claims)
	}
	if claims.UserID != userID || claims.TenantID != tenantID {
		t.Fatalf("the session is for the wrong person: %+v", claims)
	}

	// The organisation is told, in its own trail, that somebody came in.
	var started int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		  WHERE tenant_id = $1::uuid AND action = 'security.impersonation.started'`,
		tenantID).Scan(&started); err != nil {
		t.Fatalf("read the organisation's trail: %v", err)
	}
	if started != 1 {
		t.Fatalf("the organisation's trail has %d rows for the visit, want 1", started)
	}

	// A handover is worth one journey.
	if second := redeem(); second.Code != http.StatusGone {
		t.Fatalf("reusing the handover answered %d, want 410", second.Code)
	}
}
