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

	// One app, two things kept in it: records that move through an approval
	// chain, and PDFs that arrive already written and need a signature on them.
	// The PDF half was its own blueprint under slug `esign` until the two apps
	// became one; its entries are here unchanged apart from where they point.
	//
	// The order is the reading order of the work: what is waiting, what is
	// signed, what a document is started from, what a batch of them is run as.
	"io.gerege.nexus.documents": {Slug: "documents",
		Modules: []futureMenu{
			{ID: "approvals", EN: "Approval queue", Icon: "list-checks", Labels: map[string]string{
				"mn": "Батлах дараалал", "ar": "قائمة الموافقات", "zh": "审批队列", "fr": "File d'approbation", "ru": "Очередь согласования", "es": "Cola de aprobación"}},
			{ID: "pdf", EN: "PDF signing", Icon: "pen-tool", Labels: map[string]string{
				"mn": "PDF гарын үсэг", "ar": "توقيع PDF", "zh": "PDF 签名", "fr": "Signature PDF", "ru": "Подпись PDF", "es": "Firma de PDF"}},
			{ID: "logs", EN: "Signature logs", Icon: "scroll-text", Labels: map[string]string{
				"mn": "Гарын үсгийн лог", "ar": "سجلات التوقيع", "zh": "签名日志", "fr": "Journaux de signature", "ru": "Журналы подписей", "es": "Registros de firma"}},
			{ID: "templates", EN: "Document templates", Icon: "files", Labels: map[string]string{
				"mn": "Баримтын загвар", "ar": "قوالب المستندات", "zh": "文档模板", "fr": "Modèles de documents", "ru": "Шаблоны документов", "es": "Plantillas de documentos"}},
			{ID: "batch", EN: "Batch signing", Icon: "layers", Labels: map[string]string{
				"mn": "Багц баталгаажуулалт", "ar": "التوقيع المجمع", "zh": "批量签名", "fr": "Signature par lot", "ru": "Пакетное подписание", "es": "Firma por lotes"}},
		},
		Settings: []futureMenu{
			{ID: "workflows", EN: "Document workflows", Icon: "workflow", Labels: map[string]string{
				"mn": "Баримтын урсгал", "ar": "سير عمل المستندات", "zh": "文档流程", "fr": "Flux de documents", "ru": "Процессы документов", "es": "Flujos de documentos"}},
			// Two screens said "Signature policies" when these were two apps,
			// and under one menu that is a coin toss rather than a choice. This
			// one is which channel a document *type* may be signed through;
			// `rails` below is which of the two PDF machines is switched on.
			{ID: "signatures", EN: "Signature policies", Icon: "pen-tool", Labels: map[string]string{
				"mn": "Гарын үсгийн бодлого", "ar": "سياسات التوقيع", "zh": "签名策略", "fr": "Politiques de signature", "ru": "Политики подписи", "es": "Políticas de firma"}},
			{ID: "rails", EN: "PDF signing rails", Icon: "shield-check", Labels: map[string]string{
				"mn": "PDF гарын үсгийн суваг", "ar": "مسارات توقيع PDF", "zh": "PDF 签署通道", "fr": "Canaux de signature PDF", "ru": "Каналы подписания PDF", "es": "Canales de firma de PDF"}},
			{ID: "placement", EN: "Stamp placement", Icon: "move", Labels: map[string]string{
				"mn": "Тамганы байрлал", "ar": "موضع الختم", "zh": "印章位置", "fr": "Position du cachet", "ru": "Расположение печати", "es": "Posición del sello"}},
			{ID: "hsm", EN: "HSM connection", Icon: "server-cog", Labels: map[string]string{
				"mn": "HSM холболт", "ar": "اتصال HSM", "zh": "HSM 连接", "fr": "Connexion HSM", "ru": "Подключение HSM", "es": "Conexión HSM"}},
			{ID: "retention", EN: "Retention rules", Icon: "archive", Labels: map[string]string{
				"mn": "Хадгалалтын дүрэм", "ar": "قواعد الاحتفاظ", "zh": "保留规则", "fr": "Règles de conservation", "ru": "Правила хранения", "es": "Reglas de retención"}},
		}},

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
