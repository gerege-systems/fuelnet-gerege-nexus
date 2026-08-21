# Claude Code — Цөмийн хилийг татах (хэрэгжүүлэлтийн prompt-ууд)

[`CORE_BOUNDARY_PLAN.md`](CORE_BOUNDARY_PLAN.md)-ыг хэрэгжүүлэх prompt-ууд.
Үе шат бүр **нэг Claude Code session, нэг branch, нэг PR**. Дараалал нь заавал —
Үе 1 нь Үе 0-ийн маршрутын golden файлаас хамаарна, Үе 4 нь Үе 1-ийн чадварын
бүртгэлээс хамаарна.

Prompt бүрийг `## PROMPT` мөрнөөс доош бүтнээр нь хуулж Claude Code-д өгнө.

**Арваулаа 2026-08-21-нд хэрэгжсэн.** Prompt-ууд нь түүх болж үлдэв: доорх
бичвэр нь юуг даалгасныг, дараах хүснэгт нь юу гарсныг хэлнэ. Хоёр нь
хоорондоо зөрсөн газар бүр PR-ийн тайлбарт хэмжилттэйгээ бичигдсэн.

| Үе | PR | Юу гарсан | Төлөв |
| --- | --- | --- | --- |
| 0 | [#170](https://github.com/gerege-systems/open-gerege-nexus/pull/170) | Маршрутын golden (341 мөр), CODEOWNERS, хоосон ногоон CI алхам | ✅ merged |
| 1 | [#171](https://github.com/gerege-systems/open-gerege-nexus/pull/171) | `Provide[T]`/`Capability[T]`, `Bootstrap` 9→1, `Meetings()` | ✅ merged |
| 2a | [#172](https://github.com/gerege-systems/open-gerege-nexus/pull/172) | `lib/api.ts` 1831→80 мөр, 21 файл, 185 метод хэвээр | ✅ merged |
| 2b | [#173](https://github.com/gerege-systems/open-gerege-nexus/pull/173) | i18n runtime бүртгэл, 11,368 мөр байт-ижил | ✅ merged |
| 2c | [#174](https://github.com/gerege-systems/open-gerege-nexus/pull/174) | `Layout.tsx`-ийн гурван хатуу жагсаалт | ✅ merged |
| 3 | [#175](https://github.com/gerege-systems/open-gerege-nexus/pull/175) | `nexus.Migrations`, гурван хамгаалалт, миграц 00073 | ✅ merged |
| 4a | [#177](https://github.com/gerege-systems/open-gerege-nexus/pull/177) | AI "0" гэж хэлэхээ болив | ✅ merged |
| 4b | [#176](https://github.com/gerege-systems/open-gerege-nexus/pull/176) | `DefaultRoles`, `gov.*` устав, миграц 00074 | ✅ merged |
| 4c | [#178](https://github.com/gerege-systems/open-gerege-nexus/pull/178) | Өртөө **гараагүй** — [ADR 0004](adr/0004-a-pilot-that-did-not-ship.md) | ⚠️ шийдвэр |
| 5 | [#179](https://github.com/gerege-systems/open-gerege-nexus/pull/179) | Downstream ажил, хоёр canary, RELEASING | ✅ merged |

Хэрэгжүүлэлтийн үеэр prompt-оос зөрсөн хоёр зүйл, тус бүр хэмжилттэй:

* **Үе 2c** — lucide-ийн `DynamicIcon`-ыг хэрэгжүүлж, `.next/static` 3.5 MB /
  95 chunk-аас **11 MB / 1828 chunk** болохыг хэмжиж хассан. Оронд нь
  build-time generated map.
* **Үе 4c** — Өртөө гараагүй. Рельс нь 2,900 мөр `internal/` код бөгөөд
  `lifecycle_test.go` түүнийг шаарддаг; апп рельсний дөрвөн хүснэгтийг шууд
  SQL-ээр уншдаг; миграц салдаггүй. Гурван саадын дараалал ADR 0004-т.

Мөн §9-ийн хэмжүүр буруу цонх дээр ажилладаг байсныг залруулав —
[`CORE_BOUNDARY_PLAN.md` §9](CORE_BOUNDARY_PLAN.md#9-амжилтын-хэмжүүр).

**Бүх prompt-д хамаарах дүрэм** (Claude Code-д давтагдаж орсон байгаа):

* Repo хэв маяг: Go 1.26, `pgx` + гар бичсэн SQL, goose миграц, RLS-д найдсан
  тенант тусгаарлалт, `slog`. Frontend: Next.js 16 App Router, TS strict, Tailwind.
* Шалгалт: `cd backend && gofmt -l . && go vet ./... && go test -race ./... &&
  golangci-lint run`; `cd frontend && npx tsc --noEmit && npm run build`.
* `CONTRIBUTING.md` нь Conventional Commits гэж бичсэн ч **бодит түүх нь
  монголоор, өнгөрсөн цагаар** ("Цэс гадна дарахад хаагддаг болов"). Бодит
  хэвшлийг дага.
* `pkg/nexus`-ийн API өөрчлөгдвөл `go test ./pkg/nexus -update` ажиллуулж,
  `api.txt`-ийн diff-ийг PR-ийн тайлбарт **юу өөрчлөгдсөнийг үгээр** тайлбарлана.

---

## Үе 0 — Хэмжиж, хөлдөөх

Юу ч зөөхгүй. Одоогийн байдлыг тестээр бичиж, чимээгүй өсөхийг зогсооно.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b core/phase-0-freeze`.

Эхлээд контекстоо бүрдүүл:

- `backend/internal/platform/server.go` — `setupRoutes` (751–1113 мөр, ~98
  маршрут), `registerAppModuleRoutes` (1113–1118).
- `backend/pkg/nexus/api_golden_test.go` — golden файлын хэв маяг: AST-аас
  рендэрлэж, `-update` флагтай, тест бүтэлгүйтэхдээ юу хийхийг зааж өгдөг.
  Энэ файлын **толгойн тайлбарыг заавал уншиж, ижил өнгө аясаар бич**.
- `backend/internal/platform/route_policy_test.go` — `publicRoutes`, өөрөөр
  хэлбэл маршрутыг шалгах аль хэдийн байгаа арга.
- `.github/workflows/ci.yml:157–232` — "skip хийгдвэл улаан болгох" алхмууд.
- `docs/CORE_BOUNDARY_PLAN.md` §2, §3.4(C).

Зорилго: цөмд маршрут, файл чимээгүй нэмэгдэхийг **review-д харагддаг diff**
болгох. Кодын зан төлөв нэг ч байтаар өөрчлөгдөхгүй.

**Дөрвөн ажил, дөрвөн тусдаа commit.**

#### Ажил 1 — CI-ийн хоосон ногоон засах

`.github/workflows/ci.yml:163` нь `./internal/apps/gov_services/...` дээр
`TestLocalFulfilmentHappyPath`-ыг ажиллуулдаг. Тэр пакет ч, тэр тест ч
байхгүй — gov-services нь `gerege-gov` руу явсан. Алхам нь "тест дуугүй skip
хийгдэхээс хамгаалах" зорилготой байсан бөгөөд одоо өөрөө юу ч шалгахгүйгээр
ногоон болж байна.

Алхмыг устга. Устгасан шалтгааныг ci.yml дотор нэг мөр тайлбараар үлдээ:
энэ хэв маягийн алхам нь пакет явахад дагаж явах ёстой гэдгийг дараагийн хүн
мэдэх ёстой. Бусад ижил алхмуудыг (`documents`, `emailverify`, `ssoprovider`,
`reports`, route policy, lockout, tenant isolation, reporting isolation, report
sharing, urtuu) шалгаж, тэдгээрийн заасан пакет ба тест **бодитоор байгаа
эсэхийг** нэг бүрчлэн батал. Байхгүй нь олдвол устга.

#### Ажил 2 — Маршрутын golden файл

`backend/internal/platform/testdata/routes.txt` үүсгэ. Агуулга нь энэ бинарийн
мөнтөж буй **бүх** HTTP маршрут: метод + зам, эрэмбэлэгдсэн, мөр тутамд нэг.

`chi.Walk` ашиглан `NewServer`-ийн буцаасан router-ыг тойрч гарган рендэрлэ.
Тест нь `backend/internal/platform/routes_golden_test.go`:

- `TestTheRouteTableIsTheOneOnRecord` — golden-той тулгана.
- `-update` флагтай: `go test ./internal/platform -run TestTheRouteTable -update`.
- Тест бүтэлгүйтэхдээ `api_golden_test.go`-той **ижил өнгө аясаар** тайлбарла:
  цөмд маршрут нэмэх нь зөвшөөрөгдсөн бөгөөд ихэвчлэн зөв, гэхдээ санамсаргүй
  болох ёсгүй, diff нь платформын гадаргуу юу авсан/алдсаныг review-д хэлнэ.
- Аппын модулиудын маршрут (`registerAppModuleRoutes`) golden-д **орно** —
  гэхдээ `[app] ` угтвартайгаар тэмдэглэ, ингэснээр цөмийн маршрут ба аппын
  маршрутын diff хоёр өөр зүйл гэдэг нь харагдана.
- Тест нь өгөгдлийн сан шаардвал `route_policy_test.go`-ийн skip хэв маягийг
  дага, мөн ci.yml-д "skip хийгдвэл улаан" алхам нэм (Ажил 1-ийн дүрмээр).

#### Ажил 3 — CODEOWNERS

`.github/CODEOWNERS` үүсгэ (эсвэл байвал нэм). Цөмийн багийн review шаардах
файлууд:

```
backend/internal/platform/server.go
backend/internal/apps/runtime.go
backend/pkg/nexus/
backend/pkg/catalog/
backend/pkg/urtuu/
backend/pkg/platform/
backend/internal/platform/appinstaller/
backend/db/migrations/
frontend/lib/api.ts
frontend/lib/i18n/index.tsx
frontend/components/Layout.tsx
catalog/
```

Эзэн нь `@gerege-systems/core`. Байхгүй бол `@gerege-systems` ашиглаад
файлын толгойд тэмдэглэл үлдээ.

#### Ажил 4 — Баримт

`docs/CORE_BOUNDARY_PLAN.md`-ыг `docs/README.md`-ийн индекст нэм (файлын
жагсаалтын зөв хэсэгт, бусад бичлэгийн хэв маягаар). `README.md`-ийн
"Баримт бичгийн индекс" хэсэгт мөн нэм.

#### Хүлээн авах шалгуур

- `go test -race ./...` ногоон; `routes.txt` нь одоогийн маршрутын жагсаалттай
  яг таарна.
- `routes.txt`-д гараар нэг мөр нэмээд тест **улаан** болохыг батал, дараа нь
  буцаа.
- ci.yml-ийн бүх "skip хийгдвэл улаан" алхмын пакет/тест бодитоор оршдог.
- Зан төлөвийн өөрчлөлт тэг: `git diff` дотор `backend/internal/platform/`-ын
  бус-тест файлд өөрчлөлт байхгүй.

#### PR

Гарчиг: `Цөмийн маршрутын хүснэгт бичигдсэн болов`

Тайлбарт: (1) `routes.txt` юуг барихыг, (2) `gov_services` алхам яагаад
хоосон ногоон байсныг, (3) CODEOWNERS-д орсон файлууд бол `CORE_BOUNDARY_PLAN.md`
§2.1-ийн хамгийн их хөдөлдөг файлууд гэдгийг бич.

---

## Үе 1 — Чадварын бүртгэл

Хамгийн том backend ялалт. Шинэ чадвар нэмэх нь SDK-гийн засвар байхаа болино.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b core/phase-1-capabilities`.
Үе 0 (`core/phase-0-freeze`) merge хийгдсэн байх ёстой.

Эхлээд контекстоо бүрдүүл:

- `backend/pkg/nexus/link.go:122` (`UseLink`), `:133` (`Ring`), ба ялангуяа
  **`:130-132`-ын тайлбар**: чадвар нь бүтээгдэх үед биш, **хэрэглэх бүрд**
  асуугддаг. Энэ шинж чанарыг эвдэхгүй.
- `backend/pkg/nexus/documents.go:112` (`UseDocumentFiler`), `:124` (`Documents`).
- `backend/pkg/nexus/service.go:301` (`AuditSink` төрөл), `:311` (`UseAuditSink`),
  `:319` (`Audit`) — sink суугаагүй үед **алдагдана** (`AUDIT_EVENT_UNSUNK`).
- `backend/pkg/nexus/report.go:265-297` — `RegisterReport` нь sink суугаагүй
  үед **буферлэдэг**. Audit-аас өөр, зориуд өөр. Хоёуланг нь хэвээр хадгал.
- `backend/pkg/nexus/meetings.go:35` (`MeetingBooker`) ба
  `backend/internal/platform/integration/meetings_adapter.go:27` — interface
  бий, адаптер бий, **авах арга байхгүй**. Энэ бол хагас дуусаагүй чадвар.
- `backend/internal/apps/runtime.go:38-77` — `Bootstrap`, 9 позицийн параметр.
- `backend/internal/platform/server.go:184-471` (`NewServer`), ялангуяа `:223`
  (`nexus.UseLink`), `:225` (`newModulePlatform`), `:229-233` (extra modules),
  `:235-256` (`apps.Bootstrap` дуудалт).
- `backend/pkg/platform/run.go:181,187` — `UseAuditSink`, `UseReportSink` нь
  `NewServer`-ээс **өмнө** суудаг.
- `backend/pkg/nexus/api_golden_test.go` ба `testdata/api.txt`.
- `docs/CORE_BOUNDARY_PLAN.md` §3.1, §5 Үе 1.
- `docs/RELEASING.md` — deprecation журам.

Зорилго: чадвар нэмэх нь ямар ч гарын үсэг эвдэхгүй болгох, **distribution
өөрөө чадвар нийтэлж чаддаг болгох**.

**Зургаан ажил, зургаан тусдаа commit.** Ажил 1 нь бусдын урьдчилсан нөхцөл.

#### Ажил 1 — `pkg/nexus/capability.go`

Generics дээр суурилсан нэг бүртгэл:

```go
func Provide[T any](impl T)
func Capability[T any]() (T, error)
```

Шаардлага:

- `reflect.TypeFor[T]()`-оор түлхүүрлэсэн `map`, `sync.RWMutex`-тэй.
  Бүртгэл нь `registry.go`-той ижил зориудын глобал төлөв — тэр файлын
  `:14-28`-ын тайлбар яагаад глобал болохыг тайлбарласан, ижил үндэслэлийг
  давт, шинээр зохиохгүй.
- `Capability[T]()` нь **дуудагдах бүрд** хайна, буцаасан утгаа кэшлэхгүй.
  Модуль платформоос өмнө бүтээгдэж болно (`link.go:130-132`).
- Байхгүй үед буцаах алдаа нь **аль төрөл дутсаныг нэрлэнэ**:
  `nexus: this deployment provides no <T>`. `ErrNoLink`, `ErrNoDocumentFiler`
  шиг тодорхой байх ёстой.
- Сүүлд өгсөн нь ялна (`registry.go`-ийн `Register`-тэй ижил дүрэм), гэхдээ
  дарж бичихэд `slog.Warn` бичнэ — хоёр distribution нэг чадварыг өгөх нь
  зориудын байж болно, санамсаргүй ч байж болно.
- Файлын толгойн тайлбарт: **энэ бүртгэл яагаад байгааг** бич. Гол
  үндэслэл нь `Bootstrap` 11 хоногт 4→9 параметр болсон, `Use*` хос дөрөв
  удаа гараар давтагдсан, `MeetingBooker` нь interface-тэй байж авах аргагүй
  үлдсэн явдал. Тоог нь бич — маргаан биш, хэмжилт байх ёстой.

Тест `capability_test.go`: өгөх/авах, байхгүй үеийн алдаа, дарж бичих, зэрэг
хандалт (`-race`).

#### Ажил 2 — Дөрвөн `Use*`-ыг шилжүүл

`UseLink`, `UseDocumentFiler`, `UseAuditSink`, `UseReportSink` нь одоо
`Provide[T]`-ийн нимгэн бүрхүүл болно. `Ring()`, `Documents()` нь
`Capability[T]()`-ийн бүрхүүл.

- Хуучин нэрсийг **устгахгүй**. `// Deprecated:` тайлбартай, `RELEASING.md`-ийн
  журмаар нэг major мөчлөг үлдээнэ. Аль хувилбарт устахыг тайлбарт нэрлэ.
- Зан төлөв яг хэвээр: audit нь sink-гүй үед алдагдана, report нь буферлэнэ.
  `report.go:265-297`-ын буферлэх логик `Provide[ReportSink]` дуудагдах үед
  ажиллах ёстой — буфер `UseReportSink` дотор биш, `Provide`-ийн дараа
  ажилладаг байх ёстой. Хэрэв энэ нь ерөнхий `Provide`-д тусгай тохиолдол
  шаардаж байвал **бүү тусгай тохиолдол хий**: оронд нь `ReportSink` төрөлд
  `Provide`-ыг дуудсаны дараа буферыг цутгах жижиг hook-ыг `report.go`
  дотор үлдээ (`init`-д биш, `Provide`-ийн callback-аар).

#### Ажил 3 — `MeetingBooker`-ыг дуусга

`server.go`-д `integration.NewMeetingsAdapter(...)` бүтээгдэх газарт
`nexus.Provide[nexus.MeetingBooker](adapter)` нэм. `pkg/nexus/meetings.go`-д
`Meetings() (MeetingBooker, error)` нэм (`Ring()`-тэй ижил хэлбэрээр).
`meetings.go:32-34`-ын тайлбарыг шинэчил: тэнд "хамаарлын төрөл нь хамаарал
явсан газартаа хүрдэг" гэсэн сургамж бий; одоо түүн дээр "interface зарлах нь
хагас ажил, авах арга байхгүй бол хэн ч хэрэглэхгүй" гэдгийг нэм.

#### Ажил 4 — `Bootstrap`-ыг нэг параметр болго

`internal/apps/runtime.go`:

```go
func Bootstrap(p nexus.Platform) Runtime
```

`integrations`, `eidMN`, `sso`, `xyp`, `rails`, `link`, `signer`,
`installedApps` — найман хамаарлыг бүртгэлээс авна. Тус бүрд `server.go`-д
`Provide` дуудалт нэм, **`apps.Bootstrap` дуудагдахаас өмнө**.

Анхаар:

- Бүтээх дараалал `runtime.go:44-77`-д тайлбарлагдсан бөгөөд ачаа үүрсэн:
  `organisation` эхэнд, `reports` **хамгийн сүүлд** (тайлан бүртгэсэн модулиуд
  дараа нь орж чадахгүй). Дараалал хэвээр.
- `esignModule` нь өөрөө бүртгүүлдэггүй, `documents.New`-д дамжуулагддаг. Энэ
  нь repo доторх нарийн ширийн — бүртгэл рүү гаргах шаардлагагүй, `Bootstrap`
  дотор хэвээр үлдээ.
- `installedApps` нь `server` заагч дээрх closure (`server.go:254-256`).
  `Provide` хийхдээ closure-ийн шинж чанараа алдахгүй байхыг батал — нэрлэсэн
  төрөл (`type InstalledApps func(...)`) хэрэгтэй болно.
- `sso_clients.New(sso)` нь `nexus.Platform` авдаггүй цорын ганц модуль. Энэ
  ажлын хүрээнд өөрчлөхгүй, зөвхөн тэмдэглэ.

#### Ажил 5 — Хамгаалалтын тест

`internal/apps/runtime_test.go`:

```go
// TestBootstrapTakesOnlyThePlatform
```

`go/ast`-аар `runtime.go`-ыг парс хийж `Bootstrap`-ийн параметрийн тоог тоол.
1-ээс их бол улаан. Тестийн тайлбарт: энэ гарын үсэг 2026-08-09-нөөс 08-20-ны
хооронд 4-өөс 9 болж зургаан удаа эвдэрсэн, тэр нь чимээгүй болсон, учир нь
`internal/`-д байгаа тул distribution-д харагддаггүй. Одооноос эхлэн энэ нь
шийдвэр байх ёстой.

#### Ажил 6 — API-г дахин бичих

`go test ./pkg/nexus -update`. `api.txt`-ийн diff-ийг сайтар хар: `Provide`,
`Capability`, `Meetings` нэмэгдсэн, `Deprecated` дөрөв хэвээр байх ёстой.
Устсан юм байвал алдаа — deprecation мөчлөг дуусаагүй.

#### Хүлээн авах шалгуур

- `go test -race ./...`, `go vet ./...`, `golangci-lint run` ногоон.
- `routes.txt` **өөрчлөгдөөгүй** (Үе 0-ийн golden). Энэ рефактор ямар ч
  маршрутыг хөдөлгөх ёсгүй — хөдөлсөн бол ямар нэг зүйл буруу.
- `Bootstrap` нэг параметртэй, шинэ тест ногоон.
- `nexus.Meetings()` нь ажиллаж буй суулгац дээр адаптер буцаана.
- `grep -rn "func Use" backend/pkg/nexus/` нь дөрвөн Deprecated-ыг л харуулна.

#### PR

Гарчиг: `Чадвар нэмэх нь SDK-гийн засвар байхаа болив`

Тайлбарт: (1) `api.txt`-ийн diff юу гэсэн үг болохыг үгээр, (2) `Bootstrap`-ийн
4→9 түүх ба одоо 1 болсныг, (3) distribution одооноос **чадвар өгч ч чадна**
гэдгийг — жишээ кодтой (`nexus.Provide[mydist.Pricing](impl)`), (4) deprecated
дөрөв хэзээ устахыг бич.

---

## Үе 2a — `lib/api.ts` задлах

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b frontend/phase-2a-api-split`.

Контекст:

- `frontend/lib/api.ts` — 1831 мөр, ~41 метод, бүх аппын endpoint нэг файлд.
- Замын угтваруудыг өөрөө тоол: `grep -oE '"/[a-z0-9_-]+' frontend/lib/api.ts |
  sort | uniq -c | sort -rn`.
- `frontend/lib/gov.ts`, `esign.ts`, `storefront.ts`, `cp.ts` — хажуугийн
  клиентүүд, аль хэдийн салсан хэсгийн жишээ.
- `docs/ECOSYSTEM_GIT_STRATEGY.md` §2.3 — бүрхүүл цөмд үлдэх шийдвэр. **Энэ
  ажил тэр шийдвэрийг эвдэхгүй**: дэлгэцүүд байрандаа үлдэнэ, зөвхөн клиент
  хуваагдана.
- `docs/CORE_BOUNDARY_PLAN.md` §5 Үе 2a.

Зорилго: цөмийн API клиентээс **аппын endpoint-ыг гаргах**, ингэснээр аппын
ажил цөмийн файлд хүрэхээ болино.

#### Ажил 1 — Хуваалт

`frontend/lib/api/client.ts` — зөвхөн цөм:

- `request()` суурь (auth header, CSRF, алдааны боловсруулалт, tenant header);
- `/auth/*`, `/profile/*`, `/tenant/*`, `/menus`, `/store/*`, `/installed-apps`,
  `/admin/access/*`, `/admin/devices/*`, `/admin/store/*`, `/integrations/*`,
  `/verify/*`, `/push-tokens`, `/oauth2/*`.

`frontend/lib/api/<app>.ts` — апп тус бүр, `client.ts`-ийн `request()`-ыг
импортлоно: `documents`, `esign`, `egov`, `reports`, `urtuu`,
`organisation`, `sso-clients`, `ai`.

**Явсан аппуудынх** тусдаа: `contacts`, `products`, `inventory`, `billing`
(commerce), `pos`, `publisher`, `store-review`, `appstore-registry`,
`devices/shifts`. Эдгээрийг `frontend/lib/api/_departed/` хавтсанд төвлөрүүл
ба хавтасны `README.md`-д аль repo-д харьяалагдахыг нь бич. Одоо устгахгүй —
дэлгэцүүд нь ажилласаар байна (§2.3). Үе 2d-д хамт явна.

**Анхаар — эдгээр нь явсангүй:** `/store/apps`, `/admin/store/*` нь
**цөмийнх** (суулгацын дэлгүүрийн дэлгэц, `server.go:1069-1099`). `/esign/*`
мөн цөмийнх — esign нь 00058-аар `documents` руу нэгдсэн, `internal/platform/esign`
амьд. Эдгээрийг `_departed`-д битгий оруул.

#### Ажил 2 — Нийцтэй байдал

`frontend/lib/api.ts` нь re-export барьцлагч болж үлдэнэ (`export * from
'./api/client'` гэх мэт), ингэснээр 81 хуудасны import нэг ч өөрчлөгдөхгүй.
Дараагийн PR-д хуудсуудыг шууд импорт руу шилжүүлнэ — энэ PR-д биш.

#### Ажил 3 — Хамгаалалтын тест

`frontend/lib/api/client.test.ts` (эсвэл repo-д тест хэрэгсэл байхгүй бол
`frontend/scripts/check-api-boundaries.mjs` + `npm run` скрипт + ci.yml алхам):

`client.ts`-ийн эх кодыг уншиж бүх мөрөн доторх `"/..."` замыг гаргаж авна.
Дээрх цөмийн угтваруудын allowlist-д байхгүй зам олдвол унана. Алдааны
мессежэд: аппын endpoint нь аппынхаа клиентэд байх ёстой, учир нь цөмийн
клиент бол distribution бүрийн импортолдог файл.

#### Хүлээн авах шалгуур

- `npx tsc --noEmit` ба `npm run build` ногоон.
- `frontend/app/**`-д import өөрчлөлт **тэг**.
- `client.ts` 400 мөрөөс богино.
- allowlist тестийг гараар эвдээд улаан болохыг батал.

#### PR

Гарчиг: `API клиент цөм ба аппаараа хуваагдав`

Тайлбарт: `CORE_BOUNDARY_PLAN.md` §2.1-ийн тоог иш тат — `lib/api.ts` нь
аппын ажилтай хамт **40 удаа** өөрчлөгдсөн, жагсаалтын тэргүүнд байсан.

---

## Үе 2b — i18n-ийг runtime бүртгэл болгох

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b frontend/phase-2b-i18n-runtime`.

Контекст:

- `frontend/lib/i18n/index.tsx:9-33` (import-ууд), `:89-117` (spread),
  `:119` (`TranslationKey = keyof typeof dictionary`).
- `frontend/lib/i18n/addons/` — 25 файл. `frontend/lib/i18n/locales/` —
  6 файл, тус бүр ~1380 мөр.
- `frontend/lib/i18n/locales/index.ts:16` — дутуу түлхүүр англи руу уначихдаг.
- `docs/TRANSLATION_GUIDE.md` — хэлний бодлого: монгол эх, НҮБ-ын 6 хэл.
- `frontend/scripts/` — `i18n:check` скрипт (CI-д холбогдоогүй).
- `docs/CORE_BOUNDARY_PLAN.md` §3.2, §5 Үе 2b.

Зорилго: шинэ апп нэмэхэд `index.tsx`-д гар хүрэхээ болих. Цөмийн түлхүүрийн
төрлийн аюулгүй байдал **хэвээр**.

#### Ажил 1 — Бүртгэл

`frontend/lib/i18n/registry.ts`:

```ts
export function registerDictionary(appId: string, dicts: Partial<Record<Locale, Record<string,string>>>): void
```

- Дуудагдсан дарааллаас үл хамааран ажиллана (`t()` нь бүртгэлээс уншина).
- Түлхүүрийн зөрчил: `slog`-ийн frontend хувилбар байхгүй тул `console.warn`,
  зөвхөн `NODE_ENV !== 'production'`. Дарж бичихийг зөвшөөрнө, чимээгүй биш.
- Цөмийн толь (`core`, `auth`, `access`, `appearance`, `app_store`, `modules`,
  `sharing`, `cp`, `website`, `integrations`, `emailverify`) нь одоогийн
  compile-time `as const` хэлбэрээрээ үлдэнэ — `TranslationKey` нь эдгээрээс л
  үүснэ. Аппын түлхүүр нь `string` төрөлтэй, `t()` нь хоёуланг хүлээж авна.

#### Ажил 2 — Аппуудыг шилжүүл

`addons/`-ийн 25 файлаас цөмийнхөөс бусдыг (`documents`, `egov`, `esign`,
`reports`, `urtuu`, `sso_clients`, `ai`, `storefront`, ба явсан аппуудынх:
`contacts`, `products`, `inventory`, `billing`, `gov`, `appstore_modules`)
`registerDictionary` дуудалттай болго. Дуудалт нь тухайн аппын дэлгэцийн
модуль ачаалагдахад ажиллана — Next.js App Router дээр `app/<slug>/layout.tsx`
эсвэл дэлгэцийн модулийн дээд талд.

`index.tsx`-ийн import + spread жагсаалтаас тэдгээрийг хас.

#### Ажил 3 — Локал файлууд

`locales/{ar,zh,fr,ru,es}.ts` тус бүрийг цөмийн хэсэг + аппын хэсэг болгож
хуваа: `locales/<locale>/core.ts` ба `locales/<locale>/<app>.ts`. Аппын
орчуулга `registerDictionary`-аар тухайн аппаараа ирнэ. Файл бүр ~1380 мөр
байсан нь аппын ажилтай хамт **21 удаа** өөрчлөгдсөн шалтгаан.

#### Ажил 4 — `i18n:check`-ыг CI-д холбо

`frontend/scripts/`-ийн одоо байгаа скриптийг ci.yml-д алхам болгож нэм.
Дутуу түлхүүр англи руу унадаг нь зөв зан төлөв, гэхдээ **чимээгүй** байх нь
зөв биш. Эхлээд warning горимоор нэм; одоо байгаа дутуу түлхүүрийн тоог
скриптийн гаралтад хэвлэ.

#### Хүлээн авах шалгуур

- `npx tsc --noEmit`, `npm run build` ногоон.
- Долоон хэл дээр `t()` дуудсан дэлгэц бүр өмнөх мөртэйгээ **яг ижил** текст
  буцаана. Хэрэв тест хэрэгсэл байхгүй бол: build-ийн өмнө/дараа хоёр удаа
  бүх түлхүүрийг цуглуулж JSON болгож харьцуулах нэг удаагийн скрипт бич,
  тэгш байхыг батлаад скриптийг PR-д үлдээ.
- `index.tsx` 150 мөрөөс богино.
- Шинэ апп нэмэхэд `index.tsx` гар хүрэх шаардлагагүй болсныг PR-ийн
  тайлбарт жижиг жишээгээр харуул.

#### PR

Гарчиг: `Орчуулгын толь ажиллах үедээ бүртгэгддэг болов`

---

## Үе 2c — `Layout.tsx`-ыг өгөгдлөөс жолоодох

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b frontend/phase-2c-layout`.
Үе 2b merge хийгдсэн байх ёстой.

Контекст:

- `frontend/components/Layout.tsx:28-67` (`iconMap`), `:113` (`ORGANISATION_APP`),
  `:115` (`APP_ORDER` — 9 аппын ID, үүнээс `contacts`, `products`, `inventory`,
  `billing`, `esign`, `gov_services` нь явсан), `:211` (эрэмбэлэлт).
- `backend/internal/platform/menu/icons_test.go:26-87` — Go тест нь
  `Layout.tsx`-ыг **парс хийж** icon бүр байгааг шалгадаг.
- `backend/pkg/nexus/module.go:68` — `MenuDefinition` нь `Icon`, `Order`
  талбартай аль хэдийн.
- `backend/internal/platform/menu/menu.go:37-68` — цэс `nexus.List()`-ээс
  бүрдэж, `Order`-ыг хадгалдаг.
- `frontend/app/apps/page.tsx:61-65` — `appIcons`, ижил асуудлын жижиг хувилбар.

Зорилго: шинэ аппын icon, дараалал нь **цөмийн файлд бус, аппынхаа
манифест/модульд** амьдардаг болох.

#### Ажил 1 — Icon-ыг динамик болго

`iconMap`-ийн гар аргын жагсаалтыг lucide-ийн нэрээр хайх динамик lookup-аар
солино. Мэдэгдэхгүй нэр ирвэл fallback icon (одоо хоосон дөрвөлжин гардаг —
энэ нь дуугүй эвдрэл).

Bundle-ийн хэмжээ анхаарах зүйл: lucide-ийн бүхлээр нь импортлох нь болохгүй.
Хоёр сонголт, аль нь энэ репод тохирохыг **өөрөө шийдэж, шалтгааныг PR-т бич**:
(a) ашиглаж болох icon-уудын нэрлэгдсэн allowlist-ыг build-time-д үүсгэх
скрипт, (b) `next/dynamic`-аар нэрээр lazy import.

`menu/icons_test.go` нь `Layout.tsx`-ыг парс хийхээ болино. Оронд нь:
компайл хийгдсэн модуль бүрийн `Menus()`-ийн `Icon` нь зөвшөөрөгдсөн олонлогт
байгааг шалгана. Олонлог нь одоо кодоос биш, нэг эх сурвалжаас (JSON эсвэл
generated Go файл) уншигдана.

#### Ажил 2 — `APP_ORDER` устга

Дараалал `MenuDefinition.Order` болон каталогийн бичлэгээс ирнэ. Backend-д
цэсний хариултад аппын эрэмбийн утга байгаа эсэхийг шалга; байхгүй бол
`menu.go`-д нэм (`nexus.MenuDefinition.Order` аль хэдийн бий, аппын түвшний
эрэмбэ хэрэгтэй бол `catalog.CatalogApp`-д талбар нэмнэ).

Мэдэгдэхгүй ID нь одоо 999 болоод цагаан толгойн дарааллаар ордог — тэр
зан төлөв **fallback болж хэвээр** үлдэнэ, зөвхөн хатуу жагсаалт алга болно.

#### Ажил 3 — `ORGANISATION_APP` тусгай тохиолдол

`Layout.tsx:113` нь нэг аппыг апп-төмөр замаас гаргадаг. Энэ нь манифестын
шинж чанар болох ёстой: `catalog.CatalogApp`-д `pinned` эсвэл `chrome` гэх мэт
талбар (нэрийг өөрөө сонго, `pkg/catalog/manifest.go`-ийн хэв маягт нийцүүл),
цэсний API-аар дамжина. `pkg/catalog`-д талбар нэмэх нь гэрээний өөрчлөлт —
`ValidateManifest`-д заавал биш талбар болгож, `docs/`-д тэмдэглэ.

#### Ажил 4 — `app/apps/page.tsx:61-65`

`appIcons` нь ижил асуудал. Ажил 1-ийн шийдлээр сольж, устга.

#### Хүлээн авах шалгуур

- `npx tsc --noEmit`, `npm run build` ногоон; bundle хэмжээ өмнөхөөсөө
  мэдэгдэхүйц өсөөгүй (`npm run build`-ийн гаралтыг PR-т наа).
- `go test ./internal/platform/menu/...` ногоон, `icons_test.go` нь
  `Layout.tsx`-ыг уншихаа больсон.
- Хажуугийн самбарын дүр төрх өөрчлөгдөөгүй: одоогийн 6 аппын icon ба
  дараалал өмнөхтэй ижил. Скриншот эсвэл жагсаалтыг PR-т наа.
- `grep -n 'io.gerege.nexus' frontend/components/Layout.tsx` нь хоосон.

#### PR

Гарчиг: `Хажуугийн самбар цэснийхээ өгөгдлөөс жолоодогддог болов`

---

## Үе 3 — Апп өөрийн схемтэй

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b core/phase-3-app-migrations`.
Үе 1 merge хийгдсэн байх ёстой.

Контекст:

- `backend/cmd/migrate/main.go:47-70` — `MIGRATIONS_DIR`, `MIGRATIONS_TABLE`
  аль хэдийн бий, тайлбарт нь distribution-ы хэрэгцээ бичигдсэн. **Механизм
  бэлэн, модулиудад холбогдоогүй.**
- `backend/db/migrations/` — 72 файл, 104 хүснэгт.
- `backend/internal/platform/appinstaller/installer.go` — суулгах урсгал.
- `backend/internal/platform/tenant/rls_integration_test.go:14` —
  `TestEveryTenantTableHasForcedRLS`, `pg_class`/`pg_policies`-ээс уншдаг.
  CI нь `TEST_DATABASE_URL` өгдөг тул энэ **ажиллаж байгаа**.
- `backend/db/migrations/00029_tenant_rls.sql:44-67` ба
  `00037_active_organisations.sql:37-66` — хоёр удаагийн нийтийн давталт;
  00038-аас хойш хүснэгт өөрөө policy-гоо зарлана (`00063:118-133` жишээ).
- `docs/CORE_BOUNDARY_PLAN.md` §3.3, §5 Үе 3.

Зорилго: платформын миграцын хавтас зөвхөн платформынх болох.

**Энэ PR-д өгөгдлийн шилжилт ХИЙХГҮЙ.** Механизм ба хамгаалалт л. Явсан
аппуудын 28 хүснэгтийг зөөх нь тусдаа, өгөгдлийн шилжилтийн төлөвлөгөөтэй PR.

#### Ажил 1 — SDK хуук

`pkg/nexus`-д:

```go
// Модуль өөрийн embed хийсэн миграцаа авчирна.
func Migrations(moduleID string, fsys fs.FS)
```

- `embed.FS`-тэй ажиллана.
- `appinstaller` нь апп суулгах үед тухайн модулийн миграцыг ажиллуулна,
  goose-ийн version хүснэгт нь модуль тус бүрд өөрийн (`goose_db_version_<slug>`)
  — `cmd/migrate`-ийн тайлбарт яагаад ингэх ёстойг аль хэдийн бичсэн
  (`main.go:52-56`).
- Суулгах транзакцтай хэрхэн харилцахыг **зориудаар шийд**: схемийн өөрчлөлт
  нь суулгалтын транзакц дотор байх уу, өмнө нь байх уу. Шийдвэрээ кодын
  тайлбарт бич.
- `api.txt` шинэчил.

#### Ажил 2 — Хамгаалалтын тестүүд

`backend/db/migrations/ownership_test.go`:

`TestPlatformMigrationsOwnNoAppTable` — `db/migrations/`-ийн бүх
`CREATE TABLE`-ыг гаргаж авч, платформын эзэмшдэг хүснэгтийн нэрлэгдсэн
жагсаалттай тулгана. Жагсаалтад байхгүй шинэ хүснэгт нь улаан. Алдааны
мессежэд: аппын хүснэгт нь аппынхаа миграцад байх ёстой; хэрэв энэ үнэхээр
платформын хүснэгт бол жагсаалтад нэмэх нь шийдвэр, тэр шийдвэр review-д
харагдана.

Одоо байгаа 28 явсан хүснэгтийг жагсаалтад **`// TODO: явсан` тэмдэгтэй**
оруул, ингэснээр тест ногоон боловч дараагийн PR-ийн ажил нь жагсаалтад
бичигдсэн байна.

`TestPlatformSQLNamesNoAppTable` — `internal/platform/**`-ийн бүх `.go` файлын
SQL мөрөнд аппын хүснэгтийн нэр (`products`, `contacts`, `warehouses`,
`stock_levels`, `stock_movements`, `billing_invoices`, `gov_*`, `store_*`)
байгаа эсэхийг хайна. **Одоо энэ тест улаан болно** — `internal/platform/ai/`
дотор байгаа (§3.4 A). Тестийг бич, `ai` пакетыг түр зөвшөөрөх жагсаалтад
оруулж, тэр бичлэгт "Үе 4a-д устгагдана" гэж бич. Үе 4a нь тэр бичлэгийг
устгах ажил.

Энэ тест яагаад хэрэгтэйг тайлбарт бич: `boundaries_test.go` нь Go import-ыг
хардаг, SQL мөрөн доторх хүснэгтийн нэр нь **яг ижил хамаарал**, зөвхөн
компилятор хардаггүй нь.

#### Ажил 3 — Өртөө-гийн RLS зөрүү

`00061:207-208`, `00062:114-115`, `00063:127-128`, `00066:87-88`-ын policy-ууд
нь `tenant_id = current_setting('app.current_tenant')` хэвээр, 00037-ийн
`= ANY(COALESCE(app.allowed_tenants, ...))` өргөтгөлийг аваагүй. Олон
байгууллагатай session Өртөө-гийн мөрийг зөвхөн идэвхтэй байгууллагынхаа
хэмжээнд харна.

Хаагдах тал руу алддаг тул аюулгүй, гэхдээ зөрүүтэй. Шинэ миграцаар 00037-ийн
хэлбэрт нийцүүл. `TestEveryTenantTableHasForcedRLS` дээр нэмж, policy-ийн
**хэлбэр** нь ижил байхыг шалгах тест бич (`pg_policies.qual`-ыг харьцуулах).

#### Хүлээн авах шалгуур

- `go test -race ./...` ногоон, шинэ тестүүд ажиллана.
- `go run ./cmd/migrate up` цэвэр өгөгдлийн санд амжилттай.
- `routes.txt` өөрчлөгдөөгүй.
- Хэрэв туршилтын модуль дээр `nexus.Migrations` хэрэглэвэл тухайн хүснэгт
  үүсч, `goose_db_version_<slug>` тусдаа гарч ирснийг батал.

#### PR

Гарчиг: `Апп өөрийн схемээ авч явдаг болов`

Тайлбарт: 104 хүснэгтийн 28 нь явсан аппуудынх гэдгийг тоогоор бич, дараагийн
PR (өгөгдлийн шилжилт) юуг хамрахыг нэрлэ.

---

## Үе 4a — AI-г commerce схемээс салгах

Энэ бол **алдааны засвар**, рефактор биш.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b core/phase-4a-ai-capability`.
Үе 1 ба Үе 3 merge хийгдсэн байх ёстой.

Контекст ба асуудал:

- `backend/internal/platform/ai/copilot.go:201-204` (tool зарлалт), `:215`
  (`erp_summary` — `products`, `contacts`, `warehouses`, `stock_levels`),
  `:222` (`search_products`), `:272` (fallback), `:337` (`classifyIntent`
  дотор "бараа", "агуулах", "харилцагч" гэсэн эвристик).
- `backend/internal/platform/ai/inventory_forecaster.go:36-40` — ижил хамаарал.
- `backend/internal/platform/server.go:339-340` (нөхцөлгүй бүтээгддэг),
  `:1040`, `:1045` (`/ai/copilot`, `/ai/stock-forecast` нөхцөлгүй мөнтөгддөг).
- Эдгээр хүснэгт нь commerce-ийнх; commerce нь `business-gerege-nexus` руу
  явсан (`docs/ECOSYSTEM_GIT_STRATEGY.md` §2.5, миграц 00060). Хүснэгтүүд нь
  `00003_business_apps.sql:17,31,42`-оор одоо ч цөмд **үүсдэг**, тиймээс
  алдаа гарахгүй — оронд нь commerce модульгүй суулгац дээр `erp_summary` нь
  LLM-д "0 бараа, 0 харилцагч, 0 үлдэгдэл" гэж **итгэлтэйгээр худал** хэлнэ.
  `copilot.go:236-238` өөрөө "хагас хариулт нь хариулт байхгүйгээс дор" гэж
  бичсэн байдаг — яг тэр тохиолдол.
- Үе 3-ын `TestPlatformSQLNamesNoAppTable` нь `ai` пакетыг түр зөвшөөрсөн
  бичлэгтэй. **Энэ PR тэр бичлэгийг устгана.**

Зорилго: copilot нь мэдэхгүй зүйлээ "0" гэж хэлэхээ болих.

#### Ажил 1 — Чадвар тодорхойл

`pkg/nexus`-д AI-д өгөгдөл нийлүүлэх гэрээ:

```go
type AssistantSource interface {
    // ...
}
```

Загварчлахдаа `nexus.MeetingBooker`-ийн сургамжийг дага
(`pkg/nexus/meetings.go:32-34`): гэрээ нь **`internal/` төрлөөр ярьж
болохгүй**, эс бөгөөс түүнийг хэрэгжүүлэх модуль энэ репогоос гарч чадахгүй.
Талбарын тоог бага байлга — `MeetingConnector` 15 талбарын оронд 3-ыг авсан
шиг.

Хэлбэрийн санал (өөрөө шийд, шалтгаанаа бич): tool-ийн нэр, тайлбар, JSON
schema, ба гүйцэтгэгч функц. Ингэснээр commerce өөрийн `erp_summary`,
`search_products`-ыг **өөрөө нийлүүлнэ**, цөм нь юу гэж нэрлэгдэхийг мэдэхгүй.

#### Ажил 2 — Цөмөөс commerce-ийн SQL-ыг устга

`copilot.go`-оос `erp_summary`, `search_products`, `stock_levels`-ийн fallback,
`inventory_forecaster.go` бүхэлдээ — устга. Оронд нь бүртгэгдсэн
`AssistantSource`-уудаас tool-уудыг цуглуулна.

`classifyIntent`-ийн бараа/агуулах/харилцагчийн эвристик мөн адил — эдгээр
нь чадвар нийлүүлэгчийн зарласан tool-оос үүсэх ёстой.

`/ai/stock-forecast` маршрут: нийлүүлэгчгүй бол **404**, "0" биш. `routes.txt`
golden-д энэ маршрут байх эсэхийг шийдэж, шийдвэрээ бич — зөвлөмж: маршрут
хэвээр үлдэж, дотроо 404 хариулна (`server.go`-ийн бусад газарт хэрэглэсэн
"маршрутын хүснэгт орчноос хамааран хэлбэрээ өөрчилдөггүй" зарчим,
`:797-802`-ын тайлбарыг үз).

#### Ажил 3 — Үнэн хариулт

Нийлүүлэгч байхгүй үед copilot нь "энэ суулгацад бараа материалын мэдээлэл
байхгүй" гэж хэлнэ. LLM-д өгөх system prompt-д мөн тусга — тоо байхгүй үед
тоо зохиохгүй байх нь prompt-ийн ажил, кодын ажил хоёулаа.

#### Ажил 4 — Хамгаалалт

Үе 3-ын `TestPlatformSQLNamesNoAppTable`-ээс `ai`-ийн түр зөвшөөрлийг устга.
Тест одоо жинхэнээсээ ногоон болно.

#### Хүлээн авах шалгуур

- `go test -race ./...` ногоон, `ai` зөвшөөрөл устсан.
- `grep -rn "products\|stock_levels\|warehouses" backend/internal/platform/ai/`
  нь хоосон.
- Commerce-гүй суулгац дээр `/ai/copilot` нь "мэдэхгүй" гэж хариулна, "0" гэж
  биш. Тестээр батал.

#### PR

Гарчиг: `AI туслах мэдэхгүй зүйлээ 0 гэж хэлэхээ болив`

Тайлбарт: энэ нь архитектурын цэвэрлэгээ **биш**, алдааны засвар гэдгийг
тодорхой бич. Commerce модульгүй суулгац дээр өнөөдөр юу болдгийг жишээгээр
харуул.

---

## Үе 4b — Анхдагч рольд олгох эрхийг апп өөрөө зарлана

Аюулгүй байдлын хамааралтай засвар.

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b core/phase-4b-default-roles`.

Контекст ба асуудал:

- `backend/internal/platform/appinstaller/installer.go:342` — `manager` рольд
  `gov.process`, `gov.delegate`, `gov.verify`, `gov.report` олгоно; `:355` —
  `user` рольд `gov.apply`. gov-services нь `gerege-gov` руу явсан; энэ
  бинари дотор эдгээрт таарах эрх байхгүй (`internal/apps/egov/egov.go:194-198`
  нь `egov.*` зарладаг, `gov.*` биш).
- `backend/db/migrations/00008_access_control.sql:77-78,119,124` — ижил
  жагсаалт давхардсан.
- `backend/pkg/nexus/module.go:52-64` — `PermissionDefinition` дөрвөн талбартай;
  `AdminOnly` нь `installer.go:330-332`-т зөвхөн **нарийсгана**.
- Тиймээс repo-гийн гаднах модуль өөрийн эрхээ анхдагчаар аль рольд очихыг
  **зарлаж чадахгүй**. Ганц зам нь `.read`/`.manage` дагаврын нэршлийн ёс —
  `gov.apply` хэлбэрийн эрх бол цөмийн энэ файлыг засахаас өөр аргагүй.
- `backend/internal/apps/access_policy_test.go` — модуль өөрөө хариулж,
  тест нь давхар бичиж баталдаг хэв маягийн жишээ. Мөн `boundaries_test.go:94`
  дэх түүх: платформ дээрх appID-гаар түлхүүрлэсэн switch нь App Store явахад
  **чимээгүй эвдэрч**, тэр бүтээгдэхүүний маршрут бүр гишүүн бүрд нээгдсэн.
  Одоогийн `gov.*` жагсаалт бол яг тэр хэв маягийн үлдэгдэл.

Зорилго: анхдагч олголтыг платформын мэдлэгээс аппын зарлал болгох.

#### Ажил 1 — `PermissionDefinition`-д талбар

```go
type PermissionDefinition struct {
    Code        string
    Name        string
    Description string
    AdminOnly   bool
    // DefaultRoles — суулгах үед энэ эрхийг ямар системийн рольд
    // анхдагчаар олгохыг апп өөрөө зарлана.
    DefaultRoles []string
}
```

- Хүчинтэй утга: `"manager"`, `"user"`. Бусад нь суулгалтын үед алдаа.
- `AdminOnly` нь давамгайлна: `AdminOnly: true` үед `DefaultRoles` хоосон биш
  бол энэ нь **зөрчил**, суулгалт унана — зөвхөн `slog` биш. Хоёр эсрэг
  зарлалыг чимээгүй нэгтгэх нь эрх алга болох арга.
- Хоосон `DefaultRoles` нь одоогийн `.read`/`.manage` дагаврын дүрэм рүү
  унана — **нийцтэй байдал**. Тэр fallback-ыг `// Deprecated` тэмдэглэж, аль
  major хувилбарт устахыг бич. Нэршлийн ёс нь гэрээ байх ёсгүй.
- `pkg/catalog/manifest.go:52` нь ижил төрлийг ашигладаг — гадаад аппын
  манифест мөн энэ талбарыг зарлаж чадна. `ValidateManifest`-д шалгалт нэм.
- `api.txt` шинэчил.

#### Ажил 2 — `gov.*`-ыг устга

`installer.go:342,355`-аас таван `gov.*` мөрийг устга. `documents.sign` нь
**үлдэнэ** — тэр нь энэ репод амьд эрх бөгөөд `:344-354`-т яагаад энгийн
хэрэглэгчид хүрэх ёстойг сайн үндэслэсэн байгаа. Тэр эрхийг устгахын оронд
`documents` модулийн `Permissions()`-д `DefaultRoles: []string{"manager","user"}`
болгож зарлаж, `installer.go`-оос тусгай тохиолдлыг ав.

`00008_access_control.sql`-ын давхардсан жагсаалтад **гар хүрэхгүй** —
хэрэглэгдсэн миграцыг өөрчлөх нь буруу. Оронд нь шинэ миграцаар цэвэрлэ, эсвэл
(хэрэв тэдгээр эрх өгөгдлийн санд байхгүй бол) юу ч хийхгүй бөгөөд шалтгааныг
PR-т бич.

#### Ажил 3 — Хамгаалалт

`internal/platform/appinstaller/installer_test.go`:

`TestNoAppPermissionCodeIsNamedInThePlatform` — `installer.go`-ийн эх кодыг
уншиж, цэгтэй эрхийн код (`^[a-z_]+\.[a-z_]+$` хэлбэрийн мөр) байгаа эсэхийг
шалгана. Олдвол улаан. Мессежэд `boundaries_test.go:94`-ийн түүхийг иш тат:
appID-гаар түлхүүрлэсэн шийдвэр платформ дээр байх нь модуль явахад чимээгүй
эвдэрдэг.

`access_policy_test.go`-ийн хэв маягаар: компайл хийгдсэн модуль бүрийн
`DefaultRoles` зарлалыг хүснэгтэд давхар бич. Эрх чимээгүй алга болох нь
гаднаас нь алдаа шиг харагддаггүй — апп ажиллаж, хуудас нээгдэж, зөвхөн
хэрэгтэйгээс олон хүн хүрдэг.

#### Хүлээн авах шалгуур

- `go test -race ./...` ногоон.
- Шинэ тенант үүсгэхэд `manager`, `user` рольд очих эрхийн олонлог өмнөхтэй
  **яг ижил** (`documents.sign` орсон хэвээр). Интеграцын тестээр батал.
- `AdminOnly: true` + `DefaultRoles` хоосон биш үед суулгалт унана.
- `grep -nE '"[a-z_]+\.[a-z_]+"' backend/internal/platform/appinstaller/installer.go`
  нь хоосон.

#### PR

Гарчиг: `Анхдагч эрхийг апп өөрөө зарладаг болов`

Тайлбарт: (1) `gov.*` нь явсан аппынх гэдгийг, (2) энэ хүртэл гаднын модуль
`.read`/`.manage` гэж нэрлэхээс өөр аргагүй байсныг, (3) `AdminOnly` нь зөвхөн
нарийсгадаг байсныг бич.

---

## Үе 4c — Pilot: нэг аппыг distribution болгох

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b core/phase-4c-pilot-extract`.
Үе 1 ба Үе 3 merge хийгдсэн байх ёстой.

Контекст:

- `backend/internal/apps/urtuu/` — production файлууд (`envelopes.go`,
  `evidence.go`, `handlers.go`, `metrics.go`, `numbers.go`, `reports.go`,
  `tasks.go`, `urtuu.go`) нь `backend/internal/...`-ыг **огт импортлодоггүй**.
  Конструктор: `urtuu.go:60` `New(p nexus.Platform, link nexus.Link)`.
  Хамаарал нь `backend/domain/urtuu`, `backend/pkg/urtuu`, `pkg/nexus`.
- `backend/internal/apps/organisation/` — мөн адил тэг. `module.go:81`
  `New(p nexus.Platform)`. Хамаарал нь `backend/domain/organisation(/postgres)`.
- **Гэхдээ тест файлууд импортлодог:** `urtuu/lifecycle_test.go` нь
  `internal/platform/{dbguard,reporting,urtuu}`; `organisation/module_test.go`
  нь `internal/platform/rbac`.
- `backend/domain/` нь `internal/` биш тул гаднаас импортлогдоно — **гэхдээ
  `pkg/nexus`-ийн тогтвортой байдлын амлалтад хамрагдахгүй**
  (`pkg/nexus/api_golden_test.go`, `docs/RELEASING.md`).
- `docs/ECOSYSTEM_GIT_STRATEGY.md` §1 Түвшин 2 — distribution repo-гийн бүтэц.
- `backend/pkg/platform/run.go:18-33` — distribution-ы `main.go` ямар харагдахыг
  бичсэн тайлбар.

Зорилго: заадсыг **амьдаар шалгах**. Аль нэгийг сонгож (зөвлөмж: `urtuu` —
`nexus.Link`-ээр дамжуулан цөмтэй ярьдаг нь заадсыг илүү сайн шалгана)
distribution repo болгож гаргах.

#### Ажил 0 — Шийдэх асуулт: `domain/` хаашаа явах вэ

Энэ бол pilot-ийн жинхэнэ асуулт бөгөөд кодлохоос **өмнө** хариулна.

Гурван сонголт, тус бүрийн үнэ:

1. `domain/urtuu` нь энэ репод үлдэж, distribution нь түүнийг импортлоно.
   Үнэ: distribution нь `pkg/nexus`-ийн амлалтад хамрагдахгүй пакетаас
   хамаарна. Тэр пакет өөрчлөгдөхөд юу ч анхааруулахгүй.
2. `domain/urtuu` нь distribution-тайгаа хамт явна. Үнэ: `pkg/urtuu` (утасны
   гэрээ) ба `domain/urtuu` (домэйн) хоёрын хил тодорхой байх ёстой.
   `ADR 0001` нь домэйн нь аппынхаа хажууд байхыг шаарддаг тул энэ нь
   зарчимтай нийцнэ.
3. `domain/urtuu`-г `pkg/` руу гаргаж semver амлалт үүрүүлнэ. Үнэ: SDK-гийн
   гадаргуу өснө, `api_golden_test`-ийн ачаа нэмэгдэнэ.

**Зөвлөмж: (2).** Шийдвэрээ `docs/adr/` дор ADR болгож бич (одоо байгаа
ADR-уудын хэв маягаар), кодлохоос өмнө.

#### Ажил 1 — Тестийн хамаарлыг тасал

`urtuu/lifecycle_test.go`-ийн `internal/platform/{dbguard,reporting,urtuu}`
импортыг ав. Тус бүрд:

- `dbguard` — тенант тусгаарлалтыг тестлэхэд хэрэгтэй. Тестийн туслах болгож
  distribution-д хуулах уу, эсвэл `pkg/`-д жижиг тест-туслах гаргах уу гэдгийг
  шийд. **Хуулах нь энд зөв байж магадгүй** — тестийн туслах бол гэрээ биш.
- `reporting` — тайлан бүртгэгдснийг шалгадаг. `pkg/nexus`-ийн
  `RegisterReport`-оор дамжуулж шалгаж болох уу?
- `internal/platform/urtuu` (`transport`) — энэ нь жинхэнэ хамаарал.
  `nexus.Link` fake-ээр солих боломжтой эсэхийг шалга.

#### Ажил 2 — Distribution repo

`docs/ECOSYSTEM_GIT_STRATEGY.md` §1 Түвшин 2-ын бүтцээр шинэ repo. Энэ
session-д repo үүсгэж чадахгүй бол **энэ репогийн `/tmp` эсвэл sibling
хавтсанд** бүтнээр нь бэлдэж, `go build` ба `go test` амжилттай болохыг
батлаад, агуулгыг PR-ийн тайлбарт бич.

```
gerege-urtuu/
  go.mod          # require .../open-gerege-nexus/backend vX.Y.Z
  main.go         # pkg/platform.Run(Options{Modules: ...})
  modules/urtuu/  # internal/apps/urtuu-аас зөөгдсөн
  domain/urtuu/   # Ажил 0-ийн шийдвэрээр
  migrations/     # 00061-00067-аас зөөгдсөн
  catalog/
```

`main.go` нь `pkg/platform/run.go:18-33`-ын тайлбарт бичигдсэн хэлбэрээр.
Модуль нь `nexus.Link`-ыг **параметрээр биш**, `nexus.Capability[nexus.Link]()`-
ээр авна (Үе 1).

#### Ажил 3 — Цөмөөс ав

`internal/apps/urtuu/` устга. `runtime.go`-оос `appurtuu.New` дуудалт ав.
`access_policy_test.go`-ийн `corePolicies`-оос `urtuu` бичлэгийг ав
(`TestEveryModuleInThisRepositoryIsClassified` нь хавтас уншдаг тул автоматаар
таарна). `catalog/apps.json`, `catalog/manifests/`, `catalog/chronicle/`-оос
холбогдох бичлэгүүдийг ав. i18n addon (`frontend/lib/i18n/addons/urtuu.ts`) ба
дэлгэц (`frontend/app/module/urtuu/`) нь §2.3-ын дүрмээр **цөмд үлдэнэ**
(Үе 2d хүртэл).

`internal/platform/urtuu/` (transport) ба `pkg/urtuu` (гэрээ) нь **цөмд
үлдэнэ** — тэдгээр нь рельс, ганц байх ёстой.

#### Хүлээн авах шалгуур

- Цөмд: `go test -race ./...`, `go vet`, `golangci-lint` ногоон.
- Distribution-д: `go build ./...`, `go test ./...` ногоон.
- Хоёуланг нь нэг өгөгдлийн сан дээр ажиллуулж, Өртөө-гийн даалгаврын урсгал
  өмнөхтэй ижил ажиллаж байгааг батал.
- `routes.txt` golden-д `[app] /api/v1/urtuu/*` мөрүүд **алга болно** — тэр нь
  зөв diff. Гэхдээ цөмийн `/api/v1/urtuu/exchange/*` ба
  `/.well-known/urtuu.json` **үлдэнэ** (тэдгээр нь transport, апп биш).
  Golden-ыг `-update`-ээр шинэчилж, аль мөр яагаад явсныг PR-т бич.

#### PR

Гарчиг: `Өртөө апп distribution болж гарав`

Тайлбарт: (1) Ажил 0-ийн шийдвэр ба ADR-ийн холбоос, (2) тестийн гурван
хамаарлыг хэрхэн тасалсан, (3) `routes.txt`-ийн diff-ийн тайлбар, (4) энэ
pilot-оос гарсан **дараагийн заадсын асуудлууд** — юу нь бодсоноос хэцүү
байсан. Сүүлийнх нь хамгийн үнэ цэнэтэй хэсэг.

---

## Үе 5 — Downstream distribution build

### PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. `git checkout -b ci/phase-5-downstream`.
Үе 1 merge хийгдсэн байх ёстой.

Контекст:

- `backend/pkg/nexus/api_golden_test.go` — толгойн тайлбарт бичсэн: гэрээ
  эвдэрдэг нь хүн шийдэж эвдэрдэггүй, рефакторын үеэр параметрийн нэр
  солигдож, эсвэл interface-д метод нэмэгдэж эвдэрдэг. **Гэхдээ golden файл
  зөвхөн гарын үсгийг барина.** Зан төлөвийн эвдрэлийг барихгүй: метод хэвээр,
  утга нь өөр буцаах, sink дуудагдах дараалал өөрчлөгдөх, `Capability` буцаах
  алдаа өөр болох — эдгээрийн нэг нь ч api.txt-д харагдахгүй.
- Байгаа distribution-ууд: `appstore-gerege-mn`, `business-gerege-nexus`,
  `gerege-gov`, `pos-gerege-nexus`, ба Үе 4c-ийн `gerege-urtuu`.
- `.github/workflows/ci.yml` — одоогийн ажлууд.

Зорилго: "Би distribution-г эвдсэн үү?" гэсэн хариу хоногийн дараа биш,
минутын дараа ирдэг болох.

#### Ажил 1 — Downstream ажил

`.github/workflows/ci.yml`-д (эсвэл тусдаа `downstream.yml`) шинэ job:

- Нэг canary distribution repo-г clone хийнэ (хамгийн жижиг, хамгийн тогтвортой
  нь — Үе 4c дууссан бол `gerege-urtuu`, үгүй бол `appstore-gerege-mn`).
- `go.mod`-ын `open-gerege-nexus/backend` хамаарлыг `replace`-ээр **энэ PR-ийн
  commit** руу заана.
- `go build ./... && go test ./...`.
- Унавал PR улаан, мессежэд ямар distribution, аль тест эвдэрснийг нэрлэнэ.

Хэрэв distribution repo нь private бол: `secrets`-ээр token, эсвэл (илүү
энгийн) энэ репод `testdata/canary/` дор **хамгийн бага distribution**-ыг
өөрөө бичиж хадгал — `main.go` + нэг жижиг модуль, `pkg/nexus`-ийн гол
гадаргуу бүрийг хүрдэг (Module, AccessPolicy, Provide, Capability, Report,
Migrations). Тэр нь бодит distribution биш ч, гэрээг ашигладаг бодит код.
**Энэ хувилбарыг сонгосон бол PR-т яагаад сонгосныг бич.**

#### Ажил 2 — `api.txt`-ийн diff-ийг журамжуул

PR template (`.github/pull_request_template.md`) үүсгэ эсвэл нэм:
`backend/pkg/nexus/testdata/api.txt` өөрчлөгдсөн бол PR-ийн тайлбарт
"дуудагч юу хийх ёстой"-г үгээр бичих шаардлагатай гэсэн checkbox.
`api_golden_test.go`-ийн алдааны мессеж аль хэдийн үүнийг шаарддаг —
template нь тэр шаардлагыг PR хүртэл авчирна.

#### Ажил 3 — Release журам

`docs/RELEASING.md`-ыг шинэчил: (1) downstream ажил юуг барьдаг, юуг
барьдаггүйг, (2) Үе 1-ийн deprecated дөрөв (`UseLink`, `UseDocumentFiler`,
`UseAuditSink`, `UseReportSink`) аль major-т устахыг тодорхой огноо/хувилбартай
бич.

#### Хүлээн авах шалгуур

- Downstream job нь энэ PR дээр ажиллаж, ногоон.
- `pkg/nexus`-д зан төлвийн эвдрэл гараар оруулаад (жишээ:
  `Capability[T]()`-ыг байхгүй үед `nil, nil` буцаадаг болгож) downstream
  **улаан** болохыг батал, дараа нь буцаа. Энэ туршилтын үр дүнг PR-т бич —
  golden файл үүнийг барихгүй байсныг харуулах нь энэ job-ийн бүх утга учир.

#### PR

Гарчиг: `Цөмийн өөрчлөлт distribution дээр шууд шалгагддаг болов`

---

## Дараа нь

Үе 2d (`@gerege/ui`) ба явсан аппуудын дэлгэцийг гаргах нь **энэ багцад
ороогүй**. Тэдгээр нь долоо хоногийн ажил бөгөөд Үе 2a–2c дууссаны дараа
хамаагүй хямд болно — `CORE_BOUNDARY_PLAN.md` §5 Үе 2d ба §8-ыг үз.

`documents`-ыг рельс/апп болгож хуваах нь мөн ороогүй: §6-д яагаад одоо
хийвэл хамгийн их эрсдэлтэй, хамгийн бага өгөөжтэй болохыг бичсэн. Үе 1–3
дууссаны дараа дахин үнэлнэ.

**Үе бүрийн дараа §9-ийн скриптийг ажиллуулж тоог бич.** Одоо 76.6%.
Үе 1 + 2 дууссаны дараа 25%-иас доош, Үе 3 + 4 дууссаны дараа 10%-иас доош
байх ёстой. Тоо буурахгүй бол төлөвлөгөө буруу байсан — дараагийн үе рүү
орохын оронд яагаадыг нь ол.
