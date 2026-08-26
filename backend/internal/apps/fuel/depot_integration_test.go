package fuel

// The chain from the frontier to a tank, asserted against a real database.
//
// Every rule tested here is enforced by PostgreSQL or by a transaction, not by
// a Go branch that a reader could verify by looking: a partial unique index
// refuses the second receipt for one consignment, a CHECK constraint refuses an
// overfill, and a row-level policy refuses another company's depots. None of
// those are observable in a unit test, and all three are the kind of rule that
// stops holding without anybody noticing — which is the state
// benzin-gerege-mn's stock figures were already in.
//
//	FUEL_TEST_DATABASE_URL=postgres://... go test ./internal/apps/fuel/...

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/dbguard"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func openFuelPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("FUEL_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("neither FUEL_TEST_DATABASE_URL nor DATABASE_URL is set")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	// The guard the server installs. Without it every statement would run as
	// the login role, outside the row-level policies — and the isolation test
	// below would pass for the wrong reason.
	guard := &dbguard.Guard{}
	guard.Install(config)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("database unreachable: %v", err)
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := guard.Probe(probeCtx, pool); err != nil {
		pool.Close()
		t.Skipf("row-level isolation could not be enabled: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// company is one organisation with the fuel module wired to it.
type company struct {
	module   *Module
	tenantID string
	userID   string
}

// newCompany creates an organisation and a module bound to the same pool.
//
// The module is built by hand rather than through New, because New registers
// itself in the process-wide app registry and two tests doing that would leave
// the registry describing a binary that does not exist.
func newCompany(t *testing.T, pool *pgxpool.Pool, name string) *company {
	t.Helper()

	tenantID := uuid.NewString()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO platform.tenants (id, name, slug) VALUES ($1, $2, $3)`,
		tenantID, "Fuel test "+name, strings.ToLower(name)+"-"+tenantID[:8]); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM platform.tenants WHERE id = $1`, tenantID)
	})

	return &company{module: &Module{db: pool}, tenantID: tenantID, userID: uuid.NewString()}
}

// call runs a handler as a member of this organisation.
func (c *company) call(t *testing.T, handler http.HandlerFunc, method, target string,
	body any, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	encoded := []byte("{}")
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}

	req := httptest.NewRequest(method, target, strings.NewReader(string(encoded)))
	ctx := nexus.WithTenantID(req.Context(), c.tenantID)
	ctx = nexus.WithUser(ctx, nexus.UserClaims{UserID: c.userID})
	if len(params) > 0 {
		routeCtx := chi.NewRouteContext()
		for key, value := range params {
			routeCtx.URLParams.Add(key, value)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
	}

	rec := httptest.NewRecorder()
	handler(rec, req.WithContext(ctx))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return value
}

// declare records a consignment at the border and answers its id.
func (c *company) declare(t *testing.T, liters float64) string {
	t.Helper()
	rec := c.call(t, c.module.handleCreateShipment, http.MethodPost, "/shipments",
		ShipmentDraft{
			DeclarationNo:  "ГМ-TEST-" + uuid.NewString()[:8],
			BorderPort:     "Сүхбаатар",
			OriginCountry:  "ОХУ",
			Exporter:       "Роснефть",
			FuelType:       "ai92",
			DeclaredLiters: liters,
		}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("declare: %d %s", rec.Code, rec.Body.String())
	}
	return decode[Shipment](t, rec).ID
}

// base creates a depot with one tank of the given size.
func (c *company) base(t *testing.T, capacity float64) (depotID, tankID string) {
	t.Helper()

	rec := c.call(t, c.module.handleCreateDepot, http.MethodPost, "/depots",
		DepotDraft{Name: "Тест бааз " + uuid.NewString()[:8], Aimag: "Дархан-Уул"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create depot: %d %s", rec.Code, rec.Body.String())
	}
	depotID = decode[Depot](t, rec).ID

	rec = c.call(t, c.module.handleCreateTank, http.MethodPost, "/depots/x/tanks",
		TankDraft{TankNo: "Т-1", FuelType: "ai92", CapacityLiters: capacity},
		map[string]string{"id": depotID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tank: %d %s", rec.Code, rec.Body.String())
	}
	return depotID, decode[Tank](t, rec).ID
}

// clear releases a consignment from customs, which is what mints its batch.
func (c *company) clear(t *testing.T, shipmentID string) *httptest.ResponseRecorder {
	t.Helper()
	return c.call(t, c.module.handleAdvanceShipment, http.MethodPost, "/shipments/x/status",
		ClearRequest{Status: "cleared", LabStatus: "passed", QualityCertNo: "LAB-1"},
		map[string]string{"id": shipmentID})
}

// receive unloads a consignment into a tank.
func (c *company) receive(t *testing.T, depotID, tankID, shipmentID string,
	liters float64) *httptest.ResponseRecorder {
	t.Helper()
	return c.call(t, c.module.handleReceiveIntoDepot, http.MethodPost, "/depots/x/receipts",
		DepotReceiveRequest{TankID: tankID, ShipmentID: shipmentID, Liters: liters},
		map[string]string{"id": depotID})
}

// tankLevel reads a vessel's level straight out of the table.
func (c *company) tankLevel(t *testing.T, pool *pgxpool.Pool, tankID string) float64 {
	t.Helper()
	var liters float64
	ctx := nexus.WithTenantID(context.Background(), c.tenantID)
	if err := pool.QueryRow(ctx,
		`SELECT current_liters::float8 FROM fuel_depot_tanks WHERE id = $1::uuid`,
		tankID).Scan(&liters); err != nil {
		t.Fatalf("read tank: %v", err)
	}
	return liters
}

// TestClearingMintsExactlyOneBatch covers the rule the whole chain hangs from.
//
// A batch is the number every downstream record carries, so two batches for one
// consignment would mean the same litres traceable to two different origins,
// and none for a cleared one would mean fuel in the country that no document
// reaches. Clearing is idempotent because a screen can be tapped twice.
func TestClearingMintsExactlyOneBatch(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	shipmentID := acme.declare(t, 400_000)

	first := acme.clear(t, shipmentID)
	if first.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", first.Code, first.Body.String())
	}
	minted := decode[map[string]any](t, first)["batch_code"]
	if minted == "" {
		t.Fatal("clearing a shipment did not mint a batch")
	}

	// Cleared → cleared is not a legal move, so the second attempt is refused
	// before it can reach the insert. Either answer is acceptable to a caller;
	// what is not acceptable is a second batch.
	acme.clear(t, shipmentID)

	var batches int
	ctx := nexus.WithTenantID(context.Background(), acme.tenantID)
	if err := pool.QueryRow(ctx,
		`SELECT count(*)::int FROM fuel_batches WHERE customs_shipment_id = $1::uuid`,
		shipmentID).Scan(&batches); err != nil {
		t.Fatalf("count batches: %v", err)
	}
	if batches != 1 {
		t.Fatalf("one consignment minted %d batches, want 1", batches)
	}
}

// TestAShipmentCannotGoBackwards covers the state machine.
//
// The value of the record is that it cannot be back-dated. A consignment that
// reached a depot must not become "still under inspection" because a stale
// screen posted an old value — that is how a document trail stops being one.
func TestAShipmentCannotGoBackwards(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	shipmentID := acme.declare(t, 100_000)
	if rec := acme.clear(t, shipmentID); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body.String())
	}

	rec := acme.call(t, acme.module.handleAdvanceShipment, http.MethodPost, "/shipments/x/status",
		ClearRequest{Status: "border_arrived"}, map[string]string{"id": shipmentID})
	if rec.Code != http.StatusConflict {
		t.Fatalf("a cleared shipment went back to the border: %d %s", rec.Code, rec.Body.String())
	}
}

// TestUnclearedFuelCannotEnterATank is the rule the depot exists to enforce.
//
// Everything downstream — a tanker's load, a station's stock, a citizen's
// voucher — treats a tank's contents as lawfully imported. A consignment still
// under inspection may yet be turned back, and fuel that reached a tank cannot
// be.
func TestUnclearedFuelCannotEnterATank(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	depotID, tankID := acme.base(t, 500_000)
	shipmentID := acme.declare(t, 100_000)

	rec := acme.receive(t, depotID, tankID, shipmentID, 100_000)
	if rec.Code != http.StatusConflict {
		t.Fatalf("uncleared fuel reached a tank: %d %s", rec.Code, rec.Body.String())
	}
	if level := acme.tankLevel(t, pool, tankID); level != 0 {
		t.Fatalf("the tank holds %.0f litres after a refused receipt, want 0", level)
	}
}

// TestOneConsignmentUnloadsOnce covers the partial unique index.
//
// Two taps on a phone with a poor signal are two requests. A check followed by
// an insert lets both through, and the second would add the litres twice —
// which quietly breaks the one number a regulator is watching.
func TestOneConsignmentUnloadsOnce(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	depotID, tankID := acme.base(t, 500_000)
	shipmentID := acme.declare(t, 100_000)
	if rec := acme.clear(t, shipmentID); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body.String())
	}

	first := acme.receive(t, depotID, tankID, shipmentID, 100_000)
	if first.Code != http.StatusCreated {
		t.Fatalf("receive: %d %s", first.Code, first.Body.String())
	}
	receipt := decode[DepotReceipt](t, first)
	if receipt.TankAfterLiters != 100_000 {
		t.Fatalf("the tank holds %.0f litres, want 100000", receipt.TankAfterLiters)
	}
	if receipt.BatchCode == "" {
		t.Fatal("the receipt carries no batch, so the chain is broken at the depot")
	}

	second := acme.receive(t, depotID, tankID, shipmentID, 100_000)
	if second.Code != http.StatusConflict {
		t.Fatalf("the same consignment unloaded twice: %d %s", second.Code, second.Body.String())
	}
	if level := acme.tankLevel(t, pool, tankID); level != 100_000 {
		t.Fatalf("the tank holds %.0f litres after a repeat, want 100000", level)
	}
}

// TestATankCannotBeOverfilled covers the CHECK constraint, and the rollback.
//
// The refusal has to leave nothing behind: a receipt recorded for fuel that
// never entered the tank is worse than no receipt at all, because it is a
// document saying the litres arrived.
func TestATankCannotBeOverfilled(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	depotID, tankID := acme.base(t, 50_000)
	shipmentID := acme.declare(t, 80_000)
	if rec := acme.clear(t, shipmentID); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body.String())
	}

	rec := acme.receive(t, depotID, tankID, shipmentID, 80_000)
	if rec.Code != http.StatusConflict {
		t.Fatalf("a 50,000 litre tank took 80,000: %d %s", rec.Code, rec.Body.String())
	}
	if level := acme.tankLevel(t, pool, tankID); level != 0 {
		t.Fatalf("the tank holds %.0f litres after a refused overfill, want 0", level)
	}

	var receipts int
	ctx := nexus.WithTenantID(context.Background(), acme.tenantID)
	if err := pool.QueryRow(ctx,
		`SELECT count(*)::int FROM fuel_depot_receipts WHERE tank_id = $1::uuid`,
		tankID).Scan(&receipts); err != nil {
		t.Fatalf("count receipts: %v", err)
	}
	if receipts != 0 {
		t.Fatalf("%d receipts survived a rolled-back overfill, want 0", receipts)
	}
}

// TestATankBelongsToItsDepot stops the path parameter from being a way in.
//
// The depot id is in the URL and the tank id is in the body; nothing but this
// check ties them together. Without it an operator could unload a consignment
// into a tank at a base they were not writing about, and the receipt would name
// the wrong one.
func TestATankBelongsToItsDepot(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	firstDepot, _ := acme.base(t, 500_000)
	_, otherTank := acme.base(t, 500_000)

	shipmentID := acme.declare(t, 10_000)
	if rec := acme.clear(t, shipmentID); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body.String())
	}

	rec := acme.receive(t, firstDepot, otherTank, shipmentID, 10_000)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a receipt crossed depots: %d %s", rec.Code, rec.Body.String())
	}
}

// TestOneCompanyCannotSeeAnothersDepots is the promise the whole two-participant
// design rests on.
//
// A company sees its own bases and nothing else. The handlers carry no
// tenant_id in their WHERE clauses on purpose — the row-level policy is the
// answer — so this asserts the policy is really there rather than that somebody
// remembered to write a filter.
func TestOneCompanyCannotSeeAnothersDepots(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")
	rival := newCompany(t, pool, "Rival")

	acmeDepot, _ := acme.base(t, 500_000)

	rec := rival.call(t, rival.module.handleListDepots, http.MethodGet, "/depots", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list depots: %d %s", rec.Code, rec.Body.String())
	}
	for _, depot := range decode[struct {
		Depots []Depot `json:"depots"`
	}](t, rec).Depots {
		if depot.ID == acmeDepot {
			t.Fatal("a company read another company's depot")
		}
	}

	// And cannot reach it by id either: a tank added through the rival's
	// session must not attach to Acme's base.
	added := rival.call(t, rival.module.handleCreateTank, http.MethodPost, "/depots/x/tanks",
		TankDraft{TankNo: "Т-9", FuelType: "ai92", CapacityLiters: 1000},
		map[string]string{"id": acmeDepot})
	if added.Code != http.StatusNotFound {
		t.Fatalf("a company added a tank to another company's depot: %d %s",
			added.Code, added.Body.String())
	}
}

// TestTheGaugeCannotSetTheLevel is a rule about what is missing.
//
// Litres in a tank are the sum of what went in and what came out. A reading
// that could overwrite that sum would make every receipt below it advisory, so
// the update handler takes temperature, density and a safety status — and no
// way to say how full the tank is.
func TestTheGaugeCannotSetTheLevel(t *testing.T) {
	pool := openFuelPool(t)
	acme := newCompany(t, pool, "Acme")

	depotID, tankID := acme.base(t, 500_000)
	shipmentID := acme.declare(t, 120_000)
	if rec := acme.clear(t, shipmentID); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d %s", rec.Code, rec.Body.String())
	}
	if rec := acme.receive(t, depotID, tankID, shipmentID, 120_000); rec.Code != http.StatusCreated {
		t.Fatalf("receive: %d %s", rec.Code, rec.Body.String())
	}

	// Everything a reading can carry, plus a field naming the level. The field
	// is not in GaugeReading, so it is dropped — and this test is what fails if
	// somebody ever adds it.
	twenty := 20.5
	body := map[string]any{
		"temperature_c":  twenty,
		"density_kg_m3":  745.0,
		"safety_status":  "warning_temp",
		"current_liters": 999.0,
	}
	rec := acme.call(t, acme.module.handleUpdateTank, http.MethodPatch,
		"/depots/x/tanks/y", body, map[string]string{"id": depotID, "tankId": tankID})
	if rec.Code != http.StatusOK {
		t.Fatalf("gauge reading: %d %s", rec.Code, rec.Body.String())
	}

	tank := decode[Tank](t, rec)
	if tank.CurrentLiters != 120_000 {
		t.Fatalf("a gauge reading moved the level to %.0f, want 120000", tank.CurrentLiters)
	}
	if tank.SafetyStatus != "warning_temp" || tank.TemperatureC == nil || *tank.TemperatureC != twenty {
		t.Fatalf("the reading was not recorded: %+v", tank)
	}
}
