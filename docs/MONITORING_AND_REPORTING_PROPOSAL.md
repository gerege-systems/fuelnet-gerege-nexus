# Мониторинг ба тайлангийн системийн санал

**Gerege Nexus**-т ажиглалт (observability) ба тайлангийн (reporting) давхарга
нэмэх — дэлхийн шилдэг туршлагын судалгаа ба үе шаттай хэрэгжүүлэх санал.

[Баримт бичгийн төв рүү буцах](README.md)

---

## 1. Одоогийн байдал

Платформ дээр өнөөдөр байгаа зүйлс:

| Байгаа | Байрлал | Тайлбар |
| --- | --- | --- |
| `/health`, `/ready` | `cmd/api` | Docker healthcheck ба deploy шалгалтад ашиглагдана |
| `/metrics` (Prometheus) | `platform/observability/metrics.go` | Ердөө 2 хэмжүүр: `http_requests_total`, `http_request_duration_seconds`. Cardinality хамгаалалт (routed pattern-оор л label үүсгэх) зөв хийгдсэн |
| Audit лог | `platform/audit/audit.go` | `slog.Info("AUDIT_EVENT", ...)` — зөвхөн stdout руу бичигдэнэ, хадгалагдахгүй, хайж болохгүй |
| Resilience хөдөлгүүр | `platform/resilience/` | Breaker, load shedder, retry ажилладаг ч **төлөвөө хэмжүүрээр гаргадаггүй** — breaker нээгдсэнийг хэн ч мэдэхгүй |

Дутуу байгаа зүйлс — эдгээр нь энэ саналын сэдэв:

- **Хэмжүүр цуглуулагч, dashboard, alerting алга.** `/metrics` endpoint байгаа ч
  түүнийг уншдаг Prometheus, харуулдаг Grafana, дуут дохио өгдөг Alertmanager
  сервер дээр байхгүй. Шөнө API унавал өглөө хэрэглэгч л хэлж мэдэгдэнэ.
- **Логийн төвлөрсөн сан алга.** Лог `docker logs` дотор л амьдарна — контейнер
  дахин үүсэхэд алга болно, тенант/хэрэглэгчээр хайх боломжгүй.
- **Tracing, error tracking алга.** Удаан хүсэлт хаана цаг иддэгийг, panic/алдаа
  хэдэн хэрэглэгчид тохиолдсоныг харах хэрэгсэлгүй.
- **Тенантад харагдах тайлан алга.** Billing, Inventory, Documents аппууд
  өгөгдөл хуримтлуулдаг ч нэгтгэсэн тайлан, dashboard, PDF/Excel export,
  товлосон тайлан гэсэн давхарга огт байхгүй. Odoo-оос app store-ын загварыг
  авсан атлаа Odoo-гийн хамгийн их хэрэглэгддэг давхарга болох reporting-ийг
  аваагүй байна.

---

## 2. Дэлхийн шилдэг туршлага — Мониторинг

### 2.1 Гурван тулгуур + OpenTelemetry стандарт

Орчин үеийн ажиглалт metrics / logs / traces гурван тулгуур дээр тогтдог бөгөөд
салбарын стандарт нь **OpenTelemetry (OTel)** болсон: кодоо нэг л удаа OTel-ээр
instrument хийвэл backend-ээ (Grafana, Datadog, аль ч vendor) солиход код
өөрчлөгдөхгүй. Go-д `otelhttp` middleware, `otelpgx` зэрэг бэлэн сангууд бий,
`log/slog`-ийг `otelslog` гүүрээр trace ID-тай холбодог — лог мөр бүр аль
хүсэлтэд хамаарахыг харуулна.

### 2.2 Юуг хэмжих вэ: Golden signals / RED

Google SRE-ийн **дөрвөн алтан дохио** (latency, traffic, errors, saturation),
сервис талаас нь **RED** (Rate, Errors, Duration) арга. Nexus-ийн одоогийн 2
хэмжүүр нь RED-ийн R ба D-г өгдөг, гэхдээ saturation (DB pool, Go runtime,
диск) болон бизнес хэмжүүр (нэвтрэлт, нэхэмжлэх, гарын үсэг) байхгүй.

### 2.3 Alerting: threshold биш, SLO + burn rate

Шилдэг туршлага бол "CPU 80% давлаа" маягийн threshold alert биш, **SLO
(Service Level Objective) + error budget** дээр суурилсан alerting. Google
SRE Workbook-ийн зөвлөдөг **multi-window multi-burn-rate** загвар (99.9%
SLO-д):

| Түвшин | Урт цонх | Богино цонх | Burn rate | Идсэн budget |
| --- | --- | --- | --- | --- |
| Page (яаралтай) | 1 цаг | 5 мин | 14.4× | 2% |
| Page | 6 цаг | 30 мин | 6× | 5% |
| Ticket (яаралтай биш) | 3 хоног | 6 цаг | 1× | 10% |

Хоёр цонх зэрэг хэтэрсэн үед л дохио өгдөг тул түр зуурын үсрэлтэд сэрээхгүй,
жинхэнэ асуудалд хурдан сэрээнэ — alert fatigue-ээс сэргийлэх гол арга.
Alert бүр **runbook**-той (юу шалгах, яаж засах) байх ёстой.

### 2.4 Single-server-т тохирох стек: LGTM monolithic mode

Nexus нэг сервер дээр Docker Compose-оор явдаг тул Kubernetes-д зориулсан
том стек биш, **Grafana LGTM стекийн single-binary (monolithic) горим**
тохиромжтой:

| Компонент | Үүрэг | Шалтгаан |
| --- | --- | --- |
| **Prometheus** | Хэмжүүр | Нэг серверт Mimir/Cortex илүүц — Prometheus дангаараа хангалттай, local retention 30-90 хоног |
| **Loki** | Лог | Label-only индексжүүлэлт — Elasticsearch-ээс хэд дахин хөнгөн, лог мөрөө индексгүй шахаж хадгална |
| **Tempo** | Trace | Индексгүй хадгалалт, богино retention (жишээ нь 48 цаг) хангалттай |
| **Grafana** | Dashboard + alert UI | Гурвууланг нэг дэлгэцэд нэгтгэж, metric↔log↔trace хооронд үсэрч болно |
| **Alloy** | Цуглуулагч | Docker контейнеруудын логийг Loki руу, OTel-ийг Tempo руу дамжуулна |
| **Alertmanager** | Мэдэгдэл | И-мэйл, Telegram, Slack руу групплэж, давхардлыг нэгтгэж илгээнэ |

Хөнгөн горимд компонент тус бүр ~1 core / 2 GB дотор багтдаг тул нийт стек
одоогийн серверийн хажууд 3-4 GB орчим RAM-д ажиллана. Мөн хостын түвшинд
`node_exporter`, `cAdvisor`, `postgres_exporter`, `redis_exporter` гэсэн
стандарт exporter-уудыг залгаж, nginx-ийн stub_status-ийг уншуулна.

### 2.5 Нэмэлт хоёр хэрэгсэл

- **Error tracking** — panic/exception-ийг stack trace, тохиолдсон хэрэглэгчийн
  тоогоор группэлж харуулах. Self-hosted Sentry хүчирхэг ч хүнд (олон
  контейнер); **GlitchTip** нь Sentry SDK-тэй ижил протоколтой, хөнгөн
  хувилбар — нэг сервер дээр илүү тохиромжтой. Frontend (Next.js) талд ч мөн
  SDK-г нь залгаж болно.
- **Гаднаас шалгах uptime** — сервер дотроо байгаа монитор сервертэйгээ хамт
  унадаг тул `/health`-ийг гаднаас шалгах **Uptime Kuma** (эсвэл өөр хостоос
  blackbox_exporter) байх ёстой. Энэ бол "монитор өөрөө унавал хэн мэдэх вэ"
  гэсэн асуултын хариулт.

---

## 3. Дэлхийн шилдэг туршлага — Тайлан

### 3.1 Хоёр төрлийн тайлан — хоёр өөр хэрэглэгч

1. **Бизнес тайлан** (тенантын хэрэглэгчид) — борлуулалт, нэхэмжлэх, агуулах,
   гарын үсгийн статистик; dashboard, pivot, PDF/Excel export, товлосон и-мэйл.
2. **Үйл ажиллагааны тайлан** (платформ оператор) — uptime/SLO-ийн сарын
   тайлан, тенант бүрийн идэвх, апп суулгалтын статистик.

### 3.2 Odoo-гийн загвар: тайлан бол модулийн нэг хэсэг

Odoo-д тайлан тусдаа систем биш — модуль бүр өөрийн **pivot/graph view**-тэй
ирдэг, PDF нь **QWeb** template-ээр, товлосон тайлан scheduler-ээр явдаг.
Хэрэглэгч нэг л системд дотор ажиллана. Энэ нь Nexus-ийн "модуль монолит,
модуль бүр өөрийн route-тай" гэсэн философитой яг таардаг загвар.

### 3.3 Embedded BI хэрэгслүүд ба тэдгээрийн хязгаар

Metabase, Apache Superset зэрэг self-hosted BI хэрэгслийг iframe/SDK-ээр
шигтгэх нь түгээмэл туршлага. Multi-tenant үед **row-level security**-ийг
BI хэрэгслийн түвшинд давхар тохируулах шаардлагатай бөгөөд энэ нь алдаа
гаргахад амархан (нэг буруу тохиргоо = өөр тенантын өгөгдөл харагдана).
Мөн: Metabase-ийн жинхэнэ RLS нь Pro/Enterprise лицензээр ирдэг, аль ч
хэрэгсэл монгол хэлний UI бүрэн биш, тусдаа JVM/Python процесс нэмэгдэнэ.
Тиймээс эдгээр нь **дотоод админ/оператор аналитикт** сайн, харин **тенантад
харагдах тайланд** Nexus шиг өөрийн RBAC, i18n, RLS-тэй платформд in-house
модуль илүү гэж дүгнэж байна (§4.3-т харьцуулалт бий).

### 3.4 OLTP-гоо унагахгүйгээр тайлагнах

Тайлангийн query нь үндсэн үйл ажиллагааны DB-г удаашруулах эрсдэлтэй тул
шилдэг туршлага нь:

- **Materialized view** — өдөр тутмын нэгтгэлүүдийг урьдчилан тооцоолж,
  `REFRESH MATERIALIZED VIEW CONCURRENTLY`-гээр түгжээгүй сэргээх (unique
  index шаардана);
- Тайлангийн query-д **statement timeout** тавьж, удаан query-г таслах;
- Хэмжээ өссөн үед **read replica** руу тайлангийн уншилтыг салгах — энэ бол
  дараагийн шат, эхний өдрөөс хэрэггүй;
- ClickHouse/DuckDB зэрэг OLAP сан руу гарах нь сая мөрөөс дээш, секунд
  доторх real-time аналитик хэрэгтэй болсон үеийн л шийдэл.

Nexus-ийн migration 00029-ийн row-level policy + `dbguard` нь тайлангийн
давхаргад шууд ашиглагдана: тайлангийн query ч мөн тенант хамгаалалттай
role дотор ажиллах тул "WHERE tenant_id мартсан тайлан" өөр тенантын тоог
харуулж чадахгүй.

### 3.5 Тенант дамнасан нэгдсэн тайлан (cross-tenant reporting)

Бодит хэрэгцээ: **нүүрсний уурхай 100 тээврийн компанитай гэрээтэй** —
компани бүр өөрийн тенант дотроо рейс, түлш, хүргэлтээ бүртгэдэг, харин
уурхай "Тээвэр" гэсэн **ганцхан нэгдсэн тайлан** харахыг хүснэ. Энэ бол
тенант тусгаарлалтын эсрэг урсгал тул дэлхийн туршлагад дараах зарчмуудаар
шийддэг (Snowflake/Redshift data sharing, Odoo multi-company consolidation,
B2B data sharing agreement-ийн загварууд):

1. **Default deny.** Тенант дамнасан уншилт хэзээ ч далд, автоматаар
   үүсэхгүй — RLS хэвээрээ. Нэгдсэн тайлан бол тусгай, зөвшөөрөлд
   суурилсан зам.
2. **Тодорхой, буцаагдах grant.** Өгөгдөл эзэмшигч тал (тээврийн компани)
   хүлээн авагч талд (уурхай) *яг аль тайланг, ямар хүрээгээр, хэдий
   хугацаанд* харуулахаа өөрөө зөвшөөрнө. Гэрээ дуусахад grant-ыг цуцалж,
   тэр дороо харагдахаа болино.
3. **Counterparty scope — өгөгдмөл хүрээ.** Уурхай тээврийн компанийн
   *бүх* үйл ажиллагааг биш, зөвхөн **өөртэй нь холбоотой** (өөрийнх нь
   гэрээн доорх) мөрүүдийн тайланг харна. Өөр уурхайд хийсэн тээвэр нь
   grant-ын хүрээнд огт орохгүй.
4. **Түүхий мөр биш — тайлангийн үр дүн.** Хүлээн авагч тал эзэмшигчийн
   хүснэгтээс шууд уншихгүй; тайлангийн хөдөлгүүр эзэмшигч тенант бүрийн
   контекст дотор query-г ажиллуулж, зөвхөн нэгтгэсэн үр дүнг нийлүүлнэ.
   Ингэснээр RLS-ийг сулруулах шаардлагагүй.
5. **Бүрэн audit.** Хэн, хэзээ, хэний өгөгдлөөс ямар тайлан татсан нь
   хоёр талдаа бүртгэгдэнэ — эзэмшигч тал "миний өгөгдлийг хэн харав"
   гэдгээ хардаг байх ёстой.

Хоёр хэлбэр байдгийг ялгах нь чухал: **(а) шаталсан нэгтгэл** — толгой
байгууллага өөрийн салбар/охин тенантуудын тайланг нэгтгэх (аймгийн суулгац
улсын нэгдсэн рүү холбогддог одоогийн SSO federation-ий логиктой ижил
чиглэл), **(б) гэрээт талуудын хоорондох grant** — уурхай/тээврийн кейс.
Хоёулаа нэг grant механизм дээр суух боломжтой, зөвхөн scope нь өөр
(охин тенант: full, гэрээт тал: counterparty).

Мөн нэг зүйлийг ялгаж хэлье: уурхай *өөрийн* тенант дотроо байгаа өгөгдөл
(өөрийн нэхэмжлэх, гэрээ, хүлээн авалт) дээрх "тээврийн зардлын тайлан" бол
энгийн дотоод тайлан — grant огт хэрэггүй. Grant зөвхөн **тээврийн
компаниудын тенант дотор бүртгэгддэг** өгөгдлийг (рейс, жин, GPS, хүргэлтийн
төлөв г.м.) нэгтгэх үед л хэрэгтэй.

---

## 4. Nexus-д тохирсон санал — үе шаттай

### Үе шат 1. Instrument-ээ гүйцээх (код, ~1 долоо хоног)

Стек босгохоос өмнө хэмжигдэхүүнээ баяжуулна — эс бөгөөс dashboard хоосон:

- `platform/observability`-д нэмэх: Go runtime (бэлэн collector), `pgxpool`
  stat (idle/acquired/wait), resilience хэмжүүр (`breaker_state`,
  `load_shed_total`, `retry_total`), гадаад системийн дуудлага
  (`external_request_duration_seconds{system="xyp|eid|dan|esign|gemini"}`)
  — төрийн интеграцын аль нь удааширч байгааг харах хамгийн чухал хэмжүүр;
- Бизнес counter-ууд: `logins_total{method}`, `invoices_created_total`,
  `documents_signed_total{rail}`, `ai_requests_total` (label-д tenant slug
  ОРУУЛАХГҮЙ — cardinality; тенант бүрийн задаргаа §Үе шат 4-ийн тайлангаар);
- Лог: production дээр `slog`-ийг JSON handler-тай болгож, request ID,
  tenant ID-г бүх мөрөнд оруулах;
- Audit: stdout-аас гадна `audit_events` хүснэгтэд бичдэг болгох — энэ нь
  дараа нь тайлангийн түүхий эд болно.

### Үе шат 2. Мониторингийн стек босгох (~1 долоо хоног)

`deploy/docker-compose.monitoring.yml` — үндсэн стекээс тусдаа compose файл,
унавал ч платформд нөлөөгүй:

```
Prometheus (retention 60d) ← /metrics, node_exporter, cAdvisor,
                              postgres_exporter, redis_exporter, nginx
Loki (retention 31d)       ← Alloy (Docker контейнер лог)
Grafana                    ← дээрх хоёр + dashboard-ууд provisioning-оор
                              (repo-д JSON-оор хадгалагдана — dashboard-as-code)
Alertmanager               ← и-мэйл / Telegram
Uptime Kuma                ← тусдаа хямд VPS дээр /health, /ready, TLS хугацаа
```

Эхний alert-ууд: API availability SLO 99.9% дээр §2.3-ын burn-rate хоёр
түвшин, диск >85%, Postgres холболт дүүрэлт, backup гүйцэтгэл, гадаад
системийн (ХУР, eID) алдааны хувь, TLS сертификатын хугацаа. Alert бүрд
`docs/RUNBOOKS.md` дотор runbook бичнэ.

### Үе шат 3. Tracing + error tracking (~1-2 долоо хоног, 2-р шатны дараа)

- OTel SDK + `otelhttp`/`otelpgx` залгаж Tempo руу (retention 48-72 цаг,
  sampling-тай), `otelslog`-оор лог↔trace холбох;
- GlitchTip (эсвэл багийн танил бол self-hosted Sentry) — backend panic ба
  Next.js клиентийн алдаа хоёуланг нь авна.

### Үе шат 4. Тайлангийн модуль `io.gerege.nexus.reports` (~4-6 долоо хоног)

Odoo-гийн загвараар **платформын өөрийн апп модуль** болгож бүтээнэ:

- **Тайлангийн тодорхойлолт** — модуль бүр (billing, inventory, documents,
  gov_services) өөрийн тайлангуудаа Go-гийн `Report` interface-ээр бүртгүүлнэ:
  нэр (7 хэлээр, одоогийн i18n бүтцээр), параметрүүд (хугацаа, агуулах г.м.),
  гаралтын багана. Апп стор-ын каталогт тайлан нь модулийнхоо хамт ирнэ;
- **Query давхарга** — гар бичмэл SQL (одоогийн `pgx` философи), tenant
  context + RLS дотор; нэгтгэлүүд эхэндээ шууд query, удааширсан нь
  materialized view + `REFRESH ... CONCURRENTLY` (шөнийн cron + statement
  timeout);
- **UI** — frontend-д `/reports` route: хүснэгт, pivot, Recharts граф;
  тенантад идэвхтэй аппуудын тайлан л харагдана (одоогийн app gating);
- **Export** — Excel (`excelize`), CSV; PDF-ийг Documents модулийн
  туршлагаар (эсвэл Gotenberg контейнер) — албан тайланд байгууллагын
  толгой, тамга тэмдгийн загвартай;
- **Товлосон тайлан** — "сар бүрийн 1-нд энэ тайланг энэ хаягуудад PDF-ээр"
  гэсэн бүртгэл; илгээлт нь одоогийн hosted email үйлчилгээгээр;
- **Эрх** — тайлан бүр RBAC permission-тэй (`reports.billing.view` г.м.),
  үзсэн/export хийсэн бүрийг `audit`-д бичнэ.

**Тенант дамнасан тайлан — `report_grants` механизм** (§3.5-ын зарчмаар):

```sql
report_grants (
  id, grantor_tenant_id,      -- өгөгдөл эзэмшигч (тээврийн компани)
  grantee_tenant_id,          -- хүлээн авагч (уурхай)
  report_key,                 -- аль тайлан ('transport.trips' г.м.)
  scope,                      -- 'counterparty' | 'full' (охин тенантад)
  counterparty_ref,           -- талуудыг холбох түлхүүр (регистрийн дугаар)
  valid_from, valid_until, revoked_at,
  created_by, accepted_by     -- хоёр талын баталгаажуулалт
)
```

Ажиллах зарчим: уурхайн хэрэглэгч "Тээвэр — нэгдсэн" тайланг нээхэд
хөдөлгүүр идэвхтэй grant бүхий 100 тенантыг олж, **тенант тус бүрийн
контекст дотор** (RLS хэвээр) тайлангийн query-г counterparty шүүлттэй
ажиллуулж, үр дүнг компаниар нь задаргаатай эсвэл нийт дүнгээр нэг
хүснэгтэд нэгтгэнэ. 100 тенант дээр давтах нь materialized view-тэй
хослуулахад асуудалгүй; үр дүнг нь хүлээн авагч талд түр кэшлэж болно.
Гүйлт бүр хоёр талын audit-д бичигдэнэ, тээврийн компани "Хандалтын
түүх" дэлгэцээс уурхай юу татсаныг харна.

Grant үүсэх UX нь гэрээний урсгалтай уялдана: уурхай хүсэлт илгээх →
тээврийн компанийн админ хүрээг нь хараад зөвшөөрөх — эсвэл Documents
модулиар гэрээ байгуулахад тайлан хуваалцах нөхцөл нь хавсарч, гарын
үсэг зурагдмагц grant автоматаар идэвхжинэ. Аль ч тохиолдолд цуцлах эрх
эзэмшигч талдаа үлдэнэ.

**Оператор тайлан** мөн энэ модулиар: тенант бүрийн идэвх, апп суулгалт,
хадгалалтын хэмжээ; SLO-ийн сарын тайлан нь Grafana dashboard + Prometheus
өгөгдлөөс автоматаар.

### 4.3 Яагаад Metabase/Superset биш, in-house вэ?

Хэл, тусдаа нэвтрэлт хоёр нь шийдвэрлэх шалгуур биш гэж тогтсон тул
харьцуулалт үндсэн хоёр л шалгуур дээр тогтоно — **тенант тусгаарлалт**
ба **тенант дамнасан grant логик**:

| Шалгуур | In-house модуль | Metabase | Superset |
| --- | --- | --- | --- |
| Тенант тусгаарлалт | Одоогийн RLS + dbguard шууд | Sandboxing — Pro/Enterprise лиценз | RLS бий ч тохиргооны алдаанд эмзэг |
| Тенант дамнасан grant (§3.5) | Платформ дотроо бүрэн загварчилна | Байхгүй — DB view-гээр өөрсдөө дуурайлгана | Байхгүй — мөн адил |
| PDF/товлосон тайлан, гэрээтэй уялдсан урсгал | Бүрэн хяналттай | Хэсэгчлэн | Хэсэгчлэн |
| Нэмэлт ажиллагааны зардал | Байхгүй (нэг бинари) | +JVM процесс | +Python/Redis/worker |
| Хөгжүүлэлтийн зардал | Их (4-6 д.х.) | Бага | Дунд |

Гол дүгнэлт: uurhai/тээврийн кейс шиг **зөвшөөрөлд суурилсан тенант
дамнасан тайлан** бол домэйн логик — аль ч BI хэрэгсэлд ийм ойлголт
байхгүй тул grant, scope, audit-ийн бүх логикийг яаж ч байсан платформ
талдаа (эсвэл DB view-ийн давхаргад гараар) бичих хэрэгтэй болно. Тэгэх
тусмаа render хийх давхаргыг нь ч платформдоо байлгах нь нэг-бинари
философид нийцнэ. Metabase-ийг **дотоод оператор аналитикт** (интернэтэд
гаргахгүй, тенантад үзүүлэхгүй) зэрэгцээ ашиглах нь хэвээр зөв сонголт.

---

## 5. Дараалал ба хүчин чармайлтын тойм

| Үе шат | Ажил | Хугацаа | Үр дүн |
| --- | --- | --- | --- |
| 1 | Хэмжүүр/лог instrument | ~1 д.х. | Хэмжигдэхүйц платформ |
| 2 | LGTM стек + alerting + Uptime Kuma | ~1 д.х. | Унасныг хэрэглэгчээс өмнө мэднэ |
| 3 | Tracing + GlitchTip | ~1-2 д.х. | Удаан/алдаатай хүсэлтийн шинжилгээ |
| 4 | Reports модуль | ~4-6 д.х. | Тенантад тайлан, export, товлосон илгээлт |
| 4б | Тенант дамнасан grant + нэгдсэн тайлан | ~2 д.х. (4-ийн дараа) | Уурхай маягийн нэгдсэн "Тээвэр" тайлан |

1-2 үе шат нь хамгийн өндөр өгөөжтэй, бие даасан тул эхэлж хийх нь зүйтэй.
4-р шат нь бусдаасаа хамааралгүй тул зэрэгцээ явж болно.

---

## 6. Эх сурвалж

- [Google SRE Workbook — Alerting on SLOs](https://sre.google/workbook/alerting-on-slos/)
- [Self-Hosted Grafana Observability Backends: Loki vs Mimir vs Tempo](https://www.pistack.xyz/posts/2026-05-17-self-hosted-grafana-observability-backends-loki-mimir-tempo-cortex-guide/)
- [How to Build a Complete LGTM Stack with OpenTelemetry](https://oneuptime.com/blog/post/2026-02-06-lgtm-stack-opentelemetry/view)
- [OpenTelemetry Go Instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/)
- [OpenTelemetry-Native Logging in Go with otelslog](https://www.dash0.com/guides/opentelemetry-logging-in-go)
- [SRE alerting best practices — incident.io](https://incident.io/blog/sre-alerting-best-practices)
- [Building a Production Observability Stack (Grafana/Prometheus/Loki)](https://tobias-weiss.org/content/devops/production-observability-stack-grafana-prometheus-loki/)
- [Embedded Analytics: Multi-Tenancy, RLS, Tools & Pricing](https://www.toucantoco.com/en/blog/embedded-analytics-multi-tenancy-row-level-security-pricing)
- [Multi-Tenant Analytics — DataTako](https://datatako.com/blog/multi-tenant-analytics)
- [Metabase — Row and column security](https://www.metabase.com/docs/latest/permissions/row-and-column-security)
- [Apache Superset vs Metabase: 2026 Decision Framework](https://www.padiso.co/blog/apache-superset-vs-metabase-2026-decision-framework/)
- [Odoo Reports & Dashboards: QWeb, Pivot, Graph Views](https://www.odooskillz.com/blog/odoo-skillz-insights-1/odoo-custom-reports-dashboards-qweb-pivot-graph-guide-2026-342)
- [Real-time customer-facing analytics on Postgres without slowing OLTP](https://clickhouse.com/resources/engineering/real-time-analytics-postgres)
- [Refreshing PostgreSQL Materialized Views Without Downtime](https://dev.to/data_with_jelimo/refreshing-postgresql-materialized-views-without-downtime-28n6)
- [Self-Host Sentry or GlitchTip](https://danubedata.ro/blog/self-host-sentry-glitchtip-error-tracking-2026)
- [Multi-tenant patterns in Amazon Redshift using data sharing](https://aws.amazon.com/blogs/big-data/implementing-multi-tenant-patterns-in-amazon-redshift-using-data-sharing/)
- [Multitenant SaaS Patterns — Azure SQL](https://learn.microsoft.com/en-us/azure/azure-sql/database/saas-tenancy-app-design-patterns?view=azuresql)
- [Multi-Tenant Analytics: Architecture Guide for SaaS Platforms](https://www.usedatabrain.com/blog/multi-tenant-analytics)
- [Legal structures for B2B data sharing](https://gowlingwlg.com/en/insights-resources/articles/2023/data-unlocked-legal-structures)
