# Control Plane — операторын консолын төлөвлөгөө

**cp.nexus.gerege.mn** дээр ажиллах платформ операторын консол —
дэлхийн туршлагын судалгаа ба бүрэн feature төлөвлөгөө.

[Баримт бичгийн төв рүү буцах](README.md) ·
Холбоотой: [Мониторинг ба тайлангийн санал](MONITORING_AND_REPORTING_PROPOSAL.md)

---

## 1. Байрлал ба зарчим

SaaS архитектурын туршлагад платформыг **data plane** (тенантуудын ажилладаг
апп — nexus.gerege.mn) ба **control plane** (оператор платформоо удирддаг
консол — cp.nexus.gerege.mn) гэж ялгадаг. Консолоо яг энэ ойлголтын нэрээр
нь **Control Plane** гэж нэрлэнэ.

Үндсэн шийдвэрүүд:

| Шийдвэр | Сонголт | Шалтгаан |
| --- | --- | --- |
| Backend | **Нэг Go бинари дотор** тусдаа route бүлэг (`/cp/api/...`) | Нэг-бинари философи; тусдаа сервис нь deploy/мониторингийн давхар зардал |
| Frontend | Одоогийн Next.js дотор `app/cp/` хэсэг, nginx virtual host-оор `cp.` домэйнд | Компонент, дизайн систем дахин ашиглагдана |
| Хандалт | nginx түвшинд `cp.` хост **IP allowlist / VPN-ээр** хязгаарлагдана | Админ интерфэйс интернэтэд ил байх ёсгүй (NCSC зарчим) |
| Оператор бүртгэл | `operator_accounts` — тенантын хэрэглэгчээс **бүрэн тусдаа** хүснэгт, тусдаа session | Тенантын бүртгэл алдагдахад control plane өртөхгүй |
| Өгөгдөлд хандах | dbguard-д тусдаа, тодорхой **operator горим** — RLS bypass биш, зориулалтын query бүрт explicit | Default deny хэвээр; "бүхнийг хардаг" нь "бүх query нээлттэй" гэсэн үг биш |
| Env засвар | **Хийхгүй.** Динамик тохиргоо DB-д, нууц/bootstrap нь GitHub secrets + deploy pipeline-д | Өмнөх дүгнэлт: env-UI нь RCE-тэй тэнцүү эрсдэл + deploy бүрт дарагдана |

## 2. Аюулгүй байдлын суурь (бүх feature-ээс өмнө)

NCSC-ийн админ интерфэйс хамгаалах заавар, Microsoft Entra-гийн privileged
access туршлагад тулгуурлав:

1. **Нэвтрэлт**: операторт нууц үг + **WebAuthn (хамгийн сайн) эсвэл TOTP
   заавал**; session богино (жишээ нь 4 цаг idle биш 30 мин); төхөөрөмж
   бүртгэх сонголт.
2. **Операторын RBAC** — 4 үүрэг:
   `superadmin` (бүх эрх), `operator` (тенант/тохиргооны өдөр тутмын ажил,
   устгал болон нууц ил хийхгүй), `support` (унших + хэрэглэгчийн дэмжлэгийн
   үйлдлүүд), `auditor` (зөвхөн унших + audit хайх).
3. **Step-up + хоёр хүний зарчим**: аюултай үйлдэлд (тенант устгах,
   impersonate, deploy өдөөх, kill switch) тухайн агшинд дахин 2FA
   баталгаажуулалт; тенант устгалд хоёр дахь superadmin-ий батламж
   (four-eyes) шаардах тохиргоотой.
4. **Break-glass бүртгэл**: нэг онцгой бүртгэл, урт санамсаргүй нууц үг нь
   офлайн сейфэнд; ашиглагдмагц бүх операторт alert (Alertmanager-ээр).
   Ердийн үед хэзээ ч хэрэглэгдэхгүй.
5. **Append-only audit**: `operator_audit` хүснэгт — оператор, үйлдэл,
   бай (тенант/хэрэглэгч/тохиргоо), шалтгаан (заавал бичдэг талбар),
   өмнөх/дараах утга, IP, цаг. UPDATE/DELETE эрхгүй (DB түвшинд REVOKE);
   Loki руу давхар урсана. CP-ийн бүх бичих үйлдэл эндгүйгээр өнгөрөхгүй.
6. **Rate limit + alert**: CP нэвтрэлтийн оролдлого, амжилтгүй step-up
   бүр хэмжигдэж, гажигт alert.

## 3. Feature багцууд

### A. Тенантын удирдлага (гол багц)

- Жагсаалт: хайлт (нэр, slug, регистр), шүүлт (төлөв, идэвх, үүссэн огноо),
  эрэмбэ (хэрэглээ, хэрэглэгчийн тоо).
- **Үүсгэх**: нэр/slug/байгууллагын мэдээлэл + суулгах аппын багц (загвар:
  "худалдаа", "төрийн байгууллага" г.м.) + эхний админд урилгын и-мэйл.
- **Дэлгэрэнгүй хуудас**: суусан аппууд (нэмэх/хасах), хэрэглэгчид ба
  үүргүүд, хэрэглээ (хэрэглэгч, хадгалалт, API/AI дуудлага), гадаад
  интеграцын төлөв (eID, ХУР ажиллаж буй эсэх), сүүлийн идэвх, audit урсгал.
- **Амьдралын мөчлөг**: идэвхтэй → **түдгэлзүүлсэн** (нэвтрэлт хаагдана,
  өгөгдөл хэвээр, шалтгаан бичигдэнэ) → **устгал хүлээж буй** (30 хоног,
  энэ хугацаанд export татаж болно, тенантын админд мэдэгдэл) → устгасан.
  Шууд hard delete байхгүй; сэргээх нь grace хугацаанд нэг товч.
- **Quota**: тенант бүрд хэрэглэгчийн тоо, хадгалалт, AI дуудлагын сарын
  хязгаар; хэтрэлтэд зөөлөн (анхааруулга) ба хатуу (блок) горим.

### B. Хэрэглэгчийн дэмжлэг (support)

- Хэрэглэгч хайх (и-мэйл, нэр, регистр) — аль тенантуудад ямар үүрэгтэйг
  харах.
- Нууц үг reset илгээх, 2FA reset (step-up-тай), бүх session хүчингүй
  болгох, login lockout тайлах (одоогийн 00028 механизм дээр).
- **Impersonation** — "тенант дотогш нь орж харах": шалтгаан заавал
  бичигдэнэ, хугацаатай (30 мин), дэлгэцэд байнгын улбар шар banner,
  бичих үйлдэл бүр давхар тэмдэглэгдэнэ, тенантын админд мэдэгдэл очно.
  Зөвхөн `support`+ эрхтэй, step-up шаардана. (AppMaster-ийн consent +
  audit загвар.)

### C. Платформын тохиргоо

- **Нэвтрэлтийн горим: public / private** — платформ түвшний үндсэн
  тохиргоо, CP-ээс restart-гүй солигдоно:
  - **Public**: хэн дуртай бүртгүүлж нэвтэрч болно — бүртгэлийн дэлгэц
    нээлттэй, eID/ДАН-аар анх удаа нэвтэрсэн хүнд бүртгэл автоматаар
    үүснэ (JIT provisioning).
  - **Private**: зөвхөн урьдчилан бүртгэгдсэн хэрэглэгч нэвтэрнэ —
    бүртгэлийн дэлгэц хаагдана, eID/ДАН/SSO-гоор баталгаажсан ч бүртгэлгүй
    хүнд шинэ account үүсэхгүй ("Танд эрх нээгдээгүй байна" гэсэн
    ойлгомжтой хариу). Тенантын админы **урилга** private горимд ч
    ажиллана — урилгаар ирсэн хүн "урьдчилан бүртгэгдсэн"-д тооцогдоно.
  - Аюулгүй анхдагч утга нь **private**; горим солигдох бүр audit-д
    шалтгаантайгаа бичигдэнэ. Шалгалт нэг цэгт (auth давхаргад)
    төвлөрч, бүртгэл үүсгэдэг бүх зам (email signup, eID/DAN JIT,
    SSO_CLIENT_TENANT JIT) нэг дүрмээр хаагдана.
- **`platform_settings`** — DB-д хадгалагдах динамик тохиргоо: түлхүүр
  бүр төрөлтэй (bool/int/string/enum), validation-тай, тайлбартай.
  Өөрчлөлт бүр хувилбарын түүхтэй, нэг товчоор буцаана, restart шаардахгүй
  (backend 30 сек тутам эсвэл Redis invalidation-оор шинэчилнэ).
  Эхний нүүдэл: одоогийн env-ээс аюулгүй, динамик байж болох хэсэг нь
  (session idle timeout, catalog sync interval, AI model сонголт г.м.)
  шилжинэ. **Нууц утга энд хэзээ ч орохгүй.**
- **Feature flags** — Unleash/LaunchDarkly-ийн туршлагаар: flag бүр
  нэр/зориулалт/эзэмшигчтэй, тенант бүрээр эсвэл хувиар (percentage
  rollout) асаана, **kill switch** төрлийн flag нь аль ч модулийг түр
  унтраана; flag-ууд хугацаатай — хөгширсөн flag-ийг цэвэрлэх сануулга
  (flag debt-ээс сэргийлэх).
- **Maintenance mode**: платформ даяар эсвэл тенант сонгож — banner +
  зөвхөн унших горим; товлосон засварын зарлал урьдчилан харагдана.
- **Зарлал (broadcast)**: бүх/сонгосон тенантын хэрэглэгчдэд банner эсвэл
  и-мэйл мэдэгдэл.

### D. Каталог ба апп стор

- Каталог синкийн төлөв (сүүлийн амжилттай таталт, гарын үсгийн шалгалт),
  registry унасан үед кэшээс явж буйг харуулах.
- Апп хувилбарууд, тенант бүрийн суулгалтын тархалт, шинэ хувилбар руу
  албадан шилжүүлэх товч (rollout хувиар).
- Модулийн kill switch (C-гийн flag дээр суурилна).

### E. Ажиглалтын тойм (мониторингийн Үе шат 2-той уялдана)

- Нүүр хуудас = платформын эрүүл мэнд: API error rate/latency, гадаад
  системүүдийн (ХУР, eID, ДАН, eSign, Gemini) төлөв гэрэл, диск/DB/Redis,
  идэвхтэй alert-ууд. Гүнзгий шинжилгээ нь Grafana руу (нэг товч, SSO биш
  ч нэг network дотор) — CP нь Grafana-г орлохгүй, тоймлоно.
- Тенант бүрийн алдааны түвшин — аль тенант зовж байгааг support-оос өмнө
  мэдэх.
- Background ажлууд: товлосон тайлангийн гүйлт, каталог синк, миграцийн
  түүх — сүүлийн төлөв, алдаа.
- **Audit хайгч**: тенантын audit + операторын audit хоёуланг шүүж хайх UI.

### F. Үйл ажиллагаа (operations)

- **Deploy товч**: GitHub Actions *Deploy to Production* workflow-г
  workflow_dispatch API-гаар өдөөнө (tag сонгож болно) — env засахгүй,
  серверт exec хийхгүй; явц нь Actions линкээр. Step-up шаардана.
- Backup төлөв: сүүлийн амжилттай pg_dump/backup цаг, хэмжээ, **сүүлийн
  амжилттай restore test-ийн огноо** (тестлээгүй backup = backup биш).
- TLS/domain: сертификатын хугацаа, device-line origin-уудын төлөв.
- Хувилбарын мэдээлэл: одоо ажиллаж буй image tag, git sha, миграцийн
  түвшин.

### G. Metering (ирээдүйн billing-ийн суурь)

- `usage_events` — тенант бүрийн хэмжигдэхүүн (идэвхтэй хэрэглэгч,
  хадгалалт, API/AI дуудлага, илгээсэн тайлан) өдрөөр aggregate.
- CP дээр тенант бүрийн хэрэглээний график; export. Хожим төлбөрийн
  модуль энэ өгөгдөл дээр суух тул эхнээсээ цэвэр цуглуулна.

## 4. Юуг санаатайгаар ХИЙХГҮЙ вэ

| Хийхгүй | Оронд нь |
| --- | --- |
| Env засах UI | `platform_settings` (C багц) + GitHub secrets |
| Серверт shell/exec, SQL console | Зориулалтын, audit-тай үйлдлүүд л байна |
| RLS-ийг унтраасан чөлөөт query | Feature бүр өөрийн explicit query-тэй |
| Тенантын өгөгдлийг CP-д чөлөөтэй үзэх | Зөвхөн metadata + impersonation (журамтай) |
| Шууд hard delete | Түдгэлзүүлэх → 30 хоног → устгал |

## 5. Хэрэгжүүлэлтийн үе шат

| Үе шат | Агуулга | Хугацаа | Хамаарал |
| --- | --- | --- | --- |
| CP-1 | Суурь: cp хост + nginx allowlist, operator_accounts + WebAuthn/TOTP, RBAC, append-only audit, тенантын жагсаалт/дэлгэрэнгүй (зөвхөн унших) | ~2 д.х. | — |
| CP-2 | Тенантын амьдралын мөчлөг (үүсгэх/түдгэлзүүлэх/grace устгал/quota) + support багц (reset, impersonation) | ~2 д.х. | CP-1 |
| CP-3 | platform_settings + feature flags + maintenance/broadcast | ~2 д.х. | CP-1 |
| CP-4 | Ажиглалтын тойм + operations (deploy товч, backup, каталог) | ~1-2 д.х. | CP-1, мониторингийн Үе шат 2 |
| CP-5 | Metering + тенантын хэрэглээний график | ~1-2 д.х. | CP-1 |

CP-2, CP-3, CP-5 нь хоорондоо хамааралгүй тул CP-1-ийн дараа зэрэгцээ
явж болно. Мониторингийн Үе шат 1-2 (хэмжүүр + стек) нь CP-4-ийн урьдчилсан
нөхцөл тул эхэлж хийгдсэн байх нь зүйтэй.

## 6. Үндсэн платформоос Control Plane руу нүүх feature-ууд

Одоогийн кодод (`server.go`-гийн route-ууд) "тенантын админ" эрхээр
хамгаалагдсан ч үнэндээ **deployment бүхэлд нь** нөлөөлдөг хэд хэдэн
feature бий. CP нэмэгдмэгц эдгээр гурван ангилалд хуваагдана:

### А. Бүрэн CP руу нүүх (тенантын аппаас хасагдана)

| Одоогийн байрлал | Юу вэ | Яагаад |
| --- | --- | --- |
| `POST /admin/store/sync` | Registry-ээс каталог татах | Кодын өөрийнх нь comment-оор: "changes what every tenant on it is offered" — нэг тенантын админ бүх тенантад нөлөөлдөг үйлдэл хийж байна |
| `GET /admin/store/status` | Каталогийн эх сурвалж, сүүлийн синкийн төлөв | Registry-ийн нэр, алдаа — deployment-ийн мэдээлэл |
| `GET /admin/store/overview` | Бинари/каталог/суулгалтын хувилбарын зөрүү | Мөн адил — платформ бүхэлд нь харах ёстой харагдац (CP-4-ийн каталог хэсэгт очно) |

### Б. Хуваагдах (тенантын хэсэг үлдэж, платформын хэсэг CP руу)

| Feature | Тенантад үлдэх | CP руу очих |
| --- | --- | --- |
| И-мэйл баталгаажуулалт (`/admin/email-verification/overview`) | Өөрийн тенантын баталгаажуулалтын түүх | Hosted service-ийн төлөв, API key байгаа эсэх, платформ даяарх квот/алдаа |
| AI (`/admin/ai/prompts`, `ai_prompts` хүснэгт) | Тенантын өөрийн prompt override, мэдлэгийн сан | Платформын анхдагч prompt-ууд (одоо миграцын seed-ээр л солигддог), Gemini model/квотын тохиргоо |
| Integrations (`/integrations`) | Тенантын өөрийн connector-ууд | Google/Dropbox OAuth client (compose-д "clients belong to whoever operates this deployment" гэж бичсэн — одоо env-ээр), `INTEGRATION_ALLOW_PRIVATE_TARGETS`, encryption key-ийн төлөв |
| eSign | Тенантын гарын үсгийн policy дэлгэц (`/settings/policy`) | Rail-уудын deployment төлөв: ESIGN_TOKEN байгаа/mock эсэх, eID stamp fallback cert-ийн төлөв — "mock горимд SIGNED тэмдэглэгддэг" осол дахин гарахаас сэргийлэх харагдац |
| OAuth2 clients | Developer portal-аар тенантын бүртгэсэн client-ууд | Платформ эзэмшдэг client-ууд (SSO default, developer console) — жагсаалт, secret rotate |

### В. CP-д давхар (нэмэлт) харагдац үүсэх — тенантаас юу ч хасагдахгүй

- **Mock горимуудын төлөв** (EID/DAN/XYP/ESIGN) — одоогоор env-д тарсан;
  CP нүүрэнд нэг мөрөнд: аль нь mock, production-д зөрчилтэй эсэх.
- **DEMO_MODE / demo seeder-ийн төлөв** — public showcase эсэх нь нэг
  харцаар харагдана.
- **Төхөөрөмжүүд** (`/admin/devices` — киоск/POS) — тенант админ өөрийнхөө
  fleet-ийг удирдсан хэвээр; CP-д бүх тенантын төхөөрөмжийн нэгдсэн тоо,
  enrollment-ийн идэвх.
- **Login lockout / session** — тенант доторх удирдлага хэвээр; CP support
  багц (CP-2) нь тенант дамнасан хайлт, тайлах эрхтэй.

Нүүлгэлтийн дүрэм: А ангиллын endpoint-уудыг CP бэлэн болмогц
`requireAdmin`-аас `operator`-т шилжүүлж, тенантын UI-гаас цэсийг нь
хасна; Б ангилалд эхлээд CP талын шинэ дэлгэц нэмэгдэж, дараа нь тенант
талын дэлгэцээс платформын мэдээлэл алга болно. Хоёр алхмын хооронд
хуучин endpoint-ууд deprecated тэмдэгтэй ажиллана — native клиентүүд
шинэчлэгдэх хугацаа өгнө.

## 7. Эх сурвалж

- [NCSC — Protect your administration interfaces](https://www.ncsc.gov.uk/collection/secure-system-administration/protect-your-administration-interfaces)
- [AWS — Manage tenants on a single control plane](https://docs.aws.amazon.com/prescriptive-guidance/latest/patterns/manage-tenants-across-multiple-saas-products-on-a-single-control-plane.html)
- [Microsoft Entra — Secure access practices for administrators](https://learn.microsoft.com/en-us/entra/identity/role-based-access-control/security-planning)
- [Break glass account best practices](https://www.safous.com/blog/the-ultimate-guide-to-break-glass-account-security)
- [Secure admin impersonation with consent and audits](https://appmaster.io/blog/secure-admin-impersonation-controls-audit-scope)
- [Unleash — Feature flag best practices at scale](https://docs.getunleash.io/guides/best-practices-using-feature-flags-at-scale)
- [Unleash — 11 best practices for feature flag systems](https://docs.getunleash.io/guides/feature-flag-best-practices)
- [Octopus — Feature flag types (kill switch г.м.)](https://octopus.com/devops/feature-flags/)
- [Multi-tenant SaaS architecture guide 2026](https://mallary.ai/blog/multi-tenant-saas-architecture)
- [Northflank — Multi-tenant SaaS deployment production guide](https://northflank.com/blog/multi-tenant-saas-platform-deployment)
