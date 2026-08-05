package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/observability"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/platform_db?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	catalogPath := os.Getenv("CATALOG_PATH")
	if catalogPath == "" {
		catalogPath = "catalog/apps.json"
	}

	ctx := context.Background()

	shutdownTracing, err := observability.SetupTracing(ctx, "platform-erp", os.Getenv("ENVIRONMENT"))
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
	} else {
		defer shutdownTracing(ctx)
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("failed to parse db url", "error", err)
		os.Exit(1)
	}

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		slog.Error("failed to connect to db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		slog.Warn("database ping failed, attempting connection in background...", "error", err)
	} else {
		slog.Info("connected to PostgreSQL successfully")
		seedInitialData(ctx, db)
	}

	srv, err := platform.NewServer(db, catalogPath)
	if err != nil {
		slog.Error("failed to initialize platform server", "error", err)
		os.Exit(1)
	}

	slog.Info("starting ERP Platform API server", "port", port)
	if err := http.ListenAndServe(":"+port, srv.Router()); err != nil {
		slog.Error("server exited with error", "error", err)
	}
}

func seedInitialData(ctx context.Context, db *pgxpool.Pool) {
	// Seed demo tenant and admin user if empty
	var tenantID string
	err := db.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = 'demo' LIMIT 1`).Scan(&tenantID)
	if err != nil {
		tenantID = "00000000-0000-0000-0000-000000000001"
		_, err = db.Exec(ctx, `INSERT INTO tenants (id, slug, name, created_at) VALUES ($1, 'demo', 'Demo Corporation', $2) ON CONFLICT DO NOTHING`, tenantID, time.Now())
		if err != nil {
			slog.Error("failed to seed demo tenant", "error", err)
			return
		}
	}

	var userID string
	err = db.QueryRow(ctx, `SELECT id FROM users WHERE email = 'admin@example.com' LIMIT 1`).Scan(&userID)
	if err != nil {
		userID = "00000000-0000-0000-0000-000000000002"
		passHash, _ := auth.HashPassword("Password123!")
		_, err = db.Exec(ctx,
			`INSERT INTO users (id, email, password_hash, name, is_admin, created_at)
			 VALUES ($1, 'admin@example.com', $2, 'System Admin', TRUE, $3) ON CONFLICT DO NOTHING`,
			userID, passHash, time.Now())
		if err != nil {
			slog.Error("failed to seed admin user", "error", err)
			return
		}

		// Create membership
		_, _ = db.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, tenantID, userID)

		// Create admin role for tenant
		roleID := "00000000-0000-0000-0000-000000000003"
		_, _ = db.Exec(ctx, `INSERT INTO roles (id, tenant_id, code, name) VALUES ($1, $2, 'admin', 'Tenant Admin') ON CONFLICT DO NOTHING`, roleID, tenantID)

		memID := ""
		_ = db.QueryRow(ctx, `SELECT id FROM memberships WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID).Scan(&memID)
		if memID != "" {
			_, _ = db.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, memID, roleID)
		}
	}

	// Seed apps catalog into apps table
	_, _ = db.Exec(ctx,
		`INSERT INTO apps (id, slug, name, description, icon_url, category, visibility)
		 VALUES 
		 ('io.example.contacts', 'contacts', 'Contacts', 'Manage business contacts, customers, and vendors.', '/icons/contacts.png', 'CRM', 'public'),
		 ('io.example.products', 'products', 'Products', 'Product catalog management, pricing, and SKUs.', '/icons/products.png', 'Sales', 'public'),
		 ('io.example.inventory', 'inventory', 'Inventory', 'Warehouse management, stock tracking, and adjustments.', '/icons/inventory.png', 'Supply Chain', 'public')
		 ON CONFLICT (id) DO NOTHING`)

	fmt.Println("Initial platform seed completed: admin@example.com / Password123!")
}
