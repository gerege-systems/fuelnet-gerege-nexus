/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Database migration runner binary for PostgreSQL migrations using Goose.
 */

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/platform_db?sslmode=disable"
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		slog.Error("failed to open database for migrations", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}

	// Where the migrations are, and which table records what has run.
	//
	// Both are configurable for one reason: a distribution repository — a
	// product built on this platform with its own tables — has to migrate its
	// own schema, and it cannot do that through this binary while the directory
	// is a constant. The version table matters just as much: goose keeps one
	// row per applied version in one table, so a distribution's 00001 and the
	// platform's 00001 would be the same row. Each history needs its own table.
	//
	// Defaults are what they have always been, so a deployment that sets
	// neither behaves exactly as before.
	//
	//	MIGRATIONS_DIR=db/appstore MIGRATIONS_TABLE=goose_db_version_appstore migrate up
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "db/migrations"
	}
	if table := os.Getenv("MIGRATIONS_TABLE"); table != "" {
		goose.SetTableName(table)
	}

	fmt.Printf("Running goose migration command %q on %s...\n", command, migrationsDir)
	if err := goose.RunContext(context.Background(), command, db, migrationsDir); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	fmt.Println("Migrations executed successfully!")
}
