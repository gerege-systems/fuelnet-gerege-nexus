/*
 * Gerege Template Platform
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Main entry point for the HTTP API server process.
 */

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/observability"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/platform_db?sslmode=disable"
	}

	catalogPath := os.Getenv("APP_CATALOG_PATH")
	if catalogPath == "" {
		catalogPath = "catalog/apps.json"
	}

	ctx := context.Background()

	// Initialize OpenTelemetry distributed tracing
	shutdownTracing, err := observability.SetupTracing(ctx, "platform-erp", os.Getenv("ENVIRONMENT"))
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
	} else {
		defer func() {
			_ = shutdownTracing(ctx)
		}()
	}

	// Connect to shared-schema PostgreSQL database pool
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("failed to parse db url", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnIdleTime = 15 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error("failed to connect to postgres pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		slog.Warn("database ping failed on startup", "error", err)
	} else {
		// Restores the documented demo login (admin@example.com). Skipped in
		// production unless SEED_DEMO_DATA is set explicitly.
		seedInitialData(ctx, db)
	}

	// Initialize Platform Server
	srv, err := platform.NewServer(db, catalogPath)
	if err != nil {
		slog.Error("failed to initialize platform server", "error", err)
		os.Exit(1)
	}

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting Gerege Template Platform API server", "port", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listener error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down HTTP API server gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced server shutdown", "error", err)
	} else {
		slog.Info("server shutdown complete cleanly")
	}
	fmt.Println("Goodbye!")
}
