# App Store-ыг салгаж appstore.gerege.mn дээр байршуулах төлөвлөгөө

**Хамрах хүрээ:** `appstore.gerege.mn` (registry + storefront), `developer.gerege.mn`
(хөгжүүлэгчийн консол), гуравдагч талын платформыг апп болгон бүртгэх загвар,
шинэчлэлтийн механизм.

**Огноо:** 2026-08-10 · **Сүүлд шинэчилсэн:** PR #76 (`auto-update-sweep`,
commit `75769ff6`) merge болсны дараах бүрэн re-review-ээр.

---

## 0. Гүйцэтгэлийн байдал — ТӨСӨЛ ҮНДСЭНДЭЭ ДУУССАН

Гурван давалгаагаар бүрэн хэрэгжив:

| Давалгаа | PR | Юу орсон |
| --- | --- | --- |
| Nexus бэлтгэл | #68 | installed_version/app_versions засвар, upgrade endpoint + UI, remote catalog client (Ed25519/ETag/cache/fallback), external апп төрөл, SSO install gate |
| Registry + Storefront + Консол | #69–#75 | `cmd/appstore` сервис, `cmd/catalog-sign` CLI, appstore_db схем, publish/review урсгал, Next.js storefront (7 хэл, SSR), developer console (BFF + PKCE), nginx/compose/CI, OIDC root routing засвар, порт зохицуулалт |
| Auto-update бодлого | #76 | `AutoUpdate` sweep, §4.4-ийн hold дүрэм, auto-update toggle API+UI, held/pinned төлөв Settings→Apps дээр |

Үйлдвэрлэлд: appstore.gerege.mn (registry 8085 / storefront 3013),
developer.gerege.mn (3014), nexus каталогоо registry-гээс татаж байгаа.

### Амжилтын шалгуурын байдал (§8 хуучин)

1. ✅ Тенант нэг товчоор шинэчилдэг, event + түүх бичигддэг
2. ✅ appstore.gerege.mn нэвтрэлтгүй каталог, хариу бүр Ed25519 гарын үсэгтэй
3. ✅ Fallback гинж код дээр бүрэн (production дээр unplug-тест хийхийг §2-т үлдээв)
4. 🔶 Механизм бүрэн, install gate тесттэй — **бодит гуравдагч талын pilot л үлдсэн**
5. ✅ developer.gerege.mn дээрээс publisher бүртгэл, submission, review, publish ажиллана

---

## 1. Эцсийн re-review — кодын дүгнэлт (2026-08-10)

Шинээр орсон ~10k мөрийг бүрэн уншсан. Ерөнхий дүгнэлт: **чанар өндөр,
архитектурын гол шийдвэрүүд зөв.** Онцлохоор:

- **Contract нэг эх сурвалжтай:** `SignDocument` нэг функцийг endpoint, offline
  CLI, тест гурвуулаа хэрэглэдэг; snapshot-ыг гарын үсэгтэй байтаар нь DB-д
  хадгалж дахин encode хийлгүй өгдөг (миний §7-д анхааруулсан байт-нарийвчлалын
  эрсдэлийг яг зөвлөснөөр шийдсэн); ETag = sha256(document); нийтлэхийн өмнө
  `ValidateCatalog` ажиллуулж "invalid catalog зурахаас" татгалздаг.
- **JWT verifier зөв бичигдсэн:** alg токеноос уншдаггүй (RS256 хатуу), гарын
  үсгийг claims уншихаас өмнө шалгадаг, iss/aud/exp тулгадаг, JWKS 10 мин кэш +
  issuer унасан үед танил түлхүүрээр үргэлжлүүлдэг.
- **Консол BFF загвартай:** id_token зөвхөн httpOnly cookie-д, PKCE + state,
  browser-т токен огт очихгүй, refresh token зориуд байхгүй. Консолын OAuth2
  client-ийг `EnsureConsoleClient` тохиргооноос автоматаар бүртгэдэг.
- **Auto-update sweep §4.4-ийг үгчлэн хэрэгжүүлсэн:** permission нэмэгдэлт,
  scope нэмэгдэлт, launch_url-ийн **хост** солигдолт гурвыг барьж hold хийдэг;
  hold нь pin + `held` event; хуучин manifest олдохгүй бол таамаглахгүй алгасдаг
  (консерватив зөв тал); модулийн эрхийг manifest бус компиллогдсон кодоос авдаг
  тул модуль аппад hold хэрэггүй гэдгийг зөв ялгасан. Тест 224 мөр.
- **Revision зөв цэгт ахидаг:** submission биш, зөвхөн review-гийн шийдвэр
  (publish/reject/yank) каталогийн revision-ыг bump хийдэг — sync-ийн 304 зам
  ингэж хямд хэвээр үлдэнэ.
- **`nexus-oauth.conf` бодит алдаа засав:** `/oauth2/auth` өмнө нь frontend-ийн
  HTML-ээр хариулдаг байсан — гадны client-ийн нэвтрэлт огт боломжгүй байсныг
  root routing-оор шийдсэн (энэ нь external app SSO-гийн зайлшгүй нөхцөл байв).

### 1.1 Олдворууд (эрэмбээр)

**(а) Дунд — catalog_snapshots хязгааргүй өсөж болно.** `GET /api/v1/registry/catalog`
нэвтрэлтгүй бөгөөд `?platform=` бүрд snapshot мөр үүсгэж хадгалдаг
(`saveSnapshot`, түлхүүр нь channel+platform). Хорлонтой клиент semver-parseable
өөр өөр platform утга цувуулж илгээвэл DB-г мөрөөр дүүргэнэ. Санал: (1) зөвхөн
каталогт бодитоор байгаа `min_platform` цонхнуудад таарах platform-д snapshot
хадгалах, эсвэл (2) snapshot мөрийн тоог хязгаарлаж хуучныг нь janitor-оор
цэвэрлэх, (3) nginx-ийн appstore vhost-д `limit_req` зон нэмэх (одоо registry
талд ямар ч rate limit алга — nexus vhost-ийн загвар бэлэн).

**(б) Дунд/процесс — модуль аппын буруу min_platform бүх instance-ийг чимээгүй хоцроодог.**
Registry аль binary-д ямар модуль хувилбар байгааг мэдэхгүй. Publisher модуль
аппын шинэ хувилбарыг хэт сул `min_platform`-тай нийтэлбэл хуучин instance-үүд
каталог авч, `verifyCatalogVersions` бүхэл документыг гологдуулж, тэр instance
**update авахаа больдог — зөвхөн log-д warning**. Санал: (1) review дэлгэц дээр
модуль аппын хувилбар platform release-тэй тэнцүү эсэхийг анхааруулах, (2) nexus
админ UI-д "сүүлийн амжилттай sync-ийн цаг + сүүлийн алдаа" харуулах жижиг талбар
(одоо гар sync-ийн товч л хариу өгдөг).

**(в) Бага — олон replica-д sweep давхардана.** `AutoUpdate` sync хийсэн replica
бүр дээр ажиллана; advisory lock алга. Одоо нэг replica тул асуудалгүй, гэхдээ
replica нэмэхээс өмнө `pg_try_advisory_lock` нэг мөр нэмэх. (Давхардлын үр дагавар
нь давхар `upgraded` event л байх тул аюулгүй, гэхдээ бохир.)

**(г) Бага — malformed `?platform=` 500 буцаадаг.** `build`-ийн semver алдаа
клиентийн буруу боловч 500 болж гардаг; 400 болгох нь зөв бөгөөд alert-ийн
чимээг багасгана.

**(д) Бага — held шалтгаан админд бүрэн харагдахгүй тохиолдол.** `widenedGrant`
хуучин хувилбарын manifest `app_versions`-оос олдохгүй үед зөвхөн log warn хийж
алгасдаг — админ "яагаад шинэчлэгдэхгүй байгааг" UI-гаас мэдэхгүй. Ховор edge
(түүх бичигдэж эхлэхээс өмнөх суулгалт), гэхдээ (б)-ийн "sync төлөв" талбарт
хамт шийдэж болно.

**(е) Тэмдэглэл — id_token replay цонх.** Registry id_token-ыг bearer болгон
хүлээж авдаг тул хугацаа дуустал (1ц) дахин ашиглагдана; nonce/jti шалгалт алга.
BFF-cookie дотор хадгалагдаж байгаа тул бодит эрсдэл бага — mTLS/DPoP руу орох
шаардлагагүй, зөвхөн мэдэж байх.

Мөн хоёр өчүүхэн quirk: `handleKeys` каталог "warm" хийгээд үр дүнг хаядаг
(хоргүй); seed publisher автоматаар үүсдэг нь баримтжсан ба зөв.

---

## 2. Production дээр батлах сүүлчийн шалгалтууд (кодоор биш, ажиллагаагаар)

1. **Fallback unplug-тест:** registry-г түр унтрааж nexus-ийн boot + sync
   warning-ийг ажиглах; cache файл устгаад bundled file-д унахыг харах; буруу
   `APPSTORE_PUBLIC_KEY` тавьж хариу гологдохыг харах. (Код бүгдийг зөв хийхээр
   бичигдсэн, staging дээр нэг бүрчлэн үзсэн байх нь чухал.)
2. **Бүтэн publish давталт:** консолоос жинхэнэ шинэ хувилбар submit → review →
   publish → nexus дээр Update товч гарч ирэх → auto_update=true тенант дараагийн
   sync-ээр өөрөө ахих → scope нэмсэн хувилбараар held болохыг production дээр
   нэг удаа гараар батлах.
3. **Гуравдагч талын pilot:** бодит гадны нэг систем бүртгэж external аппын
   бүтэн урсгал (суулгалт → цэс → SSO → install gate) батлах. Энэ бол
   амжилтын шалгуур №4-ийн сүүлчийн алхам.
4. **Түлхүүрийн нөөцлөлт:** `APPSTORE_SIGNING_KEY`-ийн offline нөөц + хоёр дахь
   key_id-г урьдчилан үүсгэж rotation-ий дадлага нэг удаа хийх.

## 3. Үлдсэн жижиг ажлын жагсаалт

| # | Ажил | Эх сурвалж |
| --- | --- | --- |
| 1 | catalog_snapshots-ийн өсөлтийг хязгаарлах + registry-д nginx rate limit | Олдвор (а) |
| 2 | Nexus админд "сүүлийн sync төлөв" талбар; review дэлгэцэд модуль хувилбарын анхааруулга | Олдвор (б), (д) |
| 3 | Sweep-д advisory lock (replica нэмэхээс өмнө) | Олдвор (в) |
| 4 | Malformed platform → 400 | Олдвор (г) |
| 5 | Гуравдагч талын pilot апп | §2-3 |
| 6 | External `health_url` poll + Installed apps дээр төлөв | Хуучин үлдэгдэл |
| 7 | CI-д `--build-arg VERSION=$(git describe)` дамжуулж release train баримтжуулах | Хуучин үлдэгдэл |
| 8 | developer.gerege.mn ↔ eID порталын нэгтгэлийн бүтээгдэхүүний шийдвэр | Хуучин үлдэгдэл |

## 4. Лавлагаа — тогтсон архитектур

- **Wire contract:** `GET {registry}/api/v1/registry/catalog?platform=&channel=` →
  `{generated_at, key_id, apps(raw), signature=Ed25519(generated_at+'\n'+apps)}`,
  ETag=sha256, 304-conditional, 8MiB дээд. Хэрэгжилт: `appcatalog/source.go`
  (клиент) ↔ `appstore/catalog.go` (сервер) ↔ `contract_test.go` (хоёулааг
  нэг тестээр холбодог).
- **Гурван audience:** `/api/v1/registry/*` нэвтрэлтгүй; `/api/v1/dev/*` Gerege
  id_token; `/api/v1/admin/*` env-ээр нэрлэсэн reviewer.
- **Хостын порт зураглал:** nexus 3008/8082, appstore registry 8085, storefront
  3013, console 3014, appstore postgres 5439 (би хуучин 8083/3009/3010 санал
  болгосон нь Salus/app-js-тэй мөргөлдөхийг илрүүлж #73-аар зөв өөрчилсөн).
- **OIDC issuer:** nexus.gerege.mn хэвээр; root-ын `/oauth2/*`, `/.well-known/*`
  одоо API руу зөв чиглэдэг (`nexus-oauth.conf` snippet).
- Дэлгэрэнгүй ажиллагааны заавар: `docs/APPSTORE_OPERATIONS.md`.
