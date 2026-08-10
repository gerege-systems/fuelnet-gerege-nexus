# App Store-ыг салгаж appstore.gerege.mn дээр байршуулах төлөвлөгөө

**Хамрах хүрээ:** `appstore.gerege.mn` (нэгдсэн апп стор / registry), `developer.gerege.mn`
(хөгжүүлэгчийн консол), гуравдагч талын ажиллаж буй платформыг апп болгон бүртгэх
загвар, болон шинэчлэлтийн (update) механизмын бүрэн тооцоо.

**Огноо:** 2026-08-10 · **Шинэчилсэн:** 2026-08-10, PR #68 (`appstore-separation-prep`)
merge болсны дараах кодын байдлаар.

---

## 0. Гүйцэтгэлийн байдал

Nexus талын бэлтгэл ажил (анхны төлөвлөгөөний Үе шат 2 ба 3-ын runtime тал) **PR #68-аар
бүрэн хэрэгжсэн.** Кодыг дахин уншиж баталгаажуулсан байдал:

### 0.1 Хийгдсэн ✅

| Ажил | Хаана | Тэмдэглэл |
| --- | --- | --- |
| `installed_version` дахин суулгалтад ахидаг болсон, зөрүүтэй бол `upgraded` event (`from`/`to`) бичдэг | `appinstaller/installer.go` `installOrUpgrade` | Install/upgrade нэг transaction-д нэгдсэн |
| `app_versions`-д хувилбарын түүх бичигдэж эхэлсэн | `SyncCatalog` | `DO NOTHING` — нийтлэгдсэн хувилбар immutable |
| Модуль ↔ каталог ↔ manifest хувилбарын зөрүү startup error болсон | `server.go` `verifyCatalogVersions` + `appcatalog.ValidateCatalog` | esign, developer_portal-ын зөрүүг хамт зассан |
| `PlatformVersion` ldflags-аар тарьдаг, `/health`-д `platform_version` гардаг | `deploy/Dockerfile` `ARG VERSION`, `server.go` | |
| Алдааны leak засвар, permission grant-ын silent error/N+1 арилсан | `catalog_handlers.go`, `grantAppPermissions` | `ErrAppNotFound`/`ErrNotInstalled`/`ErrAlreadyCurrent` sentinel-ууд |
| `auto_update`, `pinned_version` баганууд | миграци `00033` | Доорх 0.2-ыг хар: багана бий, **бодлого нь хараахан ажилладаггүй** |
| `POST /store/apps/{slug}/upgrade` (admin, 409 already-current) | `handleUpgradeApp` | App gate invalidation-тай |
| Store жагсаалтад `latest_version`, `update_available`; UI-д Update товч, хувилбарын badge, 7 хэлний i18n | `handleListStoreApps`, `frontend/app/apps/page.tsx`, `i18n/addons/app_store.ts` | |
| Remote catalog client: `APP_CATALOG_URL` + Ed25519 verify + ETag + disk cache + bundled fallback | `appcatalog/source.go` (`Provider`) | Файл горим 100% хэвээр; түлхүүргүй бол remote асахгүй |
| Background sync (`CATALOG_SYNC_INTERVAL`, min 1m) + `POST /api/v1/admin/store/sync` | `server.go` `startCatalogSync`, `handleSyncCatalog` | 304-д DB/кэш хөндөхгүй; амжилтад бүх тенантын app gate bus-аар invalidate |
| `type: "external"` апп: manifest валидаци (absolute https launch_url, embed), модуль шалгалт алгасах, permission-ыг signed manifest-аас grant | `appcatalog/manifest.go`, `installer.go` | `catalog/manifests/example-external.json` жишээ (каталогт ороогүй — загвар) |
| External цэс: `MenuDefinition.ExternalURL`, Layout `target="_blank"` рендэр, manifest-ийн `.read` permission-оор нуугдана | `module.go`, `menu/menu.go`, `Layout.tsx`, `appReadPermission` | |
| SSO install gate: external аппын client-д зөвхөн суулгасан тенантын хэрэглэгч нэвтэрнэ | `external_apps.go`, `ssoprovider.AttachInstallGate`, `install_gate_test.go` | Тест бүрэн |
| `tenant_id` claim id_token/userinfo/introspect-д | `ssoprovider/endpoints.go` | |
| `.env.example`, `MODULE_AUTHORING_GUIDE.md`, `CHANGELOG.md` баримтжуулалт | | |

**Чухал үр дагавар:** registry-гийн wire contract одоо nexus-ийн клиент кодоор
**тогтчихсон** — appstore.gerege.mn сервисийг үүнд нийцүүлж барина (§4.1-ийн спек
одоо normative).

### 0.2 Nexus талд үлдсэн жижиг ажлууд 🔲

1. **`auto_update` бодлого хэрэгждэггүй.** Багана бий, `GetInstallationsForTenant`
   уншдаг, гэхдээ sync-ийн дараа auto_update=true суулгалтуудыг ахиулах ажил (sweep)
   байхгүй, UI toggle ч байхгүй. Registry ассаны дараа утга учиртай болно —
   `syncCatalogFromRegistry` амжилттай болсны дараа: тенант бүрийн auto_update=true,
   pinned_version IS NULL суулгалтад `installOrUpgrade` дуудах job нэмэх.
2. **`pinned_version` хэрэгждэггүй.** Upgrade дээр pin-ийг зөв зөөдөг ч суулгах/сync
   зам pin-ийг тооцдоггүй, scope өргөжсөн external аппад "дахин зөвшөөрөх" UI урсгал
   алга (§4.4). Registry-тэй хамт хийгдэнэ.
3. `tenant_slug` claim (одоо зөвхөн `tenant_id`) — гуравдагч талд slug илүү хэрэгтэй
   байж болзошгүй, жижиг нэмэлт.
4. External аппын `health_url`-ыг юу ч poll хийдэггүй (кодод "Nothing polls it yet"
   гэж тэмдэглэгдсэн) — Settings→Installed apps дээр төлөв харуулах бол жижиг job.
5. Pilot external апп: `example-external.json` каталогт ороогүй загвар тул бодит
   гуравдагч системээр бүтэн урсгалыг (бүртгэл→суулгалт→SSO нэвтрэлт) турших ажил үлдсэн.

### 0.3 Огт эхлээгүй (гол үлдэгдэл) 🔲

Registry сервис + storefront (appstore.gerege.mn), гарын үсэг зурагч хэрэгсэл,
developer.gerege.mn консол, шинэ домэйнуудын nginx/compose/CI. Доорх §5-ын шинэчилсэн
үе шатууд.

---

## 1. Суурь кодын тойм (лавлагаа)

### 1.1 App Store-ын одоогийн бүтэц

| Давхарга | Байршил | Үүрэг |
| --- | --- | --- |
| Каталог эх сурвалж | `appcatalog.Provider` (`source.go`) | remote registry → disk cache → bundled file гэсэн гурван шатлалт |
| Каталог файл | `catalog/apps.json` + `catalog/manifests/*.json` | Self-hosted-ийн эх сурвалж, бусдын нь эцсийн fallback |
| DB sync | `appinstaller.SyncCatalog` | `apps` upsert + `app_versions` түүх |
| Суулгагч | `appinstaller.installOrUpgrade` | DAG resolution, permission grant, install/upgrade event |
| App gate | `appInstalled` (memo + Redis bus) | Модуль аппад route-ийн өмнө, external аппад `/oauth2/auth` дээр |
| Store API | `/api/v1/store/apps…`, `…/upgrade`, `/api/v1/admin/store/sync` | |
| Store UI | `frontend/app/apps/page.tsx` | Update товч, хувилбарын badge-тай |
| Developer Portal | `apps/developer_portal` модуль (тенант доторх) | developer.gerege.mn руу шилжих UI |
| OIDC provider | `platform/ssoprovider` + external install gate | Issuer: `PUBLIC_ORIGIN` |

Deploy загвар: нэг хост, олон vhost (nexus, sso), олон compose stack, CI→GHCR→SSH.
Шинэ домэйнууд энэ загварыг дагана.

### 1.2 Анхны review-ийн олдворуудын байдал

Анхны төлөвлөгөөнд заасан 7 олдвороос: №1 (installed_version), №2 (app_versions),
№3 (хувилбарын зөрүү), №5 (нэвтрэлтгүй storefront-ыг registry шийднэ), №7 (error
leak, silent grant, N+1) — **бүгд засагдсан**. Үлдсэн нь: №4 `PUBLIC_ORIGIN`-ий
олон үүрэг (домэйн нэмэхэд тохиргоогоор шийдэгдэнэ, §6) ба №6 icon-ууд (storefront
хийхэд registry-гээс icon татдаг болгох — MinIO/`icon_key`, §2.2).

---

## 2. Зорилтот архитектур (өөрчлөлтгүй)

```
                        ┌──────────────────────────────┐
                        │   appstore.gerege.mn         │
                        │  ─ Storefront (Next.js, SSR) │
                        │  ─ Registry API (Go)         │──── appstore_db (Postgres)
                        │  ─ Каталог гарын үсэг зурагч │──── MinIO (icon, пакет)
                        └──────────┬───────────────────┘
                 signed catalog    │ ▲ publish (хянагдсан)
                 (pull + cache) ✅ │ │
      ┌────────────────────────────▼─┴──────────────┐
      │  nexus.gerege.mn — клиент тал БЭЛЭН ✅       │
      │  (Provider, sync, upgrade, external gate)   │
      └────────────────────────────▲────────────────┘
                                   │ OIDC (issuer: nexus, PKCE) ✅
      ┌────────────────────────────┴────────────────┐
      │  developer.gerege.mn (хөгжүүлэгчийн консол) │
      │  ─ OAuth2 client удирдлага (API бэлэн ✅)    │
      │  ─ Апп бүртгэл, хувилбар нийтлэх, review 🔲 │
      └─────────────────────────────────────────────┘
```

Үүрэг хуваарилалт, OIDC issuer-ийг нүүлгэхгүй байх шийдвэр, appstore_db схем
(`publishers`, `store_apps`, `store_app_versions`, `store_app_texts`,
`external_registrations`, `review_events`, `install_stats`) анхны төлөвлөгөөний
хэвээр. Нэг тодотгол: `store_app_versions.manifest`-ийн бүтэц одоо nexus-ийн
`appcatalog.Manifest` (Type/External талбартай)-тай яг таарах ёстой.

---

## 3. Гуравдагч талын платформыг апп болгон бүртгэх

Nexus талын дэмжлэг **бүрэн хэрэгжсэн** (§0.1). Одоо шаардлагатай нь бүртгэлийн
урсгалын бүтээгдэхүүн тал:

1. Publisher developer.gerege.mn дээр Gerege SSO-гоор нэвтэрч байгууллагаа
   баталгаажуулна (ХУР `WS100201` код бэлэн).
2. OAuth2 client үүсгэнэ — `developer_portal` модулийн одоогийн API шууд ашиглагдана.
3. Manifest v2 (`type: "external"`, `external.launch_url/sso_client_id/scopes/embed/health_url`)
   бөглөж submission → review → publish. Загвар: `catalog/manifests/example-external.json`.
4. Дараагийн sync-ээр instance бүрт очно; суулгасан тенантын цэсэнд гарч,
   SSO нэвтрэлт нь install gate-ээр хамгаалагдана — энэ бүхэлдээ ажиллаж буй код.

Шаардлагатай нэмэлт (registry талд): review queue, scope-өргөжилтийг илрүүлж
хувилбарыг "re-approval шаардана" гэж тэмдэглэх, yank урсгал.

---

## 4. Update механизм

### 4.1 Registry wire contract — одоо NORMATIVE (nexus клиент хэрэгжүүлчихсэн)

appstore.gerege.mn-ий Registry API нь `appcatalog/source.go`-гийн хүлээлттэй мөр
мөрөөрөө нийцэх ёстой:

```
GET {APP_CATALOG_URL}/catalog?platform=<semver>&channel=<stable|beta>
  Хүсэлт:  Accept: application/json, If-None-Match: <etag> (байвал)
  Хариу:   200 + ETag толгой, эсвэл 304 (өөрчлөлтгүй)
  Дээд хэмжээ: 8 MiB (nexus үүнээс томыг хаяна)

  200 body:
  {
    "generated_at": "<publish хийсэн цаг, replay-protection-д ордог>",
    "key_id": "<аль түлхүүрээр зурсан>",
    "apps": [ ...CatalogApp массив, manifest-ууд нь дотроо... ],
    "signature": "<base64(Ed25519.Sign(priv, generated_at + '\n' + apps-ийн ТҮҮХИЙ байтууд))>"
  }
```

Анхаарах зүйлс (клиент кодоос урган гарсан хатуу шаардлагууд):

- Гарын үсэг **apps массивын түүхий байтууд дээр** зурагдана — сервер apps-ыг
  serialize хийснийхээ дараа тэр байтуудаа өөрчлөхгүйгээр хариултад оруулах ёстой
  (nexus `json.RawMessage`-ээр авч шалгадаг, дахин encode хийдэггүй).
- `generated_at` гарын үсэгт орсон тул хуучин catalog-ыг replay хийж болохгүй.
- Nexus хариуг хүлээж авахдаа өөрийн `ValidateCatalog` + модуль-хувилбарын
  тулгалтыг давхар ажиллуулна — validation унасан хариу бүхэлдээ хаягдана.
  Тиймээс registry **нийтлэхээсээ өмнө** мөн адил validation ажиллуулж байх ёстой
  (буруу manifest нийтэлбэл instance-үүд зүгээр л update авахаа болино — эвдрэхгүй,
  гэхдээ мэдэгдэлгүй хоцорно).
- Түлхүүр: Ed25519; nexus-д `APPSTORE_PUBLIC_KEY` (base64) pin хийгдэнэ. Эргэлтэд
  `key_id` + `/.well-known/appstore-keys.json` (registry талд хийнэ).

Мөн хэрэгтэй: **гарын үсэг зурагч CLI** (`cmd/catalog-sign` маягийн жижиг хэрэгсэл) —
registry сервис бэлэн болтол одоогийн `apps.json`-оос signed catalog үүсгэж туршилт
хийх, мөн offline publish-ийн зам болно.

### 4.2 Дөрвөн сувгийн байдал

| Суваг | Байдал |
| --- | --- |
| (A) Каталог/manifest sync | ✅ Клиент тал бүрэн; registry сервер л дутуу |
| (B) Module апп = binary release | ✅ `min_platform` шалгалт, ldflags, drift check бэлэн; release train баримтжуулах л үлдсэн |
| (C) Платформ binary | ✅ Одоогийн CI→GHCR хэвээр; registry дээр releases feed (nice-to-have) |
| (D) PWA | ✅ Өөрчлөлт шаардахгүй |

### 4.3 Тенант түвшний урсгал

Хэрэгжсэн: install/upgrade/event/badge/Update товч (§0.1). Үлдсэн:

- **Auto-update sweep** (§0.2-1): registry sync амжилттай болмогц auto_update=true,
  pin-гүй суулгалтуудыг шинэ хувилбар руу ахиулах; event нь `upgraded`
  (`user_id: "system"` маягийн ялгаатай тэмдэглэгээтэй).
- **Pin/re-approval UI** (§0.2-2): scope өргөжсөн external хувилбарыг админд
  жагсааж, "Зөвшөөрч шинэчлэх" үйлдэл (`UpgradeApp` аль хэдийн pin-ийг зөв зөөдөг).
- Settings → Installed apps дээр auto_update toggle + pinned badge.

### 4.4 External аппын update дүрэм (өөрчлөлтгүй)

Metadata-only → шууд; scope нэмэгдсэн → pin + админы дахин зөвшөөрөл; yank →
шинээр суулгахыг хаах + шаардлагатай бол SSO client-ийн `Disabled` flag.

---

## 5. Шинэчилсэн үе шатууд

Nexus талын бэлтгэл дууссан тул дараалал өөрчлөгдөв — одоо **registry сервис нь
critical path**:

### Үе шат A — Registry сервис + гарын үсэг (1–1.5 долоо хоног) 🔲

- `services/appstore/` (Go, chi): §4.1-ийн contract-ыг яг мөрдсөн `/catalog`
  endpoint, `/.well-known/appstore-keys.json`, goose миграци (§2.2 схем),
  одоогийн `apps.json`+manifests импортын seed.
- Гарын үсэг зурагч CLI (§4.1) — эхний өдрүүдэд registry-гүйгээр ч nexus-ийн
  remote горимыг бодитоор турших боломж олгоно.
- **Integration тест:** nexus-ийг `APP_CATALOG_URL`-тэй нь registry рүү заалгаж,
  sync → store badge → upgrade бүтэн урсгалыг staging дээр батлах.
- Deploy: nginx vhost `appstore.gerege.mn.conf` (API 8083), compose stack
  `/opt/appstore`, `deploy-appstore.yml`, certbot, DNS.

### Үе шат B — Storefront (1 долоо хоног) 🔲

- Next.js SSR нээлттэй каталог (нэвтрэлтгүй), 7 хэл (`store_app_texts`),
  апп дэлгэрэнгүй + хувилбарын түүх, icon-ууд MinIO-оос (nexus-ийн icon дутуу
  асуудлыг хамт шийднэ). Storefront порт 3009.

### Үе шат C — Nexus-ийн үлдэгдэл + production асаалт (2–3 өдөр) 🔲

- §0.2-ын 1–5: auto-update sweep, pin/re-approval UI, tenant_slug claim,
  health poll, pilot external апп.
- Production: nexus-д `APP_CATALOG_URL`, `APPSTORE_PUBLIC_KEY` секрет тавьж
  remote горимд шилжүүлнэ (fallback-ууд байгаа тул эрсдэл бага; эхний долоо
  хоногт файл каталогоо registry-тэй ижил байлгаж давхар хамгаалалттай явна).

### Үе шат D — developer.gerege.mn консол (2 долоо хоног) 🔲

- Publisher бүртгэл, submission/review/publish UI; OAuth2 client console-ыг
  одоогийн `developer_portal` API дээр OAuth2 bearer-ээр (session cookie хил
  давуулахгүй). Порт 3010.
- Шилжилтийн үед nexus доторх Developer Portal апп хэвээр; консол тогтворжоод
  deprecate. eID-ийн одоогийн developer.gerege.mn агуулгатай мөргөлдөхгүй
  байдлаар subpath/таб-аар эхэлнэ (бүтээгдэхүүний шийдвэр хүлээгдэж байгаа).

### Үе шат E — Хатуужуулалт (1 долоо хоног) 🔲

- Channel-ууд registry талд (nexus `APP_CATALOG_CHANNEL` аль хэдийн дэмждэг),
  install_stats (opt-in), yank урсгал, platform releases feed, баримтжуулалт.

---

## 6. Тохиргоо (шинэчилсэн)

| Сервис | Env | Байдал |
| --- | --- | --- |
| nexus | `APP_CATALOG_URL`, `APPSTORE_PUBLIC_KEY`, `CATALOG_CACHE_PATH`, `CATALOG_SYNC_INTERVAL`, `APP_CATALOG_CHANNEL` | ✅ Хэрэгжсэн, `.env.example`-д баримтжуулагдсан |
| nexus build | `docker build --build-arg VERSION=x.y.z` | ✅ CI-д VERSION дамжуулахыг release train-д нэмэх 🔲 |
| appstore API | `DATABASE_URL`, `SIGNING_KEY`, `MINIO_*`, `PUBLIC_ORIGIN` | 🔲 |
| developer web | `NEXT_PUBLIC_API_URL` (nexus), OIDC public client + PKCE | 🔲 |
| nexus CORS | `ALLOWED_ORIGINS` += `https://developer.gerege.mn` | 🔲 асаалтын үед |

## 7. Эрсдэл (шинэчилсэн)

Анхны хүснэгт хүчинтэй хэвээр. Нэмэлт хоёр зүйл:

- **Гарын үсгийн байт-нарийвчлал:** registry apps массивыг дахин serialize хийвэл
  гарын үсэг үл тохирч бүх instance update авахаа болино. Registry-гийн хариултыг
  DB-д signed-блокоор нь бэлэн хадгалж (per channel/platform bucket), хүсэлт бүрт
  дахин угсрахгүй байх — ETag ч үүнээс шууд гарна.
- **Тест matrix:** remote горим асаахын өмнө staging дээр гурван fallback
  (registry унтраах, cache устгах, түлхүүр буруу тавих) бүгдийг нэг бүрчлэн
  батлах — код нь бүгдийг зөв хийхээр бичигдсэн ч production-д анх удаа асаж байгаа зам.

## 8. Амжилтын шалгуур

1. ✅ Тенантын админ store-оос апп-аа нэг товчоор шинэчилдэг, `upgraded` event +
   `app_versions` түүх бичигддэг. *(хэрэгжсэн, registry-гүйгээр файл каталог дээр ч ажиллана)*
2. 🔲 appstore.gerege.mn нэвтрэлтгүйгээр каталог үзүүлдэг, catalog хариу бүр
   Ed25519 гарын үсэгтэй, nexus түүнийг хүлээн авдаг.
3. 🔲 Registry унтарсан ч nexus cache/файлаараа boot хийдэг нь staging дээр батлагдсан.
4. 🔲 Гуравдагч талын бодит нэг платформ стороос суулгагдаж, зөвхөн суулгасан
   тенантын хэрэглэгч Gerege SSO-гоор нэвтэрдэг. *(механизм бэлэн, pilot үлдсэн)*
5. 🔲 developer.gerege.mn дээрээс OAuth2 client + апп хувилбар бүрэн удирдагддаг.
