# open-gerege-mn-erp

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8.svg)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-15.1-black.svg)](https://nextjs.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

An open-source, production-oriented **Modular Monolith ERP & Business Application Platform** inspired by Odoo's app ecosystem. Built for enterprise scalability using **Go 1.24**, **Chi Router**, **PostgreSQL (pgx/v5)**, **Goose Migrations**, and **Next.js (App Router)**.

---

## 👥 Authors & Maintainers

This open-source project is created and maintained by:
- 🏛️ **Gerege Systems Development Team**
- 🤖 **Gemini AI**

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
- **Observability & Async Workers**: Includes Prometheus metrics (`/metrics`), OpenTelemetry tracing, and an asynchronous OTP Mailer queue with worker pool.

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

# Frontend build check
cd frontend
npm run build
```

---

## 📚 Documentation Index

- 📘 [Module Authoring Guide](docs/MODULE_AUTHORING_GUIDE.md) - How to build custom business modules
- 🤝 [Contributing Guidelines](CONTRIBUTING.md) - How to submit bug reports and PRs
- 🛡️ [Security Policy](SECURITY.md) - Vulnerability reporting and security features
- 📜 [Code of Conduct](CODE_OF_CONDUCT.md) - Community standards
- 📋 [Changelog](CHANGELOG.md) - Release history and versioning

---

## 🙏 Acknowledgements & Inspiration

This open-source platform draws design inspiration and architectural patterns from two outstanding open-source projects:

1. **[Odoo](https://github.com/odoo/odoo)** — Inspired our modular business application system, tenant-level App Store installer, dynamic menu composition, and topological module dependency resolution.
2. **[go-zero (zeromicro/go-zero)](https://github.com/zeromicro/go-zero)** — Inspired our cloud-native resilience engine, including Google SRE adaptive circuit breaking (`resilience/breaker.go`), in-flight load shedding (`resilience/loadshedder.go`), singleflight query coalescing (`resilience/singleflight.go`), and exponential backoff retries (`resilience/retry.go`).

---

## 📄 License

Distributed under the **Apache 2.0 License**. See [`LICENSE`](LICENSE) for more information.

Copyright (c) 2026 **Gerege Systems Development Team & Gemini AI**.
