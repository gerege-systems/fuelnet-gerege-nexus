# Хоёр урсгал — нэг платформ

**Санал.** Backend-ийг хоёр салгахгүй. Нэг backend, нэг бинарь, нэг deploy
хэвээр. Дотор нь **хоёр урсгал** — нэг байгууллагын нэрийн өмнөөс ажилладаг
код, бүх байгууллагын өмнөөс ажилладаг код — хоёрыг package, өгөгдлийн сангийн
schema, маршрут, тестээр нь ялган нэрлэнэ. Урсгалуудын нэр: **`tenant`** ба
**`platform`**, тэдний доорх суурь нь **`kernel`**. `cp` гэдэг тусдаа зүйл
байхаа болино: тэр бол платформын урсгалын консол, өөр юу ч биш.

Огноо: 2026-08-24 · Статус: санал · Хамрах хүрээ: `backend/`

---

## 1. Юуг харсан бэ — review

Кодыг дахин уншлаа. Доорх нь тоо ба байршилтайгаа.

### 1.1 Хоёр урсгал аль хэдийн байгаа, гэхдээ нэргүй

Ялгаа нь кодод хэдийнэ орсон байна — зөвхөн хаана ч бичигдээгүй:

| Механизм | Тенантын тал | Платформын тал |
| --- | --- | --- |
| Postgres role | `gerege_nexus_app` | `gerege_nexus_operator` |
| Context тэмдэг | `tenant.WithTenantID` | `dbguard.AsOperator` |
| RLS бодлого | `tenant_isolation` | нэрлэсэн `SELECT` grant-ууд |
| Audit | `audit_events` | `operator_audit` |
| Session | `sessions` | `operator_sessions` |
| Host | `nexus.gerege.mn` | `cp.nexus.gerege.mn` |
| Маршрут | `/api/v1/*` | `/cp/api/*` |

Долоон газарт нэг ижил зураас татагдсан байна. Найм дахь нь — **кодын
байршил** — татагдаагүй: хоёулаа `internal/platform/` дотор сууна.

### 1.2 `internal/platform` бол багц биш, хогийн сав болсон

- Үндсэн директорт **46 `.go` файл**, тэр дотроо `server.go` дангаараа
  **1321 мөр / 66 KB**.
- `auth_handlers.go` (27 KB), `catalog_handlers.go` (23 KB),
  `sso_client_handlers.go` (20 KB), `tenant_profile_handlers.go` (16 KB),
  `access_control.go` (15 KB) — бүгд нэг package дотор, нэг `Server` struct-ын
  method-ууд.
- Дэд package **36**, нийт `internal/` дор **40 103 мөр** (тестгүйгээр).

Энэ бол урсгалын асуудлын гол цэг. `internal/platform` гэдэг нэр «платформ»
гэсэн утга биш, «бусад бүгд» гэсэн утга үүрч байна.

### 1.3 `controlplane` бол домэйн биш, «операторын хийж чадах бүхэн»

`internal/platform/controlplane/` — **20 файл, 5867 мөр**, репо дахь хамгийн
том package. Дотор нь:

```
login.go session.go totp.go operator.go middleware.go   ← операторын биеийн байцаалт
tenants.go lifecycle.go quotas.go approvals.go          ← тенантын амьдралын мөчлөг
config.go                                               ← платформын тохиргоо
usage.go export.go                                      ← metering
observability.go operations.go                          ← ажиглалт, backup, deploy
support.go impersonation.go audit.go                    ← дэмжлэг
handlers.go (27 KB)                                     ← бүгдийн маршрутын хүснэгт
```

Эдгээр нь нэг домэйн биш. Нийтлэг зүйл нь ганц: **оператор л хүрдэг**. Тэр бол
домэйны хил биш, эрхийн хил. Domain-first зарчмаараа (ADR 0001) энэ package
задрах ёстой — задрахад нь `controlplane` гэдэг нэр үлдэх шалтгаангүй болно.

### 1.4 Хүснэгтийн нэрлэлтэд дүрэм алга

66 хүснэгтийн нэрийг харвал гурван өөр загвар зэрэг явж байна:

- Модулиар угтвартай: `esign_*` (6), `urtuu_*` (6), `oauth2_*` (6),
  `store_*` (1), `ai_*` (2), `gov_*` (гарсан), `device_*` (2)
- Эзнээр угтвартай: `platform_*` (3), `operator_*` (4), `tenant_*` (2)
- Угтваргүй: `users`, `sessions`, `roles`, `permissions`, `apps`, `tenants`,
  `devices`, `integrations`, `memberships`, `announcements`, `audit_events`,
  `usage_events`, `pending_approvals`, `credential_grants`, `push_tokens` …

Аль нь угтвар авах ёстойг шийддэг дүрэм байхгүй тул шинэ хүснэгт нь хажуугийнхаа
файлаас хуулагдана — `policy_shape_test.go` дотор яг үүнтэй ижил зүйл RLS
бодлогын хэлбэрт тохиолдсоныг бичсэн байна («хуулах үедээ нээлттэй байсан
хөршөөсөө»). Нэрлэлтэд ч ижил зүйл болж байна.

### 1.5 Аль хүснэгт хэний нь болохыг ялгах ганц дохио бол `tenant_id`

Миграцуудыг скан хийхэд:

- **39 хүснэгт** `tenant_id` баганатай
- **27 хүснэгт** үгүй

Гэвч `tenant_id`-гүй нь автоматаар «платформынх» биш: `esign_batch_items`,
`membership_roles`, `role_permissions`, `installation_events`,
`oauth2_access_tokens`, `report_grants` — эдгээр нь эцэг мөрөөрөө тенантад
харьяалагддаг. Өөрөөр хэлбэл эзэмшлийн дүрэм нь **«`tenant_id` байна уу»** биш,
**«энэ мөр deployment-д ганц удаа оршдог уу, эсвэл тенант бүрд өөр өөр
байдаг уу»**. Энэ хоёрыг ялгаж бичээгүй нь одоогийн бүрхэг байдлын шалтгаан.

### 1.6 «Хилийн ширээ» аль хэдийн олдсон байна — зөвхөн тэгж нэрлээгүй

`db/migrations/policy_shape_test.go` дотор «нарийн хэлбэрийн» бодлоготой
хүснэгтүүдийг жагсаахдаа эхний тав нь **«console, FOR SELECT»** гэсэн тайлбартай:

```
announcements · feature_flag_overrides · operator_impersonations
tenant_quotas · usage_events
```

Эдгээр нь яг л хоёр урсгалын **уулзвар**: платформ бичнэ, тенант уншина.
Тэр тав нь санамсаргүй биш — хоёр урсгалын хил өөрөө өөрийгөө илрүүлсэн хэрэг.

### 1.7 Нэг зүйлийг хоёр үгээр нэрлэсээр байна — гэхдээ дүрэм нь аль хэдийн бий

Тоолж үзвэл:

| | Тоо | Хаана |
| --- | --- | --- |
| `tenant*` идентификатор | **~3 100** | `tenantID` 1331, `tenant` 812, `tenant_id` 502, `TenantID` 218, `tenants` 191 … |
| `organisation` | **588** | үүнээс **500 нь тайлбар мөрөнд**, үлдсэн нь алдааны мессеж ба тестийн текст |
| `Organisation` идентификатор | **13** | |

Репо энэ дүрмийг аль хэдийн баримталж байна, зүгээр л бичээгүй байна:

> **`tenant` = машины үг** (schema, багана, package, context, RLS бодлого).
> **«байгууллага / organisation» = хүний үг** (UI, тайлбар, алдааны мессеж,
> баримт бичиг).

Тиймээс §2-ын нэрлэлт `tenant`-ыг сонгож байна: `org` бол гурав дахь үг нэмнэ
гэсэн үг, яг тэр гурвалжингаас зайлсхийх нь энэ ажлын зорилго.

### 1.8 Гарын үсэг / иргэний баталгаажуулалт таван package-т тархсан

`esign` (3585 мөр), `eid` (530), `eidmongolia` (568), `dan` (194),
`gerege` (548), дээр нь `internal/platform/signing.go`. ADR 0002 «нэг гарын
үсгийн зам» гэж бичсэн ч кодын байршил тэрийг хараахан хэлээгүй байна.

### 1.9 «app» гэдэг үг гурван зүйл заана

- RLS role: `gerege_nexus_app` (`dbguard.AppRole`)
- Суулгадаг модуль: `apps`, `app_installations`, `appGate`, `appInstalled`
- Бинарь: `cmd/api`

Гурвуулаа өөр өөр давхаргад байгаа тул өнөөдөр асуудал үүсгээгүй ч,
`gerege_nexus_app` role-ыг унших хүн эхлээд «аль апп?» гэж бодно.

### 1.10 Жижиг олдвор: тоо таарахгүй байна

`ownership_test.go`-ийн тайлбар «Sixty-nine remain» гэж бичжээ. `platformTables`
map-д **66** бичлэг байна, миграцын скан ч **66** амьд хүснэгт олж байна.
Гурвын зөрүү. Тестийг унших хүн эхлээд тайлбарыг нь итгэдэг тул засах нь зүйтэй.

### 1.11 Одоо байгаа сайн зүйлс — эдгээр дээр л тулгуурлана

Энэ репод шилжилтэд шууд ашиглах гурван механизм аль хэдийн бий:

- `internal/apps/boundaries_test.go` — package хоорондын import-ыг тестээр
  хориглодог загвар. Хоёр урсгалын хилийг **яг үүгээр** барина.
- `db/migrations/ownership_test.go` — хүснэгт бүрийг эзэнтэй нь бүртгүүлдэг.
  Багана нэмэхэд л хангалттай.
- `internal/platform/testdata/routes.txt` + `routes_golden_test.go` +
  `movedTo(...)` — маршрут нүүлгэхэд эргэлт буцалтгүй болгодог арга хэдийнэ бий.

Дээр нь §2.9-д тайлбарласан control/data сахилга бат дөрвөн газарт хэрэгжсэн
байна. Өөрөөр хэлбэл шинэ дэд бүтэц зохиох шаардлагагүй.

---

## 2. Санал: хоёр урсгал, нэг платформ

### 2.1 Хилийн дүрэм — нэг өгүүлбэр

> **Мөр нь deployment-д ганц удаа оршдог бол платформын урсгал. Тенант бүрд
> өөр өөрөө оршдог бол тенантын урсгал.**

Гурав дахь ангилал байхгүй. Хоёуланд нь хэрэгтэй кодыг урсгал биш —
**суурь** гэж нэрлэнэ, тэр нь хүснэгт эзэмшихгүй.

### 2.2 Нэрс

| | Нэр | Утга | Схем | Package |
| --- | --- | --- | --- | --- |
| **Урсгал 1** | `tenant` | Нэг тенантын нэрийн өмнөөс | `tenant` | `internal/tenant/…` |
| **Урсгал 2** | `platform` | Бүх тенантын өмнөөс | `platform` | `internal/platform/…` |
| Суурь | `kernel` | Аль ч урсгалын өмнөөс биш — техник | — | `internal/kernel/…` |

Хоёр нэрийг ч санаатайгаар шинээр зохиогоогүй.

`tenant` нь §1.7-ийн тоогоор аль хэдийн машины үг: `tenant_id` 39 хүснэгтэд,
`tenant_isolation` бодлого, `app.current_tenant` GUC, `nexus.WithTenantID`.
Schema нэр бол машины үг тул шинэ толь бичиг нээх шаардлагагүй.

`platform` нэрийг хэвээр үлдээж байгаа нь мөн санаатай: `platform_settings`,
`platform_backups` хүснэгт аль хэдийн тэр нэртэй, `gerege_nexus_operator` role
тэднийг уншдаг. Өөрчлөлт нь **`internal/platform` жижгэрнэ** — тенантын код
тэндээс гарч `internal/tenant` руу нүүхэд үлдсэн нь үнэхээр платформ болно.
Энэ нь «бүгдийг зэрэг нүүлгэх» биш «нэгийг нь татах» ажил, эрсдэл нь бага.

**`tenant.sessions … WHERE tenant_id = $1` давхардал болох уу?** Болно, гэхдээ
ашигтай давхардал. `tenant` schema-гийн хүснэгт бүр tenant-аар шүүгдэх ёстой —
хоёр үг зэрэг харагдвал шүүлтүүр **байхгүй** query нүдэнд шууд өртөнө. 40
хүснэгтээс 34 нь `tenant_id`-тай, үлдсэн 6 нь эцэг мөрөөрөө өвлөдөг тул тэдэнд
давхардал гарахгүй.

**`platform.tenants` vs schema `tenant`.** Тенантын бүртгэл платформынх,
тенантын өгөгдөл өөрийнх нь schema-д. Ганц тооны schema (`tenant.*` = «энэ
тенантынх»), олон тооны хүснэгт (`platform.tenants` = «платформын мэддэг
тенантууд»). Миграцад нэг өгүүлбэр тайлбар бичихэд хангалттай.

### 2.3 Кодын байршил

```
backend/internal/
│
├── tenant/                   ── Урсгал 1: нэг тенант
│   ├── auth/                    нэвтрэлт, session, lockout
│   ├── access/                  role, permission, grant (access_control.go)
│   ├── profile/                 хүн ба байгууллагын тохиргоо
│   ├── identity/                eid · eidmongolia · dan · binding · sso identity
│   ├── signing/                 esign · signing.go        (ADR 0002-ыг биелүүлнэ)
│   ├── devices/                 device · staffpin · push
│   ├── ssoprovider/             oauth2 provider (тенантын client-ууд)
│   ├── ssoclient/               гадаад IdP руу холбогдох
│   ├── integration/             холбогчид
│   ├── urtuu/                   Өртөө
│   ├── reporting/               тайлан, хуваарь, хуваалцалт
│   ├── ai/                      туслах
│   ├── emailverify/
│   ├── appinstall/              суулгац (appinstaller)
│   ├── directory/ menu/ memo/ quota/
│   └── audit/                   audit_events
│
├── platform/                 ── Урсгал 2: бүх тенант
│   ├── operator/                бүртгэл, session, TOTP, step-up, impersonation
│   ├── tenants/                 үүсгэх · түдгэлзүүлэх · устгах · export · quota
│   ├── approvals/               хоёр хүний зөвшөөрөл
│   ├── settings/                динамик тохиргоо + түүх
│   ├── flags/                   feature flag ба override
│   ├── catalog/                 appcatalog · каталог синк · deprecation
│   ├── metering/                usage_events, CSV, AI квот
│   ├── backup/                  platform_backups, deploy
│   ├── announce/                зарлал
│   ├── support/                 дэмжлэгийн үйлдлүүд
│   ├── identity/                users, user_*_identities, credential_grants
│   ├── keys/                    oauth2_signing_keys
│   ├── observability/           операторын ажиглалтын тойм
│   └── audit/                   operator_audit
│
└── kernel/                   ── Суурь: хүснэгтгүй, урсгалгүй
    ├── httpx/ security/ cache/ config/ resilience/ async/
    ├── dbguard/                 хоёр role-ыг холбодог газар
    └── telemetry/               metric, trace, log — ажиглалтын анхдагч давхарга
```

`pkg/nexus`, `pkg/catalog`, `pkg/urtuu`, `pkg/platform` хэвээр — эдгээр нь өөр
репо дахь модуль уншдаг гэрээ тул хөдөлгөхгүй.

**Гурван анхаарах цэг.**

1. **`internal/platform/tenant` package устана.** Энэ бол `pkg/nexus` руу
   чиглэсэн дамжуулагч, өөрийнх нь тайлбарт ингэж бичсэн байна («These forward
   rather than reimplement»). Хэрэглэгчдийг нь шууд `nexus.WithTenantID`,
   `nexus.TenantID` руу залгавал `internal/tenant/` мод ба `tenant` package
   зэрэг оршихгүй болно. Энэ бол цорын ганц бодит мөргөлдөөн бөгөөд шийдэл нь
   кодыг **хасах** ажил.
2. **`platform/identity` доор `users` байгаа нь** эхлээд сонин санагдана. Гэхдээ
   хүн deployment-д нэг удаа оршиж, `memberships`-ээр тенантад холбогддог —
   §2.1-ийн дүрмээр энэ бол платформ. `sessions` нь эсрэгээр `tenant_id`-тай тул
   `tenant/auth`-д үлдэнэ.
3. **`observability` хоёр хуваагдана**: түүхий metric/trace цуглуулга нь
   `kernel/telemetry`, операторт зориулсан тойм дэлгэц нь
   `platform/observability`.

### 2.4 Өгөгдлийн сан — schema, угтвар биш (гол зөвлөмж)

Угтвар тавьж болно, гэхдээ угтвар бол **зөвхөн нэр**. Postgres schema бол
нэр + **эрх**:

```sql
CREATE SCHEMA tenant;
CREATE SCHEMA platform;

ALTER TABLE public.sessions          SET SCHEMA tenant;
ALTER TABLE public.operator_accounts SET SCHEMA platform;
-- … 66 мөр

-- Schema USAGE нь нэр resolve хийх эрх; хүснэгт нээх эрх биш:
GRANT USAGE ON SCHEMA tenant, platform TO gerege_nexus_tenant;
GRANT USAGE ON SCHEMA platform, tenant TO gerege_nexus_operator;

-- Хилийн таван ширээ, нэрлэсэн байдлаар:
GRANT SELECT ON platform.announcements, platform.feature_flag_overrides,
                platform.operator_impersonations, platform.tenant_quotas,
                platform.usage_events
   TO gerege_nexus_tenant;

-- Default grant зөвхөн tenant schema-д байна. Platform-д ийм grant байхгүй:
ALTER DEFAULT PRIVILEGES IN SCHEMA tenant
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gerege_nexus_tenant;
```

`platform` schema-гийн `USAGE`-ийг тенант role-д өгөх шаардлагатай: дээрх
хилийн таван хүснэгтийн нэрийг resolve хийхгүй бол нэрлэсэн `SELECT` grant ч
ажиллахгүй. `USAGE` өөрөө мөр, хүснэгт нээдэггүй. **Бодит хил нь хүснэгтийн
түвшний grant** — зөвхөн таван хүснэгтийг нэрлэж нээнэ, `operator_audit` болон
дараа нь шинээр үүсэх platform хүснэгтүүд хаалттай үлдэнэ. Үүнийг
`schema_split_test.go` бодит role-оор query хийж болон шинэ probe хүснэгт үүсгэж
батална.

Тиймээс тенантын handler `platform.operator_audit`-аас уншихыг оролдвол
**өгөгдлийн сан татгалзана** — код review хийсэн хүнээс биш. Энэ нь `dbguard`-
ийн одоогийн загвартай яг нийцнэ.

**Role-ын нэр.** Дээрх SQL-д `gerege_nexus_app` → **`gerege_nexus_tenant`**
болов. §1.9-ийн шалтгаанаар: «app» гурван зүйл заадаг, «tenant» нэгийг заана,
мөн `gerege_nexus_operator`-той хосолж уншигдана. Хийхэд хамгийн хямд үе нь
одоо — 00029-ийн бодлогуудыг ямар ч байсан гар дамжуулж байхад. Хүсэхгүй бол
хуучин нэр үлдэж болно, урсгалын хуваалт үүнээс хамаарахгүй.

**Одоо байгаа query бүтэн үлдэх арга** — `search_path`:

```sql
ALTER ROLE gerege_nexus_tenant   SET search_path = tenant, platform;
ALTER ROLE gerege_nexus_operator SET search_path = platform, tenant;
DO $$ BEGIN
  EXECUTE format('ALTER DATABASE %I SET search_path = tenant, platform',
                 current_database());
END $$;
```

`SELECT … FROM sessions` гэсэн query нэг ч тэмдэгт өөрчлөгдөхгүй ажиллана.
Дараа нь тайван, багц багцаар нь `tenant.sessions` болгож бүрэн нэрлэнэ. Өөр
репо дахь модулиуд `nexus.Migrations`-аар өөрсдийн хүснэгтээ үүсгэдэг —
`search_path`-ийн эхэнд `tenant` байгаа тул шинэ хүснэгт нь автоматаар зөв
schema-д унана; тэдэнд ямар ч өөрчлөлт хэрэггүй. Login role нь `SET ROLE NONE`
үед role-ын тохиргоог өвлөхгүй тул database-ийн default замыг мөн тавина.

**Хэрэв угтварыг илүүд үзвэл** (Хувилбар Б): `tn_*` ба `pf_*`. Зардал нь
өндөр — 66 `RENAME`, кодод байгаа бүх SQL мөр, plus нэг хувилбарын турш хуучин
нэрээр view үлдээх — үр дүн нь сул: угтвар зөрчлийг зогсоохгүй, зөвхөн
харагдуулна. Аль алиныг нь хийх шаардлагагүй: schema бол угтварын хийх ёстой
бүхнийг хийгээд, дээрээс нь баталгаа өгнө.

### 2.5 66 хүснэгтийн хуваарилалт

**Платформын урсгал — 26**

| Бүлэг | Хүснэгт |
| --- | --- |
| Оператор | `operator_accounts` `operator_sessions` `operator_audit` `operator_impersonations`\* `pending_approvals` |
| Тенантын бүртгэл | `tenants` `tenant_quotas`\* |
| Тохиргоо | `platform_settings` `platform_settings_history` `platform_backups` `feature_flags` `feature_flag_overrides`\* `announcements`\* |
| Каталог | `apps` `app_versions` `app_dependencies` `store_app_versions` |
| Хэрэглээ | `usage_events`\* |
| Хүн ба хувийн баталгаа | `users` `user_eid_identities` `user_sso_identities` `identity_binding_sessions` `eid_sign_state` `credential_grants` |
| Түлхүүр, толь | `oauth2_signing_keys` `permissions` |

\* = **хилийн ширээ**: платформ бичнэ, тенант уншина. Яг тэр таван нэр
`policy_shape_test.go` дотор «console, FOR SELECT» гэж хэдийнэ тэмдэглэгдсэн.

**Тенантын урсгал — 40**

| Бүлэг | Хүснэгт |
| --- | --- |
| Байгууллага, гишүүнчлэл | `tenant_profiles` `memberships` `membership_roles` `roles` `role_permissions` `access_change_events` |
| Session | `sessions` |
| Суулгац | `app_installations` `installation_events` |
| Холбогч | `integrations` `integration_deliveries` `integration_oauth_states` |
| Гарын үсэг | `esign_batches` `esign_batch_items` `esign_documents` `esign_settings` `esign_sign_sessions` `esign_signature_logs` |
| Төхөөрөмж | `devices` `device_telemetry` `device_enrollment_codes` `push_tokens` `staff_pin_credentials` |
| Өртөө | `urtuu_deliveries` `urtuu_inbox` `urtuu_outbox` `urtuu_peers` `urtuu_peer_codes` `urtuu_request_codes` |
| Тайлан | `report_schedules` `report_grants` |
| Туслах | `ai_knowledge` `ai_prompts` |
| Бусад | `audit_events` `email_verifications` |
| OAuth2 provider | `oauth2_clients` `oauth2_consents` `oauth2_tokens` `oauth2_authorization_codes` `oauth2_access_tokens` |

Хоёр нь маргаантай, тусад нь шийдэх ёстой:

- **`permissions`** — эрхийн нэрсийн толь. Deployment даяар нэг тул платформд
  тавьсан. Гэвч тенантын урсгал үүнийг байнга уншина → уншихын grant өгөх
  эсвэл `kernel`-д гарган толь болгох.
- **`oauth2_*`** — `oauth2_clients` нь `tenant_id`-тай (тенантынх), харин
  `oauth2_signing_keys` нь deployment-ийнх. Хилийг яг тэр хоёрын хооронд
  татсан: түлхүүр платформд, client тенантад. Токен шалгах зам хилээр нэг
  удаа гарна — тэрийг `kernel`-ийн жижиг гэрээгээр (verifier interface) шийднэ.

### 2.6 Маршрут — `cp` тусдаа зүйл байхаа болино

Өнөөдөр `server.go:941` дээр:

```go
r.Route("/cp/api", s.cp.Routes)      // s.cp — controlplane.Service
```

Санал:

```go
r.Route("/api/v1",           s.tenant.Routes)     // тенантын урсгал
r.Route("/api/platform/v1",  s.platform.Routes)   // платформын урсгал
r.Handle("/cp/api/*", movedTo("/api/platform/v1"))  // нэг хувилбар дамжуулна
```

`HostGate` (`CONTROL_PLANE_HOST`) ба nginx-ийн allowlist **хэвээр** — тэр хоёр
нь нэрлэлтийн зүйл биш, аюулгүй байдлын гурван давхаргын хоёр нь
(`docs/CONTROL_PLANE.md` §2). Өөрчлөгдөж байгаа зүйл нь: консол өөрийн гэсэн
онцгой байдалтай «cp» биш, **платформын урсгалын нэг хэрэглэгчийн нүүр** болно.
Тэр урсгалын route table-ыг маргааш ямар ч дотоод хэрэгсэл (CLI,
`operator-bootstrap`, cron) дуудаж болно, консолын route гэж тайлбарлахгүйгээр.

`routes.txt` golden test өөрчлөлтийг мөрөөр нь харуулна; `movedTo` нь хуучин
хаягийг эвдэхгүй. Frontend-ийн `cp` хэсэг нэг хувьсагч солино.

### 2.7 Нэг бинарь, хоёр урсгал — салгах шаардлагагүй, гэхдээ салган **байрлуулж** болно

Хоёр урсгал нэг `cmd/api`, нэг image, нэг deploy хэвээр. Гэхдээ mount хийх
route table нь хоёр болсноор нэг мөр нэмэхэд:

```
NEXUS_PLANES=tenant,platform   # анхдагч — өнөөдрийнхтэй яг ижил
NEXUS_PLANES=tenant            # нийтийн зангилаа: консолын код байхгүй
NEXUS_PLANES=platform          # консолын зангилаа: тенантын route байхгүй
```

Ижил код, ижил image, өөр mount. Хэрэв хожим `cp.nexus.gerege.mn`-ийг тусдаа
pod дээр гаргах шаардлага гарвал энэ нь нэг env хувьсагч болно — repo салгах ч
үгүй, бинарь хоёр болгох ч үгүй. Хэрэггүй бол хэзээ ч бүү асаа.

### 2.8 Хилийг барих тестүүд

Тав. Дөрөв нь одоо байгаа тестийн хуулбар.

1. **Import-ын хил** — `internal/boundaries_test.go`, `apps/boundaries_test.go`-
   ийн яг загвараар:
   `internal/tenant/…` нь `internal/platform/…`-ийг **импортлохгүй**, эсрэгээрээ ч
   мөн адил. Хоёулаа `internal/kernel/…` ба `pkg/…`-ийг импортолж болно.
   Онцгой тохиолдол нь `map` дотор бичигдэнэ — нэмэх нь review-д хэлэлцэх шийдвэр.

2. **Хүснэгтийн эзэмшил** — `ownership_test.go`-д багана нэмнэ:
   `table → (plane, purpose)`. Ангилаагүй хүснэгт тест унагана. Дээрх 66-г
   энэ санал өөрөө бөглөж өгч байна, тул эхний өдрөөс ногоон.

3. **SQL-ийн хил** — `TestPlatformSQLNamesNoAppTable`-ийн эгч дүү:
   `internal/tenant` доторх SQL мөр платформын хүснэгтийн нэрийг (хилийн таваас
   бусдыг) агуулж болохгүй, эсрэгээр ч мөн адил. Одоо байгаа regex-ийг
   хоёр жагсаалтад ажиллуулахад болно.

4. **Grant-ийн шалгалт** — DB тест: `gerege_nexus_tenant` нь `platform` schema-д
   `USAGE` эрхгүй, зөвхөн нэрлэсэн таван хүснэгтэд `SELECT`-тэй байх.
   `dbguard_test.go` доторх загвараар.

5. **Хамаарлын чиглэл** (§2.9) — удирдлагын хүснэгт унших боломжгүй болоход
   хүсэлтийн зам үйлчилсээр байх. `settings`, `flags`-ийн store-ыг хөлдөөж,
   `memo` кэшийг дүүргэсэн байдалд нэвтрэлт ба апп gate ажиллаж байгааг шалгана.

Таван тестийн аль нь ч шинэ хэрэгсэл шаардахгүй.

### 2.9 Хоёр дахь тэнхлэг: control plane / data plane

Control/data гэдэг хуваалт бас бий, гэхдээ тэр нь **tenant/platform-ыг орлохгүй,
түүнийг огтолно**. Дөрвөн нүд нь бүгд дүүрэн:

| | **control** (тохируулна) | **data** (гүйцэтгэнэ) |
| --- | --- | --- |
| **tenant** | `/admin/access/roles`, апп суулгах, холбогч бүртгэх, AI prompt | `/auth/login`, `/urtuu/exchange/push`, esign, `/ai/chat`, telemetry |
| **platform** | тенант үүсгэх, `platform_settings`, flag, каталог синк | `usage_events` бичилт, `oauth2_signing_keys`-ээр токен шалгах, `dbguard`-ийн role binding, kill-switch унших |

**Яагаад control/data дээд түвшний хуваалт биш вэ.** Хилийн ширээний тоо
хариулна:

- `tenant` / `platform` → 66 хүснэгтээс **5** нь хилд унана.
- `control` / `data` → `roles`-ыг control бичиж data уншина. `app_installations`
  мөн. `platform_settings` мөн. `tenants`, `memberships`, `apps`, `permissions`,
  `oauth2_clients` … **бараг бүх** чухал хүснэгт хилд унана.

Учир нь control/data бол хүснэгтүүдийн **хоорондох** хил биш, нэг хүснэгт дээрх
**үйлдлүүдийн** хоорондох хил. Тийм хилийг schema-гаар салгах, grant бичих,
өгөгдлийн сангаар барих боломжгүй — зөвхөн нэрлэж болно.

**Гэхдээ control/data үнэ цэнтэй, бас аль хэдийн хэрэгжсэн.** Дөрвөн газарт:

- `settings/store.go` — *«Reading a setting happens on the request path … so it
  must not be a query»* → 30 сек таймер + Redis invalidation
- `flags` — *«Reads are from memory … refreshed on a timer»*
- `rbac/store.go` — эрхийн шалгалт `memo` кэштэй
- `external_apps.go` — `appGate` нь `app_installations`-ийг кэшилдэг

Тиймээс control/data-г директорын бүтэц биш, **дүрэм 3** болгож бичнэ:

> Өгөгдлийн зам удирдлагын замын өгөгдлийн санд **синхроноор** хамаарахгүй.
> Удирдлагын эзэмшдэг утгыг хүсэлтийн зам зөвхөн санах ойн хуулбараас уншина;
> хуулбар нь таймер + invalidation-аар шинэчлэгдэнэ.

Ашиг нь бодит: консолын ачаалал хэрэглэгчид нөлөөлөхгүй, incident-ийн үед kill
switch удирдлагын DB-гүйгээр ажиллана. Шалгуур нь §2.8-ийн 5 дахь тест.
Дээр нь timeout ба rate limit-ийг plane-ээр нь ялгаж тавих үндэслэл гарна
(`pkg/platform/timeouts_test.go` тэр зүг рүү хэдийнэ хардаг).

**Анхааруулга.** Хэрэв «control plane = операторын консол, data plane = бусад
бүгд» гэсэн утгаар салгавал тэр нь tenant/platform-тай **бараг** ижил, гэхдээ
ирмэг дээрээ буруу: `users`, `oauth2_signing_keys`, каталогийн `apps`,
`usage_events` нь data талд унана, атал тэдгээр нь deployment-д ганц удаа
оршдог — өөрөөр хэлбэл тенантын хил хамгийн эхэлж барих ёстой яг тэр 10 орчим
хүснэгт. Нэр нь буруу зүйл амлана.

---

## 3. Шилжилтийн үе шатууд

Тус бүр нь дангаараа merge хийгдэх, дангаараа буцаагдах ёстой. Дараалал нь
санаатай: эрсдэлгүйгээс эрсдэлтэй рүү, мөн эхний хоёр нь дараагийнхыг
буруу хийхээс хамгаална.

| Үе | Юу | Эрсдэл | Буцаалт |
| --- | --- | --- | --- |
| **A** | Дүрмийг бичих: энэ баримт `docs/`-д, `ownership_test.go`-д `plane` багана нэмж 66-г ангилах, §1.10-ийн тоог засах | Байхгүй — код хөдлөхгүй | git revert |
| **B** | Import-ын хилийн тест нэмэх (одоо унана — түр allowlist-тай) | Байхгүй | git revert |
| **C** | Package нүүлгэлт: `internal/tenant/` үүсгэж тенантын код нүүх. `internal/platform` жижгэрнэ. `controlplane` домэйнээр задарна. `internal/platform/tenant` дамжуулагч устаж `pkg/nexus` шууд дуудагдана. Import-ын allowlist хоосорно | Дунд — зөвхөн нүүлгэлт, логик өөрчлөгдөхгүй | git revert (зөвхөн файл нүүлгэлт) |
| **D** | Schema: `CREATE SCHEMA` + `SET SCHEMA` + `search_path` + grant. Role `gerege_nexus_app` → `gerege_nexus_tenant`. Query өөрчлөгдөхгүй | Дунд — миграц, гэхдээ Down нь бүрэн бичигдэнэ | миграцын Down |
| **E** | Query-г бүрэн нэрлэх (`tenant.sessions`), `search_path`-ийн fallback-ийг хасах | Бага, гэхдээ өргөн | багц бүрээр revert |
| **F** | Маршрутын нэр: `/api/platform/v1` + `movedTo` дамжуулагч. Frontend-ийн base URL | Бага | `movedTo` тул хуучин хаяг ажиллана |
| **G** | Дүрэм 3-ын тест (§2.8-5), plane-ээр ялгасан timeout/rate limit | Бага | git revert |
| **H** | Сонголтоор: `NEXUS_PLANES` mount flag | Бага — анхдагч нь өнөөдрийнх | flag-ыг бүү тавь |

C-ээс цааш алхам бүр өмнөх алхмаар хамгаалагдана: package зөв нүүсэн эсэхийг
B-гийн тест, schema зөв эсэхийг A-гийн ангилал хэлнэ.

**Хамгийн бага ашигтай хэсэг:** A + B. Хоёулаа кодыг хөдөлгөхгүй, гэхдээ энэ
өдрөөс хойш бичигдэх шинэ мөр бүр аль урсгалынх болохоо хэлэх үүрэгтэй болно.
Үлдсэн нь яарах шаардлагагүй.

---

## 4. Юуг **хийхгүй** вэ

- **Репо салгахгүй.** Хоёр урсгал нэг репо, нэг `go.mod`. Апп нь өөр репод
  явдаг загвар (`nexus.Migrations`) хэвээр — тэр бол өөр асуудал, аль хэдийн
  шийдэгдсэн.
- **Хоёр бинарь болгохгүй.** `cmd/api` ганцаараа. §2.7 бол mount-ийн сонголт,
  binary-ийн салалт биш.
- **Микросервис болгохгүй.** Хоёр урсгал ижил процесс, ижил pool, ижил
  transaction-д ороход саадгүй. Хил нь нэр, эрх, тестээр татагдана — сүлжээгээр
  биш.
- **Control/data-г директорын бүтэц болгохгүй.** §2.9 — тэр бол хамаарлын
  чиглэлийн дүрэм, байршлын хуваалт биш.
- **`tenant` → `organisation` бүх нэрийг солихгүй.** §1.7-ийн дүрэм хүчинтэй:
  `tenant` бол техникийн нэр, «байгууллага / organisation» бол хэрэглэгчийн
  нүдэнд харагдах нэр. Хоёрын аль нэгийг устгах нь энэ ажлын хамрах хүрээнээс
  гадуур.
- **Одоо байгаа query-г нэг өдөрт бүгдийг нь дахин бичихгүй.** `search_path`
  байгаагийн шалтгаан яг тэр.

---

## 5. Шийдэх ёстой асуултууд

1. ~~`org` уу, `tenant` уу?~~ — **Шийдэгдсэн: `tenant`.** §1.7-ийн тоо ба
   §2.2-ын үндэслэл.
2. **Schema уу, угтвар уу?** — §2.4. Зөвлөмж: schema.
3. **Role-ын нэрийг солих уу?** — `gerege_nexus_app` → `gerege_nexus_tenant`.
   Зөвлөмж: солих, Үе D дээр. Урсгалын хуваалт үүнээс хамаарахгүй.
4. **`permissions` хаана?** — платформын толь уу, `kernel`-ийн толь уу.
5. **`/cp/api` хаягийг үнэхээр нүүлгэх үү?** — Кодын талыг (Үе C) нүүлгэх нь
   хаягийг нүүлгэхээс хамааралгүй. Хүсвэл `/cp/api` хэвээр үлдэж болно —
   `controlplane` package л алга болно.
