package egov_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/egov"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/gerege"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/staterail"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The e-Government app, tested where the interesting parts are: what it refuses
// to hand out by default, what it answers when the state rails are fixtures,
// and whether the address it used to live at still works.
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/apps/egov/...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a migrated database to run the egov tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type fixture struct {
	pool     *pgxpool.Pool
	router   chi.Router
	tenantID string
	userID   string
}

// newFixture mounts the module with the gate already passed and the caller an
// administrator: what is under test is the module, not the middleware in front
// of it. The gate itself is tested in the platform package.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testPool(t)
	ctx := context.Background()

	f := &fixture{pool: pool}
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('egov-' || substr(gen_random_uuid()::text, 1, 8), 'Egov test')
		 RETURNING id::text`).Scan(&f.tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, f.tenantID) })

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, is_admin)
		 VALUES ('egov-' || substr(gen_random_uuid()::text, 1, 8) || '@example.com', 'x', 'Egov Admin', TRUE)
		 RETURNING id::text`).Scan(&f.userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, f.userID) })

	if _, err := pool.Exec(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1, $2)`, f.tenantID, f.userID); err != nil {
		t.Fatalf("membership: %v", err)
	}

	// Two wires the server puts in place at startup, and a test has to as well:
	// the pool audit.Record writes through, and the sink nexus.Audit forwards
	// to. Miss either and the call is only a log line — AUDIT_EVENT_UNSUNK for
	// the second — and the history screen has nothing to read, which is exactly
	// what this test is checking.
	audit.UseDatabase(pool)
	nexus.Provide[nexus.AuditSink](audit.Record)

	// Mock mode, which is the state a deployment without ХУР credentials is in
	// and the only one a test can be in.
	t.Setenv("XYP_MOCK_MODE", "true")
	rails := func() []staterail.Rail {
		return []staterail.Rail{{ID: "xyp", Name: "ХУР", Mode: "mock", Endpoint: "https://xyp.example.invalid"}}
	}
	module := egov.New(nexus.NewPlatform(pool, rbac.NewSQLPermissionStore(pool)),
		gerege.NewGeregeService(), rails)

	router := chi.NewRouter()
	module.RegisterRoutes(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := nexus.WithTenantID(r.Context(), f.tenantID)
			ctx = nexus.WithUser(ctx, nexus.UserClaims{UserID: f.userID, TenantID: f.tenantID, IsAdmin: true})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	f.router = router
	return f
}

func (f *fixture) do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

// The registry lookups are administrative, and stay administrative.
//
// This is a unit test on the module's own declaration rather than on a granted
// role, because the declaration is what the installer reads: before this app
// existed the two codes were granted to the administrator alone by migration
// 00024, and the installer's default rule — anything ending `.read` goes to
// every member — would have quietly widened that on install. If AdminOnly is
// ever dropped from one of these, everybody in the organisation can look up
// any citizen by registration number, and nothing else in the suite would say
// so.
func TestTheRegistryLookupsAreNotHandedToEveryMember(t *testing.T) {
	// No database and no permission store: this test reads what the module
	// declares, which it does before it is wired to anything.
	module := egov.New(nexus.NewPlatform(nil, nil), nil, nil)

	byCode := map[string]nexus.PermissionDefinition{}
	for _, permission := range module.Permissions() {
		byCode[permission.Code] = permission
	}

	for _, code := range []string{"egov.citizen.read", "egov.company.read"} {
		permission, declared := byCode[code]
		if !declared {
			t.Fatalf("%s is no longer declared by the module", code)
		}
		if !permission.AdminOnly {
			t.Errorf("%s is not AdminOnly: installing this app would grant it to every member", code)
		}
	}

	// The screen permission is the opposite case and is meant to be ordinary.
	if byCode["egov.read"].AdminOnly {
		t.Error("egov.read is AdminOnly, so nobody but an administrator could open the app")
	}
}

func TestALookupAnswersFromTheMockRailAndIsRecorded(t *testing.T) {
	f := newFixture(t)

	res := f.do(t, http.MethodPost, "/api/v1/egov/citizen", `{"reg_number":"AA90010111"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("lookup answered %d: %s", res.Code, res.Body.String())
	}
	var info gerege.CitizenInfo
	if err := json.Unmarshal(res.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.RegNumber != "AA90010111" {
		t.Fatalf("the reply is about somebody else: %+v", info)
	}

	// Asking is the thing that is kept. The history screen reads this table, so
	// a lookup that left no row would be a lookup nobody could account for.
	var recorded int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_events
		  WHERE tenant_id = $1 AND action = 'egov.citizen_queried'`, f.tenantID).Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	if recorded != 1 {
		t.Fatalf("expected one audit row for the lookup, found %d", recorded)
	}

	// And it comes back out of the history endpoint.
	res = f.do(t, http.MethodGet, "/api/v1/egov/history", "")
	if res.Code != http.StatusOK {
		t.Fatalf("history answered %d: %s", res.Code, res.Body.String())
	}
	var history []struct {
		Action  string         `json:"action"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Action != "egov.citizen_queried" {
		t.Fatalf("the history did not carry the lookup: %+v", history)
	}
	if history[0].Details["reg_number"] != "AA90010111" {
		t.Fatalf("the history entry names the wrong subject: %+v", history[0].Details)
	}
}

// The address the lookups had while they were platform routes still answers.
//
// DEPRECATED with the alias itself: delete this test when /api/v1/xyp goes.
func TestThePreMoveLookupAddressStillAnswers(t *testing.T) {
	f := newFixture(t)

	if res := f.do(t, http.MethodPost, "/api/v1/xyp/citizen", `{"reg_number":"AA90010111"}`); res.Code != http.StatusOK {
		t.Fatalf("the old citizen address answered %d: %s", res.Code, res.Body.String())
	}
	if res := f.do(t, http.MethodPost, "/api/v1/xyp/company", `{"company_reg":"5551234"}`); res.Code != http.StatusOK {
		t.Fatalf("the old company address answered %d: %s", res.Code, res.Body.String())
	}
}

func TestConnectionsReportTheRailModeAndPointAtTheProfileForIdentities(t *testing.T) {
	f := newFixture(t)

	res := f.do(t, http.MethodGet, "/api/v1/egov/connections", "")
	if res.Code != http.StatusOK {
		t.Fatalf("connections answered %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Rails          []staterail.Rail `json:"rails"`
		IdentitiesPath string           `json:"identities_path"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rails) != 1 || body.Rails[0].Mode != "mock" {
		t.Fatalf("a mock rail must say so: %+v", body.Rails)
	}
	// The screen points at the person's own identities rather than owning
	// them — see the package comment. A build that moved them in here would
	// drop this field, and an administrator would gain the ability to unlink
	// somebody else's national identity by uninstalling an app.
	if body.IdentitiesPath != "/profile" {
		t.Fatalf("connections should send people to their own profile, got %q", body.IdentitiesPath)
	}
}
