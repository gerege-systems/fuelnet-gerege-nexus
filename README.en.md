# Gerege Template Platform 🇬🇧

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8.svg)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-15.1-black.svg)](https://nextjs.org)

An open-source, production-oriented **Modular Monolith ERP & Business Application Platform** inspired by Odoo's app ecosystem and cloud-native resilience patterns. Built for enterprise scalability, resilience, and national digital identity/data exchange integration using **Go 1.24**, **Chi Router**, **PostgreSQL (pgx/v5)**, **Goose Migrations**, and **Next.js (App Router)**.

*Үндсэн баримт бичгийг [Монгол](README.md) хэлээр унших.*

---

## 👥 Authors & Contributors

This open-source project is created and maintained by:
- 🏛️ **Gerege Systems Development Team** ([@gerege-systems](https://github.com/gerege-systems))
- 🎨 **[@craftzbay](https://github.com/craftzbay)**
- 🤖 **Gemini AI**
- 🧠 **Claude AI**

---

## 🌟 Key Advantages & Core Features

### 1. ⚡ High-Performance Modular Monolith Architecture
- **Compile-Time Go App Modules**: Applications implement a unified `Module` Go contract and compile into a single binary for zero-latency in-process execution.
- **Tenant-Level App Store Engine**: Per-tenant application gating, dynamic menu composition, and RBAC permissions managed dynamically via PostgreSQL (`app_installations`).
- **Topological Dependency Resolution Engine**: Pure Go recursive dependency resolver using Directed Acyclic Graphs (DAG) and semver constraints.

### 2. 🛡️ Cloud-Native Resilience Engine (go-zero Inspired)
- **Adaptive Circuit Breaker (`resilience/breaker.go`)**: Google SRE sliding window error-rate monitoring and rejection handling.
- **Adaptive Load Shedding (`resilience/loadshedder.go`)**: In-flight HTTP request concurrency control returning `503 Service Unavailable` with `Retry-After` headers under high load.
- **Singleflight Coalescing (`resilience/singleflight.go`)**: Thundering herd & cache stampede query suppression.
- **Exponential Backoff Retry (`resilience/retry.go`)**: `DoWithRetry` execution helper for transient network/DB failures.

### 3. 🇲🇳 National Digital Infrastructure Integration
- **State Data Exchange (ХУР)**: Official Mongolian State Data Exchange for Citizen Civil Registration (`WS100101`) and Company Legal Entity verification (`WS100201`).
- **Mongolian Government DAN SSO & E-ID Integration ([`developer.gerege.mn`](https://developer.gerege.mn) & [`eidmongolia.mn`](https://eidmongolia.mn))**: Integrates with official State DAN SSO identity verification supporting PKI Digital Signatures, Mobile OTP, Bank SSO, and Biometric Face Verification.
- **Built-in ORY Hydra Grade SSO Provider Engine (`/.well-known/openid-configuration`)**: Independent OAuth2 & OpenID Connect identity provider for issuing OAuth2 tokens to third-party applications.

### 4. 🤖 AI Copilot & Smart Business Intelligence
- **Gemini AI Assistant (`platform/ai/copilot.go`)**: Natural language ERP assistant connected to live tenant database state with intent classification and actionable UI suggestion chips.
- **AI Inventory Demand Forecaster (`platform/ai/inventory_forecaster.go`)**: Historical stock movement analysis and safety stock reorder point recommendations.

---

## 🚀 Demo Credentials

- **Admin Email**: `admin@example.com`
- **Password**: `Password123!`
- **Demo Tenant**: `Demo Corporation` (`slug: demo`)

---

## 📦 Production Business Application Suite

1. 📇 **Contacts (`io.example.contacts`)**: Customer/vendor management with instant XYP (`xyp.gerege.mn`) civil registration auto-fill (`/contacts`).
2. 📦 **Products (`io.example.products`)**: Product catalog, pricing, and tenant-scoped SKUs (`/products`).
3. 🏭 **Inventory (`io.example.inventory`)**: Warehouse management, live stock levels, append-only stock movement audit log, and transactional adjustments (`/inventory`).
4. 💳 **Public Billing & e-Barimt (`io.example.billing`)**: Public service fee invoicing, 10% VAT calculation, and e-Barimt tax receipt generation (`/billing`).
5. 📄 **Digital Documents & E-Signatures (`io.example.documents`)**: Enterprise document routing, approval workflows, and E-ID / DAN digital signature verification (`/documents`).
6. 💻 **Developer Portal & OAuth2 SSO (`io.example.developer_portal`)**: Register third-party OAuth2 client applications, issue Client IDs and Secrets (`/developer/apps`).

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
DATABASE_URL="postgres://postgres:postgrespassword@localhost:5432/platform_db?sslmode=disable" go run ./cmd/migrate up
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

## 📄 License

Copyright (c) 2026 **Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI**.
Distributed under the **Apache 2.0 License**. See [`LICENSE`](LICENSE) for more information.
