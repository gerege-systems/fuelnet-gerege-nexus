package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/core"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The organisation and its people are mostly SQL: a profile row that must exist
// for every tenant, composite foreign keys that refuse a department from
// another organisation, and the two refusals that keep somebody from locking
// the last administrator out. None of that is observable without a database.
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/apps/core/...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a migrated database to run the core tests")
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
	otherID  string // a second tenant, to prove nothing leaks between them
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testPool(t)
	ctx := context.Background()

	newTenant := func(prefix string) string {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO tenants (slug, name) VALUES ($1 || substr(gen_random_uuid()::text, 1, 8), 'Core test')
			 RETURNING id::text`, prefix).Scan(&id); err != nil {
			t.Fatalf("tenant: %v", err)
		}
		// No profile is inserted here on purpose. It arrives with the tenant, by
		// trigger, and every read below would fail if it did not — which is the
		// only way that invariant is worth relying on.
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id) })
		return id
	}

	f := &fixture{pool: pool, tenantID: newTenant("core-"), otherID: newTenant("other-")}

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, is_admin)
		 VALUES ('core-' || substr(gen_random_uuid()::text, 1, 8) || '@example.com', 'x', 'Core Admin', TRUE)
		 RETURNING id::text`).Scan(&f.userID); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, f.userID) })

	if _, err := pool.Exec(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1, $2)`, f.tenantID, f.userID); err != nil {
		t.Fatalf("membership: %v", err)
	}

	// The module's own router, with the tenant and the caller already resolved:
	// what is under test is the module, not the middleware in front of it.
	module := core.New(pool)
	router := chi.NewRouter()
	module.RegisterRoutes(router, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := tenant.WithTenantID(r.Context(), f.tenantID)
			ctx = auth.WithUserContext(ctx, auth.UserClaims{UserID: f.userID, TenantID: f.tenantID, IsAdmin: true})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	f.router = router
	return f
}

func (f *fixture) do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func TestTheOrganisationIsReadableAndEditableInParts(t *testing.T) {
	f := newFixture(t)

	res := f.do(t, http.MethodGet, "/api/v1/core/organisation", "")
	if res.Code != http.StatusOK {
		t.Fatalf("read answered %d: %s", res.Code, res.Body.String())
	}
	var before core.Organisation
	if err := json.Unmarshal(res.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	// Defaults an organisation should not have to set: this is a Mongolian
	// platform and a deadline has to be counted somewhere.
	if before.Timezone != "Asia/Ulaanbaatar" || before.Currency != "MNT" {
		t.Fatalf("expected Mongolian defaults, got %s / %s", before.Timezone, before.Currency)
	}

	if res := f.do(t, http.MethodPut, "/api/v1/core/organisation",
		`{"registration_number":"1234567","legal_name":"Жишээ ХХК"}`); res.Code != http.StatusOK {
		t.Fatalf("update answered %d: %s", res.Code, res.Body.String())
	}
	// One more edit, naming a different field. The first two must survive it —
	// a form that sends what it changed should not blank what it did not.
	if res := f.do(t, http.MethodPut, "/api/v1/core/organisation", `{"phone":"+976 11 123456"}`); res.Code != http.StatusOK {
		t.Fatalf("second update answered %d", res.Code)
	}

	var after core.Organisation
	if err := json.Unmarshal(f.do(t, http.MethodGet, "/api/v1/core/organisation", "").Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if after.RegistrationNumber != "1234567" || after.LegalName != "Жишээ ХХК" {
		t.Fatalf("an unrelated edit erased the legal identity: %+v", after)
	}
	if after.Phone != "+976 11 123456" {
		t.Fatalf("the phone was not saved: %q", after.Phone)
	}
}

func TestADepartmentFromAnotherOrganisationIsRefused(t *testing.T) {
	f := newFixture(t)

	// A department in the *other* tenant, offered here as a parent.
	var foreignID string
	if err := f.pool.QueryRow(context.Background(),
		`INSERT INTO departments (tenant_id, code, name) VALUES ($1, 'foreign', 'Foreign')
		 RETURNING id::text`, f.otherID).Scan(&foreignID); err != nil {
		t.Fatal(err)
	}

	res := f.do(t, http.MethodPost, "/api/v1/core/departments",
		`{"code":"sales","name":"Борлуулалт","parent_id":"`+foreignID+`"}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected a parent from another organisation to be refused, got %d: %s",
			res.Code, res.Body.String())
	}

	// And the same department without the foreign parent is fine, so the
	// refusal is about the tenant rather than about the request being malformed.
	if res := f.do(t, http.MethodPost, "/api/v1/core/departments",
		`{"code":"sales","name":"Борлуулалт"}`); res.Code != http.StatusCreated {
		t.Fatalf("expected the department to be created, got %d: %s", res.Code, res.Body.String())
	}
	if res := f.do(t, http.MethodPost, "/api/v1/core/departments",
		`{"code":"sales","name":"Дахин"}`); res.Code != http.StatusConflict {
		t.Fatalf("expected a duplicate code to be refused, got %d", res.Code)
	}
}

func TestYouCannotDeactivateYourselfOrTheLastAdministrator(t *testing.T) {
	f := newFixture(t)

	var membershipID string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT id::text FROM memberships WHERE tenant_id = $1 AND user_id = $2`,
		f.tenantID, f.userID).Scan(&membershipID); err != nil {
		t.Fatal(err)
	}

	// Both refusals apply to this one membership — it is the caller's, and its
	// user is the only administrator — and either is enough to stop it.
	res := f.do(t, http.MethodPost, "/api/v1/core/people/"+membershipID+"/deactivate", "")
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected self-deactivation to be refused, got %d: %s", res.Code, res.Body.String())
	}

	var active bool
	if err := f.pool.QueryRow(context.Background(),
		`SELECT active FROM memberships WHERE id = $1`, membershipID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("the membership was deactivated despite the refusal")
	}
}

func TestPeopleAreListedWithWhatThisOrganisationKnows(t *testing.T) {
	f := newFixture(t)

	created := f.do(t, http.MethodPost, "/api/v1/core/departments", `{"code":"it","name":"Мэдээллийн технологи"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("department: %d", created.Code)
	}
	var department struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &department); err != nil {
		t.Fatal(err)
	}

	var membershipID string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT id::text FROM memberships WHERE tenant_id = $1`, f.tenantID).Scan(&membershipID); err != nil {
		t.Fatal(err)
	}
	if res := f.do(t, http.MethodPut, "/api/v1/core/people/"+membershipID,
		`{"job_title":"Захирал","department_id":"`+department.ID+`"}`); res.Code != http.StatusOK {
		t.Fatalf("update person: %d %s", res.Code, res.Body.String())
	}

	var people []core.Person
	if err := json.Unmarshal(f.do(t, http.MethodGet, "/api/v1/core/people", "").Body.Bytes(), &people); err != nil {
		t.Fatal(err)
	}
	if len(people) != 1 {
		t.Fatalf("expected one person, got %d", len(people))
	}
	if people[0].JobTitle != "Захирал" || people[0].DepartmentName != "Мэдээллийн технологи" {
		t.Fatalf("the membership's own fields did not come back: %+v", people[0])
	}
	// The identity comes from the user and is shared across organisations; the
	// job title is the membership's. Both have to arrive in one row for the
	// screen to be a directory rather than two lists.
	if people[0].Email == "" || people[0].Name == "" {
		t.Fatalf("the person's identity is missing: %+v", people[0])
	}
}
