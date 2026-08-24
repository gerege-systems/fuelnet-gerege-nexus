/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package appinstall

import (
	"context"
	"net/http"
	"os"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// A module brings its own table, and its own record of having done so.
//
// This is the whole of what Үе 3 delivers: the mechanism, proved end to end
// against a real database. No app has been moved out of db/migrations yet —
// that is a separate change with a data migration behind it — so without this
// test the mechanism would ship with nothing exercising it.
//
// The second assertion is the one that is easy to skip and expensive to get
// wrong. goose keeps one row per applied version in one table; if a module's
// history shared the platform's table, a module's 00001 and the platform's
// 00001 would be the same row and whichever ran second would be recorded as
// already applied. cmd/migrate wrote that reasoning down when it learned
// MIGRATIONS_TABLE. This is the first caller to depend on it.
func TestAModuleBringsItsOwnSchemaAndItsOwnHistory(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a migrated test database to run the module migration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	const appID = "io.example.brought_its_own"
	nexus.Migrations(appID, fstest.MapFS{
		"00001_create.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE brought_its_own (id uuid PRIMARY KEY);
-- +goose Down
DROP TABLE brought_its_own;
`)},
	})
	t.Cleanup(func() {
		nexus.Migrations(appID, nil)
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS brought_its_own`)
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS goose_db_version_brought_its_own`)
	})
	// A previous failed run must not decide this one.
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS brought_its_own`)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS goose_db_version_brought_its_own`)

	installer := NewAppInstaller(pool, nil, "1.0.0")
	if err := installer.runModuleMigrations(ctx, appID); err != nil {
		t.Fatalf("run the module's migrations: %v", err)
	}

	for _, table := range []string{"brought_its_own", "goose_db_version_brought_its_own"} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = $1)`,
			table).Scan(&exists); err != nil {
			t.Fatalf("look for %s: %v", table, err)
		}
		if !exists {
			t.Errorf("%s was not created", table)
		}
	}

	// The platform's own history is untouched, which is the point of the
	// separate table rather than a happy accident of it.
	var platformVersions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM goose_db_version`).Scan(&platformVersions); err != nil {
		t.Fatalf("read the platform's migration history: %v", err)
	}
	if platformVersions == 0 {
		t.Error("the platform's own migration history is empty; the module wrote to the wrong table")
	}

	// Running again applies nothing and fails nothing: the same app installed
	// by a second tenant reaches this code a second time.
	if err := installer.runModuleMigrations(ctx, appID); err != nil {
		t.Errorf("running a module's migrations twice failed: %v", err)
	}
}

// A schema arrives even though nobody installs anything.
//
// The case this covers is an upgrade, not an install: the app is already
// installed — days ago, by the version of the platform that carried its tables
// in db/migrations — and the install path that runs a module's migrations is
// never reached again. Until MigrateModules existed, the tables a release moved
// into the module simply never appeared, and the first sign was a request into
// the app answering "relation ... does not exist".
//
// So this test installs nothing. That is the assertion.
func TestAModuleSchemaIsAppliedWithoutAnybodyInstallingTheApp(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a migrated test database to run the module migration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	nexus.Register(alreadyInstalledModule{})
	nexus.Migrations(alreadyInstalledAppID, fstest.MapFS{
		"00001_create.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE arrived_late (id uuid PRIMARY KEY);
-- +goose Down
DROP TABLE arrived_late;
`)},
	})
	drop := func() {
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS arrived_late`)
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS goose_db_version_arrived_late`)
	}
	t.Cleanup(func() {
		// The module stays in the registry — nexus has no unregister — but
		// without migrations it is skipped, so a later sweep in this binary
		// does nothing with it.
		nexus.Migrations(alreadyInstalledAppID, nil)
		drop()
	})
	drop()

	installer := NewAppInstaller(pool, nil, "1.0.0")
	if err := installer.MigrateModules(ctx); err != nil {
		t.Fatalf("apply the compiled-in modules' schemas: %v", err)
	}

	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'arrived_late')`,
	).Scan(&exists); err != nil {
		t.Fatalf("look for the table: %v", err)
	}
	if !exists {
		t.Error("a module compiled into this binary did not get its schema; an installed app would answer 500 for every request")
	}

	// The sweep runs on a timer as well as at startup, so a second round has to
	// be free rather than an error.
	if err := installer.MigrateModules(ctx); err != nil {
		t.Errorf("sweeping a second time failed: %v", err)
	}
}

const alreadyInstalledAppID = "io.example.arrived_late"

// A module of the smallest shape the registry accepts: this test is about the
// schema sweep, and the sweep asks a module for nothing but its id.
type alreadyInstalledModule struct{}

func (alreadyInstalledModule) ID() string                                { return alreadyInstalledAppID }
func (alreadyInstalledModule) Name() string                              { return "Arrived Late" }
func (alreadyInstalledModule) Version() string                           { return "1.0.0" }
func (alreadyInstalledModule) Dependencies() []nexus.Dependency          { return nil }
func (alreadyInstalledModule) Permissions() []nexus.PermissionDefinition { return nil }
func (alreadyInstalledModule) Menus() []nexus.MenuDefinition             { return nil }

func (alreadyInstalledModule) RegisterRoutes(chi.Router, func(http.Handler) http.Handler) {}

// A module with no schema of its own is the ordinary case, not a missing
// registration: every app in this repository is still in db/migrations.
func TestAModuleWithoutMigrationsInstallsQuietly(t *testing.T) {
	installer := NewAppInstaller(nil, nil, "1.0.0")
	if err := installer.runModuleMigrations(context.Background(), "io.gerege.nexus.documents"); err != nil {
		t.Errorf("a module with no registered migrations was refused: %v", err)
	}
}

// The version table name is built into DDL, and an app id can come from a
// registry rather than from this repository.
func TestAnAppIDThatCannotNameATableIsRefused(t *testing.T) {
	for _, appID := range []string{
		`io.example.drop"; DROP TABLE users; --`,
		"io.example.Mixed_Case",
		"io.example.",
		"9leading",
	} {
		if _, err := versionTable(appID); err == nil {
			t.Errorf("%q was accepted as a migration table name", appID)
		}
	}
	got, err := versionTable("io.gerege.nexus.sso-clients")
	if err != nil || got != "goose_db_version_sso_clients" {
		t.Errorf("versionTable = %q, %v; want goose_db_version_sso_clients", got, err)
	}
}
