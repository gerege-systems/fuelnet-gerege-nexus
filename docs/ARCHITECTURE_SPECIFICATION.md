# 🏛️ Gerege Template Platform — Архитектур ба Системийн Боломжийн Дэлгэрэнгүй Заавар (Technical Architecture Specification)

🇲🇳 **Үндэсний Хэл**: Монгол (Mongolian Default Language) | 🇬🇧 **Secondary Language**: English

---

## 🇲🇳 1. Системийн Ерөнхий Архитектур ба Давуу Талууд (Mongolian)

**Gerege Template Platform** нь өндөр бүтээмжтэй, Монгол улсын цахим дэд бүтэцтэй нягт холбогдох боломжтой **Modular Monolith ERP & Бизнес Аппликейшн Платформ** юм.

### ⚡ Өндөр Бүтээмжтэй Модулиар Монолит Архитектур
- **Zero-Latency Execution**: Бизнес аппликейшн модулиуд (`contacts`, `products`, `inventory`, `billing`, `documents`, `developer_portal`) нь Go хэлний `Module` контрактыг хэрэгжүүлэн нэг бинари програмд компиллогдон ажиллана.
- **Тенант Бүрийн Апп Стор Систем**: Баазад байгаа модулиуд нь тенант бүрийн хувьд идэвхжүүлсэн эсэх нь PostgreSQL сангаас динамикаар удирлагдана (`app_installations`).
- **DAG Хамаарал Шийдвэрлэх Модель**: Directed Acyclic Graph болон semver ашиглан модулиудын хамаарлыг алдаагүй тооцоолно.

### 🛡️ Cloud-Native Resilience Engine (go-zero Inspired)
- **Adaptive Circuit Breaker (`resilience/breaker.go`)**: Сүлжээ болон системд алдаа гарахад ачааллыг автоматаар тусгаарлах Google SRE стандарт.
- **Adaptive Load Shedding (`resilience/loadshedder.go`)**: Сүлжээний хэт ачааллын үед `503 Service Unavailable` буцаан системийг хамгаалах.
- **Singleflight Coalescing (`resilience/singleflight.go`)**: Давхардсан асуулгуудыг 1 удаа ажиллуулан DB ачааллыг бууруулах.

### 🇲🇳 Төрийн Мэдээлэл Солилцоо ба Танилт Нэвтрэлт
- **ХУР Төрийн Мэдээлэл Солилцооны Систем**: Иргэний бүртгэл (`WS100101`) ба Хуулийн этгээдийн мэдээлэл (`WS100201`).
- **Төрийн ДАН & E-ID Танилт Нэвтрэлт ([`eidmongolia.mn`](https://eidmongolia.mn) & [`developer.gerege.mn`](https://developer.gerege.mn))**: Төрийн нэгдсэн ДАН танилт нэвтрэлтийн системтэй холбогдон Тоон гарын үсэг, Mobile OTP, Банкны SSO, Царай танилтаар баталгаажуулах.
- **ORY Hydra SSO Provider (`/.well-known/openid-configuration`)**: Өөрийн бие даасан OAuth2 & OpenID Connect provider.

---

## 🇬🇧 2. System Architecture & Technical Specifications (English)

### ⚡ High-Performance Modular Monolith Architecture
- **In-Process Invocations**: Business modules implement a strict Go `Module` interface compiled directly into the binary for zero-latency execution.
- **Tenant App Store Engine**: Per-tenant module enablement, RBAC permissions, and dynamic menus managed via PostgreSQL (`app_installations`).

### 🛡️ Cloud-Native Resilience Engine
- **Adaptive Circuit Breaker**: Google SRE sliding-window error monitoring.
- **Adaptive Load Shedding**: Graceful load shedding under high CPU/concurrency limits.
- **Singleflight Coalescing**: Thundering herd and cache stampede query suppression.

---

## 🏛️ System Architecture Diagram

```
+-----------------------------------------------------------------------------------+
|                              Gerege Template Platform                             |
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
|  Drawer Panel | | SSO Provider  |                 | Resilience    | |  (xyp.gerege)|
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
