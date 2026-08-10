# App Store-ыг салгаж appstore.gerege.mn дээр байршуулах төлөвлөгөө

**Хамрах хүрээ:** `appstore.gerege.mn` (нэгдсэн апп стор / registry), `developer.gerege.mn`
(хөгжүүлэгчийн консол), гуравдагч талын ажиллаж буй платформыг апп болгон бүртгэх
загвар, болон шинэчлэлтийн (update) механизмын бүрэн тооцоо.

**Огноо:** 2026-08-10 · **Суурь код:** `open-gerege-nexus` (бүрэн уншиж танилцсан)

---

## 1. Одоогийн байдал — кодын review дүгнэлт

### 1.1 App Store одоо хэрхэн ажиллаж байна

Апп стор бүхэлдээ **нэг процесс дотор** амьдарч байна:

| Давхарга | Байршил | Үүрэг |
| --- | --- | --- |
| Каталог файл | `catalog/apps.json` + `catalog/manifests/*.json` | Цорын ганц эх сурвалж; boot бүрт уншина |
| Каталог ачаалагч | `platform/server.go` → `appcatalog.LoadManifestFile` | Slug шалгалт, semver шалгалт, ID тулгалт |
| DB sync | `appinstaller.SyncCatalog` | `apps` хүснэгтийг файлаас upsert хийнэ |
| Суулгагч | `appinstaller.InstallApp` | DAG dependency resolution, permission grant, event log |
| Модулийн registry | `appregistry` (compile-time) | Go модуль бинарид байгаа эсэхийг баталгаажуулна |
| App gate | `server.go` `appGate` (memo + Redis bus) | Тенант бүрийн суулгалтын кэш, 30s TTL |
| Store API | `catalog_handlers.go` (`/api/v1/store/apps…`) | Жагсаалт, суулгах/идэвхжүүлэх (admin) |
| Store UI | `frontend/app/apps/page.tsx`, `settings/apps` | Тенант доторх дэлгүүрийн дэлгэц |
| Developer Portal | `apps/developer_portal` модуль + `frontend/app/developer/**`, `module/developer/**` | OAuth2 client удирдлага (тенант доторх апп) |
| OIDC/OAuth2 provider | `platform/ssoprovider` (root `/oauth2/*`, `/.well-known/*`) | Платформын өөрийн authorization server |

Deploy: нэг хост, `docker-compose.prod.yml` (postgres + redis + migrate + backend + frontend),
nginx vhost `nexus.gerege.mn` (loopback 3008/8082), CI → GHCR → SSH rollout. Хажууд нь
`sso.gerege.mn` stack аль хэдийн зэрэгцэн ажилладаг гэдгийг nginx config-ийн тайлбар өөрөө хэлж байна —
өөрөөр хэлбэл **нэг хост, олон vhost, олон compose stack** гэсэн загвар аль хэдийн тогтсон.

### 1.2 Салгахад бэлэн болсон "заадас" (сайн тал)

- `appcatalog`-д **CatalogRepository / PackageStorage / PackageVerifier / Installer** interface-үүд
  аль хэдийн зарлагдсан ("future marketplace / remote registry boundary") — remote registry-д
  зориулж зориуд үлдээсэн залгах цэг.
- Миграци `00002_app_store.sql`-д `app_versions` (manifest JSONB, `package_url`, `package_sha256`)
  болон `app_dependencies` хүснэгтүүд **аль хэдийн бий** — гэхдээ одоогоор **юу ч бичдэггүй**.
- `PlatformVersion = "1.0.0"` тогтмол + manifest-ийн `platform` semver constraint шалгалт бий —
  апп ↔ платформын нийцлийг илэрхийлэх суурь бэлэн.
- `ALLOWED_ORIGINS` таслалаар олон origin дэмждэг, Redis bus нь олон replica-д кэш invalidation
  хийдэг — олон домэйн/олон сервис рүү тэлэхэд саад болохгүй.
- OIDC provider бүрэн ажиллаж байгаа (auth code + PKCE, consent, introspect, JWKS) — гуравдагч
  талын аппын SSO-гийн суурь **бэлэн**.

### 1.3 Салгалтад саад болох / засах ёстой олдворууд

1. **`installed_version` хэзээ ч ахидаггүй.** `installer.InstallApp`-ын дахин суулгах салбар
   (`UPDATE app_installations SET status, enabled, updated_at …`) хувилбарыг шинэчилдэггүй.
   Каталог дээр апп 1.1.0 болсон ч тенантын мөр 1.0.0 хэвээр үлдэнэ. Update механизм хийхийн
   өмнө засах **№1 blocker**.
2. **`app_versions` / `app_dependencies` ашиглагдахгүй байна.** Хувилбарын түүх, update
   илрүүлэлт, rollback бүгд эндээс эхлэх ёстой — registry салгахад энэ хүснэгт гол болно.
3. **Manifest ↔ каталог хувилбарын зөрүү шалгагддаггүй.** `LoadManifestFile` зөвхөн ID тулгадаг;
   `apps.json`-ы `version` ба manifest-ийн `version` зөрж болно. Мөн Go модулийн `Version()`
   (жишээ нь developer_portal `2.0.0`) каталогийн `1.0.0`-тэй **аль хэдийн зөрчилдсөн** байна.
4. **`PUBLIC_ORIGIN` гурван үүрэг давхар гүйцэтгэдэг** (CORS + OIDC issuer + eID callback).
   Домэйн салгахад origin бүрийг тусад нь тохируулах шаардлагатай (`ALLOWED_ORIGINS` жагсаалт,
   `SSO_ISSUER` тогтвортой үлдээх).
5. **Store API нэвтэрсэн хэрэглэгч шаарддаг** — нийтэд харагдах storefront (SEO, нээлттэй
   каталог) одоогийн API дээр боломжгүй. Registry-гийн read зам нэвтрэлтгүй байх ёстой.
6. **Frontend-ийн icon-ууд**: `apps/page.tsx` зөвхөн 3 slug-д icon хатуу кодолсон, каталогийн
   `icon_url` (`/icons/contacts.png` …) заасан файлууд `public/`-д байхгүй. Storefront хийхэд
   icon-ыг registry-гээс тат(уул)даг болгох хэрэгтэй.
7. Жижиг: `handleInstallApp` алдааг `err.Error()`-оор шууд буцаадаг (дотоод мэдээлэл алдагдах),
   installer доторх permission grant-ууд `_, _ =`-ээр алдаа залгидаг, permission бүрт admin role
   давтан query хийдэг (N+1). Салгалттай зэрэгцээд цэгцлэхэд зохимжтой.

**Ерөнхий дүгнэлт:** кодын чанар өндөр, тайлбар сайтай, салгалтын interface-үүд урьдчилан
зарлагдсан. App Store-ыг тусад нь гаргахад архитектурын хувьд "хагалах" биш "заадсаар нь
салгах" ажил болно.

---

## 2. Зорилтот архитектур

```
                        ┌──────────────────────────────┐
                        │   appstore.gerege.mn         │
                        │  ─ Storefront (Next.js, SSR) │
                        │  ─ Registry API (Go)         │──── appstore_db (Postgres)
                        │  ─ Manifest/пакет гарын үсэг │──── MinIO (icon, пакет)
                        └──────────┬───────────────────┘
                 signed catalog    │ ▲ publish (хянагдсан)
                 (pull + cache)    │ │
      ┌────────────────────────────▼─┴──────────────┐
      │  nexus.gerege.mn (runtime, өөрчлөгдөхгүй    │
      │  цөм) — catalog-оо registry-гээс татна,     │
      │  тенантын суулгалт/эрх дотроо хэвээр        │
      └────────────────────────────▲────────────────┘
                                   │ OIDC (issuer: nexus, PKCE)
      ┌────────────────────────────┴────────────────┐
      │  developer.gerege.mn (хөгжүүлэгчийн консол) │
      │  ─ OAuth2 client удирдлага                  │
      │  ─ Апп бүртгэл, хувилбар нийтлэх, review    │
      │  ─ Гуравдагч талын платформ бүртгэх         │
      └─────────────────────────────────────────────┘
```

### 2.1 Үүрэг хуваарилалт

| Сервис | Юуг эзэмшинэ | Юуг эзэмшихгүй |
| --- | --- | --- |
| **appstore.gerege.mn** | Аппын каталог, хувилбарын түүх, manifest-ийн гарын үсэг, нийтлэх суваг (stable/beta), нийтийн storefront, суулгалтын статистик | Тенантын суулгалт, эрх, session — эдгээр runtime-д үлдэнэ |
| **developer.gerege.mn** | Publisher бүртгэл, апп submission/review, OAuth2 client console (одоогийн developer_portal UI-ийн залгамжлагч), API баримтжуулалт | Token гаргах (issuer nexus/sso дээрээ үлдэнэ) |
| **nexus.gerege.mn** | Тенант, суулгалт, app gate, RBAC, бизнес модулиуд | Каталогийн эх сурвалж (registry-гээс авна), OAuth2 client-ийн UI (шилжинэ) |

Чухал шийдвэр: **OIDC issuer-ийг нүүлгэхгүй.** `SSO_ISSUER`-ийг өөрчлөх нь бүх client,
discovery, JWKS pinning-ийг хамт нүүлгэдэг тул issuer `https://nexus.gerege.mn` хэвээр
(эсвэл аль хэдийн байгаа `sso.gerege.mn` рүү тусад нь, дараагийн шатанд). developer.gerege.mn
нь issuer-ийн **удирдлагын UI** нь болохоос issuer нь өөрөө биш.

### 2.2 appstore_db схем (шинэ)

```sql
publishers            (id, name, slug, contact_email, verified, tenant_hint, created_at)
store_apps            (id, publisher_id, slug, type 'module'|'external', name,
                       description, icon_key, category, visibility, created_at)
store_app_versions    (id, app_id, version, channel 'stable'|'beta',
                       min_platform, manifest JSONB, manifest_sig,        -- Ed25519
                       package_url, package_sha256,                       -- ирээдүйд
                       status 'draft'|'in_review'|'published'|'yanked',
                       published_at, UNIQUE(app_id, version))
store_app_texts       (app_id, locale, name, description, category)       -- 7 хэл
external_registrations(app_id, launch_url, sso_client_id, scopes[],
                       embed 'new_tab'|'iframe', health_url, webhook_url)
review_events         (id, version_id, actor, action, note, created_at)
install_stats         (app_id, instance_id, tenant_count, reported_at)    -- opt-in
```

Одоогийн `apps.json` + `manifests/*.json` нь энэ хүснэгтийн **seed** болно (нэг удаагийн
импортын скрипт). Nexus доторх `apps`, `app_installations` хүснэгтүүд instance бүрийн
**локал толь** хэвээр — registry-гээс sync хийгддэг болно.

---

## 3. Гуравдагч талын ажиллаж буй платформыг апп болгон бүртгэх

Одоо ажиллаж байгаа гадны систем (жишээ нь тусдаа billing, HR,档 гэх мэт SaaS) кодоо Nexus-д
оруулахгүйгээр апп стороор дамжин тенантад "суулгагдана".

### 3.1 Manifest v2 — `type: "external"`

```json
{
  "id": "mn.example.hrms",
  "type": "external",
  "name": "Example HRMS",
  "version": "2026.8.0",
  "platform": ">=1.0.0",
  "external": {
    "launch_url": "https://hrms.example.mn/sso/gerege",
    "sso": { "client_id": "app_hrms_x1y2", "scopes": ["openid","profile","erp.read"] },
    "embed": "new_tab",
    "health_url": "https://hrms.example.mn/healthz"
  },
  "permissions": [ { "code": "hrms.read", "name": "…" } ],
  "menus": [ { "id": "hrms_home", "label": "HRMS", "path": null,
               "external_url": "https://hrms.example.mn/sso/gerege", "icon": "share-2" } ]
}
```

### 3.2 Бүртгэлийн урсгал (developer.gerege.mn дээр)

1. Publisher Gerege SSO-гоор нэвтэрч байгууллагаа баталгаажуулна (ХУР `WS100201`-ийг
   ашиглаж хуулийн этгээдийн баталгаажуулалт хийж болно — код бэлэн).
2. OAuth2 client үүсгэнэ — одоогийн `developer_portal` модулийн логик шууд ашиглагдана:
   redirect URI шалгалт (`validateRedirectURI`), public client + PKCE, secret rotation бүгд бэлэн.
3. Manifest бөглөж submission илгээнэ → review → publish (channel сонгоно).
4. Publish болмогц дараагийн catalog sync-ээр бүх Nexus instance-д харагдана.

### 3.3 Nexus талын өөрчлөлт (бага, тодорхой)

- `appinstaller.InstallApp`: `type == "external"` үед `VerifyModuleExists`-ийг **алгасна**
  (Go модуль шаардахгүй); permission-ууд manifest-аас хэвээр グrant хийгдэнэ.
- `MenuDefinition`-д `ExternalURL string` талбар нэмж, `handleMenus` дамжуулна;
  `Layout.tsx` external цэсийг `target="_blank"` + external icon-той рендэрлэнэ
  (iframe embed нь CSP/`X-Frame-Options: DENY` бодлоготой зөрчилддөг тул анхдагч нь new_tab).
- **SSO-гийн суулгалтын хаалт:** `/oauth2/auth` дээр тухайн client нь external app-тай
  холбоотой бол хэрэглэгчийн идэвхтэй тенант уг аппыг суулгасан эсэхийг `app_installations`-аас
  шалгана — суулгаагүй тенантын хэрэглэгч гадны системд "Gerege-ээр нэвтрэх" боломжгүй.
  Энэ нь app gate-ийн логикийг гадны апп руу үргэлжлүүлж байгаа хэрэг.
- `id_token`/userinfo-д `tenant_id`, `tenant_slug` claim нэмнэ (гадны платформ аль
  байгууллагын нэрийн өмнөөс нэвтэрснийг мэдэх ёстой).
- Scope өргөжсөн шинэ хувилбар гарвал тенантын админ **дахин зөвшөөрөх хүртэл** суулгалт
  хуучин manifest хувилбар дээрээ явна (доорх update бодлого §4.4).

---

## 4. Шинэчлэлтийн (update) механизм — нарийвчилсан тооцоо

Дөрвөн өөр төрлийн "шинэчлэлт" байгааг ялгаж салгаж тооцов. Тус бүр өөр сувгаар явна:

| Юу шинэчлэгдэнэ | Суваг | Хурд |
| --- | --- | --- |
| (A) Каталог/manifest (метадата, external апп) | Registry sync | Минут–цаг |
| (B) Module апп (Go код) | Платформын binary release (CI→GHCR) | Release train |
| (C) Платформ өөрөө (nexus binary + frontend) | Одоогийн deploy.yml хэвээр | Release train |
| (D) PWA / браузер клиент | `sw.js` (одоо байгаа механизм хангалттай) | Дараагийн ачаалалт |

### 4.1 (A) Каталог sync — Nexus ↔ Registry протокол

Nexus-д `APP_CATALOG_URL` тохиргоо нэмнэ (жишээ: `https://appstore.gerege.mn/api/v1/registry`).

```
GET /api/v1/registry/catalog?platform=1.0.0&channel=stable&locale=all
  Headers: If-None-Match: <etag>, X-Instance-ID: <uuid>
  200 → { generated_at, key_id, apps: [ {catalog entry + latest version/channel
          + manifest + manifest_sig + sha256} ], signature }
  304 → өөрчлөлтгүй
GET /api/v1/registry/apps/{id}/versions        — хувилбарын түүх (rollback-д)
GET /.well-known/appstore-keys.json            — гарын үсгийн нийтийн түлхүүрүүд
```

Nexus талын дараалал (эх сурвалжийн fallback-тай):

1. **Boot:** remote catalog татна → Ed25519 гарын үсэг шалгана (`APPSTORE_PUBLIC_KEY`
   тохиргоонд pin хийсэн; түлхүүр эргэлтэд `key_id`-аар well-known-оос дараагийнхыг авна)
   → диск дээрх кэш рүү бичнэ (`/var/lib/nexus/catalog.cache.json`).
2. **Remote унавал:** дискийн кэш → тэр ч байхгүй бол repo-той хамт явдаг `catalog/apps.json`
   (bundled fallback). Boot **хэзээ ч** registry-ээс болж зогсохгүй — одоогийн
   "cold database must not stop boot" зарчимтай ижил.
3. **Ажиллах үед:** `CATALOG_SYNC_INTERVAL` (анхдагч 1h) тутам ETag-тэй refresh +
   админд "Шинэчлэлт шалгах" товч (`POST /api/v1/admin/store/sync`). Sync амжилттай
   болмогц `SyncCatalog` upsert + Redis bus-аар бүх replica-гийн app gate кэш invalidate.
4. **Air-gapped / self-hosted:** `APP_CATALOG_URL` хоосон бол одоогийн файл горим 100%
   хэвээр ажиллана — нээлттэй эхийн хэрэглэгчдэд эвдрэл үгүй.

Signed catalog нь registry эвдэрсэн/шахагдсан тохиолдолд ч instance-үүд хуурамч manifest
хүлээж авахгүй байх баталгаа. `PackageVerifier` interface яг энд хэрэгжинэ.

### 4.2 (B) Module аппын шинэчлэлт — binary-тэй уялдсан тооцоо

Module апп нь бинарид компиллогддог тул **аппын шинэ хувилбар = платформын шинэ release**.
Үүнийг нуулгүй, registry дээр ил болгоно:

- `store_app_versions.min_platform` — тухайн апп хувилбар аль платформ хувилбараас
  боломжтойг заана. Nexus store UI: `binary доторх Version() >= registry latest` бол
  "Шинэчлэх" товч; үгүй бол "Платформ ≥ x.y.z шаардана" гэж идэвхгүй харуулна.
- `PlatformVersion`-ийг хатуу `"1.0.0"` байлгахаа болтой: build үед
  `-ldflags "-X platform.PlatformVersion=$(git describe)"` гэж тарааж, `/health`-д гаргана.
- CI дээр **зөрүү шалгагч** нэмнэ: Go модулийн `Version()` ↔ каталог `version` ↔ manifest
  `version` гурав тэнцүү биш бол build унана (одоогийн 2.0.0/1.0.0 зөрчил дахин гарахгүй).

### 4.3 Тенант түвшний update урсгал (одоо огт байхгүй, шинээр)

1. `installer.InstallApp`-ын дахин суулгах салбарт `installed_version = app.Version`
   нэмж засна (§1.3-ын №1 blocker).
2. Шинэ endpoint: `POST /api/v1/store/apps/{slug}/upgrade` (admin) —
   dependency-г дахин resolve → `installed_version` ахиулна → `installation_events`-д
   `'upgraded'` (`from`, `to` details-тэй) → app gate invalidate.
3. `handleListStoreApps`-ийн хариунд `latest_version`, `update_available`, `changelog_url`
   талбар нэмнэ; store UI-д "Шинэчлэх" товч + суулгасан хувилбар/сүүлийн хувилбарын badge.
4. `app_installations`-д `auto_update BOOLEAN NOT NULL DEFAULT TRUE` багана нэмнэ:
   sync бүрийн дараа auto_update=true суулгалтуудыг шууд ахиулна (external апп болон
   metadata-only өөрчлөлтөд аюулгүй), админ унтраасан тенант гараар шинэчилнэ.
5. Апп бүрийн хувилбар ахих бүрт `app_versions`-д мөр бичигдэж эхэлнэ — түүх, rollback,
   "аль тенант аль хувилбар дээр" гэсэн асуултын хариу эндээс гарна.

### 4.4 External аппын update ба зөвшөөрлийн дүрэм

- Metadata-only өөрчлөлт (нэр, тайлбар, icon, health_url): дараагийн sync-ээр шууд.
- `launch_url` домэйн солигдох / **scope нэмэгдэх**: суулгалт бүр хуучин manifest
  хувилбартаа **pin** хэвээр; тенантын админ "шинэ эрх зөвшөөрөх" дэлгэцээр баталсны дараа
  л шинэ хувилбар идэвхжинэ. (Suulгалтад `pinned_version` багана нэмэгдэнэ.)
- Publisher хувилбараа **yank** хийвэл: шинээр суулгах боломжгүй болно, суулгачихсан
  тенантуудад анхааруулга; аюулгүй байдлын шалтгаантай yank бол SSO client-ийг
  түр хаах (`Disabled` flag аль хэдийн бий) хүртэл арга хэмжээ.

### 4.5 (C)(D) Платформ ба клиентийн шинэчлэлт

- Платформын binary: одоогийн CI→GHCR→SSH урсгал хэвээр, харин registry дээр
  `GET /api/v1/registry/platform/releases` feed нэмнэ (semver, changelog, image digest) —
  self-hosted операторууд шинэ хувилбар гарсныг appstore-оос мэддэг болно.
- Rollback: image-ууд `:sha` tag-тай тул `workflow_dispatch` + tag input аль хэдийн
  rollback хийж чаддаг — өөрчлөлт хэрэггүй, баримтжуулна.
- PWA: `sw.js` нь `/api/*`-д хүрдэггүй, immutable asset-only кэштэй тул домэйн
  салгалтад нөлөөгүй; appstore storefront-д service worker **хэрэггүй** (нийтийн сайт).

---

## 5. Гүйцэтгэлийн үе шат

### Үе шат 0 — Бэлтгэл (1–2 өдөр)

- DNS: `appstore.gerege.mn`, (шаардлагатай бол) `developer.gerege.mn` A бичлэг.
- Порт хуваарилалт (одоогийн хостын loopback дүрмээр): appstore API `8083`, storefront
  `3009`; developer console `3010`. Compose: `/opt/appstore`, `/opt/developer-portal`.
- Ed25519 signing түлхүүр үүсгэж (registry гарын үсэг) secrets-д хийнэ;
  нийтийн түлхүүрийг nexus-ийн тохиргоонд pin хийнэ.
- Repo бүтэц: monorepo дотор `services/appstore/` (Go) + `services/appstore-web/` (Next.js)
  гэж эхлүүлэхийг санал болгож байна — код хуваалцах (appcatalog төрлүүд) хялбар, CI нэг.

### Үе шат 1 — Registry + Storefront (1–2 долоо хоног)

- Registry API (Go, chi — nexus-ийн загвараар): §4.1-ийн endpoint-ууд, §2.2-ын схем,
  goose миграци, `apps.json`+manifests импортын seed скрипт.
- Storefront (Next.js SSR): нээлттэй каталог (нэвтрэлтгүй!), 7 хэлний дэмжлэг
  (`store_app_texts`), апп дэлгэрэнгүй, хувилбарын түүх, "Suulgahдаа nexus дотроос" заавар.
- Nginx vhost `appstore.gerege.mn.conf` (одоогийн conf-ийн загвараар, certbot),
  compose stack, `deploy-appstore.yml` workflow (одоогийн deploy.yml-ийн хуулбар загвар).
- **Гарц:** appstore.gerege.mn амьд, одоогийн 8 апп нийтэд харагдана. Nexus-д өөрчлөлт 0.

### Үе шат 2 — Nexus-ийг registry-гээс хооллох (1 долоо хоног)

- `CatalogRepository`-гийн remote хэрэгжүүлэлт: fetch + signature verify + disk cache +
  bundled fallback (§4.1); `APP_CATALOG_URL`, `APPSTORE_PUBLIC_KEY`, `CATALOG_SYNC_INTERVAL` env.
- §4.3: `installed_version` засвар, `/upgrade` endpoint, `update_available`, store UI товч,
  `app_versions`-д бичиж эхлэх, `auto_update` багана.
- CI зөрүү шалгагч (модуль/каталог/manifest хувилбар).
- **Гарц:** nexus каталогоо appstore-оос авдаг, тенант апп-аа шинэчилж чаддаг болно.

### Үе шат 3 — External апп end-to-end (1–2 долоо хоног)

- Manifest v2 `type: external` (§3.1), installer/menu/Layout өөрчлөлт (§3.3),
  `/oauth2/auth` дээрх суулгалтын хаалт, `tenant_id` claim.
- Registry талд `external_registrations`, review урсгалын анхны (админ-гараар) хувилбар.
- Туршилтын гуравдагч тал: жинхэнэ ажиллаж буй нэг системийг (жишээ нь Gerege-ийн өөрийн
  нэг үйлчилгээ) pilot болгон бүртгэж бүтэн урсгалыг батална.
- **Гарц:** ажиллаж буй гадны платформыг стороос суулгаж, Gerege SSO-гоор нэвтэрдэг болно.

### Үе шат 4 — developer.gerege.mn консол (2 долоо хоног)

- Next.js консол + өөрийн жижиг API (эсвэл эхний ээлжид nexus API-г OAuth2 bearer-ээр
  дуудах): publisher бүртгэл, апп submission, хувилбар нийтлэх, review queue.
- Одоогийн `frontend/app/developer/**` + `module/developer/**` дэлгэцүүдийг консол руу
  нүүлгэнэ (OAuth2 client CRUD, audit, signing keys, scopes, endpoints — backend модуль
  `developer_portal` API хэвээр үлдэж, консол нь token-оор дуудна).
- Шилжилтийн үед nexus доторх Developer Portal апп хэвээр ажиллана; консол тогтворжсоны
  дараа каталог дээр "deprecated → developer.gerege.mn" тэмдэглэж, дараагийн major-оор хасна.
- **Анхаар:** README/докуудад `developer.gerege.mn` одоогоор **eID-ийн хөгжүүлэгчийн портал**
  гэж иш татагддаг. Нэг домэйн дээр нэгтгэх бол eID RP бүртгэл + Nexus app publishing +
  OAuth2 client гурвыг нэг консолын гурав таб болгох нь зөв урт хугацааны дүр зураг;
  эхний ээлжид зөрчилдөхгүйгээр subpath (`developer.gerege.mn/nexus`) юмуу шинэ таб-аар орно.

### Үе шат 5 — Хатуужуулалт (1 долоо хоног)

- Channel (stable/beta) сонголт instance түвшинд; install_stats (opt-in телеметр);
  yank урсгал; platform releases feed; баримтжуулалт (MODULE_AUTHORING_GUIDE-д external
  app бүлэг, registry API reference).

---

## 6. Тохиргооны өөрчлөлтийн хураангуй

| Сервис | Шинэ env |
| --- | --- |
| nexus backend | `APP_CATALOG_URL`, `APPSTORE_PUBLIC_KEY`, `CATALOG_SYNC_INTERVAL`, (хэвээр: `APP_CATALOG_PATH` fallback) |
| appstore API | `DATABASE_URL` (appstore_db), `SIGNING_KEY` (Ed25519 хувийн), `MINIO_*`, `PUBLIC_ORIGIN=https://appstore.gerege.mn` |
| appstore web | `NEXT_PUBLIC_REGISTRY_URL` (build arg, одоогийн загвараар) |
| developer web | `NEXT_PUBLIC_API_URL` (nexus API), OIDC client (public + PKCE, redirect `https://developer.gerege.mn/auth/callback`) |
| nexus CORS | `ALLOWED_ORIGINS`-д `https://developer.gerege.mn` нэмнэ (appstore-д хэрэггүй — registry read нь нэвтрэлтгүй, storefront API-г шууд дууддаггүй) |

Cookie хил давахгүй: developer консол nexus API-г **OAuth2 access token**-оор дуудна
(session cookie нь host-only хэвээр — `.gerege.mn` domain cookie руу орохгүй, энэ нь
аюулгүй байдлын хувьд зөв сонголт).

## 7. Эрсдэл ба шийдэх дүрэм

| Эрсдэл | Хариу |
| --- | --- |
| Registry унавал бүх instance-ийн стор "хоосорно" | Гурван шатлалт fallback (remote→disk cache→bundled); sync унасан ч суулгачихсан аппууд огт хамаарахгүй ажиллана |
| Гарын үсгийн түлхүүр алдагдах | key_id + well-known rotation, нөөц түлхүүр offline; yank + client disable зэрэг хойшлуулах арга бэлэн |
| Scope-creep — external апп эрхээ чимээгүй тэлэх | §4.4 pin + админ дахин зөвшөөрөл; consent дэлгэц scope тус бүрийг монголоор тайлбарладаг (бэлэн) |
| Нэг хостын нөөц (одоо 2 stack, 5 болно) | Stack бүр өөрийн postgres биш — appstore_db-г одоогийн postgres instance дээр тусдаа DB болгож эхлэх; ачаалал өссөн үед салгана |
| developer.gerege.mn-ийн одоогийн (eID) агуулгатай мөргөлдөх | Үе шат 4-ийн тэмдэглэл: subpath/таб-аар эхэлж, бүтээгдэхүүний шийдвэрээр нэгтгэнэ |
| Self-hosted хэрэглэгчид эвдрэх | Файл горим бүрэн хэвээр; registry нь opt-in |

## 8. Амжилтын шалгуур

1. appstore.gerege.mn нэвтрэлтгүйгээр каталог үзүүлдэг, manifest бүр гарын үсэгтэй.
2. nexus.gerege.mn каталогоо registry-гээс авч, registry унтарсан ч boot хийдэг.
3. Тенантын админ store-оос апп-аа нэг товчоор шинэчилдэг, `installation_events`-д
   `upgraded` бичигддэг, `app_versions`-д түүх үлддэг.
4. Гуравдагч талын ажиллаж буй нэг платформ стороос суулгагдаж, Gerege SSO-гоор,
   зөвхөн суулгасан тенантын хэрэглэгчдэд нэвтэрдэг.
5. developer.gerege.mn дээрээс OAuth2 client + апп хувилбар бүрэн удирдагддаг.
