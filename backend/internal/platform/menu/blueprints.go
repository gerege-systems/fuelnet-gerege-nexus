package menu

// The menu blueprint: the entries every installed app contributes to the
// sidebar beyond the one working screen its module registers itself.
//
// Each label carries all seven platform languages. It used to carry two — an
// English default and a Mongolian override — so a browser asking for Chinese
// got a sidebar in Mongolian sitting above a page body in Chinese. Server-owned
// text is the half of the interface the client cannot translate for itself, so
// a language the server cannot answer in is a language the product does not
// really offer.
//
// `en` is the struct's EN field rather than a map key, because it is also the
// fallback: a locale missing from Labels resolves to it (see LocalizedLabel).

type futureMenu struct {
	ID, EN, Icon string
	// Labels maps ISO 639-1 code → label for every locale except en.
	Labels map[string]string
}

type blueprint struct {
	Slug              string
	Modules, Settings []futureMenu
}

var blueprints = map[string]blueprint{
	// organisation has no blueprint. Its two screens — departments and people —
	// are declared by the module itself, and the three that used to be here
	// (segments, duplicates, import) went with the contact register to
	// commerce-gerege-nexus, where that module declares them with their paths.
	// A blueprint is keyed by app id in the platform, so a distribution cannot
	// add to one: the entries a product ships are the product's to name.

	// products has one working screen and nothing else to stand on: the table
	// holds sku, name, price and active, so categories, price lists, units,
	// attributes and tax profiles would each be a menu entry over no data.

	// documents had a blueprint here until 2026-08-23, ten entries over two
	// groups. The app left for client-gerege-nexus and its module declares
	// them there; a blueprint for an id no module in this binary claims is
	// unreachable — menu.go only looks one up for a registered module.

	"io.gerege.nexus.sso_clients": {Slug: "sso-clients",
		Modules: []futureMenu{
			// API, OAuth and Webhook are product vocabulary, not prose. They stay
			// Latin even in the scripts that would otherwise transliterate them.
			{ID: "api-keys", EN: "API keys", Icon: "key-round", Labels: map[string]string{
				"mn": "API түлхүүр", "ar": "مفاتيح API", "zh": "API 密钥", "fr": "Clés API", "ru": "Ключи API", "es": "Claves API"}},
			// Access audit sits under Modules rather than Settings: it is
			// something you read, not something you configure.
			{ID: "audit", EN: "Access audit", Icon: "scroll-text", Labels: map[string]string{
				"mn": "Хандалтын аудит", "ar": "تدقيق الوصول", "zh": "访问审计", "fr": "Audit des accès", "ru": "Аудит доступа", "es": "Auditoría de acceso"}},
			// No Webhooks entry: Settings -> Integrations already registers
			// webhook listeners with a target URL and a signing secret, and a
			// second screen over the same records would only disagree with the
			// first one eventually.
		},
		Settings: []futureMenu{
			{ID: "scopes", EN: "OAuth scopes", Icon: "shield-check", Labels: map[string]string{
				"mn": "OAuth scope", "ar": "نطاقات OAuth", "zh": "OAuth 权限范围", "fr": "Portées OAuth", "ru": "Области OAuth", "es": "Ámbitos OAuth"}},
			{ID: "redirects", EN: "Redirect policies", Icon: "route", Labels: map[string]string{
				"mn": "Redirect бодлого", "ar": "سياسات إعادة التوجيه", "zh": "重定向策略", "fr": "Politiques de redirection", "ru": "Политики перенаправления", "es": "Políticas de redirección"}},
			{ID: "signing-keys", EN: "Signing keys", Icon: "key-square", Labels: map[string]string{
				"mn": "Гарын үсгийн түлхүүр", "ar": "مفاتيح التوقيع", "zh": "签名密钥", "fr": "Clés de signature", "ru": "Ключи подписи", "es": "Claves de firma"}},
		}},
}

// The two group headers every app's menu hangs under.
var (
	groupModules = map[string]string{
		"mn": "Модуль", "ar": "الوحدات", "zh": "模块", "fr": "Modules", "ru": "Модули", "es": "Módulos"}
	groupSettings = map[string]string{
		"mn": "Тохиргоо", "ar": "الإعدادات", "zh": "设置", "fr": "Paramètres", "ru": "Настройки", "es": "Configuración"}
)
