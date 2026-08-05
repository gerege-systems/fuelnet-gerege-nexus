# 🏛️ Gerege Template ERP Platform — Technical Architecture & Capabilities Specification

Welcome to the comprehensive technical architecture specification for **Gerege Template ERP Platform**, an open-source, production-grade Modular Monolith ERP and business application engine built for enterprise scalability, resilience, and national digital infrastructure integration.

---

## 🚀 Key Advantages & Architecture Highlights

### 1. ⚡ High-Performance Modular Monolith Architecture
Unlike traditional microservice architectures that introduce network latency and operational complexity, or monolithic applications that suffer from tight coupling, **Gerege Template ERP Platform** implements a **Compile-Time Go Modular Monolith**:
- **Zero-Latency In-Process Invocations**: Business modules (`contacts`, `products`, `inventory`, `billing`, `documents`) implement a strictly defined Go `Module` interface and compile directly into the single binary.
- **Dynamic Tenant-Level App Store Gating**: An application being present in the Go binary does **not** mean it is enabled for a tenant. Installation, enablement, dynamic menu composition, and RBAC permissions are evaluated dynamically at runtime per tenant in PostgreSQL (`app_installations`).
- **Topological Dependency Resolution Engine**: Pure Go recursive dependency resolver using Directed Acyclic Graphs (DAG) and semver constraints. Installing `Inventory` automatically resolves and installs `Products` and `Contacts` in order, with full cycle detection.

### 2. 🛡️ Cloud-Native Resilience Engine (go-zero Inspired)
Located in [`backend/internal/platform/resilience`](../backend/internal/platform/resilience), the platform incorporates Google SRE-level resilience mechanics:
- **Adaptive Circuit Breaker (`breaker.go`)**: Implements sliding window error-rate monitoring to shed traffic automatically when downstream databases or services fail, preventing cascading failures.
- **Adaptive In-Flight Load Shedder (`loadshedder.go`)**: Monitors active HTTP concurrency and CPU load. Under extreme traffic spikes, it sheds load gracefully by returning `503 Service Unavailable` with `Retry-After` headers.
- **Singleflight Query Coalescing (`singleflight.go`)**: Suppresses duplicate concurrent queries, guaranteeing that thundering herd / cache stampede traffic executes database queries exactly **once** while sharing the result across all callers.
- **Exponential Backoff Retries (`retry.go`)**: Smart retry execution helper (`DoWithRetry`) for transient network or database failures.

### 3. 🇲🇳 National Digital Infrastructure & State Data Exchange Integration
The platform comes pre-configured with native integration layers for Mongolian enterprise ecosystem standards:
- **`xyp.gerege.mn` / `platform/gerege/xyp.go`**: Official Mongolian State Data Exchange (ХУР Төрийн мэдээлэл солилцооны систем) integration for instant Citizen Civil Registration (`WS100101`) and Legal Entity/Company verification (`WS100201`).
- **`eidmongolia.mn` & `developer.sso.mn`**: National E-ID Single Sign-On (SSO) engine supporting 4 official authentication channels:
  1. 🖊️ **PKI Digital Signature (Тоон гарын үсэг)**
  2. 📱 **Mobile OTP (Нэг удаагийн код)**
  3. 🏦 **Bank SSO Credentials (Банкны суваг)**
  4. 👤 **Biometric Face Verification (Царай танилт)**
- **`dan.gerege.mn`**: Gerege Systems DAN SSO Gateway integration service.

### 4. 🤖 AI Copilot & Smart Business Intelligence Engine
Located in [`backend/internal/platform/ai`](../backend/internal/platform/ai):
- **Gemini AI Copilot (`copilot.go`)**: Context-aware natural language assistant connected directly to tenant ERP database state. Classifies intents and returns structured answers with interactive suggestion chips.
- **AI Inventory Demand Forecaster (`inventory_forecaster.go`)**: Analyzes stock levels and historical movements to automatically generate safety stock alerts and reorder quantity recommendations.

### 5. 📊 Observability & Async Background Processing
- **Prometheus Metrics (`/metrics`)**: Exposes real-time HTTP request throughput, latency histograms, and error rates using `github.com/prometheus/client_golang`.
- **OpenTelemetry Tracing**: Distributed tracing initialization for end-to-end request tracing.
- **Asynchronous Worker Queues (`platform/mailer`)**: Non-blocking worker pool with retry logic for email OTP delivery and background notifications.

---

## 🏛️ System Architecture Diagram

```
+-----------------------------------------------------------------------------------+
|                            Gerege Template ERP Platform                           |
+-----------------------------------------------------------------------------------+
                                          |
                +-------------------------+-------------------------+
                |                                                   |
      +-------------------+                               +-------------------+
      | Next.js 15 Client |                               |  Go 1.24 Backend  |
      |   (App Router)    |                               |   (Chi Router)    |
      +-------------------+                               +-------------------+
                |                                                   |
        +-------+-------+                                   +-------+-------+
        |               |                                   |               |
+---------------+ +---------------+                 +---------------+ +---------------+
| AI Copilot UI | | E-ID / DAN    |                 | Cloud-Native  | | State Exchange|
|  Drawer Panel | | SSO Modal     |                 | Resilience    | |  (xyp.gerege)|
+---------------+ +---------------+                 +---------------+ +---------------+
                                                            |
                                                    +---------------+
                                                    | Shared-Schema |
                                                    |  PostgreSQL   |
                                                    +---------------+
```

---

## 👥 Authors & Maintainers

- 🏛️ **Gerege Systems Development Team** ([@gerege-systems](https://github.com/gerege-systems))
- 🎨 **[@craftzbay](https://github.com/craftzbay)**
- 🤖 **Gemini AI**
- 🧠 **Claude AI**
