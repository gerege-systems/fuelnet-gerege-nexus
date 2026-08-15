/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The app gate, tested through the app that made it worth testing.
 *
 * Every compiled module is mounted behind appGateMiddleware, and until now
 * nothing asserted what it does — the gate was one line in registerAppModuleRoutes
 * and the closest thing to a test was the route-policy sweep, which only asks
 * whether a stranger is refused. The e-Government app is the one where the
 * answer matters most: its endpoints read the national registry, they used to
 * be platform routes reachable by any tenant holding the permission, and the
 * move behind the gate is the thing that changed.
 *
 *	AUTH_TEST_DATABASE_URL=postgres://... go test ./internal/platform/...
 */

package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/cache"
	"github.com/jackc/pgx/v5/pgxpool"
)

type gateFixture struct {
	server   *Server
	pool     *pgxpool.Pool
	tenantID string
	userID   string
	token    string
}

func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set AUTH_TEST_DATABASE_URL to a migrated test database to run the app gate tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	t.Setenv("APP_CATALOG_URL", "")
	server, err := NewServer(pool, filepath.FromSlash("../../../catalog/apps.json"),
		cache.NewBus(ctx, nil))
	if err != nil {
		t.Fatalf("build the server: %v", err)
	}

	f := &gateFixture{server: server, pool: pool}
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('gate-' || substr(gen_random_uuid()::text, 1, 8), 'Gate test')
		 RETURNING id::text`).Scan(&f.tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, f.tenantID) })

	// An administrator, so nothing below is refused for want of a permission —
	// what is under test is the installation check, and a 403 that could be
	// either would prove nothing.
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, is_admin)
		 VALUES ('gate-' || substr(gen_random_uuid()::text, 1, 8) || '@example.com', 'x', 'Gate Admin', TRUE)
		 RETURNING id::text`).Scan(&f.userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, f.userID) })

	var membershipID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1, $2) RETURNING id::text`,
		f.tenantID, f.userID).Scan(&membershipID); err != nil {
		t.Fatalf("membership: %v", err)
	}

	// `is_admin` on the session is the tenant's admin role, not the global flag
	// on the user row — see SessionStore.Resolve. Without the role the caller is
	// an ordinary member and every assertion below would be reading a permission
	// refusal as an app-gate refusal.
	if _, err := pool.Exec(ctx,
		`WITH r AS (
		     INSERT INTO roles (tenant_id, code, name) VALUES ($1, 'admin', 'Administrator')
		     ON CONFLICT (tenant_id, code) DO UPDATE SET active = TRUE
		     RETURNING id
		 )
		 INSERT INTO membership_roles (membership_id, role_id)
		 SELECT $2::uuid, r.id FROM r ON CONFLICT DO NOTHING`,
		f.tenantID, membershipID); err != nil {
		t.Fatalf("admin role: %v", err)
	}

	token, _, err := server.sessions.Create(ctx, f.userID, f.tenantID, "password", "go-test", "127.0.0.1")
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	f.token = token
	return f
}

func (f *gateFixture) do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: f.token})
	rec := httptest.NewRecorder()
	f.server.router.ServeHTTP(rec, req)
	return rec
}

// install writes the installation row directly.
//
// Not through the installer: that grants permissions and resolves dependencies,
// which is a different mechanism with its own tests, and here it would put
// three more moving parts between the assertion and the thing being asserted.
func (f *gateFixture) install(t *testing.T, appID string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`INSERT INTO app_installations (tenant_id, app_id, installed_version, status, enabled)
		 VALUES ($1, $2, '1.0.0', 'installed', TRUE)
		 ON CONFLICT (tenant_id, app_id) DO UPDATE SET enabled = TRUE, status = 'installed'`,
		f.tenantID, appID); err != nil {
		t.Fatalf("install %s: %v", appID, err)
	}
	// The gate caches its answer for thirty seconds, which is longer than this
	// test takes.
	f.server.forgetAppGate(f.tenantID)
}

// The e-Government endpoints are behind the app, and contacts does not care.
//
// Both halves are the point. The first is the move: these two URLs used to be
// platform routes that any tenant could reach with the right permission, and a
// tenant that has removed the app must now be refused. The second is what makes
// the move safe to ship — contacts offers registry auto-fill and must not
// become unusable on a deployment that has no state integration, so it is
// deliberately not a dependent, and its own endpoints answer regardless.
func TestTheStateRegistryIsBehindItsAppAndContactsIsNot(t *testing.T) {
	f := newGateFixture(t)

	// Nothing installed yet.
	for _, target := range []string{"/api/v1/egov/citizen", "/api/v1/xyp/citizen"} {
		res := f.do(t, http.MethodPost, target, `{"reg_number":"AA90010111"}`)
		if res.Code != http.StatusForbidden {
			t.Fatalf("%s answered %d without the app installed; expected 403: %s",
				target, res.Code, res.Body.String())
		}
	}
	if res := f.do(t, http.MethodGet, "/api/v1/egov/connections", ""); res.Code != http.StatusForbidden {
		t.Fatalf("connections answered %d without the app installed; expected 403", res.Code)
	}

	// Contacts, on the other hand, is unaffected by the state integration being
	// absent. What matters here is that the answer does not depend on egov.
	//
	// The app installed to open that gate is the directory, not a contacts app:
	// the contact register was absorbed into io.gerege.nexus.organisation, so
	// that is the installation its routes are now gated on. The claim under
	// test did not move with it.
	f.install(t, "io.gerege.nexus.organisation")
	if res := f.do(t, http.MethodGet, "/api/v1/contacts", ""); res.Code == http.StatusForbidden {
		t.Fatalf("contacts was refused while the e-Government app was absent: %s", res.Body.String())
	}

	// And with the app installed the gate opens. A 403 here would mean the
	// permission was refused rather than the app; the caller is a tenant
	// administrator, who bypasses permission checks.
	f.install(t, "io.gerege.nexus.egov")
	res := f.do(t, http.MethodGet, "/api/v1/egov/connections", "")
	if res.Code != http.StatusOK {
		t.Fatalf("connections answered %d with the app installed: %s", res.Code, res.Body.String())
	}
}
