# Claude Code — Хоёр урсгалыг татах (хэрэгжүүлэлтийн prompt-ууд)

[`TWO_PLANES_PROPOSAL.md`](TWO_PLANES_PROPOSAL.md)-ыг хэрэгжүүлэх prompt-ууд.
Үе шат бүр **нэг Claude Code session, нэг branch, нэг PR**. Дараалал нь заавал:
Үе B нь A-гийн ангилалаас, C нь B-гийн тестээс, D нь C-гийн package модноос
хамаарна.

Prompt бүрийг `### PROMPT` мөрнөөс доош бүтнээр нь хуулж Claude Code-д өгнө.

| Үе | Юу | Эрсдэл | Салангид merge |
| --- | --- | --- | --- |
| A | Дүрэм + `ownership_test.go`-д `plane` багана | Байхгүй | ✅ |
| B | Import-ын хилийн тест (allowlist-тай) | Байхгүй | ✅ |
| C | Package нүүлгэлт: `internal/tenant`, `controlplane` задрал | Дунд | ✅ |
| D | Schema `tenant`/`platform`, grant, role-ын нэр | Дунд | ✅ |
| E | Query-г бүрэн нэрлэх, `search_path` fallback хасах | Бага, өргөн | ✅ багцаар |
| F | `/api/platform/v1` + `movedTo` | Бага | ✅ |
| G | Дүрэм 3-ын тест, plane-ээр ялгасан timeout | Бага | ✅ |
| H | `NEXUS_PLANES` mount flag (сонголттой) | Бага | ✅ |

**Бүх prompt-д хамаарах дүрэм** (prompt бүрд давтагдаж орсон):

* Repo хэв маяг: Go 1.26, `pgx` + гар бичсэн SQL, goose миграц, RLS-д найдсан
  тенант тусгаарлалт, `slog`. Frontend: Next.js 16 App Router, TS strict, Tailwind.
* Шалгалт: `cd backend && gofmt -l . && go vet ./... && go test -race ./... &&
  golangci-lint run`; `cd frontend && npx tsc --noEmit && npm run build`.
* Commit нь **монголоор, өнгөрсөн цагаар** ("Тенантын урсгал өөрийн модтой болов").
* Тестийн толгойн тайлбарыг энэ репод байгаа өнгө аясаар бич — `ownership_test.go`,
  `dbguard.go`, `policy_shape_test.go`-ийн толгойг эхлээд уншиж жишээ ав.
  Тайлбар нь **юу хийж байгааг биш, яагаад гэдгийг** хэлдэг.
* `pkg/nexus`-ийн API өөрчлөгдвөл `go test ./pkg/nexus -update` ажиллуулж,
  `api.txt`-ийн diff-ийг PR-ийн тайлбарт үгээр тайлбарлана.
* **Логик өөрчлөхгүй.** А–D үе шат бүр зан төлөвөөр нь тэг өөрчлөлттэй байх
  ёстой. Алдаа олдвол засахгүй — PR-ийн тайлбарт бичиж, тусад нь issue болго.

---

## Үе A — Дүрмийг бичиж, хүснэгтүүдийг ангилах

Код хөдлөхгүй. Хилийг тестийн өгөгдөл болгож буулгана.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/a-classify`.

**Заавал эхлээд унш** (эдгээрийг уншилгүйгээр бүү эхэл):

- `docs/TWO_PLANES_PROPOSAL.md` — бүтнээр нь. Энэ бол даалгаврын эх сурвалж.
- `backend/db/migrations/ownership_test.go` — `platformTables` map ба түүний
  толгойн тайлбар. Чи энэ файлыг өргөтгөнө.
- `backend/db/migrations/policy_shape_test.go` — `narrow` map, ялангуяа
  "console, FOR SELECT" гэсэн таван бичлэг.

**Хийх ажил.**

1. `ownership_test.go` доторх `platformTables map[string]string` -ыг
   `map[string]table` болго. `table` бол хоёр талбартай struct:

   ```go
   type table struct {
       plane   string // "tenant" | "platform"
       purpose string // одоо байгаа тайлбар үг хэвээр
   }
   ```

   66 бичлэг бүрд `plane`-ийг ол. Хуваарилалт нь
   `TWO_PLANES_PROPOSAL.md` §2.5-д бүтнээр бичигдсэн — түүнийг эх сурвалж болго,
   гэхдээ **дагаж хуулахгүй, шалга**: миграцаас `tenant_id` баганатай эсэхийг нь
   уншаад §2.1-ийн дүрэмтэй нийцэж байгааг батал. Зөрөх зүйл гарвал засахгүй,
   PR-ийн тайлбарт бич.

2. `plane` талбар нь `"tenant"` эсвэл `"platform"`-оос өөр утга авбал тест
   унадаг болго. Гурав дахь ангилал байхгүй гэдэг нь дүрэм.

3. Ангилаагүй хүснэгтэд өгдөг алдааны мессежийг шинэчил: одоо "энэ платформынх
   гэж мэдэгд" гэж хэлдэг, цаашид "**аль урсгалынх** гэж мэдэгд" гэж хэлж,
   §2.1-ийн нэг өгүүлбэр дүрмийг мессеж дотроо агуулна.

4. Шинэ тест нэм: **хилийн ширээ**. `policy_shape_test.go`-ийн
   "console, FOR SELECT" таван нэр (`announcements`, `feature_flag_overrides`,
   `operator_impersonations`, `tenant_quotas`, `usage_events`) нь бүгд
   `plane: "platform"` байх ёстой. Хоёр файл хоорондоо зөрвөл тест хэлнэ.
   Энэ бол дараагийн үе шатууд найдах цорын ганц хил тул түүнийг бичгээр
   тогтооно.

5. `ownership_test.go`-ийн толгойн тайлбарт **шинэ хэсэг** нэм: хоёр урсгал юу
   болох, §2.1-ийн дүрэм, хилийн таван ширээ хаанаас гарсан. Одоо байгаа
   тайлбарыг бүү устга — тэр нь 28 хүснэгтийн түүхийг үүрч байгаа.

6. `ownership_test.go`-ийн толгойд "Sixty-nine remain" гэж бичсэн. Бодит тоо
   **66** — `platformTables` мөн 66 бичлэгтэй, миграцын скан ч 66 амьд хүснэгт
   олдог. Тоог зөв болго, мөн яагаад 69 биш болохыг нэг өгүүлбэрээр тайлбарла
   (тоолол хаана алдсаныг өөрөө ол).

**Батламж.**

- `cd backend && go test ./db/migrations/...` ногоон.
- 66 бичлэг тус бүр `plane`-тэй; `platform` 26, `tenant` 40.
- `gofmt -l .` хоосон, `go vet ./...` цэвэр, `golangci-lint run` цэвэр.
- Go файлын аль нь ч өөрчлөгдөөгүй — зөвхөн `db/migrations/*_test.go`.

**PR-ийн тайлбарт:** урсгал тус бүрийн хүснэгтийн тоо, §2.5-аас зөрсөн бичлэг
бүр (байвал) шалтгаантайгаа, "Sixty-nine" тооны алдаа хаанаас гарсан.

---

## Үе B — Import-ын хилийг тестээр тавих

Мод хараахан нүүхгүй. Хилийг эхлээд бичиж, зөрчлийг нь тоолж харуулна.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/b-import-boundary`.
Үе A merge хийгдсэн байх ёстой.

**Заавал эхлээд унш:**

- `backend/internal/apps/boundaries_test.go` — бүтнээр нь. Чи үүний **эгч дүүг**
  бичнэ: ижил арга (`go/build`, `directImports`), ижил өнгө аяс, ижил
  allowlist-ийн философи ("шинэ бичлэг нэмэх нь review-д хэлэлцэх шийдвэр").
- `docs/TWO_PLANES_PROPOSAL.md` §2.3 (мод) ба §2.8 (тестүүд).

**Хийх ажил.**

1. `backend/internal/planes_test.go` (package `internal_test`) үүсгэ. Гурван
   хамаарлын дүрмийг шалгана:

   - `internal/tenant/…` нь `internal/platform/…`-ийг импортлохгүй
   - `internal/platform/…` нь `internal/tenant/…`-ийг импортлохгүй
   - Хоёулаа `internal/kernel/…`, `pkg/…`, гуравдагч талыг импортолж болно;
     `internal/kernel/…` нь хоёрын аль нэгийг ч импортлохгүй

2. Мод хараахан байхгүй тул тест **өнөөдөр хоосон дээр ажиллана**. Гурван мод
   (`internal/tenant`, `internal/kernel`) байхгүй бол тест `t.Skip` хийхгүй —
   "мод хараахан үүсээгүй" гэж `t.Log`-оор хэлээд ногоон өнгөрнө. Үе C-д мод
   үүсэхэд тэр өдрөөс өөрөө ажиллаж эхэлнэ.

3. `plannedTenantPackages` ба `plannedPlatformPackages` гэсэн хоёр жагсаалтыг
   §2.3-ын модоос бич — `internal/platform` доторх одоогийн 36 дэд package ба
   үндсэн директорын 46 файл тус бүр аль урсгалынх болох. Дараа нь тест нэм:
   **`internal/platform` доторх ангилаагүй package/файл байвал уна**. Энэ нь
   Үе C-ийн ажлын жагсаалт өөрөө болно, мөн C-ийн явцад шинэ файл нэмэгдэхэд
   чимээгүй өнгөрөхгүй.

4. Одоогийн зөрчлийг **тоол**. `internal/platform` доторх package бүр нөгөө
   урсгалын package-ыг хэдэн удаа импортолдгийг гаргаж, `t.Log`-оор хэвлэ.
   Энэ тоо нь Үе C-ийн ажлын хэмжээ бөгөөд C дуусахад тэг болох ёстой.
   Тоог PR-ийн тайлбарт бич.

5. `crossPlaneExceptions map[string]map[string]string` — хоосон map, тайлбарын
   хамт. `boundaries_test.go`-ийн `crossAppExceptions`-ийн яг тэр загвараар:
   хоосон байгаа нь санаатай, нэмэх нь шийдвэр.

**Батламж.**

- `cd backend && go test -race ./internal/...` ногоон.
- Тест мод байхгүй үед ногоон, `t.Log` нь юу байхгүйг тодорхой хэлнэ.
- `.github/workflows/ci.yml`-д нэмэх шаардлагагүй — `go test ./...` аль хэдийн
  барина. Хэрэв тусдаа алхам хэрэгтэй гэж үзвэл шалтгааныг PR-т бич.

**PR-ийн тайлбарт:** одоогийн cross-plane import-ын тоо, ангилаагүй үлдсэн
package байвал нэрсээр нь.

---

## Үе C — Модыг нүүлгэх

Хамгийн том алхам. **Зөвхөн нүүлгэлт**: нэг ч логик өөрчлөгдөхгүй.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/c-move`.
Үе A, B merge хийгдсэн байх ёстой.

**Заавал эхлээд унш:**

- `docs/TWO_PLANES_PROPOSAL.md` §2.2, §2.3 — нэр ба мод бүтнээр.
- `backend/internal/planes_test.go` — Үе B-д бичсэн ажлын жагсаалт.
- `backend/internal/platform/server.go` — `Server` struct (124 мөр орчим) ба
  `setupRoutes`. Энэ файл 1321 мөр, чи түүнийг задлана.
- `backend/internal/platform/controlplane/handlers.go` — `Routes` method.
- `backend/internal/platform/tenant/tenant.go` — толгойн тайлбарыг заавал унш.

**Хийх ажил — дараалал нь заавал.**

1. **`internal/kernel/` эхлээд.** `httpx`, `security`, `cache`, `config`,
   `resilience`, `async`, `dbguard`-ыг нүүлгэ. Эдгээр нь хүснэгт эзэмшдэггүй,
   хамаарал нь цөөн, тул эхлээд нүүхэд бусад нь дараа нь дагана.
   `observability`-г **хоёр хуваа**: түүхий metric/trace/log цуглуулга нь
   `kernel/telemetry`, операторт зориулсан тойм дэлгэц нь `platform/observability`.

2. **`internal/platform/tenant` package-ыг устга.** Энэ бол `pkg/nexus` руу
   чиглэсэн дамжуулагч — өөрийнх нь толгойд ингэж бичсэн байна. Түүнийг
   импортолдог **29 файлыг** `pkg/nexus`-ийг шууд дууддаг болго
   (`tenant.WithTenantID` → `nexus.WithTenantID`, `tenant.Require` → түүний
   `nexus` дахь хос). `rls_integration_test.go`, `tenant_test.go` нь тестүүд —
   тэдгээрийг `internal/kernel/dbguard` эсвэл шинэ байрандаа авчир, бүү устга.
   Энэ алхам чухал: үүнгүйгээр `internal/tenant/` мод ба `tenant` package
   зэрэг оршино.

3. **`internal/tenant/` мод үүсгэ** ба §2.3-ын жагсаалтаар нүүлгэ.
   `internal/platform`-ийн үндсэн директорын 46 файлын дийлэнх нь энд ирнэ:
   `auth_handlers.go`, `access_control.go`, `tenant_profile_handlers.go`,
   `device_handlers.go`, `profile_handlers.go`, `identity_*.go`,
   `google_*.go`, `emailverify_handlers.go`, `signing.go`, `access_recovery.go`,
   `access_mode.go` гэх мэт. Файл бүрийг зөв дэд package-т байрлуул — «нэг
   том `tenant` package» биш.

4. **`Server` struct-ыг задал.** Одоо энэ нь хоёр урсгалын бүх хамаарлыг нэг
   дор барьдаг. Хоёр болго: `tenant.Service` ба `platform.Service`, тус бүр нь
   өөрийн pool, өөрийн `Routes(chi.Router)` методтой. Хуваалцдаг зүйлс
   (`*pgxpool.Pool`, `dbguard.Guard`, settings/flags store) нь `kernel`-ээс
   ирж хоёуланд нь дамжина. `cmd/api/main.go` хоёуланг нь угсарна.

5. **`controlplane` package-ыг домэйнээр задал** — §2.3-ын
   `platform/*` жагсаалтаар. 20 файл, 5867 мөр нь `operator`, `tenants`,
   `approvals`, `settings`, `flags`, `catalog`, `metering`, `backup`,
   `announce`, `support`, `observability`, `audit` болж тарна.
   `handlers.go`-ийн `Routes` нь `platform.Service.Routes` болж, дэд
   package бүр өөрийн route-ийн бүлгийг зарлана. `HostGate`, `RequireAudit`,
   `RequireOperator`, `RequireCapability` нь `platform`-ийн дундын middleware
   болж үлдэнэ — тэдний дараалал бол design тул §handlers.go-ийн тайлбарыг
   хуулж авчир.

6. **`internal/planes_test.go`-ийн allowlist хоосор.** Cross-plane import
   үлдвэл засах ёстой нь код, тест биш. Үнэхээр зайлшгүй гарвал
   `crossPlaneExceptions`-д бичээд PR-ийн тайлбарт шалтгааныг тусад нь
   тайлбарла.

**Хатуу хязгаарлалт.**

- **`git mv` ашигла.** Файлын түүх тасрах ёсгүй.
- **Логик тэг өөрчлөлт.** SQL мөр, handler-ийн бие, middleware-ийн дараалал,
  алдааны мессеж — нэг ч тэмдэгт өөрчлөгдөхгүй. Зөвхөн package зарлал, import,
  экспортын нэр өөрчлөгдөнө.
- **Маршрут тэг өөрчлөлт.** `internal/platform/testdata/routes.txt` golden
  файл **байт-ижил** үлдэх ёстой. Тэр файл өөрчлөгдвөл чи нүүлгэлтээс илүү
  зүйл хийсэн байна — буц.
- Файл нүүхэд туслах функц давхардвал `kernel`-д гаргах биш, эхлээд
  **аль урсгалынх болохыг шийд**. Хоёуланд нь үнэхээр хэрэгтэй бол л `kernel`.

**Батламж.**

- `routes.txt` diff хоосон.
- `cd backend && gofmt -l . && go vet ./... && go test -race ./... &&
  golangci-lint run` бүгд цэвэр.
- `internal/planes_test.go` ногоон, `crossPlaneExceptions` хоосон.
- `internal/platform` доор тенантын нэг ч файл үлдээгүй; `internal/platform`-ийн
  үндсэн директорт `.go` файл **үлдсэнгүй** (бүгд дэд package-т).
- `docker compose up` ажиллаж, нэвтрэлт ба консол хоёулаа гарт шалгагдсан.

**PR-ийн тайлбарт:** нүүсэн файлын тоо, `internal/platform`-ийн мөрийн тоо
өмнө/дараа, `server.go` хаана хэдэн файл болж тарсан, cross-plane import
B-гийн тооноос тэг болсныг харуул.

---

## Үе D — Schema, grant, role-ын нэр

Нэрлэлт нь эрх болж хувирна.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/d-schema`.
Үе C merge хийгдсэн байх ёстой.

**Заавал эхлээд унш:**

- `docs/TWO_PLANES_PROPOSAL.md` §2.4, §2.5.
- `backend/db/migrations/00029_tenant_rls.sql` — `gerege_nexus_app` role,
  `tenant_isolation` бодлого хаанаас гарсан.
- `backend/db/migrations/00049_control_plane.sql` — `gerege_nexus_operator`-ийн
  гар бичсэн grant-ийн жагсаалт.
- `backend/internal/kernel/dbguard/dbguard.go` — толгойн тайлбар бүтнээр,
  `AppRole`, `OperatorRole`, `bindStatement`.
- `backend/pkg/nexus/migrations.go` — модуль өөрийн миграцаа хэрхэн бүртгүүлдэг.

**Хийх ажил.**

Нэг миграц: `00079_two_schemas.sql`. Goose Up/Down хоёулаа бүтэн бичигдэнэ.

1. `CREATE SCHEMA tenant; CREATE SCHEMA platform;`

2. 66 хүснэгтийг `ALTER TABLE public.<name> SET SCHEMA <plane>` —
   `ownership_test.go`-ийн `plane` талбарыг эх сурвалж болго. Дараалал нь
   хамаагүй, `SET SCHEMA` нь FK, index, бодлогыг дагуулж авч явна.

3. Role `gerege_nexus_app` → **`gerege_nexus_tenant`**. `ALTER ROLE … RENAME TO`
   нь бодлого дотор бичигдсэн role-ын нэрийг дагуулж шинэчилдэг эсэхийг
   **баталгаажуул** — эргэлзвэл бодлогуудыг дахин үүсгэ. `dbguard.AppRole`
   тогтмолыг `TenantRole` болго, түүний тайлбарт §1.9-ийн шалтгааныг бич
   («app» гурван зүйл заадаг байсан).

4. Grant:

   ```sql
   GRANT  USAGE ON SCHEMA tenant    TO gerege_nexus_tenant;
   REVOKE USAGE ON SCHEMA platform FROM gerege_nexus_tenant;
   GRANT  USAGE ON SCHEMA platform  TO gerege_nexus_operator;
   GRANT  USAGE ON SCHEMA tenant    TO gerege_nexus_operator;  -- 00049-ийн жагсаалт хэвээр
   GRANT SELECT ON platform.announcements, platform.feature_flag_overrides,
                   platform.operator_impersonations, platform.tenant_quotas,
                   platform.usage_events
      TO gerege_nexus_tenant;
   ```

   Хилийн таван ширээ бол §2.5-ын жагсаалт. Тэднээс өөр нэг ч grant
   нэмэхгүй — нэмэх шаардлага гарвал зогсоод PR-ийн тайлбарт асуу.

5. `search_path`:

   ```sql
   ALTER ROLE gerege_nexus_tenant   SET search_path = tenant, platform, public;
   ALTER ROLE gerege_nexus_operator SET search_path = platform, tenant, public;
   ```

   Login role (tenant-гүй зам, `dbguard`-ийн тайлбарын "платформын зам") ч
   мөн адил `search_path` авах ёстой — тэр зам `SET ROLE NONE`-оор ажилладаг
   тул мартвал одоо байгаа бүх query унана. `dbguard.go`-ийн толгойн тайлбарыг
   уншаад аль role-ууд оролцдогийг бүрэн жагсаа.

6. `nexus.Migrations`-аар ирдэг модулийн хүснэгтүүд `tenant` schema-д унах
   ёстой. `search_path`-ийн эхэнд `tenant` байгаа нь үүнийг хийнэ, гэхдээ
   **батал**: `backend/testdata/canary/migrations`-ийг ажиллуулж хүснэгт нь
   хаана үүссэнийг шалгах тест нэм.

7. Шинэ DB тест: `gerege_nexus_tenant` нь `platform` schema-д `USAGE`-гүй,
   зөвхөн нэрлэсэн таван хүснэгтэд `SELECT`-тэй. `dbguard_test.go`-ийн
   `openGuardedPool` загвараар, `TEST_DATABASE_URL` байхгүй бол `t.Skip`.

**Хатуу хязгаарлалт.**

- **Кодын SQL мөр нэг ч өөрчлөгдөхгүй.** `search_path` байгаагийн шалтгаан яг
  тэр. `SELECT … FROM sessions` хэвээр ажиллана. Query нэрлэх нь Үе E.
- Down нь бүрэн буцаана: `SET SCHEMA public`, role-ын нэр буцаана,
  `search_path` арилна.
- `goose_db_version` ба модулийн `goose_db_version_<slug>` хүснэгтүүд хаана
  байхыг **шийдэж, миграцын тайлбарт бич**.

**Батламж.**

- Цэвэр DB дээр `migrate up` → `migrate down` → `migrate up` гурвуулаа ажиллана.
- Бодит өгөгдөлтэй хуулбар дээр `up` ажиллаж, дараа нь бүх тест ногоон:
  `TEST_DATABASE_URL=… go test -race ./...`.
- `internal/tenant`-ийн handler `platform.operator_audit` уншихыг оролдвол
  өгөгдлийн сан татгалзаж байгааг тестээр харуул.
- Тенант тусгаарлалтын одоо байгаа тестүүд (`dbguard_test.go`,
  `tenant/rls_integration_test.go`, `policy_shape_test.go`) бүгд ногоон.

**PR-ийн тайлбарт:** нүүсэн хүснэгтийн тоо schema тус бүрээр, role-ын нэрийн
өөрчлөлт бодлогуудад хэрхэн тусав, `search_path` авсан role-уудын жагсаалт,
canary модулийн хүснэгт хаана буусан.

---

## Үе E — Query-г бүрэн нэрлэх

`search_path`-ийн түшлэгийг авах. Багц багцаар, багц бүр өөрөө merge-тэй.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/e-qualify-<багц>`.
Үе D merge хийгдсэн байх ёстой.

Нэг session-д **нэг дэд package**-ийн SQL-ийг бүрэн нэрлэ: `FROM sessions` →
`FROM tenant.sessions`. Дараалал: `tenant/auth` → `tenant/access` →
`tenant/urtuu` → `tenant/signing` → үлдсэн `tenant/*` → `platform/*`.

Дүрэм:

- Зөвхөн schema угтвар нэмнэ. `WHERE`, `JOIN`, багана, параметрийн дугаар —
  юу ч өөрчлөгдөхгүй.
- Багц бүрийн төгсгөлд тэр package-ийн тестүүд `TEST_DATABASE_URL`-тэй ногоон.
- Сүүлийн багц дуусахад `search_path`-аас `public` fallback-ийг хасах миграц
  бич, түүнийг тусдаа PR болго.
- Модулийн репод (`business-`, `client-gerege-nexus`) байгаа SQL энэ ажилд
  ороогүй — тэдэнд `search_path` үлдэнэ. Үүнийг миграцын тайлбарт бич.

Батламж: `gofmt -l .` хоосон, тестүүд ногоон, зан төлөв тэг өөрчлөлт.

---

## Үе F — Маршрутын нэр

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/f-routes`.

**Заавал унш:** `backend/internal/platform/server.go`-ийн `movedTo` туслах ба
`/api/v1/core/*` дамжуулагчид — яг тэр загварыг дага.
`backend/internal/platform/testdata/routes.txt`, `routes_golden_test.go`.
`docs/CONTROL_PLANE.md` §2 — гурван давхарга.

Хийх:

1. `/cp/api` → `/api/platform/v1`. Хуучин зам `movedTo`-оор нэг хувилбар амьд.
2. `HostGate` (`CONTROL_PLANE_HOST`) ба nginx-ийн allowlist **хэвээр**. Тэр
   хоёр нь аюулгүй байдлын давхарга, нэрлэлтийн зүйл биш. Хэрэв чи тэдгээрийг
   өөрчлөх шаардлагатай гэж үзвэл зогсоод асуу.
3. `frontend/lib/api.ts` (эсвэл консолын base URL хаана байна тэнд) нэг
   хувьсагч сольж, бусад газарт хатуу бичсэн `/cp/api` үлдээгүйг батал.
4. `routes.txt`-ийг `-update`-ээр шинэчилж, diff-ийг PR-т **үгээр** тайлбарла.
5. `docs/CONTROL_PLANE.md`, `docs/RUNBOOKS.md`, `deploy/nginx/*.conf`-д
   `/cp/api` дурдсан газруудыг шинэчил.

Батламж: хуучин болон шинэ хоёр зам хоёулаа ажиллаж байгааг гарт шалга;
`npx tsc --noEmit && npm run build` цэвэр.

---

## Үе G — Дүрэм 3: хамаарлын чиглэл

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/g-dependency-rule`.

**Заавал унш:** `docs/TWO_PLANES_PROPOSAL.md` §2.9.
`backend/internal/platform/settings/store.go` толгой,
`backend/internal/platform/flags/` толгой,
`backend/internal/tenant/access/…`-ийн `memo` кэш (хуучин `rbac/store.go`),
`external_apps.go`-ийн `appGate`. Эдгээр дөрөв нь дүрмийн **аль хэдийн байгаа**
хэрэгжилт — шинээр зохиох биш, тэднийг батлах тест бичих ажил.

Хийх:

1. Тест: удирдлагын хүснэгтийг унших боломжгүй болгосон үед хүсэлтийн зам
   үйлчилсээр байх. `settings`, `flags` store-ыг дүүргэсэн байдалд нь
   хөлдөөж, тэдний DB query-г алдаа буцаадаг болгоод — нэвтрэлт, апп gate,
   эрхийн шалгалт гурав ажиллаж байгааг батал.
2. Дүрмийг зөрчиж байгаа зам үлдсэн эсэхийг ол: хүсэлтийн зам дээр
   удирдлагын хүснэгтийг **шууд** query хийдэг газар. Олдвол засахгүй —
   жагсаагаад PR-ийн тайлбарт бич, тус бүрд issue үүсгэ.
3. `pkg/platform/timeouts_test.go`-ийг уншаад timeout ба rate limit-ийг
   plane-ээр ялгаж тавих санал бич — хэрэгжүүлэх нь энэ PR-ийн ажил биш.

---

## Үе H — `NEXUS_PLANES` (сонголттой)

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b planes/h-mount-flag`.

`cmd/api/main.go` ба `platform.Run` дээр `NEXUS_PLANES` env хувьсагч нэм:
`tenant,platform` (анхдагч, өнөөдрийнхтэй яг ижил), `tenant`, `platform`.
Зөвхөн mount хийх route table-ыг сонгоно — бинарь салахгүй, код хуваагдахгүй.

Дүрэм:

- Анхдагч зан төлөв өөрчлөгдвөл PR буруу.
- Утга танигдахгүй бол асалт дээр **унана**, чимээгүй анхдагчид буцахгүй.
- `NEXUS_PLANES=platform` үед `/health`, `/ready`, `/metrics` ажилласаар байна.
- `routes.txt` golden нь анхдагч тохиргоог л бичнэ; нөгөө хоёрыг тусдаа
  тестээр шалга.
- `docs/CONTROL_PLANE.md`-д хэзээ хэрэглэхийг бич, мөн **хэрэггүй бол бүү асаа**
  гэдгийг тодорхой хэл.
