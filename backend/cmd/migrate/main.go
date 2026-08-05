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
		dbURL = "postgres://postgres:postgres@localhost:5432/platform_db?sslmode=disable"
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

	migrationsDir := "db/migrations"
	fmt.Printf("Running goose migration command %q on %s...\n", command, migrationsDir)
	if err := goose.RunContext(context.Background(), command, db, migrationsDir); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	fmt.Println("Migrations executed successfully!")
}
