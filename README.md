# open-gerege-mn-erp

A production-oriented MVP of a modular business application platform inspired by Odoo's app ecosystem. Built using Go, Chi Router, PostgreSQL (pgx/v5), Goose migrations, and Next.js (App Router).

---

## 🚀 Demo Credentials

- **Admin Email**: `admin@example.com`
- **Password**: `Password123!`
- **Demo Tenant**: `Demo Corporation` (`slug: demo`)

---

## 🏛️ Architecture Overview

The system is structured as a **Modular Monolith**:
- **Compile-Time Go App Modules**: Business applications (`contacts`, `products`, `inventory`) implement a unified `Module` Go interface. Modules are compiled directly into the Go binary.
- **Tenant-Level App Store Engine**: An app being in the binary does not mean it is enabled for a tenant. Installation, enablement, and menu visibility are dynamically controlled per tenant via PostgreSQL (`app_installations`).
- **Dependency Resolution Engine**: Implements recursive dependency graph traversal and topological sorting. Installing `Inventory` automatically resolves and installs `Products` and `Contacts` in order.
- **Shared-Schema Multi-Tenancy**: Every business table (`contacts`, `products`, `warehouses`, `stock_levels`, `stock_movements`) contains `tenant_id` and is strictly scoped to the authenticated tenant.
- **Dynamic RBAC & Menus**: Backend endpoint access and frontend sidebar menus are filtered dynamically based on enabled tenant modules and user permissions.

---

## 📦 Business Modules

1. **Contacts (`io.example.contacts`)**:
   - Customer and vendor management (name, email, phone, company, active state).
   - Permissions: `contacts.read`, `contacts.manage`

2. **Products (`io.example.products`)**:
   - Product catalog with unique tenant-scoped SKUs and pricing.
   - Permissions: `products.read`, `products.manage`

3. **Inventory (`io.example.inventory`)**:
   - Requires `Contacts` and `Products`.
   - Warehouse management, live stock levels, append-only stock movement audit log, and transactional stock adjustments.
   - Prevents negative stock levels.
   - Permissions: `inventory.read`, `inventory.manage`

---

## 🛠️ Quick Start & Setup

### Prerequisites
- Go 1.24+
- Node.js 20+
- PostgreSQL 16+ (or Docker Compose)

### 1. Run with Docker Compose
```bash
docker-compose up -d
```

### 2. Manual Development Setup

#### Backend:
```bash
cd backend
go mod tidy
DATABASE_URL="postgres://postgres:postgres@localhost:5432/platform_db?sslmode=disable" go run ./cmd/migrate up
go run ./cmd/api
```

#### Frontend:
```bash
cd frontend
npm install
npm run dev
```
Open [http://localhost:3000](http://localhost:3000) and log in using `admin@example.com` / `Password123!`.

---

## 🧪 Running Tests

```bash
# Run backend unit & resolver tests
cd backend
go test ./...
```

---

## 🔐 Security & Remote Registry Future Boundaries

For this MVP, official manifests are seeded from `catalog/`. The system defines explicit interfaces (`CatalogRepository`, `PackageStorage`, `PackageVerifier`, `Installer`) designed for future remote registry expansion:
- Future remote packages stored as OCI artifacts in GitHub Container Registry.
- SHA-256 integrity verification.
- Sigstore/Cosign or Ed25519 publisher signature verification.
- Automated vulnerability scanning and SBOM checks before installation.
