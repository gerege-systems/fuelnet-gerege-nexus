# Changelog

All notable changes to **open-gerege-nexus** (Gerege Nexus) will be documented in
this file.

Entries below the rebrand keep the names that were true when they shipped — the
`open-gerege-mn-erp` repository, the ERP framing, and the `openerp.gerege.mn`
deployment, which has since moved to `nexus.gerege.mn`. A changelog edited to
match the present tense stops being a record.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added — Tauri v2 desktop shell ([`desktop-tauri/`](desktop-tauri))

- **A second implementation of one contract, not a second product.** The bridge
  contract ([`docs/SHELL_CONTRACT.md`](docs/SHELL_CONTRACT.md)) is the
  specification; [`desktop-mac/`](desktop-mac) is its Swift reference and this is
  the cross-platform one. Both inject the same `window.GeregeShell`, so the web
  app cannot tell them apart — it hides its own chrome and renders as a work area
  either way, and in a browser neither exists and nothing changes.
- **`platform` comes from the build target** (`macos`, `windows`, `linux`) and
  reaches the styling as `<html data-shell>`. Declared capabilities are
  `notify`, `badge`, `external.open`, `print.system`, `fs.save`, `menu.native`.
- **Native sign-in window** with email/password and both eID flows. The polling
  loop is Rust ([`auth.rs`](desktop-tauri/src-tauri/src/auth.rs)) and carries the
  same reasoning as [`EIDLogin.tsx`](frontend/components/EIDLogin.tsx): one check
  in flight at a time, a 400 ms gap between them because the server already holds
  each request for 25 s, three tolerated failures because a dropped long-poll is
  ordinary on a mobile network, and a 15-minute backstop that is a stop condition
  rather than a deadline. The QR is rendered to SVG in Rust so the window depends
  on no JavaScript library.
- **The session cookie forced the transport.** `session_token` is `HttpOnly` and
  belongs to the API origin, the web app authenticates with `credentials:
  "include"` and no bearer header, and neither Tauri nor wry can write a cookie
  into a webview from outside. The only way it lands in the right jar is for that
  webview to receive the `Set-Cookie` itself, so the sign-in requests are issued
  there ([`bridge.rs`](desktop-tauri/src-tauri/src/bridge.rs)) while the flow
  logic stays in Rust. The work-area window is created hidden and stays hidden
  until sign-in completes.
- **The native menu is the tenant's menu.** `GET /api/v1/menus` with the
  `Accept-Language` the person chose, grouped per app; `menu.changed` rebuilds
  it; choosing an item emits `shell:navigate` so the work area routes without a
  full reload. macOS maps a small set of icon names to native symbols and leaves
  the rest bare, which is steadier than half-matching them.
- **Server health lives on the tray icon**, checked every 5 s the way
  [`ServerManager.swift`](desktop-mac/src/ServerManager.swift) does it. Tauri has
  no native status bar and drawing an HTML one under the work area would put the
  shell inside the page it is supposed to stay out of. Being offline opens a
  native window that says what has to be running, not an alert that vanishes when
  dismissed.
- **Security**: main-frame navigation is confined to the Web URL's origin and
  everything else opens in the system browser; the bridge is main-frame only and
  the remote origin allowed to reach IPC is pinned in
  [`capabilities/`](desktop-tauri/src-tauri/capabilities); every native→web value
  is JSON-encoded rather than concatenated into JavaScript; `external.open`
  accepts only `http`, `https`, `mailto`, `tel`; `fs.saveAs` writes only where the
  person pointed. In a release build the API and Web URLs are compile-time
  constants — an installed shell cannot be aimed at a server it was not built for.
- **`gerege://` deep links** resolve to `shell:navigate`.
- **Auto-update is present and deliberately inert.** The plugin is left
  uninitialised with `TODO`s in three places; an updater carrying no signing key
  is a mechanism for installing unsigned code, so it stays off until a key exists.
- **Two capabilities are withheld, and why is recorded.** `secure-store` has no
  method in contract v1, so advertising it would be a claim nothing can act on —
  using it needs `secure.get`/`set`/`delete` added to the contract and a minor
  version bump. `biometric.authenticate` exists in the contract but Tauri's
  biometric plugin is mobile-only, so the capability is not declared and the call
  is rejected, which is what lets the web app fall back.
- **Not included**: installers and code signing. The shell builds; shipping it
  needs a Developer ID identity plus notarisation on macOS and an Authenticode
  certificate on Windows, both listed as TODO in
  [`desktop-tauri/README.md`](desktop-tauri/README.md).

### Added — CI for both desktop shells

- **[`desktop-tauri.yml`](.github/workflows/desktop-tauri.yml) builds on Linux,
  Windows and macOS.** Much of the shell sits behind `#[cfg(target_os = ...)]`, so
  a green build on one machine says nothing about the other two. It runs
  `cargo clippy --all-targets -- -D warnings`, `cargo build --locked` and
  `cargo test --locked`, with `fail-fast: false` because one platform failing is
  the signal the job exists to produce.
- **It found a real break on its first run**: `tauri-build` needs
  `icons/icon.ico` to generate the Windows resource, and the repository had only
  PNGs. Added `icon.ico` — sizes below 256 packed as classic DIB entries, since
  some resource compilers reject an all-PNG `.ico` — and `icon.icns` for macOS
  bundling.
- **[`desktop-mac.yml`](.github/workflows/desktop-mac.yml) compiles the Swift
  shell** and then checks the two things a successful `swiftc` cannot: that
  `build.sh` still names every file under `src/` (a source missing from that fixed
  list is not a compile error — it is code that silently never ships), and that
  the produced bundle is one macOS would launch (`Info.plist`, an executable
  Mach-O, `codesign --verify --strict`).
- **The bridge fixes are guarded, not just documented.** The job fails on a
  `WKUserScript` injected into subframes or on JavaScript built by string
  interpolation — the exact two shapes that were removed. Both guards were checked
  against the pre-fix sources to confirm they actually catch them rather than
  passing vacuously.
- **Neither workflow produces a distributable artifact**, and both are filtered by
  path. A path-filtered workflow reports no status on runs that miss its filter,
  so making either a required check needs a merge queue or a companion job.

### Added — Native Shell + Web Work Area

- **The web app now knows whether it is a whole product or part of one.** Inside a
  native shell, sign-in, the header, the menus and device access belong to the
  shell; the web app hides its own chrome and renders as a **work area**. In a
  browser there is no shell, and everything below evaluates to nothing — the
  browser rendering is unchanged to the pixel, which is the constraint the whole
  design is built around rather than an afterthought.
- **One contract, written down** ([`docs/SHELL_CONTRACT.md`](docs/SHELL_CONTRACT.md)):
  injection rules, every method's parameters, result and failure, every event's
  payload, the capability names, the versioning rule (adding is minor, changing is
  major, and the shell announces its own version), and the security requirements a
  shell must meet. Two shells written by different people meet here or not at all.
- **`window.GeregeShell` in TypeScript** ([`frontend/lib/shell.ts`](frontend/lib/shell.ts)):
  `getShell()` returns `null` during SSR and in a browser, `hasCapability()`,
  a `useShell()` hook, and `invokeShell()` — an invoke that neither throws nor
  hangs, because callers mostly need to know whether the shell took the request,
  and "not supported", "failed" and "never answered" all mean the same thing: run
  the web fallback. Method, event and capability names are constants, so renaming
  one is a compiler error rather than a silent no-op.
- **Chromeless rendering** ([`Layout.tsx`](frontend/components/Layout.tsx)): in a
  shell the top bar, sidebar, mobile tabs and drawer are not rendered at all, but
  the menu and user fetches still run — RBAC and access checks depend on them, and
  only the drawing is removed. The AI assistant stays; it is part of the work area.
- **Session expiry asks the shell first.** There is no web `/login` page inside a
  shell, so a 401 calls `auth.reLogin` and falls back to `router.push("/login")`
  only if the shell will not, cannot, or does not answer — attempted once per
  session, so a re-login that leaves the session invalid cannot loop.
- **The two halves talk over the contract, not over URLs.** A menu change tells the
  shell with `menu.changed` so it can rebuild its native menu; the shell moves the
  work area with `shell:navigate` (internal paths only — a protocol-relative
  `//host` is not one) and opens its search with `shell:search`.
- **Native-leaning styling, scoped by attribute**
  ([`theme.tsx`](frontend/lib/theme.tsx), [`globals.css`](frontend/app/globals.css)):
  the shell's platform lands on `<html data-shell>`, which switches the app to the
  host's system font stack and chrome-free spacing, with a few per-platform
  touches. The attribute is absent in a browser, so no rule can reach it. The block
  sits above the density rules on purpose — a person who chose "compact" must not
  have it overruled by being in a shell.

### Fixed — Security: the macOS shell's JavaScript bridge

- **Native results were concatenated into JavaScript.** The biometric callback was
  assembled as `onBiometricResult('\(cb)', \(success), '\(err)')`, so a single
  quote anywhere in a system error message ran as code in the work area. The
  toolbar search field went the same way, which made anything the user typed a
  script. Every native→web value is now JSON-encoded and returned through one
  entry point ([`WebViewController.swift`](desktop-mac/src/WebViewController.swift)).
- **The bridge was injected into every frame.** `WKUserScript` is now main-frame
  only, and each message is checked twice — `isMainFrame`, and that the frame's
  origin matches the platform's web origin. An embedded third-party page has no
  business reaching biometrics, files or notifications.
- **The main frame could navigate anywhere.** It is now confined to an explicit
  allowlist — the web and API origins plus named identity origins — and every other
  address opens in the system browser rather than beside our session and our
  bridge. Deployments whose integration consent screens must stay in-app can name
  those origins in `gerege_nav_allowlist`; unlisted ones continue in the browser
  rather than breaking.
- **No functional regression**: tray, toolbar, printing, downloads and
  `gerege://` deep links continue to work, and deep links and menu items now move
  the work area through the router instead of reloading it, which no longer
  discards a half-filled form.

### Added — Email verification as a platform capability

- **One flow instead of one per app**
  ([`internal/platform/emailverify`](backend/internal/platform/emailverify)):
  proving that somebody controls an address is not one module's business.
  Contacts wants it before it trusts an address, Documents before a signing link
  leaves for an outsider, Gov Services before it answers a citizen at one. Each
  is the same act, so it lives in the platform: an app module takes the service
  in its constructor the way `gov_services` takes the integration manager and
  calls `emailverify.Service.Send` with its own app id as the source.
- **The mail is sent by a hosted service, deliberately.** Delivering mail that
  arrives is not a matter of holding an SMTP password: it is SPF, DKIM, DMARC,
  reverse DNS and a sending reputation, maintained continuously. enigma.mn runs
  that, so this platform holds no mailbox credential, composes no message and
  owns no sender address. What stays here is what only this platform can know —
  which module asked, for whom, why, and whether the person came back.
- **The return is good exactly once** (migrations `00026`, `00027`): the request
  carries a single-use reference in the return address, stored as a SHA-256, and
  claimed by one conditional `UPDATE`. A browser reloading the landing page
  races itself, and a reference that travelled through a mailbox and a browser's
  history must not be replayable. A spent, expired or invented reference is
  `410` alike.
- **The platform is not an open redirector**: the onward destination is
  validated when the request is made — HTTPS only (HTTP tolerated for localhost
  outside production), no embedded credentials — not when somebody arrives, by
  which time the mail has gone.
- **Mail bombing has a cost**: a per-tenant hourly allowance in front of the
  shared key and a one-minute pause per recipient, answered `429` with a
  `Retry-After` somebody can obey — a limit we can avoid provoking upstream is
  one we do not have to explain. A request the service refuses withdraws its own
  row, so the Overview screen never shows a verification nobody was asked for.
- **Errors say who has to act**: a bad address is `400`, a missing key or an
  HTTP `PUBLIC_ORIGIN` or a rejected key is `503` (this deployment, not the
  request), and a failure at the service is `502` and retryable. An answer
  nobody documented is never read as success.
- **Settings → Email verification**: whether the service is reachable, what has
  been asked for and by whom, the verified rate, and a test send. No key
  management — keys belong to the sending service and are administered there,
  and this platform's copy is a server-side environment variable that never
  reaches a browser.
- **The page shown after a click exists in all seven platform languages.** It is
  read outside the product, by somebody who may never have seen it.
- **Known limitation, stated on the screen**: the service has no webhook yet, so
  a verification is recorded only when the person returns here. Somebody who
  confirms on another device and never comes back stays `PENDING`. That is the
  honest reading — this platform did not see it happen — and it is what the
  Overview screen says rather than something the code knows and the operator
  does not.

### Added — PDF E-Sign v2: eID Mongolia qualified remote signing

- **eID Mongolia signature client ([`internal/platform/eidsign`](backend/internal/platform/eidsign))**:
  a real relying-party client for the v3 signature API. The citizen's own device
  holds the private key and approves with PIN2, so nothing here ever touches a
  signing key: we hash the PDF, eID pushes that digest to the phone, and the
  signed document is assembled by eID's own doc-signer
  (`POST /v3/signature/stamp/{sessionId}`), which embeds the PKCS#7 together with
  OCSP and CRL data. Certificate level defaults to `QUALIFIED` — accepting
  `ADVANCED` would silently downgrade every document the ERP produces.
- **Asynchronous signing ceremony** (`/api/v1/esign/sign/init`, `/sign/{id}`,
  `/sign/{id}/download`, `/sign/{id}/cancel`): upload → verification code →
  PIN2 on the phone → long-poll → PAdES-signed PDF. Sessions carry the exact
  bytes whose digest was approved, so a document edited mid-ceremony cannot
  produce a signature that fails to verify.
- **Signing on behalf of an organisation**: representation rights are read live
  from the national registry rather than from a certificate, because a director
  who resigned yesterday still holds yesterday's certificate.
- **eID identity linkage** (`user_eid_identities`, migration `00010`): sign-in
  now records who a user is to eID. Without it every signature would make the
  citizen retype the registration number they had just authenticated with, and a
  typo would push the PIN2 prompt at somebody else's phone.
- **The five module screens are now real** — signature log (filters, pagination,
  CSV export), batch signing, stamp placement (with an A4 preview), HSM
  connection (read-only, with a connection probe) and signing policy — replacing
  the `/module/esign/*` coming-soon placeholders.
- **Signing policy**: a tenant can require qualified eID signatures and disable
  the HSM rail outright, including for callers hitting the API directly.

### Fixed — PDF E-Sign

- **The app's permissions were declared but never enforced.** `io.example.esign`
  is absent from the platform's blanket app gate
  ([`server.go`](backend/internal/platform/server.go)) and its handlers only
  checked the tenant, so anyone in a tenant could sign. Every route now asserts
  `esign.read`, `esign.sign` or `esign.manage` explicitly. Migration `00010`
  backfills the grants existing roles should already have had, so no current
  user loses access.
- **`esign.sign` was ungrantable by the installer**: the default-role rules key
  off a `.read`/`.manage` suffix, so only administrators would ever have
  received it.
- **The signature log recorded only successes**, so a refused or expired
  ceremony left no trace — exactly the event an auditor looks for. Failures,
  refusals, expiries and downloads are now recorded with an outcome.
- **Non-ASCII download filenames were mangled**: signed PDFs are now served with
  an RFC 5987 `filename*`, so a Cyrillic document keeps its name.
- **Truncated PDFs were accepted**: a valid `%PDF-` header on a truncated body
  was passed to the signing service, which returned something that would not
  open. Uploads are now checked for a trailer as well as a header.
- **Sidebar sub-menus rendered as identical grey boxes**: the icons named by the
  server's menu blueprints were never mapped in the frontend
  ([`Layout.tsx`](frontend/components/Layout.tsx)).

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
  - Production Multi-Stage Dockerfile ([`deploy/Dockerfile`](deploy/Dockerfile)) and Nginx SSL Reverse Proxy config (then `deploy/nginx/openerp.gerege.mn.conf`, since renamed to [`deploy/nginx/nexus.gerege.mn.conf`](deploy/nginx/nexus.gerege.mn.conf)).
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
