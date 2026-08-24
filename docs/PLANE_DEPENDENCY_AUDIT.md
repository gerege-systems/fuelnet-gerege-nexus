# Control/data хамаарлын аудит

2026-08-24-нд `TWO_PLANES_PROPOSAL.md` §2.9-ийн дүрэм 3-ыг шалгав:

> Өгөгдлийн зам удирдлагын эзэмшдэг шийдвэрийг хүсэлт бүрд өгөгдлийн
> сангаас синхроноор дахин уншихгүй. Санах ойн snapshot нь timer болон
> invalidation-аар шинэчлэгдэнэ.

## Баталгаа

`internal/tenant/gate_e2e_test.go` дахь
`TestARequestUsesCachedControlDecisionsWhenTheirTablesAreUnavailable` бодит
PostgreSQL, бодит session middleware ашиглана. Тест:

1. settings, flags, tenant lifecycle, app installation, permission cache-уудыг
   урьдчилан хална;
2. тэдгээрийн эх хүснэгтүүдийг өөр transaction-аар `ACCESS EXCLUSIVE` lock
   аван унших боломжгүй болгоно;
3. session баталгаажуулалт → app gate → permission gate → handler гэсэн бодит
   хүсэлтийг явуулна;
4. settings ба kill-switch snapshot-оос уншигдаж, хүсэлт `204` өгөхийг
   батална.

Түгжигдсэн хүснэгт рүү шууд probe хийж context deadline авч байгаа тул тест
зөвхөн хурдан DB дээр тохиолдлоор ногоорохгүй. Cache miss гарвал хүсэлтийн
хоёр секундын deadline тестийг унагана.

## Үлдсэн шууд хамаарал — issue backlog

Энэ үе шат тэдгээрийг засахгүй. Доорх ID бүр тусдаа issue/PR байх хэмжээний
зан төлөв, invalidation-ийн шийдвэр шаарддаг.

### PLANE-G-1 — Session бүр role хүснэгт уншдаг

`internal/tenant/auth/session.go`-ийн `SessionStore.Resolve` authenticated
хүсэлт бүрд `membership_roles` ба `roles`-оос administrator эсэхийг бодно.
Permission cache халсан байсан ч энэ join үлдэнэ.

Санал: session claim-д admin төлвийг snapshot болгох эсвэл grant cache-тай нэг
membership-decision cache ашиглах. Role өөрчлөх transaction commit хийсний
дараа тухайн tenant/user-ийн entry-г bus-аар invalidate хийх ёстой. Эрх
цуцалсны дараах 30 секундын цонхыг зөвшөөрөх эсэхийг issue дээр тусад нь
шийднэ.

### PLANE-G-2 — Tenant maintenance хүсэлт бүрд `tenants` уншдаг

`internal/tenant/auth/accessmode.go`-ийн `Maintenance` mutating хүсэлт бүрд
`platform.tenants.maintenance_*` багануудыг шууд уншина. Platform-wide
maintenance setting санах ойд байгаа боловч tenant-level шийдвэр байхгүй.

Санал: одоогийн suspension cache-ийг lifecycle snapshot
(`suspended`, `maintenance`, message) болгоно. Platform console-ийн
`TenantChanged` callback ба tenant admin-ийн profile write хоёр commit-ийн
дараа ижил cache key-г invalidate хийнэ.

### PLANE-G-3 — Announcement `/me` дээр шууд уншигддаг

`internal/tenant/auth/accessmode.go`-ийн `Notices` нь shell-ийн `/me` хүсэлт
бүрд `platform.announcements`-ийг query хийнэ.

Санал: settings/flags-ийн адил process-wide announcement snapshot нэмнэ.
Console write commit хийсний дараа bus invalidation, 30 секундийн timer
fallback хэрэглэнэ.

### PLANE-G-4 — Quota config data path дээр шууд уншигддаг

Дараах enforcement замууд `platform.tenant_quotas`-ийг synchronous уншина:

- `internal/tenant/auth/suspension.go` — шинэ хэрэглэгч нэгдэх үеийн user quota;
- `internal/tenant/auth/suspension.go` — AI quota middleware;
- `internal/tenant/quota/quota.go` — storage quota.

`platform.usage_events` бол data талын хэмжилт тул түүнийг унших нь энэ
зөрчилд орохгүй; харин операторын тогтоосон ceiling/enforcement config-ийг
tenant-аар cache хийнэ. Quota update commit хийсний дараа invalidate хийх ба
DB алдаанд одоогийн fail-open зан төлвийг хадгална.

### PLANE-G-5 — Бүх суулгасан аппын жагсаалт cache-гүй

`internal/tenant/appinstall/externals.go`-ийн `InstalledAppSet` report listing
бүрд `app_installations`-ийг шууд уншина. Нэг аппын `AppInstalled` gate cache-тэй
боловч set хэлбэрийн capability өөр замтай.

Санал: tenant-аар keyed installed-app-set snapshot нэмээд одоогийн
`ForgetGate` invalidation-д хоёуланг нь холбоно. Ингэснээр нэг аппын gate ба
report list нэг эх төлөв, нэг staleness window-той болно.

## Plane тус бүрийн timeout ба rate-limit санал

Одоогийн нэг `http.Server`-ийн `WriteTimeout` нь eID long poll-оос урт байх
ёстойг `pkg/platform/timeouts_test.go` хамгаалдаг. Түүнийг platform route-д
зориулж богиносговол tenant eID дахин 502 болно. Тиймээс transport deadline
ба handler budget-ийг салгана.

### Timeout

1. `http.Server`-ийн `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`,
   `IdleTimeout`-ийг process-wide хамгаалалт хэвээр үлдээнэ. `WriteTimeout` нь
   хамгийн урт tenant protocol (одоогоор eID poll)-оос үргэлж урт байна.
2. Plane mount бүр дээр request-context budget middleware нэмнэ:
   platform-ийн ердийн read/write 10 секунд (`operator.QueryTimeout`-тай
   адил), tenant-ийн ердийн хүсэлт 30 секунд байна.
3. Long-running route-ууд default budget-ийг нэрлэсэн exception-ээр солино:
   eID/Өртөө long poll, report query, tenant export, backup restore. Exception
   бүр өөрийн одоогийн domain timeout болон түүнээс бага PostgreSQL
   `statement_timeout`-тай байна.
4. Timeout хариуг нэг хэлбэрийн `504`/error code, plane ба route label-тай
   metric болгоно. Дараа нь бодит p95/p99 дээр тулгуурлан budget-ийг чангална;
   таамгаар process-wide deadline бууруулахгүй.
5. Test нэмэхдээ existing eID invariant-ийг хэвээр байлгаж, platform-ийн
   унтаа handler ердийн budget дээр тасрах, нэрлэсэн long route тасрахгүйг
   тусад нь батална.

### Rate limit

1. nginx-ийн `nexus_auth` ба `cp_auth` тусдаа zone хэвээр. API талын Redis
   key-г мөн `tenant:*` ба `platform:*` namespace-аар ялгана; нэг plane-ийн
   халдлага нөгөөгийн sign-in budget-ийг идэх ёсгүй.
2. Platform plane-ийн login limiter одоогийн 5 burst / минутын орчим хурдыг
   хадгална. Бусад platform write-д operator ID + IP хосолсон бага budget
   нэмж, audit/export/restore зэрэг үнэтэй үйлдлийг тусдаа bucket болгоно.
3. Tenant plane-ийн login, poll, verification limiter-үүдийг одоогийн
   route-specific shared budget-тай нь хадгална. App/module limiter-ийн key нь
   tenant ID + app ID + operation байна.
4. Process-wide load shedder нь хамгийн гадна талын ослын хамгаалалт хэвээр;
   plane budget нь түүнийг орлохгүй. Platform-д жижиг тусдаа concurrency
   reserve үлдээж, tenant traffic process-wide cap-д хүрсэн ч оператор incident
   response хийх боломжтой болгоно.
5. Эхний implementation-д config нэмэхээс өмнө reject/allow metric-ийг
   `plane`, `bucket` label-тай хэмжинэ. Default утга өнөөгийн effective
   budget-ийг өөрчлөхгүй; production хэмжилтгүй шинэ хатуу cap тавихгүй.
