package reporting_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/dbguard"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These are the tests that matter most about this package, and they need a real
// database: the whole design rests on a report's query running inside the
// caller's tenant binding, and nothing about that is observable without
// PostgreSQL enforcing the policies.

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("REPORTING_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("neither REPORTING_TEST_DATABASE_URL nor DATABASE_URL is set")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	// The same guard the server installs. Without it the pool hands out
	// connections as the login role and the row-level policies never apply —
	// which would make the isolation test below pass for the wrong reason.
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
	if !guard.Enabled() {
		pool.Close()
		t.Skip("row-level isolation is not installed on this database")
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedTenant creates an organisation with one invoice, and removes both
// afterwards.
func seedTenant(t *testing.T, pool *pgxpool.Pool, slug string, amount float64) string {
	t.Helper()
	ctx := context.Background()

	id := uuid.NewString()
	_, err := pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
		id, "Reporting test "+slug, slug)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id)
	})

	_, err = pool.Exec(ctx, `
		INSERT INTO billing_invoices
		    (tenant_id, invoice_number, contact_name, amount, vat_amount, ebarimt_status, status)
		VALUES ($1, $2, 'Test contact', $3, $4, 'SENT_TO_ETAX', 'PENDING')`,
		id, "INV-"+slug, amount, amount*0.1)
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	return id
}

// revenueFixture is a copy of billing's own report, kept here rather than
// imported: importing an app package into a platform test would make the
// platform depend on the module, which is the direction this architecture
// forbids.
type revenueFixture struct{}

func (revenueFixture) Key() string { return "test.revenue" }
func (revenueFixture) App() string { return "io.gerege.nexus.billing" }
func (revenueFixture) Titles() map[string]string {
	return map[string]string{"mn": "Орлого", "en": "Revenue"}
}
func (revenueFixture) Params() []reporting.ParamSpec {
	return []reporting.ParamSpec{{
		Key: "period", Kind: reporting.ParamDateRange,
		Titles: map[string]string{"mn": "Хугацаа"}, DefaultWindow: 365 * 24 * time.Hour,
	}}
}
func (revenueFixture) Columns() []reporting.ColumnSpec {
	return []reporting.ColumnSpec{
		{Key: "contact", Kind: reporting.ColumnText, Titles: map[string]string{"mn": "Харилцагч"}},
		{Key: "amount", Kind: reporting.ColumnMoney, Total: true, Titles: map[string]string{"mn": "Дүн"}},
	}
}

func (revenueFixture) Run(ctx context.Context, q reporting.Querier, p reporting.Params) (reporting.Result, error) {
	rows, err := q.Query(ctx, `
		SELECT contact_name, amount FROM billing_invoices
		 WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3`,
		reporting.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"))
	if err != nil {
		return reporting.Result{}, err
	}
	collected, err := reporting.Collect(rows, func() (map[string]any, error) {
		var contact string
		var amount float64
		if err := rows.Scan(&contact, &amount); err != nil {
			return nil, err
		}
		return map[string]any{"contact": contact, "amount": amount}, nil
	})
	if err != nil {
		return reporting.Result{}, err
	}
	return reporting.Result{Rows: collected}, nil
}

// leakyFixture is the mistake the second layer exists for: a report whose
// author forgot the tenant clause. It must return nothing rather than
// everything.
type leakyFixture struct{ revenueFixture }

func (leakyFixture) Key() string { return "test.leaky" }

func (leakyFixture) Run(ctx context.Context, q reporting.Querier, _ reporting.Params) (reporting.Result, error) {
	// No WHERE tenant_id. Deliberately.
	rows, err := q.Query(ctx, `SELECT contact_name, amount FROM billing_invoices`)
	if err != nil {
		return reporting.Result{}, err
	}
	collected, err := reporting.Collect(rows, func() (map[string]any, error) {
		var contact string
		var amount float64
		if err := rows.Scan(&contact, &amount); err != nil {
			return nil, err
		}
		return map[string]any{"contact": contact, "amount": amount}, nil
	})
	if err != nil {
		return reporting.Result{}, err
	}
	return reporting.Result{Rows: collected}, nil
}

func TestRunSeesOnlyItsOwnTenant(t *testing.T) {
	pool := openPool(t)
	engine := reporting.NewEngine(pool)

	mine := seedTenant(t, pool, "reporting-a-"+uuid.NewString()[:8], 1000)
	seedTenant(t, pool, "reporting-b-"+uuid.NewString()[:8], 5000)

	report := revenueFixture{}
	params, err := reporting.Bind(report, map[string]string{}, "mn")
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result, err := engine.Run(context.Background(), mine, report, params)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected one row, got %d", len(result.Rows))
	}
	if result.Totals["amount"] != 1000 {
		t.Fatalf("the other organisation's invoice was counted: total is %v", result.Totals["amount"])
	}
}

// The point of the row-level policy: a report that forgets its tenant clause
// returns nothing, not everyone's rows.
func TestAReportThatForgetsTheTenantClauseSeesNothingElse(t *testing.T) {
	pool := openPool(t)
	engine := reporting.NewEngine(pool)

	mine := seedTenant(t, pool, "reporting-c-"+uuid.NewString()[:8], 1000)
	seedTenant(t, pool, "reporting-d-"+uuid.NewString()[:8], 5000)

	report := leakyFixture{}
	params, err := reporting.Bind(report, map[string]string{}, "mn")
	if err != nil {
		t.Fatalf("bind: %v", err)
	}

	result, err := engine.Run(context.Background(), mine, report, params)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, row := range result.Rows {
		if row["amount"] == 5000.0 {
			t.Fatal("a report with no tenant clause read another organisation's invoice")
		}
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected only this organisation's row, got %d", len(result.Rows))
	}
}

// A report runs in a read-only transaction. A report that tried to write —
// through a query that happened to be a DELETE, or a function with a side
// effect — must be refused by the database rather than by review.
func TestReportsCannotWrite(t *testing.T) {
	pool := openPool(t)
	engine := reporting.NewEngine(pool)
	tenantID := seedTenant(t, pool, "reporting-e-"+uuid.NewString()[:8], 100)

	report := writerFixture{}
	params, _ := reporting.Bind(report, map[string]string{}, "mn")

	if _, err := engine.Run(context.Background(), tenantID, report, params); err == nil {
		t.Fatal("a report was allowed to write")
	}
}

type writerFixture struct{ revenueFixture }

func (writerFixture) Key() string { return "test.writer" }

func (writerFixture) Run(ctx context.Context, q reporting.Querier, _ reporting.Params) (reporting.Result, error) {
	rows, err := q.Query(ctx, `DELETE FROM billing_invoices WHERE tenant_id = $1 RETURNING contact_name, amount`,
		reporting.TenantOf(ctx))
	if err != nil {
		return reporting.Result{}, err
	}
	rows.Close()
	return reporting.Result{}, rows.Err()
}
