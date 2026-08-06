/**
 * website — The public landing page.
 */
export const website = {
  "website.menu.features": { mn: "Боломжууд", en: "Features" },
  "website.menu.architecture": { mn: "Архитектур", en: "Architecture" },
  "website.menu.modules": { mn: "Модулиуд", en: "Modules" },
  "website.menu.sso": { mn: "OIDC SSO & ДАН", en: "OIDC SSO & DAN" },

  "website.action.sign_in": { mn: "Платформ руу нэвтрэх", en: "Sign in to the platform" },
  "website.action.enter": { mn: "Нэвтрэх →", en: "Sign in →" },

  "website.view.badge": { mn: "AI Native & Үндэсний цахим танилтад бэлэн", en: "AI native & national digital identity ready" },
  "website.view.hero_lead": { mn: "Монгол Улсын цахим дэд бүтэцтэй нягт холбогдох", en: "Wired into Mongolia's national digital infrastructure" },
  "website.view.hero_highlight": { mn: "Modular Monolith ERP платформ", en: "Modular Monolith ERP platform" },
  "website.view.hero_body": { mn: "Odoo болон cloud-native экосистемээс санаа авсан, Go 1.25, Next.js 15, ДАН / E-ID, ХУР төрийн мэдээлэл солилцоо болон OAuth2 / OIDC SSO Provider агуулсан нээлттэй эх бүхий бизнес платформ.", en: "An open-source business platform inspired by Odoo and the cloud-native ecosystem: Go 1.25, Next.js 15, DAN / E-ID, the XYP state data exchange and a built-in OAuth2 / OIDC SSO provider." },
  "website.view.features_title": { mn: "Платформын үндсэн давуу талууд", en: "What the platform gives you" },
  "website.view.features_subtitle": { mn: "Байгууллагын өдөр тутмын үйл ажиллагаа, аюулгүй байдал, өндөр бүтээмжийг нэг дороос хангах цогц систем.", en: "Day-to-day operations, security and performance handled by one coherent system." },
  "website.view.modules_title": { mn: "Бэлэн бизнес аппликейшнүүд", en: "Business applications" },
  "website.view.modules_subtitle": { mn: "Апп Дэлгүүрээс тенант бүрээр идэвхжүүлэн ашиглах боломжтой Go бизнес модулиуд.", en: "Go modules you enable per tenant from the app store." },

  "website.stat.modules": { mn: "Бэлэн бизнес модуль", en: "Business modules shipped" },
  "website.stat.lint": { mn: "Lint & vet анхааруулга", en: "Lint & vet warnings" },
  "website.stat.vulns": { mn: "Мэдэгдэж буй эмзэг байдал", en: "Known vulnerabilities" },
  "website.stat.tests": { mn: "Race detector-тэй тест", en: "Tests under the race detector" },

  "website.feature.monolith_title": { mn: "Modular Monolith Engine", en: "Modular monolith engine" },
  "website.feature.monolith_body": { mn: "Go хэл дээр компиллогдох Модулиар Монолит архитектур. Сүлжээний хоцрогдолгүй (zero-latency execution), тенант бүрийн Апп Дэлгүүрийн тохиргоо ба DAG хамаарал шийдвэрлэгч.", en: "A modular monolith compiled in Go: zero-latency in-process execution, per-tenant app store configuration and a DAG dependency resolver." },
  "website.feature.resilience_title": { mn: "Cloud-Native Resilience Engine", en: "Cloud-native resilience engine" },
  "website.feature.resilience_body": { mn: "go-zero сангаас санаа авсан Adaptive Circuit Breaker, Load Shedder, Singleflight coalescing ба Exponential Backoff Retry системүүд.", en: "Adaptive circuit breaker, load shedder, singleflight coalescing and exponential backoff retry, inspired by go-zero." },
  "website.feature.identity_title": { mn: "E-ID & ДАН SSO Танилт Нэвтрэлт", en: "E-ID & DAN SSO sign-in" },
  "website.feature.identity_body": { mn: "Төрийн ДАН ба E-ID системтэй холбогдон Тоон гарын үсэг, Mobile OTP, Банкны SSO болон Царай танилтаар баталгаажуулах интеграци.", en: "Sign-in through the national DAN and E-ID systems: PKI digital signature, mobile OTP, bank SSO and biometric face verification." },
  "website.feature.provider_title": { mn: "ORY Hydra SSO Identity Provider", en: "OAuth2 / OIDC identity provider" },
  "website.feature.provider_body": { mn: "Өөрийн бие даасан OAuth2 ба OpenID Connect (OIDC) SSO Provider engine (`/.well-known/openid-configuration`) ба Developer Portal.", en: "A self-contained OAuth2 and OpenID Connect provider (`/.well-known/openid-configuration`) with a developer portal." },
  "website.feature.ai_title": { mn: "Gemini AI Copilot & Forecaster", en: "AI copilot & forecaster" },
  "website.feature.ai_body": { mn: "Байгууллагын өгөгдлийн сантай холбогдсон Gemini AI туслах болон агуулахын аюулгүйн үлдэгдэл, захиалга таамаглах систем.", en: "An AI assistant wired to live tenant data, plus safety-stock and reorder-point forecasting for the warehouse." },
  "website.feature.xyp_title": { mn: "ХУР Мэдээлэл Солилцооны Систем", en: "XYP state data exchange" },
  "website.feature.xyp_body": { mn: "Төрийн ХУР системээр иргэний бүртгэл (`WS100101`) ба Хуулийн этгээд/ААН (`WS100201`) автоматаар баталгаажуулан бөглөх модуль.", en: "Citizen registration (`WS100101`) and legal entity data (`WS100201`) verified and auto-filled straight from the state exchange." },

  "website.module.contacts_title": { mn: "Contacts Module", en: "Contacts module" },
  "website.module.contacts_body": { mn: "Харилцагч, бэлтгэн нийлүүлэгчдийн бүртгэл + ХУР төрийн системээс авто-бөглөлт.", en: "Customer and vendor directory with auto-fill from the XYP state systems." },
  "website.module.inventory_title": { mn: "Products & Inventory", en: "Products & inventory" },
  "website.module.inventory_body": { mn: "Барааны бүртгэл, SKU, агуулахын хөдөлгөөний лог болон AI үлдэгдлийн таамаглал.", en: "Product records, SKUs, warehouse movement log and AI stock forecasting." },
  "website.module.billing_title": { mn: "Public Billing & e-Barimt", en: "Public billing & e-Barimt" },
  "website.module.billing_body": { mn: "Нийтийн нэхэмжлэх, 10% НӨАТ ба e-Barimt татварын баримт хэвлэх модуль.", en: "Public invoicing, 10% VAT and printable e-Barimt tax receipts." },
  "website.module.documents_title": { mn: "Digital Documents & E-Sign", en: "Digital documents & e-sign" },
  "website.module.documents_body": { mn: "Цахим баримт бичиг, батлах workflow болон E-ID/ДАН тоон гарын үсэг.", en: "Document routing, digital signatures and approval workflows." },
  "website.module.developer_title": { mn: "Developer Portal & OAuth2", en: "Developer portal & OAuth2" },
  "website.module.developer_body": { mn: "Гуравдагч системд зориулсан OAuth2 Client App бүртгэл, Secret ба Redirect URI тохиргоо.", en: "OAuth2 client app registration, secrets and redirect URI configuration for third parties." },
  "website.module.integrations_title": { mn: "Integrations & Webhooks", en: "Integrations & webhooks" },
  "website.module.integrations_body": { mn: "HMAC-SHA256 гарын үсэгтэй асинхрон Webhook ба гадаад систем холбох Connector Manager.", en: "HMAC-SHA256 signed asynchronous webhooks and a connector manager for external systems." },

  "website.message.demo_account": { mn: "Туршилтын нэвтрэх эрх:", en: "Demo account:" },
  "website.message.stats_note": { mn: "Эдгээр үзүүлэлтийг push бүр дээр CI (golangci-lint · go vet · go test -race · govulncheck · gosec) шалгана.", en: "Every figure is enforced on each push by CI (golangci-lint · go vet · go test -race · govulncheck · gosec)." },
  "website.message.license": { mn: "— Apache 2.0 лицензээр тараагдана", en: "— Distributed under the Apache 2.0 License" },
} as const;
