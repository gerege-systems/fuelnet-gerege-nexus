# Changelog

All notable changes to **open-gerege-mn-erp** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0] - 2026-08-05

### Added
- **Modular Monolith Core Architecture**:
  - Pure Go compile-time `Module` interface and global module registry (`appregistry`).
  - Tenant-level app installation, enablement, and menu visibility engine (`appinstaller`).
  - Recursive dependency resolution algorithm with cycle detection and semver validation.
- **Shared-Schema Multi-Tenancy**:
  - Context-scoped `tenant_id` isolation across all business entities and repositories.
  - Tenant app gating middleware returning `403 Forbidden` for disabled modules.
- **Business Modules (Vertical Slices)**:
  - **Contacts (`io.example.contacts`)**: Business contacts directory with full CRUD.
  - **Products (`io.example.products`)**: Product catalog management with unique tenant-scoped SKUs.
  - **Inventory (`io.example.inventory`)**: Warehouse management, live stock levels, append-only stock movement log, and transactional stock adjustments with negative stock protection.
- **Next.js App Router Admin Shell**:
  - Top navigation bar with tenant badge (`Demo Corporation`), user profile menu, and logout.
  - Dynamic sidebar navigation driven by `/api/v1/menus`.
  - App Store (`/apps`) with search, categories, dependency badges, and Install/Enable/Disable controls.
  - Installed Apps Settings (`/settings/apps`).
  - Dedicated business UIs for `/contacts`, `/products`, and `/inventory`.
- **High-Performance Resilience Engine (go-zero Inspired)**:
  - **Adaptive Circuit Breaker (`resilience/breaker.go`)**: Google SRE style sliding window adaptive circuit breaker.
  - **Adaptive Load Shedding (`resilience/loadshedder.go`)**: In-flight HTTP request concurrency shedder returning `503 Service Unavailable` under heavy traffic spikes.
  - **Singleflight Coalescing (`resilience/singleflight.go`)**: Duplicate query suppressor preventing thundering herd cache stampedes.
  - **Exponential Backoff Retry (`resilience/retry.go`)**: `DoWithRetry` execution helper for resilient DB/network operations.
- **Observability & Async Messaging**:
  - Prometheus metrics endpoint (`/metrics`) recording HTTP request rates and latency histograms (`github.com/prometheus/client_golang`).
  - OpenTelemetry tracing initialization (`SetupTracing`).
  - Async OTP Mailer queue with worker pool, retry logic, and graceful shutdown (`internal/platform/mailer`).
- **External System Integrations & Webhook Engine ([`internal/platform/integration`](backend/internal/platform/integration))**:
  - Event Dispatcher & Connector Manager supporting HMAC-SHA256 signature signing, asynchronous webhooks, and third-party REST connectors.
  - Dedicated Integration Settings Manager UI (`/settings/integrations`) with real-time status & health tracking.
- **XYP State Data Exchange System ([`xyp.gerege.mn`](backend/internal/platform/gerege/xyp.go))**:
  - Official Mongolian State Data Exchange (ХУР Төрийн мэдээлэл солилцооны систем) integration service.
  - Citizen civil registration (`POST /api/v1/xyp/citizen`) & company legal entity verification (`POST /api/v1/xyp/company`).
  - Interactive "⚡ ХУР / XYP Auto-fill" button integration on Contacts page.
- **Database & Migrations**:
  - Goose SQL migrations (`00001_platform_core.sql`, `00002_app_store.sql`, `00003_business_apps.sql`).
  - Automated initial demo data seeder (`admin@example.com` / `Password123!`).

---

### Inspirations & Acknowledgements
- **[snykk/go-rest-boilerplate](https://github.com/snykk/go-rest-boilerplate)** by [@snykk](https://github.com/snykk): Initial Go REST API structure.
- **[Odoo](https://github.com/odoo/odoo)**: Modular app ecosystem, App Store dependency resolver, and dynamic menu architecture.
- **[go-zero](https://github.com/zeromicro/go-zero)**: High-performance cloud-native resilience engine (Adaptive Circuit Breaker, Load Shedder, Singleflight, Exponential Retry).

### Authors
- **Gerege Systems Development Team & Gemini AI**
