# Экосистемийн git стратеги

**Gerege Ecosystem**-ийн зуу зуун платформ Nexus дээр суурилахад код
давхардуулахгүй байх repo зохион байгуулалт. Апп стор (registry +
developer console) экосистемийн түгээлтийн төв болно.

[Баримт бичгийн төв рүү буцах](README.md) ·
Холбоотой: [Control Plane](CONTROL_PLANE_PLAN.md) ·
[Мониторинг ба тайлан](MONITORING_AND_REPORTING_PROPOSAL.md)

---

## 1. Гол зарчим: fork бол дайсан

Зуун платформ = зуун fork болвол засвар бүр зуун газар давтагдана. Тиймээс
нэг л дүрэм: **Nexus-ийн кодыг хэн ч хуулахгүй, зөвхөн dependency болгон
ашиглана.** "Шинэ платформ" гэдэг нь гурван түвшний аль нэг нь байхаас
өөр сонголтгүй:

### Түвшин 1 — Deployment (платформуудын 90% нь энэ байх ёстой)

Код огт байхгүй. Ижил GHCR образ, өөр `.env` + домэйн + каталог +
брэнд тохиргоо. Аймгийн суулгац, салбар байгууллагын суулгац гэх мэт.
SSO federation (`SSO_CLIENT_ISSUER`) аль хэдийн суулгац хоорондын
холбоог дэмждэг тул "тусдаа платформ" мэт харагдах олонх нь үнэндээ
зүгээр л тусдаа deployment. Тохиргоонууд нь нэг **deploy config repo**-д
(платформ бүр нэг хавтас, GitOps маягаар) хадгалагдана — код биш, YAML.

### Түвшин 2 — Distribution (өөрийн Go модультай бүтээгдэхүүн)

Nexus-ийн модуль монолит архитектур яг үүнд зориулагдсан: бүтээгдэхүүн =
цөм + өөрийн апп модулиуд. Distribution repo нь **жижигхэн**:

```
gerege-<product>/
  go.mod          # require github.com/gerege-systems/open-gerege-nexus/backend vX.Y.Z
  main.go         # цөмийг асаахдаа өөрийн модулиудаа registry-д нэмнэ
  modules/        # зөвхөн энэ бүтээгдэхүүний апп модулиуд (Go)
  catalog/        # энэ бүтээгдэхүүний апп багц + manifest
  branding/       # лого, өнгө, нэр
  deploy/         # compose override, домэйн
```

Цөмийн код энд **нэг ч мөр байхгүй** — `go.mod`-ын нэг мөр л байна.
Цөм шинэчлэгдэхэд Renovate/Dependabot автоматаар version bump PR нээж,
CI нь build+test ажиллуулна. Зуун distribution нэг өдөр цөмийн засвар
авна гэсэн үг.

### Түвшин 3 — External app (апп сторын гуравдагч апп)

Цөмтэй compile хийгдэхгүй, OAuth2/OIDC-ээр холбогдож каталогоор
түгээгддэг тусдаа систем (`catalog/manifests/example-external.json`
загвар аль хэдийн бий). Өөр хэл, өөр стек, өөр багийн апп бүгд энэ
замаар — тэдний git манай асуудал биш.

**Сонголтын дүрэм**: тохиргоо хүрэлцэхгүй болохоор л Түвшин 2 руу,
Go модуль байх шаардлагагүй бол Түвшин 3 руу. Эргэлзвэл доод түвшнийг нь
сонго.

---

## 2. Үүнийг боломжтой болгохын тулд цөмд хийх ажил

### 2.1 `internal/` хаалт — өнөөдрийн гол саад

`Module` interface одоо `backend/internal/module.go`-д байна. Go хэлний
дүрмээр **`internal/` доторх пакетыг гадны repo import хийж чадахгүй** —
өөрөөр хэлбэл өнөөдөр Nexus дээр бүтээгдэхүүн барих цорын ганц арга нь
fork. Энэ нэг зохион байгуулалт л Түвшин 2-ыг бүхэлд нь хааж байна.

Шийдэл — **нийтийн SDK давхарга** гаргах:

```
backend/pkg/nexus/        # нийтийн, semver амлалттай API
  module.go               # Module interface, ParamSpec г.м.
  registry.go             # модуль бүртгэх
  platform.go             # модульд өгөгдөх үйлчилгээний interface-ууд:
                          #   DB (tenant-scoped Querier), Auth context,
                          #   Audit, i18n, Email, Settings, Flags
```

`internal/` доторх бодит хэрэгжилт хэвээрээ — SDK нь interface л
экспортолно. Дотоод 9 апп модуль өөрсдөө энэ SDK-г хэрэглэдэг болгож
шилжүүлбэл SDK "жинхэнэ" гэдэг нь батлагдана (dogfooding).

### 2.2 Хувилбарлалт — амлалт болгох

- Цөм **semver tag**-тай release хийнэ (`backend/v1.4.0`); `pkg/nexus`
  доторх API major хувилбар дотроо эвдэрдэггүй гэсэн бичигдсэн амлалт.
- CHANGELOG (байгаа) release бүрд; API эвдэх өөрчлөлт нь deprecation
  → нэг major cycle хүлээх журамтай.
- Distribution-ууд tag-аас өөр юу ч require хийхгүй (branch/commit
  түгжихийг CI-д хориглоно).

### 2.3 Frontend — generic хэвээр нь хадгалах

Вэб клиент цэс, маршрутаа каталогоос авдаг байдлаа гүнзгийрүүлнэ:
бүтээгдэхүүн frontend-ийг fork хийх шаардлагагүй байх нь зорилго.
Брэнд (лого, өнгө, нэр) нь build-time биш **runtime тохиргоо** болно
(Түвшин 1-д зайлшгүй). Бүтээгдэхүүний өөрийн UI хэрэгтэй бол: эхлээд
модулийн metadata-аар (server-driven хуудсууд), дараагийн шат нь
`@gerege/ui` npm пакет болгож компонентуудыг гаргах. Микро-frontend
руу яарахгүй — зардал нь өндөр.

### 2.4 Гэрээнүүд тусдаа амьдрах

Repo хоорондын гэрээ болсон гурван зүйл цөмийн release-ээс хараат бус
хувилбартай байна: **каталог/manifest-ийн JSON schema** (апп сторын
гэрээ), **Shell Contract** (native бүрхүүлийн гэрээ — аль хэдийн
хувилбартай), **OIDC scope-ууд**. Эхэндээ цөм repo дотроо
байж болно, гэхдээ өөрийн version-тэй; олон баг хэрэглэж эхэлмэгц
жижиг тусдаа repo болгоно.

---

## 2.5 Одоогийн кодын хуваарилалт: юу цөмд үлдэж, юу distribution болох вэ

Ангилах гурван шалгуур: (1) **хоёроос олон төрлийн бүтээгдэхүүн хэрэглэх
үү** — тийм бол цөм; (2) **identity, тенант, стор, аюулгүй байдал, төрийн
дэд бүтцийн нэг хэсэг үү** — тийм бол цөм; (3) **нэг vertical-ын домэйн
уу** — тийм бол distribution.

Цөмд үлдэх модулиудын одоогийн нэршилд гурван алдаа бий тул салгалтаас
**өмнө** (SDK v1.0-ээс өмнө — нэр API-д баригдахаас өмнө) нэрийг нь
зөв болгоно:

### Цөмийн модулиудын шинэ нэршил

| Шинэ нэр (ID) | Хуучин | Юу өөрчлөгдөх вэ |
| --- | --- | --- |
| **`organisation`** — Байгууллага (`io.gerege.nexus.organisation`) | `apps/core` | "Core" гэдэг нэр нь платформын техникийн цөмтэй андуурагдана. Мөн апп нь хоёр давхаргыг нийлүүлчихсэн: **тенантын хуулийн профайл** (нэр, регистр, лого) нь апп биш platform-ын tenant-ийн шинж чанар тул `platform/tenant` руу нүүнэ; **хэлтэс/ажилтан** (HR-lite) нь апп хэвээр core repo-д үлдэнэ. Кодын аудитаар өөр нэг ч модуль үүнээс хамаардаггүй нь тогтоогдсон тул "устгагдашгүй" статусыг хасч, default-суудаг-гэхдээ-устгаж-болдог болгоно — платформ 0 апптай асч чаддаг байх нь экосистемийн суурийн шалгуур |
| **`egov`** — Цахим засгийн холболт (`io.gerege.nexus.egov`) | Шинэ модуль — ХУР/ДАН/eID-ийн хэрэглэгчид харагдах урсгалууд одоо `contacts` болон platform-ийн дотор тарсан байгааг нэгтгэнэ | ХУР иргэн/хуулийн этгээдийн лавлагаа (`/xyp/*`), eID/ДАН холболтын дэлгэц, баталгаажуулалтын түүх — эдгээр нь "харилцагчийн бүртгэл" биш, Gerege-ийн төрийн дэд бүтцийн нүүр. Доод түвшний клиентүүд (`platform/gerege`, `platform/eid`, `platform/dan`) хэвээр platform-д үлдэж, `egov` нь тэдний апп-нүүр нь болно |
| **`contacts`** — Харилцагч (хэвээр) | `apps/contacts` | Цэвэрлэгдэнэ: зөвхөн харилцагчийн бүртгэл үлдэнэ; ХУР авто-бөглөлт нь `egov`-ийн үйлчилгээг дууддаг болно. Олон модулийн FK суурь тул цөмд (Odoo-гийн base/contacts статус) |
| **`sso_clients`** — SSO клиентүүд (`io.gerege.nexus.sso_clients`) | `apps/developer_portal` | Энэ модуль үнэндээ платформын OIDC provider-т OAuth2 client бүртгэдэг CRUD (`/api/v1/developer`) — "Developer Portal" гэдэг нэр нь апп сторын жинхэнэ хөгжүүлэгчийн консол (developer.gerege.mn, `publisher_studio`)-той шууд мөргөлдөж төөрөгдүүлнэ. Нэрийг үүргээр нь: SSO/холболтын клиентүүд |
| **`documents`**, **`esign`** (хэвээр) | — | Нэр зөв. eID signing rail platform-д тул цөмд; хожим "gerege-docs" болгож салгах нэр дэвшигч |
| **`reports`** — Тайлан (хэвээр) | — | Хөдөлгүүр нь `platform/reporting`, апп UI нь бүх бүтээгдэхүүнд хэрэгтэй |

Нэр солилт нь өгөгдлийн миграц гэдгийг анхаар: module ID нь
`app_installations`, каталог manifest, цэсний түлхүүр, frontend route-д
хадгалагддаг. Тиймээс ID солих нь: каталогт шинэ ID + хуучин ID-г alias
болгон нэг release хадгалах + `app_installations`-ийг UPDATE хийх миграц +
route redirect. Энэ зардал нь SDK гарсны **дараа** төлбөл хэд дахин
өснө — тиймээс нэршлийн засвар нь дарааллын 1-р алхмын өмнөх ажил.

### Цөмд мөн үлдэх (модуль биш хэсгүүд)

| Хэсэг | Тайлбар |
| --- | --- |
| `internal/platform/*` бүхэлдээ | auth, SSO provider/client, tenant, RBAC, dbguard, appcatalog/appinstaller/appregistry, resilience, observability, audit, settings, flags, quota, metering, controlplane, reporting engine, eid/dan/gerege(ХУР) клиентүүд, emailverify, ai, integration — платформ гэдгийн тодорхойлолт |
| Frontend shell | login/auth/settings/profile/organisation/apps(store UI)/module framework/cp/impersonate/reports + kiosk бүрхүүл (төхөөрөмжийн туршлага нь платформынх) |
| `native-apps/*`, `catalog/` schema + tooling, `deploy/` суурь | Бүрхүүл, гэрээ, дэд бүтэц |

### Distribution болгож салгах

| Шинэ repo | Одоогийн код | Шалтгаан |
| --- | --- | --- |
| **gerege-appstore** | `apps/appstore_registry`, `apps/publisher_studio`, `apps/store_review` | Зөвхөн appstore.gerege.mn deployment-д хэрэгтэй — бусад зуун платформ энэ кодыг үхмэл ачаа болгож тээх шаардлагагүй. GitLab дээрх одоогийн appstore repo-уудтай нэгтгэж нэг бүтээгдэхүүн болгоно. **Хамгийн эхэнд салгах нь энэ** — хил нь хамгийн тод |
| **gerege-commerce** | `apps/products`, `apps/inventory`, `apps/billing` + frontend `pos/` | Худалдаа-агуулах-нэхэмжлэхийн vertical; e-Barimt, НӨАТ нь домэйн логик. products-ийг billing/inventory л ашигладаг тул хамт явна |
| **gerege-gov** | `apps/gov_services` + frontend `line/` (цахим дараалал) | Төрийн үйлчилгээний vertical — шийдвэрлэх урсгал, цаг захиалга, дараалал |

Салгасны дараа цөм нь "хоосон боловч бүрэн" платформ болно: нэвтрэлт,
байгууллага, харилцагч, баримт/гарын үсэг, тайлан, стор — дээрээс нь
каталогоор ямар ч vertical суудаг. Demo deployment нь гурван
distribution-ийг бүгдийг нь суулгасан Түвшин 2-ын нэг жишээ байдлаар
үлдэнэ.

### Салгалтын дараалал

0. **Нэршлийн засвар** (`core`→`organisation`, `developer_portal`→
   `sso_clients`, `egov` модуль ялгаж гаргах) — SDK-ээс өмнө;
1. `gerege-appstore` — хил тод, хэрэглэгч цөөн, эхний туршилт;
2. `gerege-gov` — дараагийн тод vertical;
3. `gerege-commerce` — contacts/products-ийн FK хамаарлыг SDK-ийн
   interface-ээр цэвэрлэх шаардлагатай тул хамгийн сүүлд;
4. `documents+esign` — зөвхөн жинхэнэ хэрэгцээ гарвал (өөр signing
   бүтээгдэхүүн төрөх г.м.), яарахгүй.

Салгалт бүр нэг л шалгуураар "болсон" гэж тоологдоно: цөмийн repo-д
тухайн домэйны нэр бүхий import үлдээгүй, distribution нь цөмийг зөвхөн
`go.mod`-оор авдаг, хоёул CI ногоон.

## 3. Апп сторын байр суурь

Апп стор (тусдаа байгаа `appstore-gerege-mn` registry +
`developer-gerege-nexus` console) нь экосистемийн **түгээлтийн төв**:

- **Каталог бол repo хоорондын гэрээ** — deployment аль аппуудыг
  санал болгохоо гарын үсэгтэй каталогоос авна (`APP_CATALOG_URL` +
  `APPSTORE_PUBLIC_KEY` механизм бэлэн). Түвшин 1-ийн платформууд
  **зөвхөн каталогоор** ялгарч ч чадна — нэг образ, өөр апп багц.
- Түвшин 2-ын модуль compile-time орж ирдэг ч **бүртгэл нь каталогт** —
  тиймээс distribution бүр өөрийн catalog profile-ийг registry-д
  publish хийнэ (`cmd/publish-catalog` бэлэн).
- Түвшин 3-ын аппууд developer console-оор бүртгэгдэж, гуравдагч
  хөгжүүлэгчид нээлттэй.
- Registry унасан ч deployment ажиллана (catalog cache + файл fallback
  аль хэдийн хийгдсэн) — энэ шинжийг хадгална.

## 4. Хамтын дэд бүтэц (зуун repo-г нэг баг дааж чадах нөхцөл)

- **Reusable CI**: GitHub Actions-ийн `workflow_call` — build/test/
  publish workflow нэг `gerege-ci` repo-д нэг удаа бичигдэж, бүх
  distribution нэг мөрөөр дуудна. CI засвар нэг газар хийгдэнэ.
- **Template repo**: `gerege-platform-template` — Түвшин 2-ын скелет
  (дээрх бүтэц + CI + README). Шинэ платформ = template-ээс үүсгэх +
  нэр солих, хагас өдрийн ажил.
- **Renovate**: бүх distribution-д цөмийн шинэ tag автомат PR.
- **go.work**: цөм + distribution зэрэг засах хөгжүүлэгчид local
  workspace (commit хийгдэхгүй).

## 5. Дүрмүүд (CONTRIBUTING-д орох)

1. **Upstream-first**: distribution дээр ажиллаж байгаад цөмд хэрэгтэй
   засвар олдвол цөм рүү PR илгээнэ — distribution-д хуулбарлаж
   засахыг хориглоно. "Түр хуулчихъя" бол зуун repo-гийн drift-ийн эх.
2. Цөмөөс файл хуулж авчрахыг code review-д татгалзах шалтгаан гэж үзнэ.
3. Distribution-д бизнес логик зөвхөн **модуль** хэлбэрээр — `main.go`
   дотор custom код бичихгүй.
4. Түвшин 1-ээр шийдэгдэх зүйлд Түвшин 2 нээхгүй; түвшин ахиулах
   шийдвэр нь архитектурын шийдвэр тул бичигдэж үлдэнэ.

## 6. Дараалал

| Алхам | Ажил | Үр дүн |
| --- | --- | --- |
| 1 | `pkg/nexus` SDK гаргаж, дотоод 9 модулийг түүн рүү шилжүүлэх | Гадны repo модуль бичиж чаддаг болно |
| 2 | Semver release журам + эхний `v1.0.0` tag | Dependency болж чадна |
| 3 | Брэндийг runtime тохиргоо болгох + deploy config repo | Түвшин 1 бүрэн ажиллана |
| 4 | `gerege-ci` reusable workflows + template repo | Шинэ платформ хагас өдөрт |
| 5 | Каталог schema-г хувилбаржуулах, distribution-ий catalog profile publish урсгал | Апп стор экосистемийн төв болно |

Энэ дарааллын 1-2 нь бусад бүхний урьдчилсан нөхцөл. Одоо fork хийчихсэн
юм байвал (байгаа бол) эхний ажил нь буцааж distribution хэлбэрт оруулах.

## 7. Экосистемийн хөгжүүлэлт өдөр тутамдаа хэрхэн явах вэ

Гурван төрлийн баг гурван өөр хэмнэлээр ажиллана — тэдгээрийг холбож
байгаа зүйл нь semver tag ба каталог.

**Цөмийн баг** (`open-gerege-nexus`): trunk-based — богино насалдаг
branch, `main` руу PR, бүрэн CI (одоогийн lint/test/govulncheck/gosec).
Release train: **2 долоо хоног тутам minor** (`v1.5.0`), яаралтай засвар
patch (`v1.5.1`) хэлбэрээр. `pkg/nexus` SDK-д нөлөөлөх өөрчлөлт бүр
өмнө нь богино design doc (яг одоогийн `docs/*_PROPOSAL.md` маягаар)
+ CODEOWNERS-ийн review шаардана — API бол зуун repo-гийн гэрээ тул
кодоос удаан амьдарна.

**Бүтээгдэхүүний багууд** (distribution repo-ууд): өөрийн модулиудаа
өөрийн хэмнэлээр хөгжүүлнэ — цөмийн release хүлээхгүй, өөрсдөө хүссэн
үедээ deploy хийнэ. Цөмтэй харьцах нь хоёрхон урсгал:

1. *Цөм шинэчлэгдэхэд*: Renovate шинэ tag-ийн PR автоматаар нээнэ →
   distribution-ий CI (өөрийн модулийн тест + compose асааж `/health`
   smoke) ногоон бол merge. Улаан бол цөмийн release-д асуудал байна
   гэсэн дохио — цөмийн багт issue очно. Ингэснээр "цөмийн шинэ хувилбар
   экосистемээ эвдсэн эсэх" нь release хийснээс хойш цагийн дотор, зуун
   CI-гийн үр дүнгээр харагдана.
2. *Цөмд өөрчлөлт хэрэгтэй болоход*: upstream-first — цөм рүү PR; local
   орчинд `go.work`-оор цөмийн branch-тайгаа зэрэг ажиллаж тестэлнэ,
   merge болмогц дараагийн tag-ийг хүлээж авна. Яаралтай бол цөмийн баг
   patch release гаргана — distribution дотор цөмийн кодыг хуулж
   "түр засах" зам байхгүй.

Жишээ — "тээврийн бүтээгдэхүүнд шинэ тайлан нэмэх": тайлан нь модулийн
код тул `gerege-transport` repo-д л бичигдэнэ, өнөөдөр л deploy болно.
Харин "тайлангийн хөдөлгүүрт шинэ chart төрөл хэрэгтэй" бол цөмийн
`pkg/nexus` руу PR → 2 долоо хоногийн train → Renovate bump. Багууд энэ
хоёр замын ялгааг мэддэг байх нь л жинхэнэ сургалт.

**Гуравдагч хөгжүүлэгчид** (Түвшин 3): developer console-оор бүртгэж,
нийтийн **sandbox deployment** (Түвшин 1-ийн нэг суулгац, DEMO_MODE)
дээр аппаа тестэлж, каталогт publish хүсэлт өгнө. Тэдний код экосистемийн
git-д огт орж ирэхгүй — гэрээ нь зөвхөн OIDC + manifest.

**Орчнууд**: хөгжүүлэгч бүр local compose; цөмийн `main` тутмын build
автоматаар нэг **staging deployment** дээр суудаг (экосистемийн бүх
шинэ зүйл эхэлж энд уулзана); production-ууд нь тус тусын deploy config
repo-гийн хэмнэлээр. Мониторингийн стек (энэ баримтын хос төлөвлөгөө)
staging дээр мөн адил ажиллаж, release-ийн регрессийг эрт барина.

## 8. Эх сурвалж

- [Managing white-label solutions — git workflow](https://medium.com/flawless-app-stories/managing-white-label-solutions-8ed8ce9d7fa8)
- [GitOps architecture, patterns and anti-patterns](https://platformengineering.org/blog/gitops-architecture-patterns-and-anti-patterns)
- [Git branching strategy guidance — Microsoft](https://learn.microsoft.com/en-us/azure/devops/repos/git/git-branching-guidance?view=azure-devops)
- Go-гийн `internal` пакетын дүрэм: https://go.dev/doc/go1.4#internalpackages
- Загвар болгосон туршлагууд: Odoo (core + addons-path), Kubernetes
  (core + distributions), GitLab (CE + omnibus packaging)
