# App Store — ажиллуулах гарын авлага

> **Код энэ репод байхгүй.** Registry ба storefront нь
> [`appstore-gerege-mn`](https://gitlab.gerege.mn/gerege-line/gerege-core/appstore-gerege-mn),
> консол нь [`developer-gerege-nexus`](https://gitlab.gerege.mn/gerege-line/gerege-core/developer-gerege-nexus)
> репод, deploy нь тэдний GitLab CI-аар явна. Энэ баримт нь **nexus талын**
> тохиргоо (remote горим, түлхүүр pin) болон гэмтэл олоход хэвээр хэрэгтэй;
> сервисийн өөрийнх нь дэлгэрэнгүй нь тухайн репод.

`appstore.gerege.mn` (registry + storefront) ба `developer.gerege.mn` (хөгжүүлэгчийн
консол)-ыг асаах, шинэчлэх, буцаах бүх алхам. Архитектурын үндэслэл
`APPSTORE_SEPARATION_PLAN.md`-д; энд зөвхөн гар дээрх ажил.

---

## 1. Юу хаана ажилладаг вэ

| Сервис | Процесс | Loopback порт | Домэйн |
| --- | --- | --- | --- |
| Registry API | `/app/appstore` (backend image) | 8085 | `appstore.gerege.mn/api`, `/.well-known` |
| Storefront | `appstore-web` image | 3013 | `appstore.gerege.mn` |
| Хөгжүүлэгчийн консол | `developer-web` image | 3014 | `developer.gerege.mn` |
| Registry DB | `gerege_appstore_postgres` | 5439 | — |

Stack: `/opt/appstore`, compose файл нь `deploy/appstore/docker-compose.prod.yml`.

**Порт сонголт:** энэ хост дээр зургаан stack ажилладаг (nexus, nexus-ds, sso, salus,
app-js, appstore). 3008–3012, 5434–5438, 8082–8084, 8095–8096 аль хэдийн эзэлсэн тул
дээрх дөрвөн портыг сонгосон. Порт солих бол compose файл **ба** харгалзах nginx vhost
хоёуланг зэрэг өөрчилнө; rollout нь порт эзэлсэн эсэхийг урьдчилан шалгаж, хэн эзэлж
байгааг нэрлэж хэлнэ.
Nexus-ийн stack (`/opt/open-gerege-nexus`) хөндөгдөхгүй — тусдаа DB, тусдаа сүлжээ.

Registry нь платформын **яг тэр image**-ээс өөр binary-гаар ажиллана. Учир нь
каталогийн форматыг хоёр процесс хуваалцдаг: contract-ыг хоёр газар бичихийн оронд
хоёулаа нэг `appcatalog` төрлүүдийг ашиглана (`internal/appstore/contract_test.go`
үүнийг батална).

---

## 2. Анх удаа асаах

### 2.1 Гарын үсгийн түлхүүр

```bash
# appstore-gerege-mn репод:
go run ./cmd/catalog-sign -genkey
# SIGNING_KEY=<base64 private>        → registry-д
# APPSTORE_PUBLIC_KEY=<base64 public> → Nexus instance бүрт pin хийнэ
```

Хувийн түлхүүрийг **зөвхөн** registry мэднэ. Алдагдвал: шинэ хос үүсгэж, шинэ
`key_id`-тайгаар registry-д тавиад, instance бүрийн `APPSTORE_PUBLIC_KEY`-г
шинэчилнэ; хуучин түлхүүрээр зурагдсан каталог тэр даруй хүчингүй болно
(instance-ууд хаяад өөрсдийн cache/файл руу унана — эвдрэхгүй, зүгээр л update
авахаа болино).

### 2.2 Хаана ямар тохиргоо байдаг вэ

Сервисийн тохиргоо нь **түүний өөрийн репод** (GitLab CI/CD Variables):

| Репо | Хувьсагч |
| --- | --- |
| `appstore-gerege-mn` | `APPSTORE_SIGNING_KEY`, `APPSTORE_POSTGRES_PASSWORD`, `APPSTORE_ADMIN_EMAILS`, `DEPLOY_SSH_KEY`, `DEPLOY_HOST`, `DEPLOY_USER` |
| `developer-gerege-nexus` | `DEPLOY_SSH_KEY`, `DEPLOY_HOST`, `DEPLOY_USER` |

Nexus (GitHub) талд зөвхөн **хэрэглэгчийн** тал үлдэнэ:

| Нэр | Төрөл | Утга |
| --- | --- | --- |
| `APPSTORE_PUBLIC_KEY` | secret | Registry-гийн **нийтийн** түлхүүр (pin) |
| `APP_CATALOG_URL` | variable | `https://appstore.gerege.mn/api/v1/registry` |
| `CATALOG_SYNC_INTERVAL` | variable | Анхдагч `1h` |
| `CONSOLE_ORIGIN` | variable | Консолын OAuth2 client-ийг бүртгэхэд |

**Түлхүүр эргүүлэх үед хоёуланг зэрэг солино:** GitLab дахь `APPSTORE_SIGNING_KEY`
ба GitHub дахь `APPSTORE_PUBLIC_KEY` нь нэг хосын хоёр хэсэг. Ганцыг нь сольвол
instance-ууд каталогийг хаяж, өөрсдийн cache/файлаараа үлдэнэ — эвдрэхгүй ч
шинэчлэлт зогсоно.

### 2.3 DNS ба TLS

`appstore.gerege.mn`, `developer.gerege.mn` → production хостын A бичлэг.
`deploy-appstore.yml` нь vhost-уудыг `/etc/nginx/sites-available` руу тавиад
certbot ажиллуулна. Deploy хэрэглэгчид passwordless sudo байхгүй бол алхам нь
**warning өгөөд өнгөрнө** (аппууд loopback дээр аль хэдийн боссон байна) —
дараах хоёр командыг гараар:

```bash
sudo cp /opt/appstore/nginx/appstore.gerege.mn.conf /etc/nginx/sites-available/
sudo ln -sf /etc/nginx/sites-available/appstore.gerege.mn.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d appstore.gerege.mn        # developer.gerege.mn-д мөн адил
```

Certbot нь vhost файлыг өөрөө засварладаг тул дараагийн deploy нь
"managed by Certbot" гэсэн файлыг **дахин бичихгүй** (TLS блокийг устгахгүйн тулд).
Vhost-ын агуулгыг өөрчлөх шаардлагатай бол гараар нэгтгэнэ.

### 2.4 Эхний өгөгдөл

Registry анх боссондоо DB хоосон бол image дотор явж буй `catalog/apps.json`-оос
8 аппыг `gerege` publisher нэрээр импортолж, stable сувагт нийтэлнэ. Дахин
ажиллуулахад юу ч давхардуулахгүй (нийтлэгдсэн хувилбар өөрчлөгддөггүй).

---

## 3. Nexus-ийг registry-гээс хооллох

Nexus талын код бэлэн (`appcatalog.Provider`). Асаах:

```
APP_CATALOG_URL=https://appstore.gerege.mn/api/v1/registry
APPSTORE_PUBLIC_KEY=<base64 public>
CATALOG_SYNC_INTERVAL=1h            # анхдагч
CATALOG_CACHE_PATH=/var/lib/nexus/catalog.cache.json
```

Асаахын өмнө шалгах гурван зүйл (§7-ийн test matrix — бүгд туршигдсан):

1. **Registry унтраа** → instance дискний cache-ээ үзүүлнэ.
2. **Cache устсан + registry унтраа** → image доторх `catalog/apps.json`.
3. **Түлхүүр буруу** → remote хариуг бүхэлд нь хаяна.

Гурвуулан дээр **boot амжилттай**, суулгачихсан аппууд ажиллаж байх ёстой.
Эхний долоо хоногт `catalog/apps.json`-оо registry-тэй ижил байлгах нь давхар
хамгаалалт болно.

Гараар sync: `POST /api/v1/admin/store/sync` (тенантын админ).

---

## 4. Хувилбар нийтлэх урсгал

1. Publisher `developer.gerege.mn` дээр Gerege бүртгэлээрээ нэвтэрнэ.
2. Publisher бүртгэл үүсгэнэ (нэг хүнд нэг publisher).
3. Апп нэмнэ (id нь reverse-DNS, өөрчлөгдөхгүй).
4. Manifest илгээнэ → `in_review`.
5. `APPSTORE_ADMIN_EMAILS`-д байгаа хүн хяналтын дараалалаас нийтэлнэ.
6. Nexus instance-ууд дараагийн sync-ээр (эсвэл админы товчоор) авна.

Manifest-ыг registry нь Nexus-ийн ашигладаг **яг тэр** шалгалтаар шалгана
(`appcatalog.ValidateManifest`). Тиймээс энд зөвшөөрөгдсөн зүйл тэнд бас
зөвшөөрөгдөнө — instance-ууд чимээгүй хоцрох эрсдэлийг үүнээр хаасан.

---

## 4.1 Түлхүүрийн нөөц ба эргэлт

`APPSTORE_SIGNING_KEY`-ийн хувийн хэсгийг **offline** нөөцлөнө (secret manager
эсвэл битүүмжилсэн хуулбар). Алдагдвал сэргээх зам байхгүй — шинэ хос үүсгэж,
шинэ `SIGNING_KEY_ID`-тайгаар registry-д тавиад, instance бүрийн
`APPSTORE_PUBLIC_KEY`-г шинэчилнэ. Хуучин түлхүүрээр зурсан каталогийг
instance-үүд шууд хаяж, өөрсдийн cache/файл руугаа унана — эвдрэхгүй, харин
шинэчлэлт зогсоно.

## 4.2 `VERSION` build-arg (одоогоор бүү дамжуул)

`deploy/Dockerfile` нь `--build-arg VERSION=` дэмждэг ба уг утга нь
`platform.PlatformVersion` болж, **manifest шалгалтаар semver гэж уншигддаг**.
Repo-д git tag байхгүй тул `git describe` нь `31e63f30` маягийн утга буцаана —
энэ нь semver биш тул instance boot дээр `invalid platform version` алдаагаар
унана. Тиймээс CI-д дамжуулахын өмнө эхлээд semver tag-ийн схем (жишээ нь
`v1.2.0`) тогтоож, `git describe --tags --abbrev=0` хэлбэрээр авах ёстой.

## 5. Гэмтэл олох

| Шинж тэмдэг | Хаанаас харах |
| --- | --- |
| Instance update авахаа болив | `docker logs gerege_appstore_registry`; instance дээр `catalog:` гэсэн WARN мөрүүд |
| `signature does not verify` | Instance-ийн `APPSTORE_PUBLIC_KEY` ↔ registry-ийн `SIGNING_KEY` хос таарахгүй байна |
| Storefront хоосон | `curl 127.0.0.1:8085/api/v1/registry/apps` — registry үү, storefront уу гэдгийг ялгана |
| Консол `401` | id_token хугацаа дуусав (1 цаг) — дахин нэвтэрнэ |
| Консол `403` | `APPSTORE_ADMIN_EMAILS`-д байхгүй хүн хяналтын үйлдэл хийхийг оролдов |
| Каталог хоцорсон | `catalog_snapshots`-ын `revision` ба `registry_state.revision` — зөрвөл дараагийн хүсэлтэд дахин угсрагдана |
| Sync ажиллаж байгаа эсэх | Nexus-д Тохиргоо → Суулгасан аппууд дээрх мөр: эх сурвалж, сүүлд шалгасан цаг, алдаа. API: `GET /api/v1/admin/store/status` |
| Апп шинэчлэгдэхгүй байна | Тухайн мөрөнд "тогтоосон" badge ба шалтгаан харагдана; "Зөвшөөрч шинэчлэх" дарж шийднэ |

Буцаах: `deploy-appstore.yml` → `workflow_dispatch` → өмнөх commit sha-г `tag`-аар
өгнө. Image бүр sha tag-тай тул буцаалт нь шинэ build шаарддаггүй.
