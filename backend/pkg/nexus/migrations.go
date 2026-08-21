/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package nexus

import (
	"io/fs"
	"log/slog"
	"sync"
)

// A module's own schema.
//
// db/migrations holds 72 files and 104 tables, and 28 of those tables belong to
// apps that are no longer in this repository — commerce, point of sale, the
// government services workflow. They cannot leave, because the only place a
// table can be created is the platform's migration directory, so every
// deployment carries the schema of every app anybody ever wrote here.
//
// The mechanism to fix that has existed since cmd/migrate learned MIGRATIONS_DIR
// and MIGRATIONS_TABLE; it was never connected to modules. This is the
// connection. A module embeds its migrations and hands them over from its
// constructor, beside Register:
//
//	//go:embed migrations/*.sql
//	var migrations embed.FS
//
//	func New(p nexus.Platform) *Module {
//	    m := &Module{db: p.DB()}
//	    nexus.Register(m)
//	    nexus.Migrations(m.ID(), must(fs.Sub(migrations, "migrations")))
//	    return m
//	}
//
// Each module gets its own goose version table — goose_db_version_<slug> — for
// the reason cmd/migrate already wrote down: goose keeps one row per applied
// version in one table, so a module's 00001 and the platform's 00001 would be
// the same row.
var (
	migrationMu sync.RWMutex
	migrationFS = map[string]fs.FS{}
)

// Migrations records the schema a module brings with it.
//
// Called from a constructor, like Register. Last writer wins and an overwrite
// is logged, the same rule and the same reasoning as Provide: two modules
// claiming one id is a build mistake, and there is nothing later that will
// surface this one.
//
// A nil filesystem withdraws the registration rather than storing an empty one,
// so a test can undo itself.
func Migrations(moduleID string, fsys fs.FS) {
	migrationMu.Lock()
	defer migrationMu.Unlock()
	if fsys == nil {
		delete(migrationFS, moduleID)
		return
	}
	if _, replaced := migrationFS[moduleID]; replaced {
		slog.Warn("a module registered migrations twice and the later ones win", "module", moduleID)
	}
	migrationFS[moduleID] = fsys
}

// MigrationsOf returns what a module registered, if anything.
//
// Most modules register nothing: the platform's own apps still live in
// db/migrations and will until each is moved deliberately, with a data
// migration. A module with no schema of its own is the ordinary case, not a
// missing registration.
func MigrationsOf(moduleID string) (fs.FS, bool) {
	migrationMu.RLock()
	defer migrationMu.RUnlock()
	fsys, ok := migrationFS[moduleID]
	return fsys, ok
}
