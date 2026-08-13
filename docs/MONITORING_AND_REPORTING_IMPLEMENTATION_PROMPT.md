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
- `docs/CONTROL_PLANE_PLAN.md` — Үе шат 5 (Control Plane)-ийн дизайн
  баримт; мөн адил эх сурвалжийн статустай.
- `docs/ARCHITECTURE_SPECIFICATION.md`, `docs/MODULE_AUTHORING_GUIDE.md`
- `backend/internal/platform/observability/metrics.go`,
  `backend/internal/platform/audit/audit.go`,
  `backend/internal/platform/resilience/`
- `docker-compose.prod.yml`, `deploy/` хавтас
- Миграцийн дугаарлалт: `backend/db/migrations/` доторх сүүлийн дугаараас
  үргэлжлүүлнэ.

Дараах үе шатуудыг **дэс дарааллаар** нь хэрэгжүүл. Үе шат бүрийн төгсгөлд
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

### Үе шат 5 — Control Plane (cp.nexus.gerege.mn)

`docs/CONTROL_PLANE_PLAN.md`-ийн дизайнаар, доторх CP-1 … CP-5 дарааллаар.
CP-1 бусдынх нь урьдчилсан нөхцөл; CP-4 нь Үе шат 2 (мониторингийн стек)
хийгдсэн байхыг шаардана. Үе шат бүр тусдаа commit.

**CP-1 — Суурь, нэвтрэлт, audit:**

1. **Хост тусгаарлалт**: nginx-д `cp.nexus.gerege.mn` virtual host
   (`deploy/nginx/`-д config + IP allowlist-ийн жишээ snippet, allowlist
   нь env/include файлаар солигддог). Backend-д `/cp/api/*` route бүлэг —
   зөвхөн cp хостоос ирсэн (nginx-ийн тавьсан толгойгоор, TRUST_PROXY
   зарчмаар) хүсэлтэд нээгдэнэ. Frontend-д `app/cp/` route бүлэг, cp
   хостоос бусад дээр 404.
2. **Миграцууд**: `operator_accounts` (id, email, name, role
   `superadmin|operator|support|auditor`, totp_secret, webauthn
   credentials, disabled_at, created_at), `operator_sessions` (богино
   хугацаатай, SHA-256 хэш — одоогийн sessions загвараар),
   `operator_audit` (operator_id, action, target_type, target_id, reason,
   before jsonb, after jsonb, ip, created_at) — **append-only**: app
   role-оос UPDATE/DELETE REVOKE хийсэн байх. Эдгээр нь платформ-түвшний
   хүснэгт тул tenant RLS үл хамаарна — comment-оор тэмдэглэ.
3. **Нэвтрэлт**: нууц үг + TOTP заавал (WebAuthn боломжтой бол нэм,
   үгүй бол TOTP-оор эхэл); анхны superadmin-ийг CLI-гаар үүсгэнэ
   (`cmd/api`-д subcommand эсвэл тусдаа `cmd/operator-bootstrap`) — вэб
   бүртгэл байхгүй. Session idle 30 мин. Амжилтгүй оролдлого хэмжигдэж
   (`cp_login_attempts_total`), lockout үйлчилнэ.
4. **Step-up механизм**: аюултай үйлдлийн өмнө TOTP дахин асуудаг
   middleware — дараагийн үе шатууд үүнийг ашиглана.
5. **CP audit дүрэм**: `/cp/api`-ийн бичих үйлдэл бүр `operator_audit`-д
   бичигдээгүй бол огт хэрэгжихгүй гэсэн зарчмаар — handler-т биш, дундын
   давхаргад шийд (audit бичилт + үйлдэл нэг transaction-д).
6. **Тенантын жагсаалт/дэлгэрэнгүй (зөвхөн унших)**: хайлт, шүүлт; дэлгэрэнгүйд
   суусан аппууд, хэрэглэгчид, сүүлийн идэвх. Query бүр dbguard-ын тусдаа
   operator горимоор — RLS-ийг бүхэлд нь унтраахгүй, зориулалтын query
   бүр explicit байна.

**CP-2 — Тенантын амьдралын мөчлөг + support:**

1. Тенант үүсгэх (нэр/slug/байгууллагын мэдээлэл + аппын багц + эхний
   админд урилга — одоогийн email verify урсгалаар), түдгэлзүүлэх
   (шалтгаантай; түдгэлзсэн тенантын нэвтрэлт, API бүгд 403),
   устгал: `deletion_scheduled_at` + 30 хоног grace, энэ хугацаанд
   сэргээх товч, export (тенантын өгөгдлийг JSON/CSV багцаар), хугацаа
   дуусахад цэвэрлэх job. Устгал товч нь step-up + хоёр дахь superadmin-ий
   батламж (`pending_approvals` хүснэгт) шаардана.
2. Quota: `tenant_quotas` (хэрэглэгчийн тоо, хадгалалт MB, AI дуудлага/сар),
   зөөлөн/хатуу горим; шалгалт нь холбогдох handler-уудад.
3. Support: хэрэглэгч хайх, нууц үг reset илгээх, session-ууд хүчингүй
   болгох, lockout тайлах — бүгд step-up + audit.
4. **Impersonation**: шалтгаан заавал, 30 минутын хугацаатай тусгай
   session; тенантын UI-д улбар шар banner (frontend-д session төрлөөс
   мэдэрнэ); бичих үйлдэл бүр тенантын audit-д давхар "impersonated"
   тэмдэгтэй; тенантын админд мэдэгдэл. Зөвхөн support+ эрх, step-up-тай.

**CP-3 — Динамик тохиргоо + feature flags:**

0. **Нэвтрэлтийн горим `platform.access_mode`** (`public|private`,
   анхдагч `private`) — platform_settings-ийн эхний бөгөөд жишиг тохиргоо:
   - Auth давхаргад нэг л шалгалт: бүртгэл ҮҮСГЭДЭГ бүх зам — email
     signup, eID/ДАН-ы JIT provisioning, `SSO_CLIENT_TENANT`-ийн JIT,
     `EID_JIT_TENANT_SLUG` — private үед хаагдана. Байгаа бүртгэлтэй
     хэрэглэгчийн нэвтрэлтэд горим нөлөөлөхгүй.
   - Тенантын админы урилгын урсгал private үед хэвээр ажиллана (урилга =
     урьдчилан бүртгэл). Урилгаар account үүсэх нь JIT-д тооцогдохгүй.
   - Private үед бүртгэлгүй хүн нэвтрэх гэвэл 403 + i18n-тэй ойлгомжтой
     мессеж ("Энэ платформ хаалттай горимд байна...");
     frontend нэвтрэх дэлгэц горимоо public config endpoint-оос уншиж
     бүртгүүлэх товч/холбоосыг нуана.
   - `DEMO_MODE=true` үед demo seeder ажиллахын тулд горим нь public
     байхыг шаардана — зөрчилтэй тохиргоог boot үед лог + CP нүүрэнд
     анхааруулга болго.
   - Тест: private үед signup 403, eID mock-оор шинэ хүн JIT үүсэхгүй,
     урилгаар үүснэ, public руу шилжмэгц (restart-гүй) signup нээгдэнэ —
     4 integration тест.
1. `platform_settings`: түлхүүр бүр төрөл/validation/тайлбартай registry
   Go кодод; утга DB-д; өөрчлөлт бүр түүхтэй (`platform_settings_history`),
   нэг товчоор rollback; backend Redis invalidation-оор (эсвэл 30 сек TTL)
   restart-гүй шинэчилнэ. Эхний нүүдэл: SESSION_IDLE_TIMEOUT,
   CATALOG_SYNC_INTERVAL, GEMINI_MODEL зэрэг аюулгүй env-үүд (env нь
   fallback хэвээр — DB утга байвал давуу). **Нууц утга registry-д
   бүртгэгдэхийг Go типийн түвшинд боломжгүй болго** (secret төрөл байхгүй).
2. `feature_flags`: нэр, тайлбар, эзэмшигч, төрөл (`release|kill_switch|
   experiment`), төлөв, тенант бүрийн override, хувиар rollout
   (tenant_id-ийн hash-аар тогтвортой), `expires_at` — хугацаа хэтэрсэн
   flag CP нүүрэнд сануулга болж гарна. Go-д `flags.Enabled(ctx, "key")`
   helper. Модулийн kill switch нь app gating-тэй уялдана.
3. Maintenance mode (платформ/тенант түвшин, banner + зөвхөн унших) ба
   зарлал broadcast (banner, сонголтоор и-мэйл).

**CP-4 — Ажиглалтын тойм + operations:**

1. Нүүр: API error rate/latency, гадаад системүүдийн төлөв гэрэл
   (Prometheus-ын API-гаас query хийж эсвэл backend-ийн өөрийн хэмжүүрээс),
   идэвхтэй alert-ууд (Alertmanager API), диск/DB/Redis. Гүнзгий линкүүд
   Grafana руу.
2. Тенант бүрийн алдааны түвшин, background ажлын төлөв (товлосон
   тайлан, каталог синк), миграцийн түүх, ажиллаж буй image tag/sha.
3. **Deploy товч**: GitHub Actions workflow_dispatch API (`GITHUB_DEPLOY_
   TOKEN` env, зөвхөн энэ workflow-д эрхтэй fine-grained token) — tag
   сонгож өдөөнө, явцын линк харуулна. Step-up шаардана. Серверт exec,
   env засвар ОГТ хийхгүй.
4. Backup төлөв: сүүлийн backup цаг/хэмжээ (одоогийн backup механизмаас
   уншина; байхгүй бол эхлээд энгийн pg_dump cron + status файл нэмж
   тэмдэглэ), сүүлийн restore test огноог гараар бүртгэх талбар.
5. Каталог: синкийн төлөв, хувилбарын тархалт, kill switch (CP-3-ын flag).

**CP-5 — Metering:**

1. `usage_events` (tenant_id, metric, value, day, RLS-тэй) — өдөр тутмын
   aggregation job: идэвхтэй хэрэглэгч, хадгалалт, API дуудлага (одоогийн
   http хэмжүүрээс биш — DB-д тоолж), AI дуудлага, илгээсэн тайлан.
2. CP дээр тенант бүрийн хэрэглээний график (Recharts), quota-тай
   харьцуулсан харагдац, CSV export. Quota-гийн шалгалт (CP-2) энэ
   өгөгдлөөс уншдаг болгож нэгтгэ.

**Тест (Үе шат 5 нийтдээ)**: cp route-ууд энгийн хостоос 404/403; operator
auth + step-up урсгал; audit-гүйгээр бичих үйлдэл үл хэрэгжих; түдгэлзсэн
тенант 403; grace устгал сэргээгдэх; flag rollout hash тогтвортой;
impersonation banner/audit — эдгээрт integration тест заавал.

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
