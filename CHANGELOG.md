# Changelog

All notable changes to **open-gerege-mn-erp** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- **PDF E-Sign App Module ([`io.example.esign`](backend/internal/apps/esign))**:
  - PDF document upload with tenant-scoped storage (migration `00009`), page-count detection, and original/signed download endpoints (`/api/v1/esign`).
  - Digital signature (тоон гарын үсэг) certificate validation and PKCS#7 PDF signing via the Gerege eSign HSM platform client ([`internal/platform/gerege/esign.go`](backend/internal/platform/gerege/esign.go)) — the private signing key never leaves the HSM.
  - Visible signature stamp placement with last-page auto-targeting and signature audit log (`esign_signature_logs`).
  - Frontend signing flow (`/esign`): certificate check → canvas signature pad → HSM signing → signed PDF download.
  - Mock mode by default (`ESIGN_MOCK_MODE`); configure `ESIGN_LOGIN_URL`, `ESIGN_SIGN_URL`, and `ESIGN_TOKEN` for live signing.

### Fixed — CI/CD pipeline

- **Go toolchain mismatch broke every job**: `backend/go.mod` requires `go 1.25.7`
  while the workflows pinned `go-version: "1.24"` and both Dockerfiles used
  `golang:1.24-alpine` (which sets `GOTOOLCHAIN=local`, so the build hard-fails).
  All jobs now resolve the version from `backend/go.mod` and the image builder
  sets `GOTOOLCHAIN=auto`.
- **Security workflow could never pass**: `govulncheck ./...` and `gosec ./...`
  ran at the repository root, which contains no `go.mod`. They now run against
  `backend/`. The workflow also only triggered on PRs to `master` while the
  default branch is `main`.
- **GHCR push lacked `packages: write`**, so deployment failed with `denied:
  installation not allowed to Write the repository`.
- **Removed the `swag-drift` job**: the sources carry no swagger annotations and
  `backend/docs/` is untracked, so it could only fail or pass vacuously.
- `deploy.yml` no longer duplicates lint/test from `ci.yml`; it runs migrations
  before swapping the API over, uses `docker compose`, lower-cases the GHCR
  image path, and skips cleanly when deployment secrets are absent.
- Added a **frontend CI job** (`npm ci` + `tsc --noEmit` + `next build`) — the
  Next.js app was never built by CI.
- Added `.dockerignore`, a pinned `backend/.golangci.yml` (v2), deleted the
  duplicate `backend/Dockerfile`, untracked committed `.DS_Store` files, and
  `gofmt`-ed the 11 files that had drifted.
- `docker-compose.yml`: added a one-shot **migration service** (the API used to
  start against an empty schema), health checks, and build-time
  `NEXT_PUBLIC_API_URL`; `frontend/Dockerfile` now uses `npm ci` with the lock
  file. Database credentials are consistent across compose, `.env.example` and
  the Makefile.

### Fixed — Security

- **Session tokens were the user's UUID**, the same value returned by
  `/api/v1/auth/me`. Replaced with opaque 256-bit `crypto/rand` tokens stored as
  SHA-256 digests in a new `sessions` table, with expiry and real revocation on
  logout (logout previously only dropped the cookie).
- **Mock national-identity mode was on by default** (`os.Getenv(...) != "false"`),
  so in production `/auth/eid/login` and `/auth/dan/login` accepted any
  registration number and logged the caller in as the first user in the table
  with `is_admin: true`. Mock mode is now refused in production unless requested
  explicitly, and identities are matched against a real ERP user.
- **OAuth2 token endpoint accepted any known `client_id` with no secret**
  (`clientSecret != "" && ...` skipped the check entirely). Client
  authentication is now mandatory and constant-time, supports HTTP Basic,
  validates the grant type, and is also enforced on `/oauth2/introspect` and
  `/oauth2/revoke`.
- Removed the **hard-coded client secret** `secret_gerege_dev_2026`;
  `ListClients` no longer discloses client secrets.
- App install/enable/disable and integration registration now require a **tenant
  administrator** — any authenticated user could previously reconfigure the
  tenant.
- Login rate limiting no longer trusts `X-Forwarded-For` unless
  `TRUST_PROXY_HEADERS=true`.
- Halved-entropy `generateRandomString` (hex output truncated back to `n`) fixed.
- `/metrics` no longer labels unmatched routes with the raw request path —
  unbounded Prometheus cardinality driven by unauthenticated requests.

### Fixed — App store & modules

- **Billing, Documents and the Developer Portal could not be installed**: their
  modules were never registered in `appregistry`, their rows were missing from
  the `apps` table (foreign-key violation), and `developer_portal` was rejected
  by the slug validator, which forbade underscores.
- The `apps` table is now **synchronised from `catalog/apps.json` on boot**
  instead of a hand-maintained INSERT that listed three of six apps.
- **Three manifests were malformed** (`"dependencies": {}` instead of an array,
  permissions as plain strings, `depends`/`sequence`/`action` keys). They parsed
  into a silent stub with no dependencies, permissions or menus. Manifests are
  fixed, and a manifest that fails to load is now a startup error.
- Billing and Documents no longer create their tables at boot with the error
  discarded; the schema moved into migration `00004`. Both stop answering failed
  writes with **fabricated demo records** (`inv_demo_100`, `doc_demo_200`).
- The app store reported disabled apps as "not installed" because `installed`
  and `enabled` were both derived from the enabled-only query.
- Fixed a nil-interface panic in `InstallApp` when a module was missing from the
  registry, and a nil-pointer dereference in the E-ID/DAN login handlers that
  called `err.Error()` on a nil error.

### Fixed — Reliability

- `AsyncOTPMailer.Shutdown` closed the queue while workers and retries could
  still send on it (**panic: send on closed channel**) and dropped already-queued
  mail; it now drains, is idempotent, and refuses post-shutdown enqueues.
- AI Copilot intent classification was case-sensitive against a lowercase
  keyword table, so "Stock" never matched.
- Restored demo-data seeding (dropped from `cmd/api`), now idempotent and
  disabled in production unless `SEED_DEMO_DATA` is set.

---

## [0.1.0] - 2026-08-05

### Added
- **Modular Monolith Core Architecture**:
  - Pure Go compile-time `Module` interface and global module registry (`appregistry`).
  - Tenant-level app installation, enablement, and menu visibility engine (`appinstaller`).
  - **ORY Hydra-Grade OAuth2 & OpenID Connect (OIDC) SSO Provider ([`internal/platform/ssoprovider`](backend/internal/platform/ssoprovider))**:
  - OpenID Connect Discovery (`/.well-known/openid-configuration`), JWKS URI (`/.well-known/jwks.json`), and OAuth2 Authorization Server (`/oauth2/token`, `/oauth2/introspect`, `/oauth2/revoke`).
  - Supports `authorization_code`, `client_credentials`, and `refresh_token` grant flows.
- **Developer Portal App Module ([`io.example.developer_portal`](backend/internal/apps/developer_portal))**:
  - Developer portal interface (`/developer/apps`) to register third-party OAuth2 client applications, issue Client IDs and Client Secrets, and manage redirect URIs.
- **Automated Production Deployment & CI/CD Pipeline ([`openerp.gerege.mn`](.github/workflows/deploy.yml))**:
  - Continuous Integration & Automated Deployment pipeline building GHCR Docker images and deploying to `openerp.gerege.mn`.
  - Production Multi-Stage Dockerfile ([`deploy/Dockerfile`](deploy/Dockerfile)) and Nginx SSL Reverse Proxy config ([`deploy/nginx/openerp.gerege.mn.conf`](deploy/nginx/openerp.gerege.mn.conf)).
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
- **Public Billing & e-Barimt Module ([`io.example.billing`](backend/internal/apps/billing))**:
  - Public service fee invoices, 10% VAT calculation for Mongolia e-Barimt, and status tracking (`/billing`).
- **Gerege DAN SSO Gateway System ([`dan.gerege.mn`](backend/internal/platform/dan))**:
  - Official Gerege Systems DAN SSO Gateway integration service (`POST /api/v1/auth/dan/login`).
  - Citizen identity verification and session token validation against `https://dan.gerege.mn/api/v1`.
- **E-ID Digital Identity & DAN SSO Authentication ([`internal/platform/eid`](backend/internal/platform/eid))**:
  - Aligned 100% with official **[eidmongolia.mn](https://eidmongolia.mn)** & **[developer.gerege.mn](https://developer.gerege.mn)** OAuth2 and OpenID Connect (OIDC) specifications.
  - Supports 4 official Mongolian authentication channels: PKI Digital Signature (Тоон гарын үсэг), Mobile OTP, Bank SSO, and Biometric Face Verification.
- **External System Integrations & Webhook Engine ([`internal/platform/integration`](backend/internal/platform/integration))**:
  - Event Dispatcher & Connector Manager supporting HMAC-SHA256 signature signing, asynchronous webhooks, and third-party REST connectors.
  - Dedicated Integration Settings Manager UI (`/settings/integrations`) with real-time status & health tracking.
- **XYP State Data Exchange System ([`xyp.gerege.mn`](backend/internal/platform/gerege/xyp.go))**:
  - Official Mongolian State Data Exchange (ХУР Төрийн мэдээлэл солилцооны систем) integration service.
  - Citizen civil registration (`POST /api/v1/xyp/citizen`) & company legal entity verification (`POST /api/v1/xyp/company`).
  - Interactive "ХУР / XYP Auto-fill" button integration on Contacts page.
- **Database & Migrations**:
  - Goose SQL migrations (`00001_platform_core.sql`, `00002_app_store.sql`, `00003_business_apps.sql`).
  - Automated initial demo data seeder (`admin@example.com` / `Password123!`).

---

### Inspirations & Acknowledgements
- **[snykk/go-rest-boilerplate](https://github.com/snykk/go-rest-boilerplate)** by [@snykk](https://github.com/snykk): Initial Go REST API structure.
- **[Odoo](https://github.com/odoo/odoo)**: Modular app ecosystem, App Store dependency resolver, and dynamic menu architecture.
- **[go-zero](https://github.com/zeromicro/go-zero)**: High-performance cloud-native resilience engine (Adaptive Circuit Breaker, Load Shedder, Singleflight, Exponential Retry).

### Authors & Contributors
- **Gerege Systems Development Team**
- **Gemini AI**
- **Claude AI**
