/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The service behind appstore.gerege.mn: the published catalogue every Nexus
 * instance pulls, and the publishing side that fills it.
 *
 * It is a separate process with a separate database and a separate domain, but
 * it lives in this module so that it and the client that consumes it share one
 * definition of the catalogue format. Two programs agreeing about bytes is
 * easier to keep true when they share the structs than when they share a
 * document describing them.
 */

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/appstore"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/async"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	port := envOr("PORT", "8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/appstore_db?sslmode=disable"
	}

	// The signing key is the whole trust story: a Nexus instance believes a
	// catalogue because of this signature and nothing else. Refusing to start
	// without one is deliberate — a registry that served unsigned catalogues
	// would be a registry every instance silently ignored.
	signer, err := appstore.NewSigner(envOr("SIGNING_KEY_ID", "appstore-2026"), os.Getenv("SIGNING_KEY"))
	if err != nil {
		slog.Error("APPSTORE signing key is missing or invalid; refusing to start", "error", err)
		os.Exit(1)
	}
	slog.Info("catalog signing key loaded",
		"key_id", envOr("SIGNING_KEY_ID", "appstore-2026"), "public_key", signer.PublicKey())

	ctx := context.Background()

	if err := migrateUp(dbURL); err != nil {
		slog.Error("failed to migrate the registry schema", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to the registry database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	server := appstore.NewServer(pool, signer, appstore.Config{
		Origin:            envOr("PUBLIC_ORIGIN", "https://appstore.gerege.mn"),
		StorefrontOrigins: splitList(envOr("ALLOWED_ORIGINS", "https://appstore.gerege.mn,https://developer.gerege.mn")),
		Issuer:            envOr("SSO_ISSUER", "https://nexus.gerege.mn"),
		ConsoleAudience:   envOr("CONSOLE_CLIENT_ID", "gerege-developer-console"),
		AdminSubjects:     splitList(os.Getenv("APPSTORE_ADMIN_SUBJECTS")),
		AdminEmails:       lowerAll(splitList(os.Getenv("APPSTORE_ADMIN_EMAILS"))),
	})

	// A registry with nothing in it is not worth deploying and then remembering
	// to fill, so the apps this platform ships with are imported on first boot.
	seedCtx, cancelSeed := context.WithTimeout(ctx, 30*time.Second)
	defer cancelSeed()
	store := appstore.NewStore(pool)
	if empty, err := store.IsEmpty(seedCtx); err != nil {
		slog.Warn("could not check whether the registry is empty", "error", err)
	} else if empty {
		catalogPath := envOr("APP_CATALOG_PATH", "/app/catalog/apps.json")
		if err := store.SeedFromCatalogFile(seedCtx, catalogPath,
			envOr("SEED_PUBLISHER_SLUG", "gerege"),
			envOr("SEED_PUBLISHER_NAME", "Gerege Systems"),
			"seed:bundled-catalog"); err != nil {
			slog.Error("could not seed the registry from the bundled catalog", "error", err)
		}
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Router(),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	async.Go("appstore-http", func() {
		slog.Info("starting the Gerege App Store registry", "port", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("registry listener error", "error", err)
			os.Exit(1)
		}
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down the registry")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced registry shutdown", "error", err)
	}
}

// migrateUp applies the embedded schema. goose wants database/sql, which is
// why this opens a second, short-lived connection rather than reusing the pool.
func migrateUp(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(appstore.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	// A cold database on a fresh host is normal, so this waits rather than
	// failing the first boot of a stack whose Postgres is still starting.
	var lastErr error
	for attempt := range 30 {
		if lastErr = db.Ping(); lastErr == nil {
			break
		}
		if attempt == 0 {
			slog.Info("waiting for the registry database")
		}
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		return fmt.Errorf("database unreachable: %w", lastErr)
	}
	return goose.Up(db, "migrations")
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	list := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return list
}

func lowerAll(values []string) []string {
	for i := range values {
		values[i] = strings.ToLower(values[i])
	}
	return values
}
