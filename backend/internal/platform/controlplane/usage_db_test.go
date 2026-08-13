package controlplane

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/metering"
)

// CP-5's promises: the numbers are counted from the database rather than from
// metrics, the console can read them and cannot write them, and the two
// metrics that are not sums — storage and active users — are not summed.

func TestUsageIsCountedFromTheDatabase(t *testing.T) {
	pool := openPool(t)
	service := configService(t, pool)
	tenantID, _ := newTenant(t, pool)
	userID, _ := newPerson(t, pool, tenantID)
	ctx := context.Background()

	// Two acts and one AI call today, and a session that was used today.
	for _, action := range []string{"contacts.create", "ai.chat"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO audit_events (tenant_id, user_id, action, resource)
			 VALUES ($1::uuid, $2, $3, 'test')`, tenantID, userID, action); err != nil {
			t.Fatalf("write an audit row: %v", err)
		}
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, tenant_id, expires_at, last_seen_at)
		 VALUES (repeat('b', 64), $1::uuid, $2::uuid, NOW() + INTERVAL '1 hour', NOW())`,
		userID, tenantID); err != nil {
		t.Fatalf("write a session: %v", err)
	}

	metering.New(pool).CollectDay(ctx, time.Now())

	usage, err := service.UsageFor(ctx, tenantID)
	if err != nil {
		t.Fatalf("read the usage: %v", err)
	}

	got := map[string]int64{}
	for _, series := range usage.Series {
		got[series.Metric] = series.Total
	}
	if got[metering.Actions] != 2 {
		t.Fatalf("actions counted %d, want 2", got[metering.Actions])
	}
	if got[metering.AICalls] != 1 {
		t.Fatalf("AI calls counted %d, want 1", got[metering.AICalls])
	}
	if got[metering.ActiveUsers] != 1 {
		t.Fatalf("active users counted %d, want 1", got[metering.ActiveUsers])
	}
	if usage.Collected == nil {
		t.Fatal("the collection time is missing, so the screen cannot tell empty from uncounted")
	}

	// Collecting the same day again rewrites rather than doubles: the job runs
	// several times a day and on every restart.
	metering.New(pool).CollectDay(ctx, time.Now())
	usage, err = service.UsageFor(ctx, tenantID)
	if err != nil {
		t.Fatalf("read the usage: %v", err)
	}
	for _, series := range usage.Series {
		if series.Metric == metering.Actions && series.Total != 2 {
			t.Fatalf("a second collection made it %d", series.Total)
		}
	}
}

// Active users is a peak and storage is a reading. Summing either would be a
// number that grows for ever and means nothing.
func TestTheMetricsThatAreNotSumsAreNotSummed(t *testing.T) {
	pool := openPool(t)
	service := configService(t, pool)
	tenantID, _ := newTenant(t, pool)
	ctx := context.Background()

	for day, values := range map[string][2]int64{
		"2026-08-01": {3, 100},
		"2026-08-02": {5, 120},
		"2026-08-03": {4, 90},
	} {
		for metric, value := range map[string]int64{
			metering.ActiveUsers: values[0],
			metering.StorageMB:   values[1],
		} {
			if _, err := pool.Exec(ctx,
				`INSERT INTO usage_events (tenant_id, day, metric, value)
				 VALUES ($1::uuid, $2::date, $3, $4)
				 ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value`,
				tenantID, day, metric, value); err != nil {
				t.Fatalf("write a usage row: %v", err)
			}
		}
	}

	usage, err := service.UsageFor(ctx, tenantID)
	if err != nil {
		t.Fatalf("read the usage: %v", err)
	}
	for _, series := range usage.Series {
		switch series.Metric {
		case metering.ActiveUsers:
			if series.Total != 5 {
				t.Fatalf("active users totalled %d, want the peak of 5", series.Total)
			}
		case metering.StorageMB:
			// The last day in the window, which is the third of August here.
			if series.Total != 90 {
				t.Fatalf("storage totalled %d, want the last reading of 90", series.Total)
			}
		}
	}
}

// The console reads usage and cannot write it — which is the answer to the
// first question anybody asks of a metering system in a billing dispute.
func TestTheConsoleCannotWriteUsage(t *testing.T) {
	pool := openPool(t)
	tenantID, _ := newTenant(t, pool)
	ctx := scoped(context.Background())

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM usage_events WHERE tenant_id = $1::uuid`, tenantID).Scan(&count); err != nil {
		t.Fatalf("the operator role cannot read the usage: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO usage_events (tenant_id, day, metric, value)
		 VALUES ($1::uuid, CURRENT_DATE, 'actions', 999999)`, tenantID); err == nil {
		t.Fatal("the operator role wrote a usage row")
	}
	if _, err := pool.Exec(ctx,
		`UPDATE usage_events SET value = 0 WHERE tenant_id = $1::uuid`, tenantID); err == nil {
		t.Fatal("the operator role changed a usage row")
	}
}

func TestTheUsageExportIsOneRowPerDay(t *testing.T) {
	pool := openPool(t)
	service := configService(t, pool)
	tenantID, _ := newTenant(t, pool)
	ctx := context.Background()

	for _, day := range []string{"2026-08-01", "2026-08-02"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO usage_events (tenant_id, day, metric, value)
			 VALUES ($1::uuid, $2::date, 'actions', 7)
			 ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value`,
			tenantID, day); err != nil {
			t.Fatalf("write a usage row: %v", err)
		}
	}

	var out strings.Builder
	if err := service.WriteUsageCSV(ctx, &writerTo{&out}, tenantID); err != nil {
		t.Fatalf("export: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("the export has %d lines, want a header and two days: %q", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "day,") {
		t.Fatalf("the header is %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "2026-08-01") || !strings.HasPrefix(lines[2], "2026-08-02") {
		t.Fatalf("the days are out of order: %q", out.String())
	}
}

// writerTo is the smallest http.ResponseWriter that can hold a CSV.
type writerTo struct{ into *strings.Builder }

func (w *writerTo) Header() http.Header         { return http.Header{} }
func (w *writerTo) Write(p []byte) (int, error) { return w.into.Write(p) }
func (w *writerTo) WriteHeader(int)             {}
