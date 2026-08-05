# Gerege Template Platform 🇲🇳

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8.svg)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-15.1-black.svg)](https://nextjs.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Gerege Template Platform** нь Odoo болон cloud-native экосистемээс санаа авсан, өндөр бүтээмжтэй, Монгол улсын цахим дэд бүтэц (ДАН, E-ID, ХУР / XYP)-тэй шууд холбогдох боломжтой **Монгол Хэлний Үндсэн Сонголттой (Mongolian Default Language)** Нээлттэй Эх бүхий **Modular Monolith ERP & Бизнес Аппликейшн Платформ** юм.

*Read this in [English](README.en.md).*

---

## 👥 Хөгжүүлэгчид ба Зохиогчид (Authors & Contributors)

Төслийг хамтран хөгжүүлэгчид:
- 🏛️ **Gerege Systems Development Team** ([@gerege-systems](https://github.com/gerege-systems))
- 🎨 **[@craftzbay](https://github.com/craftzbay)**
- 🤖 **Gemini AI**
- 🧠 **Claude AI**

---

## 🌟 Үндсэн Боломжууд ба Давуу Талууд

### 1. ⚡ Өндөр Бүтээмжтэй Модулиар Монолит Архитектур
- **Compile-Time Go Апп Модулиуд**: Модулиуд (`contacts`, `products`, `inventory`, `billing`, `documents`, `developer_portal`) нь нэг бинари програмд компиллогдон ажиллах тул сүлжээний хоцрогдолгүй (zero-latency execution).
- **Тенант Бүрийн Апп Стор Систем**: PostgreSQL дээр тенант бүрийн аппликейшний эрх, меню болон RBAC тохиргоо динамикаар удирлагдана (`app_installations`).
- **Модулийн Хамаарал Шийдвэрлэх Модель**: DAG (Directed Acyclic Graph) болон semver ашигласан хамаарал шийдвэрлэх систем.

### 2. 🛡️ Cloud-Native Resilience Engine (go-zero Сангаас Санаа Авсан)
- **Adaptive Circuit Breaker (`resilience/breaker.go`)**: Google SRE стандартын алдааны харьцааг хянах систем.
- **Adaptive Load Shedding (`resilience/loadshedder.go`)**: Сүлжээний хэт ачааллын үед `503 Service Unavailable` буцаан хамгаалах хөдөлгүүр.
- **Singleflight Coalescing (`resilience/singleflight.go`)**: Давхардсан хүсэлтүүдийг 1 удаа ажиллуулж кэш дээр ачаалал бууруулах.
- **Exponential Backoff Retry (`resilience/retry.go`)**: Сүлжээний саатлын үеийн давтан ажиллуулах функц.

### 3. 🇲🇳 Төрийн Цахим Дэд Бүтцийн Интеграци
- **Төрийн Мэдээлэл Солилцооны ХУР Систем**: Иргэний бүртгэл (`WS100101`) ба Хуулийн этгээд/ААН (`WS100201`) баталгаажуулалт.
- **Төрийн ДАН & Үндэсний E-ID Системтэй Холбогдох Интеграци ([`developer.gerege.mn`](https://developer.gerege.mn) & [`eidmongolia.mn`](https://eidmongolia.mn))**:
  1. 🖊️ **Тоон гарын үсэг (PKI Digital Signature)**
  2. 📱 **Нэг удаагийн код (Mobile OTP)**
  3. 🏦 **Банкны суваг (Bank SSO)**
  4. 👤 **Царай танилт (Biometric Face Verification)**
- **Платформын Өөрийн ORY Hydra Grade SSO Provider (`/.well-known/openid-configuration`)**: Байгууллага өөрөө гуравдагч системүүдэд OAuth2 / OpenID Connect OIDC танилт нэвтрэлт олгох бие даасан сервер.

### 4. 🤖 AI Copilot & Бизнес Аналитик
- **Gemini AI Туслах (`platform/ai/copilot.go`)**: Байгууллагын өгөгдлийн сантай холбогдсон AI Copilot туслах.
- **AI Агуулахын Захиалга Таамаглагч (`platform/ai/inventory_forecaster.go`)**: Үлдэгдэл болон захиалгын хэмжээг AI-аар таамаглах.

---

## 🚀 Туршилтын Нэвтрэх Эрх (Demo Credentials)

- **И-мэйл хаяг**: `admin@example.com`
- **Нууц үг**: `Password123!`
- **Тенант**: `Demo Corporation` (`slug: demo`)

---

## 📦 Бэлэн Бизнес Аппликейшнүүд

1. 📇 **Contacts (`io.example.contacts`)**: Харилцагчийн бүртгэл + ХУР авто-бөглөлт (`/contacts`).
2. 📦 **Products (`io.example.products`)**: Бараа бүтээгдэхүүн, үнэ ба SKU бүртгэл (`/products`).
3. 🏭 **Inventory (`io.example.inventory`)**: Агуулахын хөдөлгөөн ба аюулгүйн үлдэгдэл (`/inventory`).
4. 💳 **Public Billing & e-Barimt (`io.example.billing`)**: Нийтийн нэхэмжлэх, 10% НӨАТ ба e-Barimt татварын баримт (`/billing`).
5. 📄 **Digital Documents & E-Signatures (`io.example.documents`)**: Цахим баримт бичиг, тоон гарын үсэг ба баталгаажуулалт (`/documents`).
6. 💻 **Developer Portal & OAuth2 SSO (`io.example.developer_portal`)**: Хөгжүүлэгчийн OAuth2 Client апп бүртгэл ба тохиргоо (`/developer/apps`).

---

## 🛠️ Төслийг Ажиллуулах Заавар

### Шаардлагатай Програмууд
- Go 1.24+
- Node.js 20+
- PostgreSQL 16+ (эсвэл Docker Compose)

### 1. Docker Compose-оор ажиллуулах
```bash
docker-compose up -d
```

### 2. Гараар ажиллуулах

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
Вэб хөтөч дээрээ [http://localhost:3000](http://localhost:3000) хаягаар орон `admin@example.com` / `Password123!` эрхээр нэвтэрнэ үү.

---

## 🧪 Тест Ажиллуулах

```bash
# Backend нэгж тестүүдийг ажиллуулах
cd backend
go test -v -race ./...

# Frontend build шалгах
cd frontend
npm run build
```

---

## 📚 Баримт Бичгүүдийн Индекс

- 🏛️ [Архитектурын Дэлгэрэнгүй Заавар (Mongolian / English)](docs/ARCHITECTURE_SPECIFICATION.md)
- 📘 [Модуль Хөгжүүлэх Заавар](docs/MODULE_AUTHORING_GUIDE.md)
- 🤝 [Хамтран Ажиллах Заавар](CONTRIBUTING.md)
- 🛡️ [Аюулгүй Байдлын Бодлого](SECURITY.md)
- 📋 [Өөрчлөлтийн Түүх (Changelog)](CHANGELOG.md)
- 🇬🇧 [English Readme](README.en.md)

---

## 🙏 Ашигласан & Санаа Авсан Төслүүд

1. **[snykk/go-rest-boilerplate](https://github.com/snykk/go-rest-boilerplate)** by **[@snykk](https://github.com/snykk)** — Go REST API суурь архитектур.
2. **[Odoo](https://github.com/odoo/odoo)** — Модулиар Апп Стор болон хамаарал шийдвэрлэх модель.
3. **[go-zero (zeromicro/go-zero)](https://github.com/zeromicro/go-zero)** — Cloud-native resilience engine (Circuit Breaker, Load Shedder, Singleflight).

---

## 📄 Лиценз

Copyright (c) 2026 **Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI**.
Distributed under the **Apache 2.0 License**. See [`LICENSE`](LICENSE) for more information.
