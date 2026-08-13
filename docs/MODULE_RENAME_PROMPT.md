# Claude Code — Цөмийн модулиудын нэршлийн засвар (Салгалтын 0-р алхам)

Экосистемийн салгалтаас өмнө хийгдэх нэршлийн ажил. Доорх prompt-ыг
Claude Code-д бүтнээр нь өгнө.

---

## PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. Эхлээд контекстоо бүрдүүл:

- `docs/ECOSYSTEM_GIT_STRATEGY.md` §2.5 — энэ ажлын дизайн шийдвэр,
  эх сурвалж нь энэ баримт.
- `docs/MODULE_AUTHORING_GUIDE.md`, `backend/internal/module.go`,
  `backend/internal/apps/` бүтэц, `catalog/` (apps.json, manifests,
  chronicle), `backend/internal/platform/appcatalog`, `appinstaller`,
  `appregistry`.
- Module ID хаана хадгалагддагийг өөрөө мөшгиж жагсаа: `app_installations`,
  `apps`, `installation_events`, каталог файлууд, цэсний түлхүүр, RBAC
  permission-ий префикс, frontend route/цэс, native цэсний түлхүүр —
  энэ жагсаалт чинь миграцын чеклист болно.

Зорилго: цөмийн модулиудын нэршлийг салгалтаас өмнө нэг мөр болгох.
**Гурван ажил, гурван тусдаа commit** — дараалал нь чухал биш ч тус
бүр нь бие даан ногоон байх ёстой.

### Заавал мөрдөх дүрмүүд

1. Репогийн хэв маяг: `pgx` + гар SQL, goose миграц, шинэ хүснэгтгүй
   бол RLS-д нөлөөгүй; i18n нь монгол эх + англи, бусад нь
   TRANSLATION_GUIDE журмаар.
2. **Alias зарчим**: хуучин module ID, хуучин route хоёулаа нэг release
   ажиллаж байгаад дараагийнхад хасагдана. Native клиентүүд болон гадаад
   каталог хэрэглэгчид шууд эвдэрч болохгүй. Alias бүр код дээр
   `// DEPRECATED: remove in vNEXT` тэмдэгтэй.
3. Миграц бүр **буцаах (down) хэсэгтэй**, мөн хоосон/хуучин өгөгдөлгүй
   DB дээр ч, өгөгдөлтэй DB дээр ч ажилладаг байх (idempotent UPDATE).
4. Шалгалт: `cd backend && go test -race ./... && go vet ./... &&
   golangci-lint run`, `cd frontend && npm run build`, мөн доорх
   grep-шалгуурууд.

---

### Ажил 1 — `apps/core` → `apps/organisation`

1. Хавтас, пакет, төрлийн нэрс: `apps/core` → `apps/organisation`;
   Module ID `io.gerege.nexus.core` → `io.gerege.nexus.organisation`;
   Name нь "Organisation & People" хэвээр (i18n түлхүүрүүд шинэ ID-гаар).
2. Каталог: `catalog/manifests/core.json` → `organisation.json` шинэ
   ID-тэй; `chronicle/core.json` мөн адил; `apps.json`-д шинэчилнэ.
   `appcatalog`-д **ID alias map** нэм: `io.gerege.nexus.core` →
   `io.gerege.nexus.organisation` — хуучин ID-тай каталог/суулгалт
   уншигдвал шинэ рүү нь резолв хийнэ (DEPRECATED тэмдэгтэй).
3. Миграц: `app_installations`, `apps`, `installation_events` доторх
   хуучин ID-г UPDATE; permission префикс өөрчлөгдөж байвал
   `permissions`/`role_permissions`-ийг мөн адил.
4. **Давхарга салгах**: тенантын хуулийн профайл (нэр, регистр, хаяг,
   лого — CP, ХУР баталгаажуулалт, SSO consent-д апп суусан эсэхээс үл
   хамааран хэрэгтэй өгөгдөл) нь `platform/tenant` руу нүүнэ; аппад
   зөвхөн хэлтэс/ажилтан (HR-lite) үлдэнэ.
5. **Устгагдашгүй статусыг хас**: `appinstaller`-ийн `CoreApps`
   жагсаалтаас гаргаж, default-оор суудаг ч устгаж болдог болго
   (кодын аудитаар өөр модуль үүнээс хамаардаггүй нь тогтоогдсон).
   Тест: 0 апптай тенант дээр платформ бүрэн асч, нэвтрэлт/тохиргоо
   ажиллана; organisation-ийг устгаад дахин суулгахад өгөгдөл
   алдагдахгүй (хүснэгтүүд арчигдахгүй, зөвхөн gating хаагдана).
5. Frontend `/organisation` route аль хэдийн зөв — module ID ашигласан
   газруудыг л шинэчил.

### Ажил 2 — `apps/developer_portal` → `apps/sso_clients`

1. Хавтас/пакет/төрөл: `DeveloperPortalModule` → `SSOClientsModule`;
   ID `io.gerege.nexus.developer_portal` → `io.gerege.nexus.sso_clients`;
   Name "Developer Portal & OAuth2 SSO" → "SSO клиентүүд" (en: "SSO
   Clients"). Каталог, миграц, alias — Ажил 1-тэй ижил журмаар.
2. Route: `/api/v1/developer/*` → `/api/v1/sso-clients/*`; хуучин зам
   нэг release-д 308 redirect эсвэл давхар mount (DEPRECATED).
3. Frontend: `app/developer/` → `app/sso-clients/` (эсвэл одоогийн
   Тохиргооны бүтцэд нийцүүлж байрлуул — байгаа UI хэв маягийг дага);
   цэсний нэр, i18n шинэчил.
4. Грэп-шалгуур: `grep -ri "developer_portal\|DeveloperPortal" backend
   frontend` — үр дүн зөвхөн alias/миграц/CHANGELOG дотор үлдэнэ.
   Анхаар: appstore-ын `publisher_studio` болон гадаад
   developer.gerege.mn консолд бүү хүр — тэд өөр зүйл.

### Ажил 3 — `egov` модуль ялгаж гаргах

1. Шинэ `apps/egov`: ID `io.gerege.nexus.egov`, Name "Цахим засгийн
   холболт" (en: "e-Government Link"). Дараахыг **нүүлгэж** авчир:
   - ХУР лавлагааны endpoint-ууд: одоо `server.go`-д байгаа
     `/xyp/citizen`, `/xyp/company` → `/api/v1/egov/citizen`,
     `/api/v1/egov/company` (хуучин зам нэг release DEPRECATED alias).
     Permission: `xyp.citizen.read` → `egov.citizen.read` (миграцаар
     UPDATE, хуучин нэр alias-гүй — permission нь дотоод тул шууд).
   - eID/ДАН identity холболтын хэрэглэгчийн урсгалууд (profile-ийн
     identities хэсгээс апп-нүүр рүү) ба баталгаажуулалтын түүхийн
     харагдац — байгаа handler-уудыг platform-аас module руу зөө,
     доод түвшний клиентүүд (`platform/gerege`, `platform/eid`,
     `platform/dan`) байрандаа үлдэнэ.
2. `contacts` цэвэрлэгээ: ХУР авто-бөглөлт нь `egov`-ийн Go үйлчилгээг
   (module-аас экспортолсон interface) дууддаг болго; contacts дотор
   төрийн интеграцын шууд дуудлага үлдэхгүй. `egov` нь `contacts`-ын
   dependency БИШ — contacts нь egov суугаагүй үед авто-бөглөлтгүй,
   энгийнээр ажиллана (graceful degradation, тестээр батал).
3. Каталог: `egov`-ийг суурь профайлуудад default-оор суудаг апп болгож
   нэм (устгаж болно — organisation шиг устгагдашгүй биш).
4. Frontend: `/egov` route — лавлагаа, холболтууд, түүх гэсэн гурван
   дэлгэц; цэс i18n 2 хэлээр.
5. Тест: egov суугаагүй тенантад `/api/v1/egov/*` 403 (app gating),
   contacts хэвийн; суусан үед лавлагаа ажиллана (mock горимд);
   permission миграц хуучин role-уудыг эвдээгүй.

---

### Дуусгах шалгуур

- Гурван commit, бүх тест/lint/build ногоон, CI эвдрээгүй.
- Grep: `io.gerege.nexus.core`, `io.gerege.nexus.developer_portal`,
  `developer_portal`, `/xyp/` — зөвхөн alias, миграц, CHANGELOG-д.
- Хуучин ID-тай өгөгдөлтэй DB дээр миграц ажиллуулахад суулгалт,
  цэс, эрхүүд бүрэн шилжсэн байх integration тест.
- README-гийн аппын хүснэгт, `docs/ARCHITECTURE_SPECIFICATION.md`,
  `docs/MODULE_AUTHORING_GUIDE.md`-ийн жишээнүүд, CHANGELOG шинэчлэгдсэн.
- Дараагийн release-д хасах DEPRECATED жагсаалтыг CHANGELOG-д тусад нь
  бичсэн байх.

Эхлэхийн өмнө ID хадгалагддаг газруудын бүрэн жагсаалтаа надад
харуулж баталгаажуулаад, дараа нь Ажил 1-ээс эхэл.

---

## PROMPT төгсөв
