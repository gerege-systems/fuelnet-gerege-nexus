# Claude Code — Мониторинг ба тайлангийн системийг хэрэгжүүлэх prompt

Доорх текстийг бүтнээр нь хуулж Claude Code-д өгнө. Үе шат бүрийг тусад нь
өгч болно (санал болгож буй арга — нэг чатад нэг үе шат), эсвэл бүтнээр нь
өгөөд дараалан хийлгэж болно.

---

## PROMPT (эндээс доошхыг хуулна)

Чи Gerege Nexus репод ажиллаж байна. Эхлээд дараах баримтуудыг уншиж
контекстоо бүрдүүл:

- `docs/MONITORING_AND_REPORTING_PROPOSAL.md` — энэ ажлын дизайн баримт.
  Бүх шийдвэр энэ баримтад тулгуурлана; зөрчилдвөл энэ баримт нь эх сурвалж.
- `docs/ARCHITECTURE_SPECIFICATION.md`, `docs/MODULE_AUTHORING_GUIDE.md`
- `backend/internal/platform/observability/metrics.go`,
  `backend/internal/platform/audit/audit.go`,
  `backend/internal/platform/resilience/`
- `docker-compose.prod.yml`, `deploy/` хавтас
- Миграцийн дугаарлалт: `backend/db/migrations/` доторх сүүлийн дугаараас
  үргэлжлүүлнэ.

Дараах 5 үе шатыг **дэс дарааллаар** нь хэрэгжүүл. Үе шат бүрийн төгсгөлд
шалгалтуудыг ажиллуулж, ногоон болсны дараа тухайн үе шатыг тусдаа commit
болгож байж дараагийнх руу шилж.

### Заавал мөрдөх дүрмүүд (бүх үе шатад)

1. **Репогийн хэв маягийг дага**: ORM хэрэглэхгүй — `pgx` + гар бичмэл SQL;
   схемийн өөрчлөлт зөвхөн goose миграцаар; шинэ хүснэгт бүрт migration
   00029-тэй ижил загвараар tenant RLS policy бичнэ (тенантад хамааралгүй
   платформ-түвшний хүснэгт бол тайлбар comment-оор яагаад гэдгийг бич).
2. **Нэг бинари философи**: шинэ Go процесс, шинэ микросервис үүсгэхгүй.
   Мониторингийн стек нь тусдаа compose файлд, платформын кодоос ангид.
3. **Prometheus label-д тенант ID/slug, хэрэглэгч ID, чөлөөт текст
   ОРУУЛАХГҮЙ** — cardinality хамгаалалт. Одоогийн `metrics.go`-ийн routed
   pattern зарчмыг хадгал.
4. **UI текст** одоогийн i18n бүтцээр: монгол эх + англи орчуулгатай нэм;
   бусад 5 хэлийг `docs/TRANSLATION_GUIDE.md`-ийн журмаар (байхгүй бол
   англиар fallback).
5. Үе шат бүрд unit test бич; `cd backend && go test -race ./...`,
   `go vet ./...`, `golangci-lint run`, frontend өөрчлөлт орсон бол
   `cd frontend && npm run build` — бүгд ногоон байх ёстой.
6. Compose файлд өөрчлөлт орвол `docker compose -f <файл> config`-оор
   синтаксыг шалга.
7. Нууц утга (Grafana admin нууц үг г.м.) хатуу бичихгүй — env хувьсагчаар,
   `.env.example` ба `deploy/.env.prod.example`-д жишээ утгатай нэм.

---

### Үе шат 1 — Instrumentation гүйцээх (зөвхөн backend код)

`backend/internal/platform/observability/`-г өргөтгө:

1. **Go runtime ба process collector** default registry дээр бүртгэлтэй
   эсэхийг шалгаж, байхгүй бол нэм.
2. **pgxpool stats** — 15 секунд тутам pool-ийн `AcquiredConns`, `IdleConns`,
   `TotalConns`, `EmptyAcquireCount`-ийг gauge/counter болгож экспортлох
   collector.
3. **Resilience хэмжүүр** — `resilience/` доторх breaker, load shedder,
   retry-д hook нэмж: `resilience_breaker_state{name}` (0=closed,1=open,
   2=half-open), `resilience_load_shed_total`, `resilience_retry_total{name}`.
   Resilience пакетын API-г эвдэхгүйгээр, observer/callback хэлбэрээр залга.
4. **Гадаад системийн дуудлага** — ХУР (`xyp.go`), eID, ДАН, eSign HSM,
   Gemini, email verify зэрэг бүх outbound HTTP клиентэд нэг wrapper:
   `external_request_duration_seconds{system,operation,status}` histogram.
   `system` label нь `xyp|eid|dan|esign|gemini|emailverify` гэсэн хаалттай
   жагсаалт.
5. **Бизнес counter** — `logins_total{method,result}`,
   `invoices_created_total`, `documents_signed_total{rail,result}`,
   `ai_requests_total{kind}`. Тухайн handler-уудад нэг мөр increment.
6. **Structured лог** — `ENVIRONMENT=production` үед slog-ийг JSON handler
   болго. Request middleware-д request ID үүсгэж (байхгүй бол), slog-ийн
   бүх мөрөнд `request_id`, `tenant_id` attr автоматаар орох болго
   (context-оос уншдаг slog.Handler wrapper).
7. **Audit-ийг DB-д** — шинэ миграц: `audit_events` хүснэгт (id, tenant_id,
   user_id, action, resource, details jsonb, created_at; tenant_id индекстэй,
   RLS-тэй). `audit.Record`-ийг stdout лог + DB insert хоёулаа хийдэг болго
   (insert алдаа нь үндсэн үйлдлийг унагахгүй — best effort, алдааг логло).
   Хуучин дуудлагын газрууд өөрчлөгдөхгүй байхаар signature-ээ хадгал,
   context-оор DB хүрдэг болго.

Тест: collector-ууд бүртгэгдсэн, хэмжүүрүүд increment хийгддэг, audit DB
insert ажилладаг (testcontainer эсвэл одоогийн тестийн загвараар).

---

### Үе шат 2 — Мониторингийн стек (deploy)

`deploy/monitoring/` хавтас + `deploy/docker-compose.monitoring.yml` үүсгэ:

1. **Сервисүүд**: `prometheus` (retention 60d, scrape: backend `/metrics`,
   өөрийн стекийн exporter-ууд), `loki` (single-binary, retention 31d,
   filesystem storage), `alloy` (Docker контейнеруудын логийг Loki руу —
   docker discovery), `grafana`, `alertmanager`, `node_exporter`, `cadvisor`,
   `postgres_exporter` (одоогийн postgres руу read-only role-оор — миграцаар
   `monitoring` гэсэн login role үүсгэ, зөвхөн `pg_monitor` эрхтэй),
   `redis_exporter`. Бүх порт `127.0.0.1`-д bind хийгдэнэ; Grafana-г nginx
   snippet-ээр (жишээ config `deploy/nginx/`-д) гаргана.
2. **Volume**: prometheus/loki/grafana өгөгдөл нэрлэсэн volume-тэй,
   `docker-compose.prod.yml`-ийн volume нэршлийн загварыг дага
   (`gerege_nexus_monitoring_*`).
3. **Dashboard-as-code**: `deploy/monitoring/grafana/provisioning/` дотор
   datasource + dashboard provisioning; дараах dashboard-уудыг JSON-оор бич:
   *API тойм* (RED: rps, error rate, p50/p95/p99 latency, top routes),
   *Гадаад системүүд* (system тус бүрийн latency/error), *Инфра* (CPU, RAM,
   диск, контейнер бүрийн хэрэглээ, Postgres холболт/удаан query, Redis),
   *Resilience* (breaker төлөв, shed/retry).
4. **Alert rules** (`deploy/monitoring/prometheus/rules/`): API availability
   99.9% SLO дээр proposal-ын §2.3-ын multi-window burn-rate 3 түвшин;
   диск >85%; Postgres холболт >80%; гадаад систем бүрийн error rate;
   backend `/health` scrape fail; TLS хугацаа <14 хоног (blackbox биш бол
   node_exporter textfile эсвэл prometheus-ын cert metric ашигла).
   Alertmanager: SMTP env-ээр, Telegram webhook env-ээр (хоосон бол тухайн
   receiver идэвхгүй).
5. **Runbook**: `docs/RUNBOOKS.md` — alert бүрд: юу болсон бэ, эхний 5
   минутад юу шалгах, яаж засах, хэзээ escalate хийх. Монголоор.
6. **Баримт**: `docs/MONITORING.md` — стекийг асаах/унтраах, Grafana руу
   орох, шинэ хэмжүүр нэмэх журам; README-гийн баримтын индекст нэм.
   Uptime Kuma-г гадаад хостоос ажиллуулах зааврыг мөн энд бич (энэ репод
   deploy хийхгүй, зөвхөн заавар).

---

### Үе шат 3 — Tracing + error tracking

1. **OpenTelemetry**: `go.opentelemetry.io/otel` + `otelhttp` (chi
   middleware), `otelpgx` (pgx tracer), OTLP exporter → Tempo.
   `OTEL_EXPORTER_OTLP_ENDPOINT` хоосон бол tracing бүрэн унтарсан байх
   (no-op, ямар ч overhead-гүй). Sampling: `OTEL_TRACES_SAMPLER_ARG`-аар,
   default 10%.
2. **Tempo**-г monitoring compose-д нэм (single-binary, retention 72h,
   filesystem storage); Grafana datasource + trace↔log↔metric холболт
   (derived fields: логийн `trace_id` → Tempo линк).
3. **otelslog**: slog handler-ийг otel bridge-тэй болгож лог бүрд
   `trace_id`, `span_id` орох болго.
4. **GlitchTip**: monitoring compose-д нэм (postgres-ээ дундаа хэрэглэхгүй —
   өөрийн жижиг postgres volume-тэй). Backend-д sentry-go SDK:
   `SENTRY_DSN` хоосон бол унтарсан. Panic recovery middleware-ээс event
   илгээнэ. Frontend-д `@sentry/nextjs` мөн env-gated. PII scrub: user
   email/register дугаар event-д орохгүй.

---

### Үе шат 4 — Тайлангийн модуль `io.gerege.nexus.reports`

`docs/MODULE_AUTHORING_GUIDE.md`-ийн журмаар шинэ апп модуль үүсгэ
(каталог manifest, chronicle, `catalog/apps.json` бүртгэл орно):

1. **Report interface** (`backend/internal/platform/reporting/`):

   ```go
   type Report interface {
       Key() string                 // "billing.revenue" г.м.
       App() string                 // аль аппад харьяалагдах
       Titles() map[string]string   // i18n нэр
       Params() []ParamSpec         // date_range, warehouse_id г.м.
       Columns() []ColumnSpec
       Run(ctx context.Context, q Querier, p Params) (Result, error)
   }
   ```

   Модулиуд тайлангаа init үедээ registry-д бүртгүүлнэ. Тенантад тухайн
   апп идэвхгүй бол тайлан нь жагсаалтад гарахгүй (одоогийн app gating).
2. **Эхний тайлангууд** (модуль тус бүрд 2-оос доошгүй): billing — орлого
   сараар, нэхэмжлэхийн төлөв; inventory — үлдэгдэл агуулахаар, хөдөлгөөний
   тойм; documents/esign — гарын үсгийн статистик rail-аар; core —
   хэрэглэгчийн идэвх. Бүгд гар бичмэл SQL, tenant context дотор,
   `statement_timeout` 30s.
3. **API**: `GET /api/v1/reports` (жагсаалт), `GET /api/v1/reports/{key}`
   (метадата), `POST /api/v1/reports/{key}/run` (JSON үр дүн),
   `POST /api/v1/reports/{key}/export?format=xlsx|csv`. RBAC:
   `reports.view` + тайлангийн app-ын эрх. Бүх run/export `audit`-д бичигдэнэ.
4. **Export**: xlsx — `excelize`, csv — stdlib. Толгой мөр тайлангийн
   i18n нэрээр, тоо/огнооны форматтай.
5. **Товлосон тайлан**: миграц `report_schedules` (tenant_id, report_key,
   params jsonb, cron хэлбэрийн хуваарь, format, recipients text[], RLS).
   Backend дотор goroutine scheduler (шинэ процесс биш) — минут тутам
   шалгаад болзсон тайланг ажиллуулж, export-ийг одоогийн hosted email
   үйлчилгээгээр илгээнэ. Давхар илгээлтээс хамгаалж `last_run_at`
   advisory lock-той шинэчил.
6. **Frontend** `/reports`: тайлангийн жагсаалт (аппаар бүлэглэсэн),
   параметрын форм, үр дүнгийн хүснэгт + Recharts граф (тайлангийн
   ColumnSpec-д chart hint), export товч, товлосон тайлангийн CRUD.
   Одоогийн UI компонент, дизайны хэв маягийг дага.

---

### Үе шат 4б — Тенант дамнасан grant + нэгдсэн тайлан

Proposal-ын §3.5-ын зарчмаар (default deny, тодорхой буцаагдах grant,
counterparty scope, түүхий мөр биш үр дүн, хоёр талын audit):

1. **Миграц** `report_grants`: proposal-д байгаа схемээр (grantor/grantee
   tenant, report_key, scope `counterparty|full`, counterparty_ref,
   valid_from/until, revoked_at, created_by, accepted_by). RLS: grantor
   БА grantee хоёулаа өөрт хамаарах мөрөө харна.
2. **Grant урсгал**: API — grantee хүсэлт үүсгэх → grantor-ын админ
   зөвшөөрөх/цуцлах. Хоёр үйлдэл хоёулаа audit-д. UI: Тохиргоо дотор
   "Тайлан хуваалцах" дэлгэц — ирсэн хүсэлт, өгсөн grant, авсан grant,
   цуцлах товч, "Хандалтын түүх" (миний өгөгдлөөс хэн юу татсан).
3. **Нэгдсэн тайлан ажиллуулах**: reporting хөдөлгүүрт `RunConsolidated` —
   grantee-гийн хүсэлтээр идэвхтэй grant бүхий grantor бүр дээр давтаж,
   **grantor-ын tenant context дотор** (dbguard-ын одоогийн механизмаар,
   RLS сулруулахгүй) тайланг counterparty шүүлттэй ажиллуулж, үр дүнг
   компани тус бүрийн задаргаа + нийт дүнтэй нэг Result болгож нэгтгэнэ.
   Grantor бүрийн гүйлт тус тусдаа audit event (хоёр талд). Аль нэг
   grantor дээр алдаа гарвал бусад нь үргэлжилж, үр дүнд тухайн компани
   "алдаатай" гэж тэмдэглэгдэнэ (бүхэлдээ унахгүй).
   Counterparty холбоос: тенантуудыг регистрийн дугаараар
   (`tenants`-ийн байгууллагын мэдээлэл) тулгаж холбоно — энэ тулгалтыг
   grant үүсгэх үед нэг удаа шийдэж `counterparty_ref`-д хадгал.
4. **Frontend**: нэгдсэн тайлан нь `/reports` дотор "Нэгдсэн" гэсэн
   тэмдэгтэй харагдана; компаниар задлах/нэгтгэх toggle, export мөн адил.
5. **Тест**: grant-гүй үед юу ч харагдахгүй, цуцалсны дараа шууд хаагдана,
   counterparty scope өөр уурхайн мөрийг оруулдаггүй, audit хоёр талд
   бичигддэг — эдгээр 4 integration тест заавал.

---

### Дуусгах шалгуур (Definition of Done)

- Бүх тест, lint, build ногоон; CI workflow эвдрээгүй.
- `docker compose up -d` + monitoring compose хоёулаа цэвэр асдаг.
- Шинэ env хувьсагч бүр `.env.example`, `deploy/.env.prod.example`,
  README-гийн хүснэгтэд бичигдсэн.
- `docs/MONITORING.md`, `docs/RUNBOOKS.md` үүссэн, README индекст орсон;
  `CHANGELOG.md`-д тэмдэглэгдсэн.
- Үе шат бүр тусдаа commit, тайлбар нь монголоор, юу ба яагаад гэдгийг
  хэлсэн байна.

Эхлэхийн өмнө үе шат бүрийн төлөвлөгөөгөө towch гаргаж надаар
баталгаажуулаад, дараа нь Үе шат 1-ээс эхэл.

---

## PROMPT төгсөв
