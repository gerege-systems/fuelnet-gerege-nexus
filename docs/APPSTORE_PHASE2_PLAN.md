# 2-р үе: «Шастир» ба платформ нэгдлийн төлөвлөгөө

**Хамрах хүрээ:** (1) Nexus дээр модулийн хөгжүүлэлт/шинэчлэлтийг хянадаг, хийгдсэн
ажлын товч тайлбартай update log систем — нэршил, байршил, appstore руу урсах зам;
(2) аппад байж болох бүх метадатаг (хэн үүсгэсэн, хэн шинэчилсэн г.м.) хоёр талд
хадгалах; (3) appstore.gerege.mn ба developer.gerege.mn-ийг Nexus платформ дээр
ажиллуулах нэгдэл.

**Огноо:** 2026-08-11 · Суурь: `open-gerege-nexus` @ `75769ff6` (хоёр талын кодыг
бүрэн уншсан байдлаар)

---

## 1. Нэршил: «Шастир» (Chronicle)

Модулийн хөгжүүлэлтийн түүхийг хөтөлдөг дэд системийг **Шастир** гэж нэрлэе —
монголоор түүх бичгийн нэр, англиар `chronicle`. Кодод `chronicle` гэсэн нэрээр
явна: `catalog/chronicle/`, `GET …/chronicle`, `chronicle-check`.

Гол санаа: **хувилбар бүр өөрийн шастирын бичлэгтэй, бичлэг нь manifest дотор
аялдаг.** Manifest аль хэдийн гарын үсэгтэйгээр registry → instance хооронд
зорчдог тул шинэ wire contract хэрэггүй — одоо байгаа сувгаар түүх өөрөө урсана.

### 1.1 Эх сурвалж — `catalog/chronicle/<slug>.json` (анхны эх нь repo)

Нэгдүгээр эх сурвалж нь код өөрчилдөг газар байх ёстой тул first-party
модулиудын шастир repo дотор, manifest-ийн хажууд амьдарна:

```json
{
  "app_id": "io.example.contacts",
  "entries": [
    {
      "version": "1.1.0",
      "released_at": "2026-08-11",
      "kind": "feature",
      "summary": { "mn": "ХУР авто-бөглөлтийг хуулийн этгээдэд нэмэв", "en": "XYP autofill for legal entities" },
      "details": { "mn": "WS100201-ээр …", "en": "…" },
      "authors": ["craftzbay"],
      "refs": ["#82"]
    }
  ]
}
```

- `kind`: `feature | fix | security | breaking | docs`
- `summary` заавал (mn + en); `details`, бусад 5 хэл сайн дурын — одоогийн
  орчуулгын бодлоготой ижил.
- **CI сахиул (`chronicle-check`):** модулийн хувилбар ахисан мөртлөө тухайн
  хувилбарын бичлэг шастирт байхгүй бол build унана. Одоогийн
  хувилбар-зөрүүний шалгагчийн (`verifyCatalogVersions`) яг хажууд, Go тест
  хэлбэрээр — version bump хийсэн хүн ажлаа нэг өгүүлбэрээр бичихээс өөр
  аргагүй болно. (`frontend/scripts/i18n-check.mjs`-ийн жишгээр CI-д шинэ job
  биш, байгаа lint шатанд нэмнэ.)

### 1.2 Manifest v2.1 — метадата ба шастир нэг дор

`appcatalog.Manifest`-д нэмэгдэх талбарууд (бүгд optional, хуучин manifest
хөндөгдөхгүй):

```json
{
  "publisher": "gerege",
  "authors":     [ { "name": "…", "email": "…", "gerege_sub": "…" } ],
  "maintainers": [ { "name": "…", "email": "…" } ],
  "repository": "https://github.com/gerege-systems/open-gerege-nexus",
  "homepage": "https://…", "license": "Apache-2.0",
  "release_notes": {
    "kind": "feature",
    "summary": { "mn": "…", "en": "…" },
    "details": { "mn": "…", "en": "…" },
    "authors": ["…"], "refs": ["#82"]
  }
}
```

- `release_notes` = **тухайн хувилбарын** шастирын бичлэг. Каталог ачаалагч
  (file горим) болон registry-гийн catalog builder хоёул manifest-д шастираас
  автоматаар шигтгэнэ — гараар хоёр газар бичихгүй.
- Бүрэн түүх хувилбар хувилбараараа хадгалагдана: nexus талд `app_versions.manifest`
  (аль хэдийн бичигдэж байгаа), registry талд `store_app_versions.manifest` —
  өөрөөр хэлбэл **түүхийн сан хоёр талд аль хэдийн баригдчихсан, дутуу нь
  зөвхөн бичлэгийн агуулга ба харуулах гадаргуу.**
- `ValidateManifest` өргөтгөл: release_notes байвал summary.mn заавал; authors
  дүрэм; license SPDX хэлбэр.

### 1.3 Nexus дээрх гадаргуу (хянадаг систем)

1. **Store дэлгэц:** Update товчтой картад "Юу шинэчлэгдсэн" —
   `latest.release_notes.summary` (локалиар). Товч дарж болох эсэхийг шийдэхэд
   хамгийн их хэрэгтэй ганц өгүүлбэр яг тэнд.
2. **Settings → Суулгасан аппууд → «Түүх» drawer:** апп бүрд нэгтгэсэн он дараалал —
   `installation_events` (installed/upgraded/held, хэн, хэзээ; `system` sweep
   тусдаа тэмдэгтэй) + `app_versions`-ийн release_notes нийлж нэг timeline болно.
   API: `GET /api/v1/store/apps/{slug}/history` (нэвтэрсэн гишүүн уншина,
   RBAC-ийн `.read`-ээр).
3. **Админ тойм:** `GET /api/v1/admin/store/overview` — модуль бүрийн
   binary-version / catalog-version / суулгасан тенантын тоо / сүүлийн sync
   төлөв, сүүлийн алдаа. (Өмнөх review-гийн (б), (д) олдворыг энд нэг мөсөн
   шийднэ: "sync чимээгүй хоцордог" асуудал энэ дэлгэцээр ил болно.)

### 1.4 Appstore руу урсах зам

- **Гуравдагч тал:** консолын submission маягтад Release notes хэсэг нэмэгдэнэ
  (mn/en заавал) → manifest-д орж ирдэг тул registry талд өөрчлөлт бага.
- **First-party (Nexus-ийн өөрийн модулиуд):** release tag дээр CI-гийн шинэ
  алхам `publish-catalog` — өөрчлөгдсөн модуль бүрийн manifest(+шастир)-ыг
  registry-гийн dev API руу автоматаар submit хийнэ. Түр хугацаанд registry-д
  **бот-издательгчийн токен** (`PUBLISH_TOKEN`, seed "gerege" publisher-т
  хатуу холбогдсон, зөвхөн submit эрхтэй — publish шийдвэр хэвээр хүний гарт);
  §3-ын нэгдэл дууссаны дараа энэ нь in-process болж токен хэрэггүй болно.
- Ингэснээр урсгал: **код + шастир нэг PR-д → merge → release → appstore-ийн
  review queue-д автоматаар → хүн publish дарна → бүх instance sync-ээр авна →
  auto_update тенантуудад өөрөө тархана.** Хийгдсэн ажил бичигдээгүй бол CI
  унадаг тул түүх алдагдах зам байхгүй.

### 1.5 Registry/storefront талын өөрчлөлт

- `store_apps`-д: `authors JSONB`, `maintainers JSONB`, `repository`, `homepage`,
  `license`, `created_by`, `updated_by` (Caller-ийн sub/email-ээс), `created_at/updated_at` (бий).
- `store_app_versions`-д: `release_notes JSONB` (manifest-аас хуулбарлаж
  query-д хялбар болгоно), `authors JSONB`.
- Registry API: `GET /api/v1/registry/apps/{slug}/chronicle` — бүх нийтэд
  нээлттэй, хувилбар дараалсан шастир.
- Storefront аппын хуудас: Publisher badge (verified тэмдэгтэй), authors,
  maintainers, license, repo линк, хувилбарын түүх бүлэг бүр release_notes-той,
  хэн submit хийж хэн publish хийснийг `review_events`-ээс. 7 хэлээр.

---

## 2. «Хэн юу хийсэн» — метадатагийн бүрэн зураглал

Зорилго: аппын амьдралын мөчлөгийн алхам бүр нэртэй, цагтай, шалтгаантай байх.
Ихэнх нь аль хэдийн бичигдэж байгаа — дутуу нь нэгтгэж харуулах тал:

| Асуулт | Хаана хадгалагдана | Байдал |
| --- | --- | --- |
| Хэн зохиосон / арчилдаг | manifest `authors`/`maintainers` | 🔲 шинэ (§1.2) |
| Хэн publisher, баталгаажсан уу | `publishers` (owner_sub, verified) | ✅ бий |
| Хэн энэ хувилбарыг submit хийсэн | `store_app_versions.submitted_by` | ✅ бий |
| Хэн review хийж publish/yank хийсэн | `review_events` (actor, action, note) | ✅ бий — storefront-д харуулах нь 🔲 |
| Аппын бүртгэлийг хэн үүсгэж/зассан | `store_apps.created_by/updated_by` | 🔲 шинэ |
| Тенант дээр хэн суулгаж/шинэчилсэн | `installation_events.details.user_id`, audit.Record | ✅ бий — UI 🔲 (§1.3-2) |
| Автомат шинэчлэлт үү, хүн үү | `user_id: "system"` ялгаа | ✅ бий |
| Юу өөрчлөгдсөн (агуулга) | Шастир → manifest `release_notes` | 🔲 §1 |

---

## 3. Appstore ба developer-ийг Nexus платформ дээр ажиллуулах

### 3.1 Одоогийн байдал ба зорилтот дүр зураг

Одоо: registry = тусдаа Go binary (`cmd/appstore`), консол = тусдаа Next.js BFF,
storefront = тусдаа Next.js SSR. Ажиллаж байгаа ч Nexus-ийн session, RBAC, 7
хэлний i18n, E-ID нэвтрэлт, меню shell, audit зэргийг **давхардуулан дутуу
хувилбараар** дахин бүтээсэн хэлбэртэй.

Зорилт: **appstore.gerege.mn = Nexus-ийн тусдаа instance**, дээр нь 3 шинэ
компайл-модуль ажиллана. Nexus нь чухамдаа "апп тээдэг платформ" тул аппын
дэлгүүр өөрөө түүн дээр ажиллах нь платформын хамгийн үнэмшилтэй демо нь мөн.

```
appstore.gerege.mn  = Nexus instance (яг тэр binary, өөрийн DB, өөрийн catalog)
  ├── io.gerege.appstore_registry   каталог, snapshot+гарын үсэг, нийтийн API
  ├── io.gerege.publisher_studio    апп бүртгэл, хувилбар submit, шастир засварлагч
  └── io.gerege.store_review        review queue, publisher баталгаажуулалт
developer.gerege.mn = мөн энэ instance-ийн frontend (тусдаа vhost, нэг stack)
```

**Publisher = Tenant.** Гуравдагч тал appstore instance дээр байгууллагаа
тенантаар бүртгэнэ. Үүгээр дараах бүхэн үнэгүй ирнэ: E-ID/ХУР-аар хуулийн
этгээдийн баталгаажуулалт (`WS100201` код бэлэн), гишүүдийн membership + RBAC
(хэн submit хийх эрхтэйг publisher өөрөө роль дээрээ шийднэ), audit, 7 хэл,
session удирдлага, "хэн үүсгэсэн/зассан" нь memberships-ээс автоматаар. Одоогийн
"нэг хүн = нэг publisher" хязгаар (кодод өөрөө "a team belongs to an
organisation … modelling that properly is worth more" гэж тэмдэглэсэн) яг
энэ загвараар шийдэгдэнэ.

### 3.2 Цөмд шаардлагатай өөрчлөлт (бага)

1. **Нийтийн route-ийн журам:** модуль `RegisterRoutes`-доо root router авдаг тул
   gate-гүй нийтийн зам mount хийх нь техникийн хувьд боломжтой —
   `appstore_registry` модуль `/api/v1/registry/*`, `/.well-known/appstore-keys.json`-ыг
   нийтэд, админ/публишерийн замаа gate-тэй mount хийнэ. Үүнийг MODULE_AUTHORING_GUIDE-д
   журамлан бичих (private-гаа андуурч нийтэд гаргахаас CI-гийн route-тест хамгаална).
2. **Instance profile = catalog файл.** Аль instance ямар модуль санал болгохыг
   аль хэдийн `catalog/apps.json` шийддэг тул цөмд "profile" ойлголт нэмэх
   шаардлагагүй: appstore instance-ийн catalog нь өөрийн 3 модультай тусдаа
   файл (`catalog/profiles/appstore/apps.json`). Нэг binary — хоёр өөр catalog.
3. **Signer платформын үйлчилгээ болно:** `appstore.Signer`-ыг registry модульд
   шилжүүлж SIGNING_KEY-ийг instance-ийн env-д үлдээнэ. `/catalog` contract
   **байтын түвшинд хэвээр** — `contract_test.go` үүнийг өмнө нь ч хоёр талыг
   холбож баталдаг, нэгдлийн дараа ч хэвээр батална (энэ тест л migration-ийн
   аюулгүйн бүс).

### 3.3 Юу хаана үлдэх вэ

| Хэсэг | Шийдвэр | Учир |
| --- | --- | --- |
| Registry backend | Nexus модуль болно (код `internal/appstore`-оос модулийн бүрхүүлд нүүнэ, логик хэвээр) | Session/RBAC/audit давхардал арилна |
| Консол (developer-web) | **Татан буугдана** — publisher_studio модулийн дэлгэцүүд nexus frontend-д (`app/module/publisher/…`) | BFF, id_token verifier, тусдаа OAuth client бүгд илүүц болно: хэрэглэгч instance-ийн өөрийн session-тэй |
| Storefront (appstore-web) | **Үлдэнэ** (толгойн шийдвэр) — нэвтрэлтгүй, SEO-той нийтийн хуудас нь платформ-shell-ийн бус тусдаа асуудал; platform UI-ийн өнгө/типографикаар нэг мөр загварчилна | Nexus frontend нь нэвтрэлт шаарддаг client-side shell; нийтийн SSR-ийг түүнд оруулах нь ашиггүй том ажил |
| OIDC issuer | nexus.gerege.mn хэвээр (өөрчлөхгүй) | Гадны бүх client үүн дээр бүртгэлтэй |
| Registry-гийн auth | Instance-ийн өөрийн session (E-ID, и-мэйл) | id_token-verifier зам гадны CLI/CI дуудлагад л үлдэнэ |
| Nexus↔appstore sync contract | Огт өөрчлөгдөхгүй | Талбар дахь instance-үүд юу ч мэдрэхгүй |

Developer_portal (OAuth2 клиентүүд) nexus.gerege.mn дээрээ үлдэнэ — issuer тэнд
байгаа болохоор. developer.gerege.mn-ийн цэснээс хөндлөн линкээр холбоно;
хоёр консол нэг харагдацтай байх нь одоо platform shell дундын учир өөрөө гарна.

### 3.4 Data migration

1. `appstore_db`-д платформын цөм миграциуд (00001…00033) + appstore миграциуд
   нэг DB-д тавигдана (`platform_core` + `store_*` хүснэгтүүд зэрэгцэнэ).
2. Publisher → Tenant скрипт: publisher бүрд tenant үүсгэж (slug хэвээр),
   `owner_sub`/`owner_email`-ээр хэрэглэгч үүсгэн admin membership өгнө;
   `publishers.tenant_id` FK нэмж хуучин баганууд deprecated болно.
3. `store_apps.publisher_id` хэвээр (publisher нь одоо тенантын "профайл" мөр).
4. Cutover: шинэ instance зэрэгцээ босгож `/catalog`-ийн хариуг байтаар нь
   хуучинтай тулгасны (diff = 0) дараа nginx upstream-ыг солино; хуучин stack
   DNS буцаах rollback болж 2 долоо хоног амьд үлдэнэ.

---

## 4. Үе шатууд

| Шат | Ажил | Хугацаа | Хамаарал |
| --- | --- | --- | --- |
| Ш1 | «Шастир»: файл формат + loader + `chronicle-check` CI + Manifest v2.1 + байгаа 8 модульд эхний бичлэгүүд | 4–5 өдөр | — |
| Ш2 | Nexus гадаргуу: store картын "Юу шинэчлэгдсэн", Түүх drawer + history API, админ тойм (sync төлөвтэй) | 4–5 өдөр | Ш1 |
| Ш3 | Registry/storefront метадата: шинэ баганууд, chronicle API, storefront түүх+authors+publisher badge, консолын submission маягтад release notes | 4–5 өдөр | Ш1 |
| Ш4 | CI автомат publish (бот токен, зөвхөн submit) | 2 өдөр | Ш3 |
| Ш5 | Платформ нэгдэл: 3 модуль бичих (код нүүлгэлт), нийтийн route журам, appstore instance profile | 2 долоо хоног | Ш2, Ш3 |
| Ш6 | Publisher→Tenant migration + байт-тулгалттай cutover + консол татан буулгалт | 1 долоо хоног | Ш5 |
| Ш7 | Цэвэрлэгээ: `cmd/appstore` HTTP давхарга, developer-web устгах; өмнөх review-гийн үлдэгдэл (rate limit, snapshot цэвэрлэгээ, advisory lock) энд хамт | 3 өдөр | Ш6 |

Ш1–Ш4 нь Ш5-аас хамааралгүй тул **Шастир шууд эхэлж болно** — нэгдлийг хүлээх
шаардлагагүй, дараа нь модуль руу кодтойгоо хамт нүүнэ.

## 5. Эрсдэл

| Эрсдэл | Хариу |
| --- | --- |
| Нэгдлийн үеэр `/catalog` contract эвдрэх | contract_test хоёр талыг нэг тестээр барьдаг хэвээр; cutover-ын байт-тулгалт (diff=0) заавал |
| Publisher→Tenant migration дутуу холбогдох | Скрипт idempotent, dry-run-тай; хуучин баганууд устгалгүй deprecated |
| Registry модулийн нийтийн route андуурагдах | Route-жагсаалтын unit тест: нийтэд гарах замын цагаан жагсаалтаас гадуурх бүх зам 401 буцаадгийг батална |
| Шастир бичих сахилга | CI-гаас өөр зам байхгүй болгоно (version bump = бичлэг заавал); бичлэг нь 1 өгүүлбэр байхад хангалттай гэдгийг guide-д онцлон дарамт болгохгүй |
| Storefront үлдээх шийдвэр эргэлзээ төрүүлэх | Хэрэв бүх юмыг нэг дор гэвэл: nexus frontend-д public layout нэмэх хувилбарыг Ш5-ын үед дахин үнэлж болно — одоо шийдэх албагүй |

## 6. Амжилтын шалгуур

1. Модулийн хувилбар ахих бүрд шастирын бичлэг заавал үүсдэг, CI-гүйгээр давах
   боломжгүй; тэр бичлэг store карт, түүх drawer, storefront гурвууланд нэг эхээс харагдана.
2. Апп бүрийн "хэн зохиосон / хэн submit хийсэн / хэн publish хийсэн / аль тенант
   хэзээ хэний гараар (эсвэл system-ээр) шинэчилсэн" гинж бүрэн мөрдөгддөг.
3. Nexus release → appstore review queue хүртэл гар ажиллагаагүй; publish хэвээр хүний шийдвэр.
4. appstore.gerege.mn Nexus instance болж, publisher байгууллагууд тенантаар
   E-ID баталгаажуулалттай ажилладаг; developer-web консол устсан; `/catalog`-ийн
   хариу байтын түвшинд өөрчлөгдөөгүй.
