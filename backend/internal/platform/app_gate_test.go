/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
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

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/cache"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
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

// An app's routes are behind its installation, and another app's are not.
//
// Both halves are the point. The first is what the app gate is for: a tenant
// that has not installed an app must be refused its endpoints, whatever
// permission the caller holds. The second is what makes the gate safe — one
// app being absent must not make another unusable.
//
// This was written about the e-Government link, whose routes had been platform
// routes before it became an app. That module moved to client-gerege-nexus on
// 2026-08-23; documents, the organisation, Өртөө's task board and the reports
// app followed it the same day. The claim has now outlived five of its
// subjects, which is the argument for keeping it: what is being asserted is the
// gate, not any app in particular. It asks about the assistant and the SSO
// client register today.
func TestAnAppsRoutesAreBehindItsInstallationAndAnothersAreNot(t *testing.T) {
	f := newGateFixture(t)

	// Nothing installed yet: sso-clients is refused.
	if res := f.do(t, http.MethodGet, "/api/v1/sso-clients/apps/", ""); res.Code != http.StatusForbidden {
		t.Fatalf("sso-clients answered %d without the app installed; expected 403: %s",
			res.Code, res.Body.String())
	}

	// Platform routes are unaffected by whether optional apps are installed.
	res := f.do(t, http.MethodGet, "/api/v1/admin/ai/prompts", "")
	if res.Code != http.StatusOK {
		t.Fatalf("platform core AI answered %d: %s", res.Code, res.Body.String())
	}

	// And with the app installed the gate opens.
	f.install(t, "io.gerege.nexus.sso_clients")
	res = f.do(t, http.MethodGet, "/api/v1/sso-clients/apps/", "")
	if res.Code == http.StatusForbidden {
		t.Fatalf("sso-clients was refused after installation: %s", res.Body.String())
	}
	if res.Code == http.StatusNotFound {
		t.Fatalf("sso-clients answered 404; this test asserts nothing unless the route is served")
	}
}
