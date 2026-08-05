"use client";

import React, { createContext, useContext, useEffect, useMemo, useState } from "react";

export type Locale = "mn" | "en";

export const LOCALES: { code: Locale; label: string; flag: string }[] = [
  { code: "mn", label: "Монгол", flag: "/icons/flag-mn.png" },
  { code: "en", label: "English", flag: "/icons/flag-en.png" },
];

const STORAGE_KEY = "locale";
export const DEFAULT_LOCALE: Locale = "mn";

/**
 * Every user-facing string in the app. Mongolian is the source language;
 * `en` is the translation. Keys are namespaced by screen.
 */
const dictionary = {
  // ─── Shared ────────────────────────────────────────────────────────────────
  "common.loading": { mn: "Ачаалж байна...", en: "Loading..." },
  "common.save": { mn: "Хадгалах", en: "Save" },
  "common.cancel": { mn: "Болих", en: "Cancel" },
  "common.create": { mn: "Үүсгэх", en: "Create" },
  "common.edit": { mn: "Засах", en: "Edit" },
  "common.delete": { mn: "Устгах", en: "Delete" },
  "common.search": { mn: "Хайх", en: "Search" },
  "common.actions": { mn: "Үйлдэл", en: "Actions" },
  "common.status": { mn: "Төлөв", en: "Status" },
  "common.name": { mn: "Нэр", en: "Name" },
  "common.email": { mn: "И-мэйл", en: "Email" },
  "common.phone": { mn: "Утас", en: "Phone" },
  "common.company": { mn: "Байгууллага", en: "Company" },
  "common.active": { mn: "Идэвхтэй", en: "Active" },
  "common.inactive": { mn: "Идэвхгүй", en: "Inactive" },
  "common.amount": { mn: "Дүн", en: "Amount" },
  "common.date": { mn: "Огноо", en: "Date" },
  "common.close": { mn: "Хаах", en: "Close" },
  "common.empty": { mn: "Бичлэг олдсонгүй", en: "No records found" },
  "common.error": { mn: "Алдаа гарлаа", en: "Something went wrong" },
  "common.language": { mn: "Хэл", en: "Language" },

  // ─── App shell ─────────────────────────────────────────────────────────────
  "shell.loadingPlatform": { mn: "Платформыг ачаалж байна...", en: "Loading ERP Platform..." },
  "shell.appStore": { mn: "Апп Стор", en: "App Store" },
  "shell.installedApps": { mn: "Суулгасан аппууд", en: "Installed Apps" },
  "shell.integrations": { mn: "Интеграцууд", en: "Integrations" },
  "shell.developerApps": { mn: "Хөгжүүлэгчийн аппууд", en: "Developer Apps" },
  "shell.settings": { mn: "Тохиргоо", en: "Settings" },
  "shell.logout": { mn: "Гарах", en: "Sign out" },
  "shell.modules": { mn: "Модулиуд", en: "Modules" },
  "shell.platform": { mn: "Платформ", en: "Platform" },

  // ─── Login ─────────────────────────────────────────────────────────────────
  "login.subtitle": { mn: "Модуль бүтэцтэй байгууллагын платформ", en: "Modular Enterprise Application Platform" },
  "login.email": { mn: "И-мэйл хаяг", en: "Email Address" },
  "login.password": { mn: "Нууц үг", en: "Password" },
  "login.submit": { mn: "Платформ руу нэвтрэх", en: "Sign In to Platform" },
  "login.submitting": { mn: "Нэвтэрч байна...", en: "Signing in..." },
  "login.or": { mn: "ЭСВЭЛ", en: "OR" },
  "login.eid": { mn: "E-ID Mongolia (Танилт Нэвтрэлт)", en: "E-ID Mongolia (National Identity)" },
  "login.eidTitle": { mn: "E-ID Mongolia танилт (eidmongolia.mn)", en: "E-ID Mongolia Identity (eidmongolia.mn)" },
  "login.regNumber": { mn: "Регистрийн дугаар", en: "Registration number" },
  "login.otp": { mn: "Нэг удаагийн код", en: "One-time code" },
  "login.demoCredentials": { mn: "Туршилтын эрх:", en: "Demo credentials:" },
  "login.backToHome": { mn: "Нүүр хуудас руу буцах", en: "Back to home" },

  // ─── Landing ───────────────────────────────────────────────────────────────
  "landing.nav.features": { mn: "Боломжууд", en: "Features" },
  "landing.nav.architecture": { mn: "Архитектур", en: "Architecture" },
  "landing.nav.modules": { mn: "Модулиуд", en: "Modules" },
  "landing.nav.sso": { mn: "OIDC SSO & ДАН", en: "OIDC SSO & DAN" },
  "landing.cta.signIn": { mn: "Платформ руу нэвтрэх", en: "Sign in to the platform" },
  "landing.cta.enter": { mn: "Нэвтрэх →", en: "Sign in →" },
  "landing.badge": { mn: "AI Native & Үндэсний цахим танилтад бэлэн", en: "AI native & national digital identity ready" },
  "landing.hero.lead": { mn: "Монгол Улсын цахим дэд бүтэцтэй нягт холбогдох", en: "Wired into Mongolia's national digital infrastructure" },
  "landing.hero.highlight": { mn: "Modular Monolith ERP платформ", en: "Modular Monolith ERP platform" },
  "landing.hero.body": {
    mn: "Odoo болон cloud-native экосистемээс санаа авсан, Go 1.25, Next.js 15, ДАН / E-ID, ХУР төрийн мэдээлэл солилцоо болон OAuth2 / OIDC SSO Provider агуулсан нээлттэй эх бүхий бизнес платформ.",
    en: "An open-source business platform inspired by Odoo and the cloud-native ecosystem: Go 1.25, Next.js 15, DAN / E-ID, the XYP state data exchange and a built-in OAuth2 / OIDC SSO provider.",
  },
  "landing.demoBanner": { mn: "Туршилтын нэвтрэх эрх:", en: "Demo account:" },
  "landing.stats.modules": { mn: "Бэлэн бизнес модуль", en: "Business modules shipped" },
  "landing.stats.lint": { mn: "Lint & vet анхааруулга", en: "Lint & vet warnings" },
  "landing.stats.vulns": { mn: "Мэдэгдэж буй эмзэг байдал", en: "Known vulnerabilities" },
  "landing.stats.tests": { mn: "Race detector-тэй тест", en: "Tests under the race detector" },
  "landing.stats.note": {
    mn: "Эдгээр үзүүлэлтийг push бүр дээр CI (golangci-lint · go vet · go test -race · govulncheck · gosec) шалгана.",
    en: "Every figure is enforced on each push by CI (golangci-lint · go vet · go test -race · govulncheck · gosec).",
  },
  "landing.features.title": { mn: "Платформын үндсэн давуу талууд", en: "What the platform gives you" },
  "landing.features.subtitle": {
    mn: "Байгууллагын өдөр тутмын үйл ажиллагаа, аюулгүй байдал, өндөр бүтээмжийг нэг дороос хангах цогц систем.",
    en: "Day-to-day operations, security and performance handled by one coherent system.",
  },
  "landing.modules.title": { mn: "Бэлэн бизнес аппликейшнүүд", en: "Business applications" },
  "landing.modules.subtitle": {
    mn: "Апп Стороос тенант бүрээр идэвхжүүлэн ашиглах боломжтой Go бизнес модулиуд.",
    en: "Go modules you enable per tenant from the app store.",
  },
  "landing.footer.license": { mn: "— Apache 2.0 лицензээр тараагдана", en: "— Distributed under the Apache 2.0 License" },

  // ─── Landing cards ─────────────────────────────────────────────────────────
  "landing.feature1.title": {
    mn: "Modular Monolith Engine",
    en: "Modular monolith engine",
  },
  "landing.feature2.title": {
    mn: "Cloud-Native Resilience Engine",
    en: "Cloud-native resilience engine",
  },
  "landing.feature3.title": {
    mn: "E-ID & ДАН SSO Танилт Нэвтрэлт",
    en: "E-ID & DAN SSO sign-in",
  },
  "landing.feature4.title": {
    mn: "ORY Hydra SSO Identity Provider",
    en: "OAuth2 / OIDC identity provider",
  },
  "landing.feature5.title": {
    mn: "Gemini AI Copilot & Forecaster",
    en: "AI copilot & forecaster",
  },
  "landing.feature6.title": {
    mn: "ХУР Мэдээлэл Солилцооны Систем",
    en: "XYP state data exchange",
  },
  "landing.feature1.body": {
    mn: "Go хэл дээр компиллогдох Модулиар Монолит архитектур. Сүлжээний хоцрогдолгүй (zero-latency execution), тенант бүрийн Апп Стор тохиргоо ба DAG хамаарал шийдвэрлэгч.",
    en: "A modular monolith compiled in Go: zero-latency in-process execution, per-tenant app store configuration and a DAG dependency resolver.",
  },
  "landing.feature2.body": {
    mn: "go-zero сангаас санаа авсан Adaptive Circuit Breaker, Load Shedder, Singleflight coalescing ба Exponential Backoff Retry системүүд.",
    en: "Adaptive circuit breaker, load shedder, singleflight coalescing and exponential backoff retry, inspired by go-zero.",
  },
  "landing.feature3.body": {
    mn: "Төрийн ДАН ба E-ID системтэй холбогдон Тоон гарын үсэг, Mobile OTP, Банкны SSO болон Царай танилтаар баталгаажуулах интеграци.",
    en: "Sign-in through the national DAN and E-ID systems: PKI digital signature, mobile OTP, bank SSO and biometric face verification.",
  },
  "landing.feature4.body": {
    mn: "Өөрийн бие даасан OAuth2 ба OpenID Connect (OIDC) SSO Provider engine (`/.well-known/openid-configuration`) ба Developer Portal.",
    en: "A self-contained OAuth2 and OpenID Connect provider (`/.well-known/openid-configuration`) with a developer portal.",
  },
  "landing.feature5.body": {
    mn: "Байгууллагын өгөгдлийн сантай холбогдсон Gemini AI туслах болон агуулахын аюулгүйн үлдэгдэл, захиалга таамаглах систем.",
    en: "An AI assistant wired to live tenant data, plus safety-stock and reorder-point forecasting for the warehouse.",
  },
  "landing.feature6.body": {
    mn: "Төрийн ХУР системээр иргэний бүртгэл (`WS100101`) ба Хуулийн этгээд/ААН (`WS100201`) автоматаар баталгаажуулан бөглөх модуль.",
    en: "Citizen registration (`WS100101`) and legal entity data (`WS100201`) verified and auto-filled straight from the state exchange.",
  },
  "landing.module1.title": {
    mn: "Contacts Module",
    en: "Contacts module",
  },
  "landing.module2.title": {
    mn: "Products & Inventory",
    en: "Products & inventory",
  },
  "landing.module3.title": {
    mn: "Public Billing & e-Barimt",
    en: "Public billing & e-Barimt",
  },
  "landing.module4.title": {
    mn: "Digital Documents & E-Sign",
    en: "Digital documents & e-sign",
  },
  "landing.module5.title": {
    mn: "Developer Portal & OAuth2",
    en: "Developer portal & OAuth2",
  },
  "landing.module6.title": {
    mn: "Integrations & Webhooks",
    en: "Integrations & webhooks",
  },
  "landing.module1.body": {
    mn: "Харилцагч, бэлтгэн нийлүүлэгчдийн бүртгэл + ХУР төрийн системээс авто-бөглөлт.",
    en: "Customer and vendor directory with auto-fill from the XYP state systems.",
  },
  "landing.module2.body": {
    mn: "Барааны бүртгэл, SKU, агуулахын хөдөлгөөний лог болон AI үлдэгдлийн таамаглал.",
    en: "Product records, SKUs, warehouse movement log and AI stock forecasting.",
  },
  "landing.module3.body": {
    mn: "Нийтийн нэхэмжлэх, 10% НӨАТ ба e-Barimt татварын баримт хэвлэх модуль.",
    en: "Public invoicing, 10% VAT and printable e-Barimt tax receipts.",
  },
  "landing.module4.body": {
    mn: "Цахим баримт бичиг, батлах workflow болон E-ID/ДАН тоон гарын үсэг.",
    en: "Document routing, digital signatures and approval workflows.",
  },
  "landing.module5.body": {
    mn: "Гуравдагч системд зориулсан OAuth2 Client App бүртгэл, Secret ба Redirect URI тохиргоо.",
    en: "OAuth2 client app registration, secrets and redirect URI configuration for third parties.",
  },
  "landing.module6.body": {
    mn: "HMAC-SHA256 гарын үсэгтэй асинхрон Webhook ба гадаад систем холбох Connector Manager.",
    en: "HMAC-SHA256 signed asynchronous webhooks and a connector manager for external systems.",
  },

  // ─── App store ─────────────────────────────────────────────────────────────
  "store.title": { mn: "Апп Стор", en: "App Store" },
  "store.subtitle": {
    mn: "Тенантдаа хэрэгтэй бизнес модулиудыг суулгаж, идэвхжүүлнэ.",
    en: "Install and enable the business modules your tenant needs.",
  },
  "store.install": { mn: "Суулгах", en: "Install" },
  "store.installing": { mn: "Суулгаж байна...", en: "Installing..." },
  "store.enable": { mn: "Идэвхжүүлэх", en: "Enable" },
  "store.disable": { mn: "Идэвхгүй болгох", en: "Disable" },
  "store.installed": { mn: "Суулгасан", en: "Installed" },
  "store.version": { mn: "Хувилбар", en: "Version" },

  // ─── Contacts ──────────────────────────────────────────────────────────────
  "contacts.title": { mn: "Харилцагчид", en: "Contacts" },
  "contacts.subtitle": { mn: "Үйлчлүүлэгч, нийлүүлэгчдийн бүртгэл", en: "Customer and vendor directory" },
  "contacts.new": { mn: "Шинэ харилцагч", en: "New contact" },
  "contacts.xypLookup": { mn: "ХУР-аас татах", en: "Look up in XYP" },

  // ─── Products ──────────────────────────────────────────────────────────────
  "products.title": { mn: "Бараа бүтээгдэхүүн", en: "Products" },
  "products.subtitle": { mn: "Барааны каталог, үнэ ба SKU", en: "Catalog, pricing and SKUs" },
  "products.new": { mn: "Шинэ бараа", en: "New product" },
  "products.sku": { mn: "SKU код", en: "SKU" },
  "products.price": { mn: "Үнэ", en: "Price" },

  // ─── Inventory ─────────────────────────────────────────────────────────────
  "inventory.title": { mn: "Агуулах", en: "Inventory" },
  "inventory.subtitle": { mn: "Агуулах, үлдэгдэл ба хөдөлгөөн", en: "Warehouses, stock levels and movements" },
  "inventory.warehouses": { mn: "Агуулахууд", en: "Warehouses" },
  "inventory.stockLevels": { mn: "Үлдэгдэл", en: "Stock levels" },
  "inventory.movements": { mn: "Хөдөлгөөн", en: "Movements" },
  "inventory.adjust": { mn: "Тохируулга хийх", en: "Adjust stock" },
  "inventory.quantity": { mn: "Тоо хэмжээ", en: "Quantity" },

  // ─── Billing ───────────────────────────────────────────────────────────────
  "billing.title": { mn: "Нэхэмжлэх ба e-Barimt", en: "Billing & e-Barimt" },
  "billing.subtitle": { mn: "Нэхэмжлэх үүсгэх, НӨАТ ба татварын баримт", en: "Invoicing, VAT and tax receipts" },
  "billing.new": { mn: "Шинэ нэхэмжлэх", en: "New invoice" },
  "billing.invoiceNumber": { mn: "Нэхэмжлэхийн дугаар", en: "Invoice number" },
  "billing.vat": { mn: "НӨАТ", en: "VAT" },
  "billing.contact": { mn: "Харилцагч", en: "Contact" },

  // ─── Documents ─────────────────────────────────────────────────────────────
  "documents.title": { mn: "Баримт бичиг ба цахим гарын үсэг", en: "Documents & e-signatures" },
  "documents.subtitle": { mn: "Баримт бичгийн урсгал ба баталгаажуулалт", en: "Document routing and approvals" },
  "documents.new": { mn: "Шинэ баримт", en: "New document" },
  "documents.type": { mn: "Төрөл", en: "Type" },
  "documents.signedBy": { mn: "Гарын үсэг зурсан", en: "Signed by" },

  // ─── Developer portal ──────────────────────────────────────────────────────
  "developer.title": { mn: "Хөгжүүлэгчийн аппууд", en: "Developer apps" },
  "developer.subtitle": {
    mn: "Гуравдагч системд зориулсан OAuth2 / OIDC client бүртгэл",
    en: "OAuth2 / OIDC client registration for third-party systems",
  },
  "developer.new": { mn: "Шинэ апп бүртгэх", en: "Register app" },
  "developer.clientId": { mn: "Client ID", en: "Client ID" },
  "developer.redirectUris": { mn: "Redirect URI-ууд", en: "Redirect URIs" },
  "developer.scopes": { mn: "Scope-ууд", en: "Scopes" },
  "developer.secretHidden": {
    mn: "үүсгэх үед нэг удаа харагдана",
    en: "shown once, at creation",
  },

  // ─── Integrations ──────────────────────────────────────────────────────────
  "integrations.title": { mn: "Интеграц ба Webhook", en: "Integrations & webhooks" },
  "integrations.subtitle": { mn: "Гадаад системтэй холбогдох тохиргоо", en: "Connections to external systems" },
  "integrations.new": { mn: "Интеграц нэмэх", en: "Add integration" },
  "integrations.targetUrl": { mn: "Хүлээн авах хаяг", en: "Target URL" },
  "integrations.type": { mn: "Төрөл", en: "Type" },

  // ─── Screens (generated) ───────────────────────────────────────────────────
  "apps.platformAppStore": { mn: "Платформын Апп Стор", en: "Platform App Store" },
  "apps.loadingAppsCatalog": { mn: "Апп каталог ачаалж байна...", en: "Loading apps catalog..." },
  "apps.noAppsFoundMatchingYourQuery": { mn: "Хайлтад тохирох апп олдсонгүй.", en: "No apps found matching your query." },
  "apps.searchApps": { mn: "Апп хайх...", en: "Search apps..." },
  "apps.enableApp": { mn: "Идэвхжүүлэх", en: "Enable App" },
  "apps.disableApp": { mn: "Идэвхгүй болгох", en: "Disable App" },
  "contacts.contactsDirectory": { mn: "Харилцагчийн бүртгэл", en: "Contacts Directory" },
  "contacts.loadingContacts": { mn: "Харилцагчдыг ачаалж байна...", en: "Loading contacts..." },
  "contacts.newContact": { mn: "Шинэ харилцагч", en: "New Contact" },
  "contacts.createNewContact": { mn: "Шинэ харилцагч үүсгэх", en: "Create New Contact" },
  "contacts.name": { mn: "Нэр", en: "Name" },
  "contacts.email": { mn: "И-мэйл", en: "Email" },
  "contacts.phone": { mn: "Утас", en: "Phone" },
  "contacts.company": { mn: "Байгууллага", en: "Company" },
  "contacts.status": { mn: "Төлөв", en: "Status" },
  "products.productCatalog": { mn: "Барааны каталог", en: "Product Catalog" },
  "products.manageSkusProductNamesAndPricing": { mn: "SKU, барааны нэр, үнийн удирдлага", en: "Manage SKUs, product names, and pricing" },
  "products.loadingProducts": { mn: "Бараануудыг ачаалж байна...", en: "Loading products..." },
  "products.newProduct": { mn: "Шинэ бараа", en: "New Product" },
  "products.createNewProduct": { mn: "Шинэ бараа үүсгэх", en: "Create New Product" },
  "products.productName": { mn: "Барааны нэр", en: "Product Name" },
  "products.unitPrice": { mn: "Нэгж үнэ", en: "Unit Price" },
  "products.status": { mn: "Төлөв", en: "Status" },
  "products.egProd001": { mn: "жишээ: PROD-001", en: "e.g. PROD-001" },
  "inventory.inventoryWarehouseOperations": { mn: "Агуулах ба нөөцийн үйл ажиллагаа", en: "Inventory & Warehouse Operations" },
  "inventory.loadingInventoryData": { mn: "Агуулахын мэдээлэл ачаалж байна...", en: "Loading inventory data..." },
  "inventory.newWarehouse": { mn: "Шинэ агуулах", en: "New Warehouse" },
  "inventory.createWarehouse": { mn: "Агуулах үүсгэх", en: "Create Warehouse" },
  "inventory.liveStockLevels": { mn: "Одоогийн үлдэгдэл", en: "Live Stock Levels" },
  "inventory.stockMovementsHistory": { mn: "Хөдөлгөөний түүх", en: "Stock Movements History" },
  "inventory.stockAdjustment": { mn: "Үлдэгдлийн тохируулга", en: "Stock Adjustment" },
  "inventory.adjustStock": { mn: "Тохируулга хийх", en: "Adjust Stock" },
  "inventory.availableQuantity": { mn: "Боломжит тоо хэмжээ", en: "Available Quantity" },
  "inventory.referenceReason": { mn: "Баримт / Шалтгаан", en: "Reference / Reason" },
  "inventory.referenceNote": { mn: "Тайлбар", en: "Reference Note" },
  "inventory.dateTime": { mn: "Огноо, цаг", en: "Date & Time" },
  "inventory.warehouse": { mn: "Агуулах", en: "Warehouse" },
  "inventory.product": { mn: "Бараа", en: "Product" },
  "inventory.change": { mn: "Өөрчлөлт", en: "Change" },
  "inventory.address": { mn: "Хаяг", en: "Address" },
  "inventory.egPo98421OrPhysicalCountAdjustment": { mn: "жишээ: PO-98421 эсвэл тооллогын зөрүү", en: "e.g. PO-98421 or Physical count adjustment" },
  "billing.publicBillingEbarimtTaxReceipts": { mn: "Нэхэмжлэх ба e-Barimt татварын баримт", en: "Public Billing & e-Barimt Tax Receipts" },
  "billing.stateFeeInvoicesBillingAndTaxReceipts": { mn: "Улсын тэмдэгтийн хураамж, нэхэмжлэх ба татварын баримт", en: "State fee invoices, billing, and tax receipts" },
  "billing.loadingInvoices": { mn: "Нэхэмжлэхүүдийг ачаалж байна...", en: "Loading invoices..." },
  "billing.createInvoice": { mn: "Нэхэмжлэх үүсгэх", en: "Create Invoice" },
  "billing.createStateFeeInvoice": { mn: "Улсын хураамжийн нэхэмжлэх үүсгэх", en: "Create State Fee Invoice" },
  "billing.contactClient": { mn: "Харилцагч", en: "Contact / Client" },
  "billing.totalAmount": { mn: "Нийт дүн", en: "Total Amount" },
  "billing.paymentStatus": { mn: "Төлбөрийн төлөв", en: "Payment Status" },
  "documents.digitalDocumentsEsignatures": { mn: "Цахим баримт ба тоон гарын үсэг", en: "Digital Documents & E-Signatures" },
  "documents.loadingDocuments": { mn: "Баримтуудыг ачаалж байна...", en: "Loading documents..." },
  "documents.createDocument": { mn: "Баримт үүсгэх", en: "Create Document" },
  "documents.createDigitalDocument": { mn: "Цахим баримт үүсгэх", en: "Create Digital Document" },
  "documents.documentTitle": { mn: "Баримтын гарчиг", en: "Document Title" },
  "documents.documentCategory": { mn: "Баримтын ангилал", en: "Document Category" },
  "documents.legalContract": { mn: "Гэрээ", en: "Legal Contract" },
  "documents.officialRequest": { mn: "Албан хүсэлт", en: "Official Request" },
  "documents.internalApproval": { mn: "Дотоод батламж", en: "Internal Approval" },
  "documents.digitalSignatureEidDan": { mn: "Тоон гарын үсэг (E-ID / ДАН)", en: "Digital Signature (E-ID / DAN)" },
  "documents.pendingSignature": { mn: "Гарын үсэг хүлээж буй", en: "Pending Signature" },
  "documents.createdDate": { mn: "Үүсгэсэн огноо", en: "Created Date" },
  "documents.status": { mn: "Төлөв", en: "Status" },
  "settingsApps.installedAppsSettings": { mn: "Суулгасан аппуудын тохиргоо", en: "Installed Apps Settings" },
  "settingsApps.loadingInstalledApps": { mn: "Суулгасан аппуудыг ачаалж байна...", en: "Loading installed apps..." },
  "settingsApps.applicationName": { mn: "Аппликейшны нэр", en: "Application Name" },
  "settingsApps.moduleId": { mn: "Модулийн ID", en: "Module ID" },
  "settingsApps.installedVersion": { mn: "Суулгасан хувилбар", en: "Installed Version" },
  "settingsApps.installedDate": { mn: "Суулгасан огноо", en: "Installed Date" },
  "settingsApps.status": { mn: "Төлөв", en: "Status" },
  "settingsApps.actions": { mn: "Үйлдэл", en: "Actions" },
  "integrations.externalSystemIntegrationsWebhooks": { mn: "Гадаад системийн интеграц ба Webhook", en: "External System Integrations & Webhooks" },
  "integrations.loadingIntegrations": { mn: "Интеграцуудыг ачаалж байна...", en: "Loading integrations..." },
  "integrations.addIntegration": { mn: "Интеграц нэмэх", en: "Add Integration" },
  "integrations.registerIntegrationConnector": { mn: "Интеграц холбогч бүртгэх", en: "Register Integration Connector" },
  "integrations.integrationType": { mn: "Интеграцын төрөл", en: "Integration Type" },
  "integrations.webhookListener": { mn: "Webhook хүлээн авагч", en: "Webhook Listener" },
  "integrations.governmentGateway": { mn: "Төрийн гарц", en: "Government Gateway" },
  "integrations.paymentGateway": { mn: "Төлбөрийн гарц", en: "Payment Gateway" },
  "integrations.customRestEndpoint": { mn: "Захиалгат REST хаяг", en: "Custom REST Endpoint" },
  "integrations.secretKeySigning": { mn: "Нууц түлхүүр (гарын үсэг)", en: "Secret Key (Signing)" },
  "integrations.optionalHmacSecret": { mn: "HMAC нууц түлхүүр (заавал бус)", en: "Optional HMAC secret" },
  "integrations.egSalesWebhook": { mn: "жишээ: Борлуулалтын Webhook", en: "e.g. Sales Webhook" },
  "copilotUI.aiCopilot": { mn: "AI Туслах", en: "AI Copilot" },
  "copilotUI.erpAiAssistant": { mn: "ERP AI туслах", en: "ERP AI Assistant" },
  "copilotUI.poweredByGeminiAiEngine": { mn: "Gemini AI хөдөлгүүр дээр ажиллана", en: "Powered by Gemini AI Engine" },
  "copilotUI.askAiCopilot": { mn: "AI туслахаас асуу...", en: "Ask AI Copilot..." },

  // ─── AI copilot ────────────────────────────────────────────────────────────
  "copilot.title": { mn: "AI Туслах", en: "AI Copilot" },
  "copilot.placeholder": { mn: "Асуултаа бичнэ үү...", en: "Ask a question..." },
  "copilot.send": { mn: "Илгээх", en: "Send" },
  "copilot.thinking": { mn: "Бодож байна...", en: "Thinking..." },
  // ─── Screens (follow-up) ───────────────────────────────────────────────────
  "login.eidMongoliaIdentityEidmongoliamn": { mn: "E-ID Mongolia танилт (eidmongolia.mn)", en: "E-ID Mongolia Identity (eidmongolia.mn)" },
  "login.nationalDanIdentityChannel": { mn: "Үндэсний ДАН Танилт Нэвтрэх Суваг", en: "National DAN identity channel" },
  "billing.ebarimtStatus": { mn: "e-Barimt төлөв", en: "e-Barimt Status" },
  "developer.loadingOauth2ClientApps": { mn: "OAuth2 client аппуудыг ачаалж байна...", en: "Loading OAuth2 client apps..." },
  "developer.oidcDiscoveryEndpoint": { mn: "OIDC Discovery хаяг", en: "OIDC Discovery Endpoint" },
  "developer.registerNewOauth2ClientApp": { mn: "Шинэ OAuth2 client апп бүртгэх", en: "Register New OAuth2 Client App" },
  "developer.registerOauth2Client": { mn: "OAuth2 client бүртгэх", en: "Register OAuth2 Client" },
  // ─── App store (follow-up) ─────────────────────────────────────────────────
  "apps.installEnableAndManage": { mn: "Компиляцын үеийн бизнес модулиудыг суулгаж, идэвхжүүлж, удирдана", en: "Install, enable, and manage compile-time business modules" },
  "apps.installedEnabled": { mn: "Суулгасан ба идэвхтэй", en: "Installed & Enabled" },
  "apps.disabled": { mn: "Идэвхгүй", en: "Disabled" },
  "apps.requires": { mn: "Шаардлага: ", en: "Requires:" },
  "apps.allCategories": { mn: "Бүгд", en: "All" },
  // ─── Shell, login, appearance (follow-up) ──────────────────────────────────
  "shell.closeMenu": { mn: "Цэс хаах", en: "Close menu" },
  "shell.openMenu": { mn: "Цэс нээх", en: "Open menu" },
  "shell.tenantActive": { mn: "· идэвхтэй", en: "· active" },
  "shell.toggleTheme": { mn: "Theme солих", en: "Toggle theme" },
  "shell.appearance": { mn: "Харагдац", en: "Appearance" },
  "login.identityChannel": { mn: "Танилт Нэвтрэх Суваг", en: "Identity channel" },
  "login.methodPki": { mn: "Тоон Гарын Үсэг (PKI)", en: "PKI digital signature" },
  "login.methodOtp": { mn: "Нэг удаагийн код (Mobile OTP)", en: "Mobile OTP" },
  "login.methodBank": { mn: "Банкны системээр нэвтрэх", en: "Bank SSO" },
  "login.methodBiometric": { mn: "Царай танилт (Biometric Face)", en: "Biometric face verification" },
  "login.regNumberRequired": { mn: "Иргэний Регистрийн Дугаар *", en: "Registration number *" },
  "login.otpCode": { mn: "Баталгаажуулах Код (OTP / Pin)", en: "Verification code (OTP / PIN)" },
  "login.cancel": { mn: "Цуцлах", en: "Cancel" },
  "login.verifyAndSignIn": { mn: "Баталгаажуулж Нэвтрэх", en: "Verify and sign in" },
  "contacts.xypVerifiedCitizen": { mn: "ХУР Баталгаажсан Иргэн", en: "XYP verified citizen" },
  "contacts.xypAutofill": { mn: "ХУР / XYP Auto-fill", en: "XYP auto-fill" },
  "billing.contactPlaceholder": { mn: "e.g. Гэрэгэ Системс ХХК", en: "e.g. Gerege Systems LLC" },
  "documents.titlePlaceholder": { mn: "e.g. Хамтран ажиллах гэрээ 2026", en: "e.g. Partnership agreement 2026" },
  "appearance.light": { mn: "Гэгээн", en: "Light" },
  "appearance.dark": { mn: "Харанхуй", en: "Dark" },
  "appearance.system": { mn: "Системийн", en: "System" },
  "appearance.teal": { mn: "Хөх ногоон", en: "Teal" },
  "appearance.violet": { mn: "Нил ягаан", en: "Violet" },
  "appearance.emerald": { mn: "Маргад", en: "Emerald" },
  "appearance.title": { mn: "Харагдац", en: "Appearance" },
  "appearance.subtitle": { mn: "Энэ төхөөрөмж дээр Gerege ERP хэрхэн харагдахыг тохируулна.", en: "Choose how Gerege ERP looks on this device." },
  "appearance.reset": { mn: "Анхны төлөв", en: "Reset to defaults" },
  "appearance.themeStyle": { mn: "Theme загвар", en: "Theme style" },
  "appearance.themeStyleHint": { mn: "ERP-ийн анхны харагдац эсвэл Gerege дизайн системийг сонгоно.", en: "Pick the classic ERP look or the Gerege design system." },
  "appearance.classicHint": { mn: "Odoo-маягийн анхны ERP интерфэйс", en: "The classic Odoo-style ERP interface" },
  "appearance.geregeHint": { mn: "Gerege-ийн cobalt дизайн систем", en: "The Gerege cobalt design system" },
  "appearance.colorMode": { mn: "Өнгөний горим", en: "Colour mode" },
  "appearance.colorModeHint": { mn: "Гэгээн, харанхуй эсвэл төхөөрөмжийн тохиргоог дагана.", en: "Light, dark, or follow the device setting." },
  "appearance.accent": { mn: "Онцлох өнгө", en: "Accent colour" },
  "appearance.accentHint": { mn: "Cobalt нь Gerege-ийн үндсэн брэнд өнгө.", en: "Cobalt is the primary Gerege brand colour." },
  "appearance.density": { mn: "Дэлгэцийн нягтрал", en: "Display density" },
  "appearance.comfortable": { mn: "Тав тухтай", en: "Comfortable" },
  "appearance.compact": { mn: "Нягт", en: "Compact" },
} as const;

export type TranslationKey = keyof typeof dictionary;

interface I18nValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: TranslationKey, vars?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nValue | null>(null);

function isLocale(value: string | null): value is Locale {
  return value === "mn" || value === "en";
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  // Server and first client render must agree, so the stored preference is
  // applied in an effect rather than during initial state.
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);

  useEffect(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (isLocale(stored)) {
      setLocaleState(stored);
    }
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value = useMemo<I18nValue>(() => {
    const setLocale = (next: Locale) => {
      window.localStorage.setItem(STORAGE_KEY, next);
      setLocaleState(next);
    };

    const t = (key: TranslationKey, vars?: Record<string, string | number>) => {
      const entry = dictionary[key];
      let text: string = entry ? entry[locale] : key;
      if (vars) {
        for (const [name, replacement] of Object.entries(vars)) {
          text = text.replace(new RegExp(`\\{${name}\\}`, "g"), String(replacement));
        }
      }
      return text;
    };

    return { locale, setLocale, t };
  }, [locale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used inside <I18nProvider>");
  }
  return ctx;
}
