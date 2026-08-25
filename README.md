> **Nexus FuelNet — `fuelnet.gerege.mn`**
>
> Энэ репо бол [Gerege Nexus](https://github.com/gerege-systems/open-gerege-nexus)
> платформын **форк**: шатахуун түгээх сүлжээнд зориулсан байрлуулалт.
> Хожуулсан код нь цөмийнхтэй ижил — ялгаа нь брэнд, домэйн, гаргалтын урсгал:
>
> * `deploy/nginx/fuelnet.gerege.mn.conf` — энэ стекийн vhost
> * `.github/workflows/deploy.yml` — 38.180.120.144 руу гаргадаг
> * Нэвтрэлт нь `nexus.gerege.mn` (SSO_CLIENT_*), өөрөө хэн болохыг шийдэхгүй
>
> Цөмөөс өөрчлөлт татахдаа `git fetch upstream && git merge upstream/main`.
> Дээрх гурваас өөр газар салангид явах бүр нэгдэх ажлыг л нэмнэ.
>
> **Байрлуулалт.** Хост `38.180.120.144`, `/opt/fuelnet-gerege-nexus/`.
> Портууд цөмийн анхдагчаараа (5434 · 6381 · 8082 · 3008) — энэ машин
> зөвхөн FuelNet-ийнх. Образууд GHCR дээр энэ репогийн нэрээр. `main`
> дээр push хийхэд CI явж, амжилттай бол Deploy өөрөө гарна.
>
> **Нэвтрэлт.** `nexus.gerege.mn`-ий клиент — `SSO_CLIENT_LOCAL_LOGIN=false`,
> өөрөөр хэлбэл дотоод нууц үгээр орох замгүй. Provider унасан үед буцаж
> ороход тэр хувьсагчийг `true` болгоод backend-ыг дахин асаана; апп
> суулгаж, тенант үүсгэх ажил бүр яг тэр замаар хийгддэг.

# Gerege Nexus

**Үйлчилгээ, үйл ажиллагаа, системийн нэгдсэн платформ**

**Gerege Nexus** нь төрийн болон хувийн хэвшлийн байгууллагын үйлчилгээ, үйл
ажиллагаа, систем, өгөгдлийг нэгтгэх модульт платформ юм. Cloud-native
экосистемээс санаа авсан, өндөр бүтээмжтэй, Монгол Улсын цахим дэд бүтэц (ДАН,
E-ID, ХУР / XYP)-тэй шууд холбогдох боломжтой, **монгол хэлийг үндсэн хэл
болгосон** нээлттэй эхийн шийдэл.

*Nexus* гэдэг нь холбох цэг — байгууллага, үйлчилгээ, ажлын урсгал, систем,
хэрэглэгч, өгөгдөл нэг дор уулзах цэгийг хэлнэ. Платформ өөрөө нэг салбарт
зориулагдаагүй: дээр нь ажиллах модулиуд л тухайн байгууллагын хэрэгцээг
тодорхойлно.

Нэг Go бинари дотор модулиуд компиллогдож, тенант бүрт аль апп
идэвхтэйг PostgreSQL дээрх апп стор шийднэ — сүлжээний нэмэлт дуудлагагүй,
микросервисийн нарийн төвөгтэй байдалгүйгээр модуль хуваарилалт хийнэ.

**Хэлний бодлого: монгол хэл + НҮБ-ын албан ёсны 6 хэл** — араб, хятад, англи,
франц, орос, испани. Нийт 7 хэл. Монгол хэл эх сурвалж; баримт бичиг долуулаа
байдаг бол програм хангамж нь монгол, англи хоёроор ирж, үлдсэнийг нь
**Тохиргоо → Харагдац** дотроос асаана. Дэлгэрэнгүйг
[орчуулгын гарын авлага](docs/TRANSLATION_GUIDE.md)-аас үзнэ үү.

<p>
  <img src="docs/assets/icons/flag-mn.png" width="18" height="18" alt=""> <b>Монгол</b>
  &nbsp;·&nbsp;
  <a href="docs/README_AR.md"><img src="docs/assets/icons/flag-ar.png" width="18" height="18" alt=""> العربية</a>
  &nbsp;·&nbsp;
  <a href="docs/README_ZH.md"><img src="docs/assets/icons/flag-zh.png" width="18" height="18" alt=""> 中文</a>
  &nbsp;·&nbsp;
  <a href="docs/README_EN.md"><img src="docs/assets/icons/flag-en.png" width="18" height="18" alt=""> English</a>
  &nbsp;·&nbsp;
  <a href="docs/README_FR.md"><img src="docs/assets/icons/flag-fr.png" width="18" height="18" alt=""> Français</a>
  &nbsp;·&nbsp;
  <a href="docs/README_RU.md"><img src="docs/assets/icons/flag-ru.png" width="18" height="18" alt=""> Русский</a>
  &nbsp;·&nbsp;
  <a href="docs/README_ES.md"><img src="docs/assets/icons/flag-es.png" width="18" height="18" alt=""> Español</a>
</p>

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-16.3-black.svg)](https://nextjs.org)
[![CI](https://github.com/gerege-systems/open-gerege-nexus/actions/workflows/ci.yml/badge.svg)](https://github.com/gerege-systems/open-gerege-nexus/actions/workflows/ci.yml)
[![Security](https://github.com/gerege-systems/open-gerege-nexus/actions/workflows/security.yml/badge.svg)](https://github.com/gerege-systems/open-gerege-nexus/actions/workflows/security.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Docs](https://img.shields.io/badge/Баримт-gerege--systems.github.io-0050b0.svg)](https://gerege-systems.github.io/open-gerege-nexus/)

**Баримт бичиг:
[gerege-systems.github.io/open-gerege-nexus](https://gerege-systems.github.io/open-gerege-nexus/)**
— энэ репод байгаа бүх баримт долоон хэлээр, хайхад хялбар вэб хэлбэрээр.

---


## Хамаарах сервисүүд

Апп стор нь тусдаа репод байрладаг бөгөөд энэ платформ түүнээс каталогоо
**гарын үсэгтэйгээр** татдаг (`APP_CATALOG_URL`; тохируулаагүй бол
`catalog/apps.json` файлаараа ажиллана):

- [`appstore-gerege-mn`](https://gitlab.gerege.mn/gerege-line/gerege-core/appstore-gerege-mn)
  — registry API ба нээлттэй storefront (appstore.gerege.mn)
- [`developer-gerege-nexus`](https://gitlab.gerege.mn/gerege-line/gerege-core/developer-gerege-nexus)
  — хөгжүүлэгчийн консол (developer.gerege.mn)

## Агуулга

- [Хөгжүүлэгчид](#хөгжүүлэгчид)
- [Үндсэн боломжууд](#үндсэн-боломжууд)
- [Бэлэн бизнес аппликейшнүүд](#бэлэн-бизнес-аппликейшнүүд)
- [Төслийн бүтэц](#төслийн-бүтэц)
- [Desktop бүрхүүлүүд](#desktop-бүрхүүлүүд)
- [Ажиллуулах заавар](#ажиллуулах-заавар)
- [Тохиргооны хувьсагчид](#тохиргооны-хувьсагчид)
- [API-н тойм](#api-н-тойм)
- [Тест ба чанарын хяналт](#тест-ба-чанарын-хяналт)
- [Аюулгүй байдал](#аюулгүй-байдал)
- [Баримт бичгийн индекс](#баримт-бичгийн-индекс)

---

## Хөгжүүлэгчид

| Оролцогч | Үүрэг |
| --- | --- |
| **Gerege Systems Development Team** ([@gerege-systems](https://github.com/gerege-systems)) | Архитектур, платформын цөм |
| **Gemini AI** | Код үүсгэлт, баримтжуулалт |
| **Claude AI** | Код шинжилгээ, аюулгүй байдлын аудит |

---

## Үндсэн боломжууд

### 1. Өндөр бүтээмжтэй модуль монолит архитектур

- **Compile-time Go апп модулиуд** — модулиуд (`contacts`, `products`,
  `inventory`, `billing`, `documents`, `sso_clients`) нэг бинарид
  компиллогдож, процесс дотроо дуудагдана.
- **Тенант бүрийн апп стор** — тенант тус бүрийн апп эрх, меню, RBAC тохиргоо
  PostgreSQL (`app_installations`) дээр динамикаар удирдагдана.
- **Хамаарал шийдвэрлэх хөдөлгүүр** — DAG (Directed Acyclic Graph) дээр
  тулгуурласан рекурсив шийдвэрлэлт, мөчлөг илрүүлэлт, semver шалгалт.
- **Каталог синк** — `catalog/apps.json` нь цорын ганц эх сурвалж; `apps`
  хүснэгт ачаалал бүрт үүнээс шинэчлэгдэнэ.

### 2. Cloud-native тэсвэрлэлтийн хөдөлгүүр

| Модуль | Зориулалт |
| --- | --- |
| `resilience/breaker.go` | Google SRE загварын adaptive circuit breaker |
| `resilience/loadshedder.go` | Ачаалал хэтэрсэн үед `503` + `Retry-After` |
| `resilience/singleflight.go` | Давхардсан хүсэлтийг нэгтгэж кэшийн ачаалал бууруулах |
| `resilience/retry.go` | Экспоненциал ухралттай давталт |

### 3. Төрийн цахим дэд бүтцийн интеграци

- **ХУР — Төрийн мэдээлэл солилцооны систем** (`platform/gerege/xyp.go`):
  иргэний бүртгэл (`WS100101`), хуулийн этгээдийн баталгаажуулалт (`WS100201`).
  Клиент нь платформд үлдэж, хэрэглэгчид харагдах нүүр нь `apps/egov` —
  лавлагаа, сувгийн төлөв, лавлагааны түүх гурван дэлгэц (`/egov`).
- **Үндэсний E-ID ба ДАН** ([`developer.gerege.mn`](https://developer.gerege.mn),
  [`eidmongolia.mn`](https://eidmongolia.mn)) — тоон гарын үсэг (PKI), нэг
  удаагийн код (Mobile OTP), банкны суваг (Bank SSO), царай танилт (Biometric).
- **Платформын өөрийн OAuth2 / OIDC provider**
  (`/.well-known/openid-configuration`) — гуравдагч системд client credentials
  урсгалаар токен олгоно.
- **Мөн өөр провайдерийн SSO клиент болж чадна** (`SSO_CLIENT_ISSUER`) — өөр
  Gerege Nexus суулгац ч байж болно. Хоёр хагас нь бие биеэсээ хамааралгүй:
  аймгийн суулгац улсын нэгдсэн рүү дээшээ холбогдоод, өөрөө өөр дээрээ суусан
  аппуудад identity өгсөөр байна. Клиент болсон үед эндэх нэвтрэлт хаагдаж,
  гарах үед провайдер дээрээс гарч буцаж ирнэ —
  [`docs/SSO_FEDERATION.md`](docs/SSO_FEDERATION.md).
- **И-мэйл баталгаажуулалт** (`platform/emailverify`) — хаяг эзэмшлийг батлах
  нэгдсэн урсгал, платформын бүх апп модуль Go дуудлагаар ашиглана. Захидлыг
  хостинг үйлчилгээ (`enigma.mn`) илгээх тул платформ SMTP нууц үг, илгээгчийн
  хаяг эзэмшихгүй. Хэрэглэгч буцаж ирэхэд баталгаажуулалт бүртгэгдэнэ — буцах
  утга нэг л удаа ажиллана. Тохиргоо → И-мэйл баталгаажуулалт дотор харагдана.

> **Анхаар.** E-ID / ДАН / ХУР-ын mock горим зөвхөн хөгжүүлэлтийн орчинд
> ажиллана. `ENVIRONMENT=production` үед mock горим автоматаар унтарч,
> хуурамч иргэний мэдээллээр нэвтрэх боломжгүй болно.

### 4. AI Copilot ба бизнес аналитик

- **AI туслах** (`platform/ai/copilot.go`) — тенантын өгөгдлийн сангийн бодит
  төлөвт холбогдсон, зорилго ангилдаг харилцан яриа.
- **Агуулахын эрэлт таамаглагч** (`platform/ai/inventory_forecaster.go`) —
  түүхэн хөдөлгөөнд тулгуурлан аюулгүйн үлдэгдэл ба дахин захиалгын цэгийг
  санал болгоно.

---

## Бэлэн бизнес аппликейшнүүд

| # | Апп | ID | Зам | Тайлбар |
| --- | --- | --- | --- | --- |
| 1 | Organisation & People | `io.gerege.nexus.organisation` | `/organisation` | Хэлтэс нэгж, ажилтнуудын бүртгэл. Шинэ тенантад default-оор суух ч устгаж болно; байгууллагын хуулийн профайл нь апп биш, платформын хэсэг |
| 2 | e-Government Link | `io.gerege.nexus.egov` | `/egov` | ХУР-ын иргэн/хуулийн этгээдийн лавлагаа, eID ба ДАН сувгийн төлөв, лавлагааны түүх. Default-оор суух ч устгаж болно |
| 3 | Contacts | `io.gerege.nexus.contacts` | `/contacts` | Харилцагчийн бүртгэл, ХУР авто-бөглөлт |
| 4 | Products | `io.gerege.nexus.products` | `/products` | Бараа, үнэ, тенантад хамаарах SKU |
| 5 | Inventory | `io.gerege.nexus.inventory` | `/inventory` | Агуулах, үлдэгдэл, хөдөлгөөний бүртгэл |
| 6 | Public Billing & e-Barimt | `io.gerege.nexus.billing` | `/billing` | Нэхэмжлэх, 10% НӨАТ, e-Barimt баримт |
| 7 | Digital Documents & E-Sign | `io.gerege.nexus.documents` | `/documents` | Цахим баримт, гарын үсэг, батламжийн урсгал |
| 8 | SSO Clients | `io.gerege.nexus.sso_clients` | `/sso-clients` | Энэ платформоор дамжуулан нэвтрэх системүүдийн OAuth2 клиент бүртгэл |
| 9 | State Services | `io.gerege.nexus.gov_services` | `/gov-services` | Тохируулж болох шийдвэрлэх урсгал, шилжүүлэлт, баталгаажуулалт, цаг захиалга |
| 10 | PDF цахим гарын үсэг | `io.gerege.nexus.esign` | `/esign` | eID Mongolia (PIN2) хуулийн хүчин төгөлдөр цахим гарын үсэг, Gerege eSign HSM, багц баталгаажуулалт, гарын үсгийн лог |

Апп бүр тенантад суулгагдаж идэвхжсэн үед л маршрутууд нээгдэнэ. Суулгаагүй апп
руу хандвал `403 Forbidden` буцна.

---

## Төслийн бүтэц

```
backend/
  cmd/api/            HTTP API сервер (+ demo seeder)
  cmd/migrate/        Goose миграцийн ажиллуулагч
  db/migrations/      SQL миграцууд
  internal/
    module.go         Модулийн Go гэрээ (Module interface)
    apps/             Бизнес модулиуд
    platform/         Платформын цөм үйлчилгээнүүд
frontend/             Next.js 16 (App Router) вэб клиент
native-apps/          Swift, C# ба Kotlin native клиентүүд (Linux нь PWA)
catalog/              Апп сторын каталог ба manifest-ууд
deploy/               Production Dockerfile, Nginx тохиргоо
docs/                 Баримт бичиг ба орчуулгууд
```

---

## Desktop бүрхүүлүүд

Архитектур нь **Native Shell + Web Work Area**: native бүрхүүл нь session-ий
мөчлөг, толгой хэсэг, цэс, tray, төхөөрөмжийн хандалтыг эзэмшинэ; вэб клиент нь бүрхүүл
дотор ажиллахдаа өөрийн chrome-оо нуугаад зөвхөн **ажлын муж** болж
рендерлэгдэнэ. Хөтчөөр орвол бүрхүүл байхгүй тул вэб клиент урьдын адил бүрэн
аппликейшн хэвээрээ ажиллана.

Бүрхүүл ба вэб клиентийн хооронд бичигдсэн гэрээ бий —
[`docs/SHELL_CONTRACT.md`](docs/SHELL_CONTRACT.md) нь `window.GeregeShell`-ийн
method, event, capability, хувилбарын дүрэм, аюулгүй байдлын шаардлагыг
тодорхойлно. Вэб клиент бүрхүүлийн дотоод бүтцийг мэдэхгүй — зөвхөн гэрээг л
мэднэ.

Клиентүүд [`native-apps/`](native-apps) дотор гурван native сангаар
хөгжинө: Swift (macOS/iOS/iPadOS), C# (Windows desktop/kiosk/POS), Kotlin
(Android mobile/tablet/kiosk/POS). Linux desktop нь PWA хэвээр.

Бүх native клиент нэвтрэлтийг өөрийн native UI-аар хийж, session cookie-г
webview store-д тарина. `/login` нь browser/PWA горимд л ашиглагдана:

```bash
make run-mac        # macOS хөгжүүлэлтийн горим
make build-mac      # Swift/AppKit компиляц
```

### Хөтчөөс суулгах (Linux болон бусад)

Native клиентгүй платформ дээр вэб клиент нь PWA
(`/manifest.webmanifest`) тул хөтчөөс шууд суулгаж болно: Chrome/Edge дээр
хаягийн мөрний суулгах товч, Safari дээр **File → Add to Dock**. Суулгасан
хувилбар нь dock эсвэл taskbar-т орж, өөрийн цонхоор нээгддэг — татаж авах
файлгүй, дэлгүүргүй, вэбтэй яг ижил хуудсуудыг үзүүлнэ.

Платформ бүрийн урьдчилсан шаардлага, runtime endpoint, enrollment, code
signing болон auto-update сувгийн зааврыг
[`native-apps/README.md`](native-apps/README.md)-ээс үзнэ үү.

> Native CI нь macOS Swift ба Windows .NET build-ийг тус тусын runner дээр
> шалгана. Installer нь signing identity оруулсны дараах release ажил.

---

## Ажиллуулах заавар

### Шаардлагатай програмууд

- Go 1.26+
- Node.js 20+
- PostgreSQL 16+ (эсвэл Docker Compose)

### 1. Docker Compose (хамгийн хялбар)

```bash
docker compose up -d
```

Миграц нь тусдаа `migrate` service-ээр автоматаар ажиллаж дуусмагц API асна.

### 2. Гараар ажиллуулах

**Backend:**

```bash
cd backend
go mod download
DATABASE_URL="postgres://postgres:postgrespassword@localhost:5432/platform_db?sslmode=disable" \
  go run ./cmd/migrate up
PUBLIC_ORIGIN=http://nexus.localhost:3000 \
ALLOWED_ORIGINS=http://nexus.localhost:3000,http://cp.localhost:3000 \
CONTROL_PLANE_HOST=cp.localhost \
  go run ./cmd/api
```

**Frontend:**

```bash
cd frontend
npm ci
CONTROL_PLANE_HOST=cp.localhost \
NEXT_PUBLIC_API_URL=http://nexus.localhost:8080/api/v1 \
NEXT_PUBLIC_CONTROL_PLANE_API_URL=http://cp.localhost:8080/api/platform/v1 \
  npm run dev
```

Вэб хөтөч дээр тенантын урсгалыг
[http://nexus.localhost:3000](http://nexus.localhost:3000), операторын консолыг
[http://cp.localhost:3000](http://cp.localhost:3000) хаягаар нээнэ. `localhost`-
ын дэд домэйнүүд loopback руу шийдэгдэх тул `/etc/hosts` өөрчлөхгүй.

### Туршилтын нэвтрэх эрх

| Талбар | Утга |
| --- | --- |
| И-мэйл | `admin@example.com` |
| Нууц үг | `Password123!` |
| Тенант | `Demo Corporation` (`slug: demo`) |

Энэ бүртгэл зөвхөн хөгжүүлэлтийн орчинд үүснэ. Production дээр
`SEED_DEMO_DATA=true` гэж тодорхой заагаагүй бол огт үүсэхгүй.

---

## Автомат deploy

`main` салбар руу push хийх бүрд [`deploy.yml`](.github/workflows/deploy.yml)
ажиллана:

1. Backend ба frontend образыг GHCR руу угсарч илгээнэ (`:latest` ба `:<sha>`).
2. `docker-compose.prod.yml`-ийг серверт хуулна.
3. Серверт `.env`-ийг GitHub secret-ээс шинээр бичиж, образуудыг татна.
4. Миграц бүрэн дуусмагц API ба frontend солигдоно.
5. `/health` ба `/ready`-г шалгаж, амжилтгүй бол лог хэвлээд алдаа өгнө.

Гараар ажиллуулахдаа Actions → *Deploy to Production* → **Run workflow**
(шаардвал тодорхой tag зааж болно).

Шаардлагатай repository secrets:

| Secret | Заавал | Тайлбар |
| --- | --- | --- |
| `DEPLOY_SSH_KEY` | Тийм | Deploy хэрэглэгчийн хувийн түлхүүр. Байхгүй бол rollout алгасана |
| `POSTGRES_PASSWORD` | Тийм | Сервер дэх өгөгдлийн сангийн нууц үг |
| `SSO_DEFAULT_CLIENT_SECRET` | Тийм | Production дээр OAuth2 client-д зайлшгүй |
| `DEPLOY_HOST` / `DEPLOY_USER` / `DEPLOY_PORT` | Үгүй | Анхдагч: `nexus.gerege.mn` / `deploy` / `22` |
| `PUBLIC_ORIGIN` | Үгүй | Анхдагч: `https://nexus.gerege.mn` |

> Production домэйн нь `nexus.gerege.mn`. Өмнөх `openerp.gerege.mn` домэйныг
> Gerege Nexus нэршилд шилжихэд орлуулсан. `PUBLIC_ORIGIN` нь CORS, OIDC issuer,
> eID callback гурвыг нэг дор тодорхойлдог тул түүнийг өөрчлөхөд DNS, TLS
> гэрчилгээ, issuer-т тулгуурласан client бүр хамт шилжинэ.

Серверт зөвхөн Docker шаардлагатай — эх код ч, Go/Node ч хэрэггүй. Утгуудын
жишээг [`deploy/.env.prod.example`](deploy/.env.prod.example)-ээс үзнэ үү.

### Анхны байгууллага — шинэ deployment дээр эхлээд хийх зүйл

Шинэ deployment дээр **бүртгүүлэх дэлгэц байхгүй**, demo бүртгэл production
дээр үүсэхгүй. Тэгэхээр миграц дуусаад контейнерууд асахад өгөгдлийн сан
хоосон, `/ready` ногоон, нэвтрэх дэлгэц гарч ирдэг ч **хэн ч нэвтэрч чадахгүй**.
Асах үед лог үүнийг хэлж, хоёр замыг зааж өгнө.

**1. Тохиргооны шидтэн (вэб).** Сервер асахдаа нэг удаагийн setup токен үүсгээд
хаягийг нь лог руу бичнэ:

```
WARN this deployment has no organisation ... setup_url=https://.../setup?token=<токен>
```

Тэр хаягаар орвол гурван алхам: байгууллага -> админ -> нууц үг. Регистрийн
дугаараар **Gerege Core**-оос байгууллагын нэр, албан ёсны нэр, админы нэр,
и-мэйлийг татна (`GEREGE_CORE_TOKEN` тохируулсан үед). Токен нь зөвхөн санах
ойд, дискэнд бичигдэхгүй, байгууллага үүссэн даруйд хүчингүй болно — сервер
дахин асвал шинэ токен гарна.

**2. Команд (терминал).** Консол ч, хөтөч ч хэрэггүй:

```bash
docker exec -it gerege_nexus_backend /app/tenant-bootstrap \
  -org "Байгууллагын нэр" -slug baiguullaga \
  -email you@example.mn -name "Таны нэр"
```

Энэ нь эхний байгууллага, түүний админ хэрэглэгч, гишүүнчлэл, `admin` эрхийг
нэг transaction-д үүсгэнэ. Нууц үгийг TTY-ээс хоёр удаа асууна — flag эсвэл
env-д бүү дамжуул (`docker exec -it`, `docker exec` биш): shell history,
process list, container inspect-д үлдэнэ.

**Нэг л удаа ажиллана.** Байгууллага аль хэдийн байвал команд татгалзана —
дараагийнхыг нь [control plane консолоос](docs/CONTROL_PLANE.md) үүсгэнэ.
Аппуудыг админ нь нэвтрээд дэлгүүрээс суулгана.

---

## Тохиргооны хувьсагчид

Бүрэн жагсаалтыг [`.env.example`](.env.example)-ээс үзнэ үү.

| Хувьсагч | Анхдагч | Тайлбар |
| --- | --- | --- |
| `DATABASE_URL` | localhost | PostgreSQL холболтын мөр |
| `PORT` | `8080` | API сонсох порт |
| `ENVIRONMENT` | `development` | `production` үед аюулгүй байдлын хатуу горим |
| `APP_CATALOG_PATH` | `catalog/apps.json` | Апп сторын каталогийн зам |
| `ALLOWED_ORIGINS` | `nexus.localhost`, `cp.localhost` | Хоёр browser урсгалын CORS зөвшөөрөгдсөн эх сурвалж |
| `TRUST_PROXY_HEADERS` | `false` | `X-Forwarded-For`-д итгэх эсэх |
| `CONTROL_PLANE_HOST` | `cp.localhost` | Операторын консолын хост. Production дээр хоосон бол консол огт байхгүй ([`docs/CONTROL_PLANE.md`](docs/CONTROL_PLANE.md)) |
| `CONTROL_PLANE_ALLOWED_CIDRS` | — | Консолд хүрэх хаягууд. **Зөвхөн платформ хаалттай (private) горимд** шалгагдана — нээлттэй үед хаягаар хязгаарлахгүй. Хоосон эсвэл `open` бол огт хязгаарлахгүй |
| `PROMETHEUS_URL` / `ALERTMANAGER_URL` / `GRAFANA_URL` | — | Консолын нүүр хуудсанд хэмжүүр, дохио, гүнзгий линк. Хоосон бол тэр хэсэг "тохируулаагүй" гэж харагдана |
| `GITHUB_DEPLOY_TOKEN` / `GITHUB_REPOSITORY` | — | Консолын deploy товч. Токен нь зөвхөн deploy workflow-д эрхтэй fine-grained байх ёстой |
| `SEED_DEMO_DATA` | production-оос бусад үед идэвхтэй | Туршилтын бүртгэл үүсгэх. Платформ хаалттай (private) горимтой бол зөрчилдөх тул boot дээр анхааруулна |
| `SSO_DEFAULT_CLIENT_SECRET` | — | Production дээр заавал шаардлагатай |
| `SSO_CLIENT_ISSUER` / `SSO_CLIENT_ID` | — | Тохируулбал энэ суулгац нэрлэсэн провайдерийн клиент болно: эндэх нэвтрэлт хаагдаж, гарах нь провайдер дээр дуусна |
| `SSO_CLIENT_TENANT` | — | Провайдерийн баталгаажуулсан ч энд бүртгэлгүй хүнийг үүсгэх байгууллага. Хоосон бол үүсгэхгүй |
| `GEREGE_CORE_TOKEN` / `GEREGE_CORE_URL` | — / `https://core.gerege.mn` | Байгууллага, хүнийг регистрээр хайхад ашиглах Gerege Core-ийн токен. Консолоос ч тавьж болно (`core.api_token`). Хоосон бол талбаруудыг гараар бөглөнө |
| `INTEGRATION_ENCRYPTION_KEY` | — | Консолд хадгалсан түлхүүрүүд болон холбогчийн эрхийг битүүмжлэх AES түлхүүр. Хоосон бол консолоос түлхүүр хадгалах боломжгүй — цэвэр текстээр хадгалахын оронд бичилт татгалзана |
| `GEMINI_API_KEY` | — | AI chat, voice, TTS, орчуулгыг идэвхжүүлэх түлхүүр |
| `GEMINI_MODEL` / `GEMINI_TTS_MODEL` | Gemini 2.5 Flash загварууд | Chat ба дууны model сонголт |
| `EID_MOCK_MODE` / `DAN_MOCK_MODE` / `XYP_MOCK_MODE` | production-оос бусад үед идэвхтэй | Төрийн системийн mock горим |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Trace-ийг Tempo руу илгээх хаяг. Хоосон бол tracing бүрэн унтарсан |
| `OTEL_TRACES_SAMPLER_ARG` | `0.1` | Trace-ийн хэдэн хувийг хадгалах (0-1) |
| `SENTRY_DSN` | — | Алдааны бүртгэл (GlitchTip эсвэл Sentry). Хоосон бол унтарсан |
| `REPORT_SMTP_URL` / `REPORT_MAIL_FROM` | — | Товлосон тайланг илгээх SMTP. Хоосон бол тайлан бэлтгэгдэнэ, илгээгдэхгүй |

### Консолоос тохируулах

Дээрх хувьсагчдын нэг хэсэг нь одоо **удирдлагын консолын** `/cp/config` дэлгэцээс
өөрчлөгддөг — сервер дахин ачаалахгүйгээр. Env хувьсагч нь хэвээрээ ажиллах бөгөөд
консолоос өгсөн утга л түүнийг дардаг: консол огт нээгээгүй суулгац юу ч мэдэрэхгүй.

| Дэлгэц | Юу | Тайлбар |
| --- | --- | --- |
| Тохиргоо | `ai.model`, `ai.tts_model`, `brand.name`, `catalog.sync_interval`, `observability.*`, `platform.access_mode`, `platform.maintenance*`, `session.idle_timeout` | Утга бүр хаанаас ирснийг (консол/env/анхдагч) харуулна, өөрчлөлт бүр шалтгаантай, түүхтэй, нэг товчоор буцаана |
| Түлхүүрүүд | `ai.gemini_api_key`, `core.api_token`, `reports.smtp_url` | **Буцааж харагдахгүй.** Утга нь AES-256-GCM-ээр битүүмжлэгдэж хадгалагдана, дэлгэц зөвхөн эх сурвалж болон сүүлийн дөрвөн тэмдэгтийг харуулна. Бичихэд хоёр дахь хүчин зүйл дахин шаардана |

Хаяг, нэр гэх мэт нууц бус утга нь тохиргооны хүснэгтэд, нууц утга нь тусдаа
битүүмжилсэн хүснэгтэд байдаг. Энэ хоёрын хил нь код дээр барьцаалагдсан:
`internal/kernel/settings` нь нууц мэт унших түлхүүрийг бүртгэхээс `panic`-даг.

---

Мониторингийн стек нь **өөрийн орчинтой** — платформын `.env`-д хамаарахгүй.
`GRAFANA_ADMIN_PASSWORD`, `MONITORING_DB_PASSWORD` болон дохиоллын сувгийн
хувьсагчдыг [`deploy/.env.monitoring.example`](deploy/.env.monitoring.example)
ба [`docs/MONITORING.md`](docs/MONITORING.md)-ээс үзнэ үү. Тусдаа байгаа
шалтгаан нь стек өөрөө тусдаа: платформ түүнгүйгээр бүрэн ажиллана.

---

## API-н тойм

| Аргачлал | Зам | Тайлбар |
| --- | --- | --- |
| `GET` | `/health`, `/ready` | Амьд ба бэлэн байдлын шалгалт |
| `GET` | `/metrics` | Prometheus хэмжүүрүүд |
| `POST` | `/api/v1/auth/login` | И-мэйл/нууц үгээр нэвтрэх |
| `POST` | `/api/v1/auth/eid/login` | Үндэсний E-ID-аар нэвтрэх |
| `POST` | `/api/v1/auth/dan/login` | ДАН гарцаар нэвтрэх |
| `POST` | `/api/v1/auth/logout` | Session-ийг цуцлах |
| `GET` | `/api/v1/auth/tenants` | Хэрэглэгчийн харьяалагдах байгууллагууд |
| `POST` | `/api/v1/auth/switch-tenant` | Session-ийг өөр байгууллага руу шилжүүлэх |
| `GET` | `/api/v1/menus` | Тенантад идэвхтэй цэсүүд |
| `GET` | `/api/v1/store/apps` | Апп сторын жагсаалт |
| `POST` | `/api/v1/ai/chat`, `/stt`, `/tts`, `/translate` | Tenant-safe Gemini AI pipeline |
| `GET/PUT` | `/api/v1/admin/ai/prompts/{key}` | AI prompt тохируулах (админ) |
| `GET/POST` | `/api/v1/admin/ai/knowledge` | AI мэдлэгийн сан (админ) |
| `POST` | `/api/v1/store/apps/{slug}/install` | Апп суулгах (админ) |
| `POST` | `/api/v1/verify/send` | Хостинг үйлчилгээнээс баталгаажуулах холбоос хүсэх |
| `GET` | `/api/v1/verify/landed` | Баталгаажуулсан хэрэглэгчийг хүлээн авах — нэг л удаа ажиллана |
| `GET` | `/api/v1/admin/email-verification/overview` | Баталгаажуулалтын тойм ба үйлчилгээний төлөв (админ) |
| `POST` | `/oauth2/token` | OAuth2 client credentials токен |
| `GET` | `/oauth2/logout` | RP-initiated logout — session хааж, бүртгэлтэй хаяг руу буцаана |
| `GET` | `/api/v1/auth/sso/config` | Энэ суулгац хэрхэн нэвтрүүлдэг — нэвтрэх дэлгэц уншина |
| `GET` | `/api/v1/auth/sso/start` | Провайдер дээр нэвтрэлт эхлүүлнэ (PKCE, state, nonce) |
| `GET` | `/api/v1/auth/sso/callback` | Провайдерээс буцаж ирэх цэг |

Нэвтрэлтийн токен нь HttpOnly cookie эсвэл `Authorization: Bearer <token>`
толгойгоор дамжина.

---

## Тест ба чанарын хяналт

```bash
# Backend нэгж тестүүд (race detector-тэй)
cd backend && go test -race ./...

# Статик шинжилгээ
cd backend && go vet ./... && golangci-lint run

# Эмзэг байдлын шалгалт
cd backend && govulncheck ./...

# Frontend build
cd frontend && npm run build
```

CI нь push ба pull request бүр дээр lint, тест, frontend build, Docker образ
угсралт, govulncheck ба gosec шалгалтыг ажиллуулна.

---

## Аюулгүй байдал

- Session токен нь 256 бит санамсаргүй утга бөгөөд өгөгдлийн санд зөвхөн
  SHA-256 хэш нь хадгалагдана.
- Нууц үг bcrypt-ээр хэшлэгдэнэ; нэвтрэх хүсэлтэд IP-д суурилсан хурдны
  хязгаарлалт үйлчилнэ.
- Апп суулгах, идэвхжүүлэх, интеграц бүртгэх үйлдэл тенантын админ эрх шаардана.
- OAuth2 client танилт тогтмол хугацааны харьцуулалтаар (constant-time)
  шалгагдана.

Эмзэг байдал мэдээлэх журмыг [`SECURITY.md`](SECURITY.md)-ээс үзнэ үү.

---

## Баримт бичгийн индекс

| Баримт | Тайлбар |
| --- | --- |
| [Баримт бичгийн төв](docs/README.md) | Бүх баримтын индекс ба орчуулгууд |
| [Архитектурын тодорхойлолт](docs/ARCHITECTURE_SPECIFICATION.md) | Платформын давхаргууд ба шийдвэрүүд |
| [Цөмийн хилийн төлөвлөгөө](docs/CORE_BOUNDARY_PLAN.md) | Юу цөмд үлдэж, юу апп болж гарах вэ — хэмжилт ба үе шатууд |
| [Модуль хөгжүүлэх заавар](docs/MODULE_AUTHORING_GUIDE.md) | Шинэ апп модуль бичих алхмууд |
| [Bridge Contract v1](docs/SHELL_CONTRACT.md) | Native бүрхүүл ба вэб ажлын мужийн гэрээ |
| [Хамтран ажиллах заавар](CONTRIBUTING.md) | Хувь нэмэр оруулах журам |
| [Аюулгүй байдлын бодлого](SECURITY.md) | Эмзэг байдал мэдээлэх |
| [Ёс зүйн дүрэм](CODE_OF_CONDUCT.md) | Хамт олны хэм хэмжээ |
| [Өөрчлөлтийн түүх](CHANGELOG.md) | Хувилбар бүрийн өөрчлөлт |

---

## Ашигласан ба санаа авсан төслүүд

1. **[snykk/go-rest-boilerplate](https://github.com/snykk/go-rest-boilerplate)**
   by **[@snykk](https://github.com/snykk)** — Go REST API суурь архитектур.
2. **[Odoo](https://github.com/odoo/odoo)** — модуль апп стор ба хамаарал
   шийдвэрлэх загвар.
3. **[go-zero](https://github.com/zeromicro/go-zero)** — cloud-native
   resilience хөдөлгүүр.

---

## Лиценз

Copyright (c) 2026 **Gerege Systems Development Team, Gerege Nomadica Foundation**. Apache 2.0 лицензээр тараагдана — [`LICENSE`](LICENSE)-ийг үзнэ үү.

Тугны дүрсийг [Flaticon](https://www.flaticon.com/)-оос авсан
([оруулсан хувь нэмэр](docs/assets/icons/ATTRIBUTION.md)).
