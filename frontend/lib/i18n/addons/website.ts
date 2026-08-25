/**
 * website — The public landing page: what the platform is, before anyone
 * signs in.
 */
export const website = {
  // The menu follows the order of the page, so a reader who picks the first
  // item lands near the top rather than at the bottom. Which of these appear
  // depends on which sections the deployment renders — see lib/landing.ts.
  "website.menu.architecture": { mn: "Архитектур", en: "Architecture" },
  "website.menu.applications": { mn: "Аппууд", en: "Applications" },
  "website.menu.platform": { mn: "Платформын суурь", en: "Platform" },
  "website.menu.trust": { mn: "Аюулгүй байдал", en: "Security" },
  "website.menu.technology": { mn: "Технологи", en: "Technology" },
  // The section is `#features`; what it argues is that identity is the floor.
  "website.menu.identity": { mn: "Нэвтрэлт", en: "Identity" },
  // Leaves the product for the published documentation. The other menu items
  // scroll within this page, so this one is marked as leaving.
  "website.menu.docs": { mn: "Баримт бичиг", en: "Documentation" },
  "website.menu.toggle": { mn: "Цэс", en: "Menu" },

  "website.action.sign_in": { mn: "Нэвтрэх", en: "Sign in" },
  "website.action.eid_sign_in": { mn: "eID-ээр нэвтрэх", en: "Sign in with eID" },
  "website.action.see_features": { mn: "Боломжийг үзэх", en: "See what it does" },

  // The hero headline is one sentence with a highlighted middle, so it is
  // stored in three parts rather than as markup inside a translation. The split
  // falls mid-phrase on purpose: what is highlighted is the claim, not a whole
  // clause, and each language chooses its own break.
  "website.view.hero_title_lead": { mn: "Байгууллагын бүх систем", en: "Every system an organisation runs" },
  "website.view.hero_title_highlight": { mn: "нэг цэгт", en: "meets in one" },
  "website.view.hero_title_tail": { mn: "уулзана", en: "place" },
  "website.view.hero_lede": {
    mn: "{brand} нь байгууллагын үйлчилгээ, үйл ажиллагаа, систем, өгөгдлийг нэгтгэх модульт платформ юм. Модулиуд нэг Go бинарид компиллогдож, тенант бүрт аль апп идэвхтэйг апп стор шийднэ.",
    en: "{brand} is a modular platform that brings an organisation's services, operations, systems and data together. Modules compile into a single Go binary, and an app store decides which of them each tenant runs.",
  },

  // Three numbers that hold still. Keep the app count in step with the base
  // catalogue; distributions can override it with BRAND_COPY.
  //
  // The figures are keys rather than literals so that a deployment counting
  // something else can say so: an identity provider shipping four modules does
  // not have nine business applications, and a number the deployment cannot
  // correct is a number its landing page states wrongly. Same value in every
  // language — they are digits — but they go through `t()` so BRAND_COPY can
  // reach them like any other line.
  "website.stat.apps_count": { mn: "1", en: "1" },
  "website.stat.apps": { mn: "үндсэн репод багтсан апп", en: "app included in the base repository" },
  "website.stat.languages_count": { mn: "7", en: "7" },
  "website.stat.languages": { mn: "хэл — монгол + НҮБ-ын 6", en: "languages: Mongolian plus the UN six" },
  "website.stat.binary_count": { mn: "1", en: "1" },
  "website.stat.binary": { mn: "суулгах бинари", en: "binary to deploy" },

  "website.view.features_eyebrow": { mn: "GEREGE IDENTITY LAYER", en: "GEREGE IDENTITY LAYER" },
  "website.view.features_title": {
    mn: "Нэвтрэлт бол тусдаа дэлгэц биш, платформын суурь",
    en: "Sign-in is not a screen, it is the floor the platform stands on",
  },
  "website.view.features_lede": {
    mn: "Gerege Platform-ийн батлагдсан урсгалыг {brand}-ийн tenant, role, audit болон SSO загварт нэгтгэлээ.",
    en: "The Gerege Platform's proven flow, folded into {brand}'s tenants, roles, audit trail and SSO model.",
  },

  "website.feature.instant_title": { mn: "Цахим үнэмлэхээр хормын дотор", en: "Digital ID in seconds" },
  "website.feature.instant_body": {
    mn: "Регистрийн дугаараар eID апп руу хүсэлт илгээх, компьютер дээр QR уншуулах, утсан дээр App2App холбоосоор нэвтэрнэ.",
    en: "Push a request to the eID app by registration number, scan a QR on the desktop, or hand off App2App on the phone.",
  },
  "website.feature.sso_title": { mn: "Нэг нэвтрэлт — олон систем", en: "One sign-in, many systems" },
  "website.feature.sso_body": {
    mn: "Платформын баталгаажсан session нь OAuth2/OIDC provider-тэй нэг trust boundary ашиглана. Холбогдсон апп бүр дахин нэвтрүүлэх шаардлагагүй.",
    en: "The platform session and the OAuth2/OIDC provider share one trust boundary, so no connected app asks again.",
  },
  "website.feature.passwordless_title": { mn: "Нууц үггүй, сервер талын хамгаалалт", en: "Passwordless, guarded server-side" },
  "website.feature.passwordless_body": {
    mn: "RP secret зөвхөн backend-д хадгалагдана. Browser-д identity credential ил гарахгүй, session token hash хэлбэрээр хадгалагдана.",
    en: "The RP secret never leaves the backend, no identity credential reaches the browser, and session tokens are stored hashed.",
  },
  "website.feature.channels_title": { mn: "Апп ба вэбийн нэг урсгал", en: "One flow across app and web" },
  "website.feature.channels_body": {
    mn: "Desktop cross-device, mobile same-device callback, push болон QR бүгд ижил start/poll contract-аар ажиллана.",
    en: "Desktop cross-device, mobile same-device callback, push and QR all run on the same start/poll contract.",
  },

  "website.view.trust_eyebrow": { mn: "ИДЭВХТЭЙ ХАМГААЛАЛТ", en: "ACTIVE PROTECTION" },
  "website.view.trust_title": {
    mn: "Танилтаас эрх хүртэл нэг баталгааны гинж",
    en: "One chain of proof, from identity to permission",
  },
  "website.view.trust_lede": {
    mn: "eID identity → серверийн session → tenant membership → RBAC → OIDC client. Алхам бүр сервер талд шалгагдана.",
    en: "eID identity → server session → tenant membership → RBAC → OIDC client. Every link is checked on the server.",
  },
  "website.trust.cookie": { mn: "httpOnly, SameSite session cookie", en: "httpOnly, SameSite session cookie" },
  "website.trust.rbac": { mn: "Tenant-аар тусгаарласан role ба permission", en: "Roles and permissions isolated per tenant" },
  "website.trust.allowlist": { mn: "RP callback origin allowlist", en: "RP callback origin allowlist" },
  "website.trust.audit": { mn: "Login ба access audit event", en: "Login and access audit events" },

  // Key kept as-is: it is internal, and renaming it would touch every caller
  // for no user-visible gain. The value is what reaches the screen.
  "website.tech.erp_body": { mn: "Модульт аппууд, tenant тусгаарлалт, RBAC", en: "Modular apps, tenant isolation, RBAC" },
  "website.tech.eid_body": { mn: "Push, QR, App2App, баталгаажсан identity", en: "Push, QR, App2App, verified identity" },
  "website.tech.sso_body": { mn: "Холбогдсон аппууд, нэг session", en: "Connected applications, one session" },

  // ─── The platform itself ───────────────────────────────────────────────────
  // Everything above this line argues for the sign-in. Everything below argues
  // for the platform behind it, and was moved here from the documentation site,
  // which used to make the same case in a second place that could drift.

  "website.arch.eyebrow": { mn: "ЯАГААД ЭНЭ АРХИТЕКТУР", en: "WHY THIS ARCHITECTURE" },
  "website.arch.title": {
    mn: "Модульт монолит — микросервисийн уян хатан, монолитын хурд",
    en: "A modular monolith: the flexibility of microservices at the speed of one process",
  },
  "website.arch.lede": {
    mn: "Модуль бүр өөрийн домэйн, миграц, эрхтэй. Гэвч бүгд нэг процесс дотор ажилладаг тул модуль хоорондын дуудлага сүлжээгээр явахгүй — Go-ийн функцийн дуудлага л болно.",
    en: "Every module owns its domain, its migrations and its permissions. They all run in one process, so a call between two of them is a function call, not a network hop.",
  },
  "website.arch.modules_title": { mn: "Компиллогдсон Go модулиуд", en: "Compiled-in Go modules" },
  "website.arch.modules_body": {
    mn: "Модуль бүр нэг Go гэрээг хэрэгжүүлж нэг бинарид компиллогдоно. Маршрут, цэс, эрх, миграц бүгд модулийн өөрийнх.",
    en: "Each module implements one Go contract and compiles into a single binary. Routes, menus, permissions and migrations all belong to the module itself.",
  },
  "website.arch.store_title": { mn: "Тенант бүрийн апп стор", en: "An app store for each tenant" },
  "website.arch.store_body": {
    mn: "Аль байгууллагад аль апп идэвхтэйг өгөгдлийн сан шийднэ. Суулгаагүй апп руу хандвал хориглоно — код нь байгаа ч хаалга нь хаалттай.",
    en: "The database decides which apps an organisation runs. An app that is not installed refuses the request: the code is there, the door is not open.",
  },
  "website.arch.dag_title": { mn: "Хамаарал шийдвэрлэгч", en: "Dependency resolution" },
  "website.arch.dag_body": {
    mn: "Рекурсив шийдвэрлэлт, мөчлөг илрүүлэлт, хувилбарын шалгалт. Апп суулгахад түүний хамаарал бүр тохирох хувилбартайгаа хамт орно.",
    en: "Recursive resolution with cycle detection and version checks, so installing an app brings every dependency it needs at a version that fits.",
  },
  "website.arch.catalog_title": { mn: "Нэг эх сурвалжтай каталог", en: "One catalogue, one source" },
  "website.arch.catalog_body": {
    mn: "Апп сторын каталог нь цорын ганц эх сурвалж бөгөөд гарын үсэгтэйгээр татагдана. Систем асах бүрд апп жагсаалт түүнээс шинэчлэгдэнэ.",
    en: "The app catalogue is the single source of truth and is fetched signed. The list of apps is refreshed from it every time the system starts.",
  },

  "website.apps.eyebrow": { mn: "ҮНДСЭН DISTRIBUTION", en: "BASE DISTRIBUTION" },
  "website.apps.title": { mn: "Нэг built-in апп, нэмэгдэх боломжтой платформ", en: "One built-in app, an extensible platform" },
  "website.apps.lede": {
    mn: "Энэ репогийн каталогт SSO клиент удирдах апп л байна. Бизнес аппуудыг тусдаа distribution репо компиллож, өөрийн каталогоор нэмдэг.",
    en: "This repository's catalogue contains only SSO client management. Product distributions compile in business apps and publish their own catalogue.",
  },
  "website.apps.sso_clients": { mn: "SSO клиентүүд — OAuth2/OIDC клиент бүртгэл", en: "SSO Clients — OAuth2/OIDC client registration" },

  "website.depth.eyebrow": { mn: "ПЛАТФОРМЫН СУУРЬ", en: "UNDER THE PLATFORM" },
  "website.depth.title": {
    mn: "Бүтээгдэхүүн болгонд дахин бичих шаардлагагүй зүйлс",
    en: "The parts you would otherwise rewrite for every product",
  },
  "website.depth.lede": {
    mn: "Эдгээрийн аль нь ч нэмэлт биш. Платформын цөмд байрлах тул апп бүр тэднийг өвлөж авна.",
    en: "None of this is an add-on. It sits in the core, so every application inherits it.",
  },
  "website.depth.resilience_title": { mn: "Хүсэлтийн хамгаалалт", en: "Request protection" },
  "website.depth.resilience_body": {
    mn: "Хэт олон зэрэг хүсэлтийг 503-аар хязгаарлах load shedder, гадаад дуудлагын timeout, зориулалтын retry бодлого платформд хэрэгжсэн.",
    en: "The platform implements concurrency load shedding, outbound timeouts and operation-specific retry policies.",
  },
  "website.depth.gov_title": { mn: "Төрийн систем рүү шууд", en: "Straight into state systems" },
  "website.depth.gov_body": {
    mn: "Иргэн, хуулийн этгээдийн лавлагаа төрийн мэдээлэл солилцооны системээс; тоон гарын үсэг, нэг удаагийн код, банкны суваг, царай танилт eID ба ДАН-аар.",
    en: "Citizen and legal-entity lookups through the state exchange, with signatures, one-time codes, bank channels and biometrics via eID and DAN.",
  },
  "website.depth.security_title": { mn: "Анхдагчаараа хатуу", en: "Secure by default" },
  "website.depth.security_body": {
    mn: "Session токен зөвхөн хэшээрээ хадгалагдана, нууц үг bcrypt-ээр, OAuth2 танилт тогтмол хугацаанд шалгагдана, асуулга бүр байгууллагаараа хязгаарлагдана.",
    en: "Session tokens are stored only as hashes, passwords use bcrypt, OAuth2 clients are compared in constant time, and every query is bounded by organisation.",
  },
  "website.depth.ai_title": { mn: "Өөрийн өгөгдөлд холбогдсон AI", en: "AI wired to your own data" },
  "website.depth.ai_body": {
    mn: "Gemini түлхүүр өгвөл чат, яриа таних, унших, орчуулга ажиллана. Бизнес өгөгдөлд хандах хэрэгслийг тухайн distribution-ийн апп өөрөө бүртгэнэ.",
    en: "With a Gemini key, chat, speech, text-to-speech and translation are available. Product apps register the tools that expose their own business data.",
  },
  "website.depth.i18n_title": { mn: "Долоон хэл, цоорхойгүй", en: "Seven languages, no gaps" },
  "website.depth.i18n_body": {
    mn: "Монгол, англи эх мөрүүд дээр НҮБ-ын бусад таван хэлний overlay нэмэгдэнэ. Орчуулга дутвал англи руу fallback хийж, CI үлдсэн цоорхойг тайлагнана.",
    en: "Mongolian and English source strings are joined by five UN-language overlays. Missing translations fall back to English and CI reports the remaining gaps.",
  },
  "website.depth.observability_title": { mn: "Ажиглалт ба аудит", en: "Observability and audit" },
  "website.depth.observability_body": {
    mn: "Хэмжүүр, амьд ба бэлэн байдлын шалгалт, хэн юуг хэзээ өөрчилснийг бүртгэсэн ул мөр — эхний өдрөөс.",
    en: "Metrics, liveness and readiness probes, and a trail of who changed what and when — from the first day.",
  },

  "website.message.footer_note": {
    mn: "eID-д суурилсан · Нээлттэй стандарт · Secure by design",
    en: "Built on eID · Open standards · Secure by design",
  },

  // Shown only by a deployment running under its own name — see SiteFooter.
  //
  // Deliberately not `{brand}`: this sentence names the platform underneath,
  // and interpolating the deployment's own name would have Gerege Salus
  // announcing that it is powered by Gerege Salus.
  "website.message.powered_by": {
    mn: "Gerege Nexus дээр суурилсан",
    en: "Powered by Gerege Nexus",
  },
} as const;
