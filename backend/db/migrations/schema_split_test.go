/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package migrations_test

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	tenantRole    = "gerege_nexus_tenant"
	oldTenantRole = "gerege_nexus_app"
)

func schemaPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var migrated bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'tenant')
		   AND EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'platform')`).Scan(&migrated); err != nil {
		t.Fatal(err)
	}
	if !migrated {
		t.Skip("the database is not migrated through 00079_two_schemas")
	}
	return pool
}

// The database result is checked against the same ownership decision that
// generated the migration. Counting 26 and 40 is useful in a review, but the
// names are the invariant: swapping one table each way would preserve both
// counts and still put both on the wrong plane.
func TestEveryPlatformMigrationTableLandsOnItsDeclaredPlane(t *testing.T) {
	pool := schemaPool(t)
	rows, err := pool.Query(context.Background(), `
		SELECT schemaname, tablename
		  FROM pg_tables
		 WHERE schemaname IN ('tenant', 'platform')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	found := make(map[string]string, len(platformTables))
	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err != nil {
			t.Fatal(err)
		}
		if _, ours := platformTables[name]; ours {
			found[name] = schema
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	for name, declared := range platformTables {
		actual, exists := found[name]
		if !exists {
			t.Errorf("%s is absent from both tenant and platform schemas", name)
			continue
		}
		counts[actual]++
		if actual != declared.plane {
			t.Errorf("%s is in %s, ownership_test.go declares %s", name, actual, declared.plane)
		}
	}
	if counts["platform"] != 26 || counts["tenant"] != 40 {
		t.Errorf("schema counts: platform=%d tenant=%d; want 26 and 40", counts["platform"], counts["tenant"])
	}
}

// PostgreSQL stores policy roles by OID. Renaming the role should therefore
// change the displayed name without rebuilding forty policies; this test is
// the proof that lets the migration leave their USING/WITH CHECK expressions
// untouched.
func TestTenantRoleRenameReachedEveryIsolationPolicy(t *testing.T) {
	pool := schemaPool(t)
	var oldExists bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, oldTenantRole).Scan(&oldExists); err != nil {
		t.Fatal(err)
	}
	if oldExists {
		t.Errorf("the old database role %s still exists", oldTenantRole)
	}

	rows, err := pool.Query(context.Background(), `
		SELECT schemaname, tablename, roles
		  FROM pg_policies
		 WHERE policyname = 'tenant_isolation'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		var schema, table string
		var roles []string
		if err := rows.Scan(&schema, &table, &roles); err != nil {
			t.Fatal(err)
		}
		if _, ours := platformTables[table]; !ours {
			continue
		}
		seen++
		if !slices.Contains(roles, tenantRole) || slices.Contains(roles, oldTenantRole) {
			t.Errorf("%s.%s policy roles = %v, want only the renamed tenant role", schema, table, roles)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("no platform-owned tenant_isolation policies were found")
	}
}

// A schema's USAGE privilege only permits resolving names inside it; the
// table's own privilege is what opens a row. The five boundary tables require
// both, so platform USAGE cannot be revoked from the tenant role. The boundary
// is instead proved at the table that must stay shut: operator_audit.
func TestTenantRoleReadsTheBoundaryButNotOperatorAudit(t *testing.T) {
	pool := schemaPool(t)
	ctx := context.Background()

	for _, table := range []string{
		"announcements", "feature_flag_overrides", "operator_impersonations",
		"tenant_quotas", "usage_events",
	} {
		var allowed bool
		if err := pool.QueryRow(ctx, `SELECT has_table_privilege($1, $2, 'SELECT')`,
			tenantRole, "platform."+table).Scan(&allowed); err != nil {
			t.Fatal(err)
		}
		if !allowed {
			t.Errorf("%s cannot SELECT platform.%s", tenantRole, table)
		}
	}

	var auditAllowed bool
	if err := pool.QueryRow(ctx, `SELECT has_table_privilege($1, 'platform.operator_audit', 'SELECT')`,
		tenantRole).Scan(&auditAllowed); err != nil {
		t.Fatal(err)
	}
	if auditAllowed {
		t.Fatal("the tenant role has SELECT on platform.operator_audit")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE gerege_nexus_tenant`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT 1 FROM platform.operator_audit LIMIT 1`); err == nil {
		t.Fatal("a query running as the tenant role read platform.operator_audit")
	} else if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("operator_audit was refused for an unexpected reason: %v", err)
	}
}

func TestLoginPathSearchesBothPlanes(t *testing.T) {
	pool := schemaPool(t)
	var path string
	if err := pool.QueryRow(context.Background(), `SHOW search_path`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if path != "tenant, platform, public" {
		t.Errorf("login role search_path = %q, want tenant, platform, public", path)
	}
}
