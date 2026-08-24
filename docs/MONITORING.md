# Мониторинг — ажиглалтын стек

Gerege Nexus-ийн хэмжүүр, лог, дохиоллын систем: юу хаана байдаг, яаж асаах,
шинэ хэмжүүр яаж нэмэх.

[Баримт бичгийн төв рүү буцах](README.md) ·
[Дизайны санал](MONITORING_AND_REPORTING_PROPOSAL.md) ·
[Runbook-ууд](RUNBOOKS.md)

---

## 1. Юу байдаг вэ

Стек нь **үндсэн платформоос бүрэн тусдаа** compose файлд байна. Энэ нь
санамсаргүй биш: мониторинг бүхэлдээ унасан ч платформ мэдэхгүй, ямар ч
хүсэлтийн зам дээр эдгээрийн нэг нь ч байхгүй, `down` хийх нь ямар ч цагт
аюулгүй үйлдэл.

| Компонент | Контейнер | Порт (loopback) | Үүрэг |
| --- | --- | --- | --- |
| Prometheus | `gerege_nexus_prometheus` | 9091 | Хэмжүүр, 60 хоног (20 GB тааз) |
| Alertmanager | `gerege_nexus_alertmanager` | 9093 | Групплэх, дарах, илгээх |
| Loki | `gerege_nexus_loki` | 3100 | Лог, 31 хоног |
| Alloy | `gerege_nexus_alloy` | 12345 | Docker логийг Loki руу |
| Tempo | `gerege_nexus_tempo` | 3200, 4318 | Trace, 72 цаг |
| Grafana | `gerege_nexus_grafana` | 3009 | Dashboard, лог, trace |
| node_exporter | `gerege_nexus_node_exporter` | 9100 | Хостын CPU, RAM, диск |
| cAdvisor | `gerege_nexus_cadvisor` | 8085 | Контейнер бүрийн хэрэглээ |
| postgres_exporter | `gerege_nexus_postgres_exporter` | 9187 | `pg_stat_*` |
| redis_exporter | `gerege_nexus_redis_exporter` | 9121 | Redis |

Бүх порт **зөвхөн 127.0.0.1** дээр bind хийгдсэн. Гаднаас хандах ганц зам нь
nginx-ээр гарсан Grafana (§4) эсвэл SSH tunnel.

### Хэмжүүр хаанаас ирдэг вэ

Платформ өөрөө `/metrics` дээр дараахыг гаргана
(`backend/internal/kernel/telemetry/`):

| Хэмжүүр | Тайлбар |
| --- | --- |
| `http_requests_total{method,path,status}` | `path` нь chi-ийн routed pattern — түүхий URL биш |
| `http_request_duration_seconds{method,path}` | |
| `pgxpool_*` | Холболтын pool: эзлэгдсэн, сул, нийт, хүлээлт |
| `external_request_duration_seconds{system,operation,status}` | ХУР, eID, ДАН, eSign, Gemini, и-мэйл баталгаажуулалт |
| `logins_total{method,result}` | password, eid, dan, google, sso |
| `invoices_created_total` | |
| `documents_signed_total{rail,result}` | rail: EID, DAN, HSM |
| `ai_requests_total{kind}` | copilot, chat, stt, tts, translate, forecast |
| `resilience_load_shed_total`, `resilience_in_flight_requests` | |
| `resilience_retry_total{name}` | |
| `go_*`, `process_*` | client_golang-ийн бэлэн collector-ууд |

**Аль ч label-д тенант байхгүй.** Тенант ID эсвэл slug нь label болвол
time series-ийн тоо байгууллагын тоогоор үржинэ, гарсан байгууллагын series нь
retention дуустал үлдэнэ. Тенантаар задарсан тоог тайлангийн модулиас авна —
тэнд мөр устгаж болдог, time series-д тэр боломж байхгүй.

---

## 2. Босгох

### 2.1 Урьдчилсан нөхцөл

Платформын стек ажиллаж байх ёстой (`docker-compose.prod.yml`), миграцууд
**00044** хүртэл хийгдсэн байх ёстой.

### 2.2 Нэг удаагийн алхам — өгөгдлийн сангийн нууц үг

Миграц 00044 нь `monitoring` гэсэн role-ыг `pg_monitor` эрхтэй, **нууц үггүй**
үүсгэдэг. Нууц үг нь репод байх боломжгүй тул оператор нэг удаа өгнө:

```bash
PASSWORD=$(openssl rand -base64 24)
docker exec -i gerege_nexus_postgres psql -U postgres -d platform_db \
  -c "ALTER ROLE monitoring WITH PASSWORD '$PASSWORD'"
echo "MONITORING_DB_PASSWORD=$PASSWORD"
```

Гарсан утгыг `.env.monitoring`-д бичнэ.

Энэ role нь `pg_stat_*` уншина, өөр юу ч биш — ямар ч хүснэгт, ямар ч тенантын
мөрийг харахгүй. Exporter-ыг `postgres` superuser-ээр ажиллуулах нь хамгийн
түгээмэл алдаа: тэр үед мониторингийн контейнер эвдрэлд орох нь өгөгдлийн
сангийн бүрэн хандалт алдагдсантай тэнцэнэ.

### 2.3 Орчин

```bash
cd /opt/open-gerege-nexus
cp deploy/.env.monitoring.example .env.monitoring
# GRAFANA_ADMIN_PASSWORD ба MONITORING_DB_PASSWORD хоёрыг бөглөнө
chmod 600 .env.monitoring
```

### 2.4 Асаах

```bash
docker compose -f deploy/docker-compose.monitoring.yml \
  --env-file .env.monitoring up -d
```

Шалгах:

```bash
# Бүх scrape target "up" эсэх
curl -s localhost:9091/api/v1/targets | jq '.data.activeTargets[] | {job:.labels.job, health}'

# Alertmanager ямар сувагтай болсон
docker logs gerege_nexus_alertmanager 2>&1 | grep 'notification channels'

# Loki лог хүлээж авч байгаа эсэх
curl -s 'localhost:3100/loki/api/v1/labels' | jq
```

### 2.5 Унтраах

```bash
docker compose -f deploy/docker-compose.monitoring.yml down
```

Өгөгдөл нэрлэсэн volume-д үлдэнэ (`gerege_nexus_monitoring_*`). `down -v` нь
60 хоногийн хэмжүүр, 31 хоногийн логийг устгана — буцаах арга байхгүй.

---

## 3. Сүлжээ

Мониторингийн стек нь платформын Docker сүлжээнд **гаднаас нэгдэнэ**
(`platform` network, `NEXUS_NETWORK`). Ингэснээр Prometheus нь
`gerege_nexus_backend:8080`-ыг, postgres_exporter нь `gerege_nexus_postgres`-ыг
контейнерийн нэрээр шууд дуудна — хостын порт руу тойрохгүй.

Сүлжээний нэр нь платформын стек байрлах хавтаснаас гаралтай. Өөр бол:

```bash
docker network ls | grep nexus
# ...дараа нь .env.monitoring дотор NEXUS_NETWORK=<нэр>
```

---

## 4. Grafana руу орох

**SSH tunnel (санал болгож буй).** Нэмэлт хандалтын гадаргуу үүсгэхгүй:

```bash
ssh -L 3009:127.0.0.1:3009 <server>
# дараа нь http://localhost:3009
```

**nginx-ээр.** Гаднаас байнга хэрэгтэй бол vhost-д snippet-ийг оруулна:

```nginx
include snippets/nexus-monitoring.conf;
```

Дараа нь `https://nexus.gerege.mn/grafana/`. Snippet нь Grafana-г л гаргана —
Prometheus, Loki, cAdvisor, exporter-ууд гарахгүй. Prometheus-д ямар ч
нэвтрэлт байхгүй бөгөөд түүний query API нь асуусан хүн бүрт энэ суулгацын
бүх бүтцийг задлан хэлнэ.

Dashboard-ууд **"Gerege Nexus"** гэсэн хавтсанд өөрөө үүснэ:

| Dashboard | Хэзээ харах |
| --- | --- |
| **API тойм** | "Ямар нэг зүйл эвдэрсэн үү" — эхний дэлгэц |
| **Гадаад системүүд** | Асуудал бидний тал уу, тэдний тал уу |
| **Инфраструктур** | Удаашрал — хост, контейнер, Postgres, Redis |
| **Тэсвэрлэлт ба эзлэхүүн** | Ачаалал, pool, бизнесийн тоо |

---

## 5. Dashboard-as-code

Dashboard-ууд нь `deploy/monitoring/grafana/dashboards/*.json` дотор,
provisioning-оор ачаалагдана. `allowUiUpdates: false` — **браузераас хийсэн
засварыг хадгалж болохгүй**.

Энэ нь төвөгтэй боловч зориудаар: 02:00 цагт ослын үеэр хэн нэгний засварласан
панел үнэ цэнэтэй бөгөөд түүнийг хадгалах арга нь commit, дараагийн
`docker compose down -v` устгачих volume доторх мөр биш.

Засварлах урсгал:

1. Grafana дотор панелаа засна;
2. Dashboard → **Export → Save to file** (эсвэл JSON Model-ыг хуулна);
3. Репо доторх файлыг солино;
4. Grafana хавтсыг 30 секунд тутам дахин уншдаг тул серверт файл шинэчлэхэд
   restart шаардлагагүй.

Datasource-ийн `uid` (`nexus-prometheus`, `nexus-loki`) нь тогтмол.
Өөрчилвөл тэдгээрийг нэрлэсэн панел бүр чимээгүй эвдэрнэ.

---

## 6. Лог хайх

Loki-д **label-ууд нь индекс**: `container`, `service`, `level`, `deployment`,
`job`. Лог мөрийн доторх зүйл индексжээгүй бөгөөд query үед шүүгдэнэ.

```logql
# Backend-ийн бүх алдаа — level нь label тул энэ нь индексээр шүүгдэнэ
{container="gerege_nexus_backend", level="error"}

# Нэг хүсэлтийн бүх мөр — request_id нь хариуны X-Request-Id толгойд байдаг
{container="gerege_nexus_backend"} | json | request_id = "abc123"

# Нэг байгууллагын үйлдлүүд
{container="gerege_nexus_backend"} | json | tenant_id = "<uuid>"

# Audit-ийн мөрүүд
{container="gerege_nexus_backend"} | json | msg = "AUDIT_EVENT"
```

`request_id`, `tenant_id`-г **label болгож болохгүй**. Тэдгээр нь Loki-г
өөрийнхөө зайлсхийхийг зорьсон бүрэн текстийн индекс болгож хувиргана —
утга бүр тусдаа stream үүсгэнэ.

---

## 7. Шинэ хэмжүүр нэмэх

1. **Cardinality-г эхэлж бод.** Label бүрийн боломжит утгын тоог үржүүл.
   Тенант, хэрэглэгч, ID, чөлөөт текст, түүхий зам — эдгээрийн аль нь ч
   label болохгүй. Утгын багц нь кодод бичигдсэн тогтмол байх ёстой.
2. Хэмжүүрийг `observability` пакетад зарлаж, `init()`-д бүртгэ
   (`business.go`, `external.go`, `resilience.go` жишээ).
3. Нэмэгдүүлэх дуудлагыг **бүх зам нийлдэг ганц цэгт** тавь — handler бүрт
   биш. Жишээ: Google-ийн бүх татгалзал `failGoogle`-аар, гарын үсгийн хоёр
   rail `store.markSigned`-аар өнгөрдөг.
4. Тест бич: `instrumentation_test.go` доторх `counterValue` туслах
   функцүүдийг ашигла.
5. Хэрэгтэй бол dashboard JSON-д панел нэм, alert дүрэм нэмбэл
   `RUNBOOKS.md`-д runbook **заавал** бич.

Гадаад систем нэмэх бол `external.go` доторх тогтмолд нэрийг нь нэмнэ —
`knownSystems`-д байхгүй нэр нь `other` болж нугалагдана, энэ нь label-ын
багцыг тогтмолгүй өргөжихөөс хамгаалдаг санаатай зан төлөв.

---

## 8. TLS-ийн хугацаа

Сертификат нь хостын nginx дээр байдаг тул түүнийг хэмжих ганц зам нь
node_exporter-ийн textfile collector. Дараах cron ажлыг **хост дээр** тавина
(blackbox exporter нэмэхгүй байх шалтгаан: сертификат нь энэ л хост дээр
байгаа, гаднаас шалгах ажлыг Uptime Kuma §9 хийнэ):

```bash
sudo mkdir -p /var/lib/node_exporter
sudo tee /usr/local/bin/nexus-tls-expiry.sh >/dev/null <<'EOF'
#!/bin/sh
# TLS-ийн дуусах хугацааг node_exporter-ийн textfile collector-т бичнэ.
# Атомик бичилт: node_exporter хагас бичигдсэн файлыг уншиж болохгүй.
set -eu
OUT=/var/lib/node_exporter/nexus_tls.prom
TMP=$(mktemp)
echo '# HELP nexus_tls_not_after_timestamp_seconds Certificate expiry, unix seconds' > "$TMP"
echo '# TYPE nexus_tls_not_after_timestamp_seconds gauge' >> "$TMP"
for dir in /etc/letsencrypt/live/*/; do
    domain=$(basename "$dir")
    [ -f "$dir/cert.pem" ] || continue
    end=$(openssl x509 -enddate -noout -in "$dir/cert.pem" | cut -d= -f2)
    epoch=$(date -d "$end" +%s)
    echo "nexus_tls_not_after_timestamp_seconds{domain=\"$domain\"} $epoch" >> "$TMP"
done
mv "$TMP" "$OUT"
chmod 644 "$OUT"
EOF
sudo chmod +x /usr/local/bin/nexus-tls-expiry.sh
sudo sh -c 'echo "17 4 * * * root /usr/local/bin/nexus-tls-expiry.sh" > /etc/cron.d/nexus-tls-expiry'
sudo /usr/local/bin/nexus-tls-expiry.sh
```

Энэ ажил ажиллахгүй бол `NexusTLSExpiryUnknown` дохио өгнө — "хэмжилт байхгүй"
нь "бүх зүйл хэвийн"-тэй яг адилхан харагддаг гэдгийг сануулах зорилготой.

---

## 9. Гаднаас шалгах — Uptime Kuma

**Энэ репод deploy хийгдэхгүй, зориудаар.** Энэ хост дээр ажиллаж байгаа
монитор нь хост унахад хамт унана — "монитор өөрөө унавал хэн мэдэх вэ"
гэсэн асуултын хариулт нь өөр хаяган дээр байх ёстой.

Хямд VPS эсвэл өөр үүлэн дээр:

```bash
docker run -d --restart unless-stopped \
  -p 3001:3001 \
  -v uptime-kuma:/app/data \
  --name uptime-kuma louislam/uptime-kuma:1
```

Тохируулах шалгалтууд:

| Төрөл | Хаяг | Давтамж | Тайлбар |
| --- | --- | --- | --- |
| HTTP(s) | `https://nexus.gerege.mn/health` | 60с | `"status":"ok"` гэсэн үг агуулсан эсэхийг шалга |
| HTTP(s) | `https://nexus.gerege.mn/ready` | 60с | Өгөгдлийн сангийн хүртээмж |
| TLS хугацаа | дээрхтэй адил | — | Kuma өөрөө сертификатыг хардаг, 14 хоногийн сануулга |
| HTTP(s) | `https://nexus.gerege.mn/grafana/login` | 300с | Мониторинг өөрөө амьд эсэх |

Мэдэгдлийг **энэ хостоос гарсан** суваг руу тохируул — SMTP нь энэ серверийнх
байвал уналтын үед мэдэгдэл ч явахгүй.

---

## 10. Ослын үед

1. **Grafana → "API тойм"** — алдааны хувь, p95, аль зам.
2. Гадаад систем сэжигтэй бол **"Гадаад системүүд"**.
3. Удаашрал бол **"Инфраструктур"** → DB pool, Postgres холболт, диск.
4. Дохио ирсэн бол [`RUNBOOKS.md`](RUNBOOKS.md) доторх тухайн alert-ын
   хэсгийг нээ — alert бүрийн `runbook` annotation нь яг тэр холбоос.
5. Лог хэрэгтэй бол Grafana → Explore → Loki, §6-ийн query-үүд.

---

## 11. Trace — OpenTelemetry ба Tempo

### Асаах

Tempo нь мониторингийн стектэй хамт үргэлж асдаг ч **платформ түүн рүү юу ч
илгээхгүй** — тэр нь платформын шийдвэр. Асаахын тулд үндсэн стекийн `.env`-д:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://gerege_nexus_tempo:4318
OTEL_TRACES_SAMPLER_ARG=0.1
```

...дараа нь backend-ыг дахин асаана. Хоосон бол tracing нь **үнэхээр**
унтарсан: exporter байхгүй, batch processor байхгүй, background goroutine
байхгүй, код доторх span бүр no-op. Tempo огт ажиллуулахгүй суулгац
хэмжигдэхүйц зардал төлөхгүй.

### Sampling

Өгөгдмөл нь 10%. Бүгдийг авах нь ямар ч эзлэхүүн дээр буруу хариулт: хүсэлт
бүрийн доторх query бүр нэг span бөгөөд бүгд retention дуустал хадгалагдана.
10% нь хоцролтыг тодорхойлж, давтагдах удаан замыг барихад хангалттай — тэр
хоёр л зүйлийн төлөө trace уншдаг. **Тодорхой нэг** удаан хүсэлтийг олох
хэрэгтэй бол тэр нь логийн `request_id`-ийн ажил.

`/health`, `/ready`, `/metrics` гурав нь огт trace хийгддэггүй: Docker ба
Prometheus тэднийг 10-15 секунд тутам дууддаг тул 10% дээр ч бүх trace-ийн
дийлэнх нь тэд байх байсан.

### Гурвыг хооронд нь холбох

Grafana-д гурван datasource **хоорондоо холбогдсон**:

- Логийн мөр дэх `trace_id` → **Trace** товч (derived field);
- Trace дотроос → тухайн container-ийн лог, тухайн үеийн;
- Trace дотроос → үйлчилгээний RED хэмжүүр;
- Хоцролтын графикийн удаан цэг → түүнийг удаашруулсан trace (exemplar).

Энэ гурвалсан аялал бол зөвхөн эхнийхийг нь биш гурвууланг нь ажиллуулж
байгаагийн бүх шалтгаан.

### Юуг trace хийхгүй вэ

**Query-ийн параметр огт бичигдэхгүй** (`otelpgx`-ийн өгөгдмөл, түүнийг
өөрчилж болохгүй). Тэдгээр нь query ямар мөрийн тухай болохыг хэлдэг —
и-мэйл хаяг, регистрийн дугаар, орж ирж буй нууц үгийн хэш. Span нь Tempo-гийн
retention-ий турш хадгалагдаж, Grafana нээж чадах хүн бүрт уншигдана. SQL
текст өөрөө аюулгүй: тэнд зөвхөн `$1`, `$2` байна.

---

## 12. Алдааны бүртгэл — GlitchTip

Sentry-ийн протокол ярьдаг, түүнээс олон дахин хөнгөн self-hosted хувилбар.
**Тусдаа compose файл**: `deploy/docker-compose.glitchtip.yml`. Тусдаа
байгаа шалтгаан нь дөрвөн нэмэлт контейнер ба хоёр дахь Postgres — үүнийг
хүсэхгүй суулгац тугийн тухай уншилгүй татгалзаж чадах ёстой.

### Босгох

```bash
cd /opt/open-gerege-nexus
# .env.monitoring дотор GLITCHTIP_DB_PASSWORD ба GLITCHTIP_SECRET_KEY бөглөнө
docker compose -f deploy/docker-compose.glitchtip.yml --env-file .env.monitoring up -d

# Эхний хэрэглэгч (нээлттэй бүртгэл зориудаар хаалттай)
docker exec -it gerege_nexus_glitchtip_web ./manage.py createsuperuser
```

Дараа нь `http://localhost:8000` (SSH tunnel) дээр орж, байгууллага ба
төсөл үүсгээд DSN-ийг хуулна.

### Залгах

Backend — платформын `.env`-д:

```bash
SENTRY_DSN=http://<key>@gerege_nexus_glitchtip_web:8000/1
```

Frontend — DSN нь bundle дотор **build үед** шигтгэгддэг тул runtime
хувьсагч биш, image-ийн build argument:

```bash
docker build --build-arg NEXT_PUBLIC_SENTRY_DSN=<dsn> ...
```

Rendering server-ийн өөрийн алдааг runtime-ийн `FRONTEND_SENTRY_DSN`
хувьсагчаар өгнө (compose дотор `SENTRY_DSN` болж дамжина).

### Юу илгээгддэггүй вэ

Энэ бол PII-ийн заавал мөрдөх хил. Backend талд
`observability.scrubEvent`, frontend талд `beforeSend`:

| Хасагддаг | Яагаад |
| --- | --- |
| Query string | `/api/v1/verify/landed?ref=…` нь нэг удаагийн итгэмжлэл |
| Cookie, `Authorization` толгой | Амьд session, bearer token |
| Хүсэлтийн бие | Регистрийн дугаар, нууц үг, гарын үсэг зурах PDF |
| Хэрэглэгчийн и-мэйл, нэр, IP | Хүнийг нэрлэх шаардлагагүй |
| Session Replay (frontend) | Хараад байсан дэлгэцийн DOM-ыг бичдэг |

**Үлддэг**: tenant ID (хэдэн байгууллагад нөлөөлснийг тоолоход),
`request_id`, `trace_id`, route pattern, `User-Agent`. Толгойн жагсаалт нь
**allow-list** — ирээдүйд proxy нэмсэн толгой автоматаар гарахгүй.
`errortracking_internal_test.go` энэ бүхнийг шалгана.

---

## 13. Дараагийн үе шат

Тайлангийн модуль нь
[дизайны саналын](MONITORING_AND_REPORTING_PROPOSAL.md) 4-р үе шат.
