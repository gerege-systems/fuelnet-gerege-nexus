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
	"io.example.contacts": {Slug: "contacts",
		Modules: []futureMenu{
			{ID: "segments", EN: "Segments", Icon: "users", Labels: map[string]string{
				"mn": "Сегментүүд", "ar": "الشرائح", "zh": "客户分组", "fr": "Segments", "ru": "Сегменты", "es": "Segmentos"}},
			{ID: "activity", EN: "Contact activity", Icon: "activity", Labels: map[string]string{
				"mn": "Харилцааны түүх", "ar": "سجل التواصل", "zh": "联系人动态", "fr": "Activité des contacts", "ru": "История контактов", "es": "Actividad de contactos"}},
		},
		Settings: []futureMenu{
			{ID: "fields", EN: "Custom fields", Icon: "settings", Labels: map[string]string{
				"mn": "Нэмэлт талбар", "ar": "حقول مخصصة", "zh": "自定义字段", "fr": "Champs personnalisés", "ru": "Дополнительные поля", "es": "Campos personalizados"}},
			{ID: "duplicates", EN: "Duplicate rules", Icon: "copy", Labels: map[string]string{
				"mn": "Давхардлын дүрэм", "ar": "قواعد التكرار", "zh": "重复规则", "fr": "Règles de doublons", "ru": "Правила дубликатов", "es": "Reglas de duplicados"}},
			{ID: "import", EN: "Import mapping", Icon: "upload", Labels: map[string]string{
				"mn": "Импортын зураглал", "ar": "تعيين الاستيراد", "zh": "导入映射", "fr": "Mappage d'import", "ru": "Сопоставление импорта", "es": "Asignación de importación"}},
		}},

	"io.example.products": {Slug: "products",
		Modules: []futureMenu{
			{ID: "categories", EN: "Categories", Icon: "tags", Labels: map[string]string{
				"mn": "Ангилал", "ar": "الفئات", "zh": "分类", "fr": "Catégories", "ru": "Категории", "es": "Categorías"}},
			{ID: "price-lists", EN: "Price lists", Icon: "badge-dollar-sign", Labels: map[string]string{
				"mn": "Үнийн жагсаалт", "ar": "قوائم الأسعار", "zh": "价目表", "fr": "Listes de prix", "ru": "Прайс-листы", "es": "Listas de precios"}},
		},
		Settings: []futureMenu{
			{ID: "units", EN: "Units of measure", Icon: "ruler", Labels: map[string]string{
				"mn": "Хэмжих нэгж", "ar": "وحدات القياس", "zh": "计量单位", "fr": "Unités de mesure", "ru": "Единицы измерения", "es": "Unidades de medida"}},
			{ID: "attributes", EN: "Product attributes", Icon: "sliders", Labels: map[string]string{
				"mn": "Барааны шинж", "ar": "خصائص المنتج", "zh": "产品属性", "fr": "Attributs de produit", "ru": "Атрибуты товара", "es": "Atributos de producto"}},
			{ID: "taxes", EN: "Tax profiles", Icon: "percent", Labels: map[string]string{
				"mn": "Татварын тохиргоо", "ar": "ملفات الضرائب", "zh": "税务配置", "fr": "Profils fiscaux", "ru": "Налоговые профили", "es": "Perfiles fiscales"}},
		}},

	"io.example.inventory": {Slug: "inventory",
		Modules: []futureMenu{
			{ID: "transfers", EN: "Stock transfers", Icon: "arrow-right-left", Labels: map[string]string{
				"mn": "Бараа шилжүүлэг", "ar": "تحويلات المخزون", "zh": "库存调拨", "fr": "Transferts de stock", "ru": "Перемещения запасов", "es": "Transferencias de stock"}},
			{ID: "replenishment", EN: "Replenishment", Icon: "refresh-cw", Labels: map[string]string{
				"mn": "Нөхөн дүүргэлт", "ar": "إعادة التزويد", "zh": "补货", "fr": "Réapprovisionnement", "ru": "Пополнение", "es": "Reabastecimiento"}},
		},
		Settings: []futureMenu{
			{ID: "warehouses", EN: "Warehouse settings", Icon: "warehouse", Labels: map[string]string{
				"mn": "Агуулахын тохиргоо", "ar": "إعدادات المستودع", "zh": "仓库设置", "fr": "Paramètres d'entrepôt", "ru": "Настройки склада", "es": "Ajustes de almacén"}},
			{ID: "routes", EN: "Logistics routes", Icon: "route", Labels: map[string]string{
				"mn": "Ложистикийн маршрут", "ar": "مسارات اللوجستيات", "zh": "物流路线", "fr": "Itinéraires logistiques", "ru": "Логистические маршруты", "es": "Rutas logísticas"}},
			{ID: "valuation", EN: "Valuation methods", Icon: "calculator", Labels: map[string]string{
				"mn": "Өртөг тооцоолол", "ar": "طرق التقييم", "zh": "计价方法", "fr": "Méthodes de valorisation", "ru": "Методы оценки", "es": "Métodos de valoración"}},
		}},

	"io.example.billing": {Slug: "billing",
		Modules: []futureMenu{
			{ID: "payments", EN: "Payments", Icon: "wallet", Labels: map[string]string{
				"mn": "Төлбөрүүд", "ar": "المدفوعات", "zh": "付款", "fr": "Paiements", "ru": "Платежи", "es": "Pagos"}},
			{ID: "reports", EN: "Revenue reports", Icon: "chart-column", Labels: map[string]string{
				"mn": "Орлогын тайлан", "ar": "تقارير الإيرادات", "zh": "收入报表", "fr": "Rapports de revenus", "ru": "Отчёты о доходах", "es": "Informes de ingresos"}},
		},
		Settings: []futureMenu{
			{ID: "numbering", EN: "Invoice numbering", Icon: "list-ordered", Labels: map[string]string{
				"mn": "Нэхэмжлэх дугаарлалт", "ar": "ترقيم الفواتير", "zh": "发票编号", "fr": "Numérotation des factures", "ru": "Нумерация счетов", "es": "Numeración de facturas"}},
			// e-Barimt is the Mongolian tax authority's receipt system: a product
			// name, so it survives translation untouched in every language.
			{ID: "ebarimt", EN: "e-Barimt connection", Icon: "receipt", Labels: map[string]string{
				"mn": "e-Barimt холболт", "ar": "اتصال e-Barimt", "zh": "e-Barimt 连接", "fr": "Connexion e-Barimt", "ru": "Подключение e-Barimt", "es": "Conexión e-Barimt"}},
			{ID: "payment-methods", EN: "Payment methods", Icon: "credit-card", Labels: map[string]string{
				"mn": "Төлбөрийн хэлбэр", "ar": "طرق الدفع", "zh": "支付方式", "fr": "Moyens de paiement", "ru": "Способы оплаты", "es": "Métodos de pago"}},
		}},

	"io.example.documents": {Slug: "documents",
		Modules: []futureMenu{
			{ID: "approvals", EN: "Approval queue", Icon: "list-checks", Labels: map[string]string{
				"mn": "Батлах дараалал", "ar": "قائمة الموافقات", "zh": "审批队列", "fr": "File d'approbation", "ru": "Очередь согласования", "es": "Cola de aprobación"}},
			{ID: "templates", EN: "Document templates", Icon: "files", Labels: map[string]string{
				"mn": "Баримтын загвар", "ar": "قوالب المستندات", "zh": "文档模板", "fr": "Modèles de documents", "ru": "Шаблоны документов", "es": "Plantillas de documentos"}},
		},
		Settings: []futureMenu{
			{ID: "workflows", EN: "Document workflows", Icon: "workflow", Labels: map[string]string{
				"mn": "Баримтын урсгал", "ar": "سير عمل المستندات", "zh": "文档流程", "fr": "Flux de documents", "ru": "Процессы документов", "es": "Flujos de documentos"}},
			{ID: "signatures", EN: "Signature policies", Icon: "pen-tool", Labels: map[string]string{
				"mn": "Гарын үсгийн бодлого", "ar": "سياسات التوقيع", "zh": "签名策略", "fr": "Politiques de signature", "ru": "Политики подписи", "es": "Políticas de firma"}},
			{ID: "retention", EN: "Retention rules", Icon: "archive", Labels: map[string]string{
				"mn": "Хадгалалтын дүрэм", "ar": "قواعد الاحتفاظ", "zh": "保留规则", "fr": "Règles de conservation", "ru": "Правила хранения", "es": "Reglas de retención"}},
		}},

	"io.example.esign": {Slug: "esign",
		Modules: []futureMenu{
			{ID: "logs", EN: "Signature logs", Icon: "scroll-text", Labels: map[string]string{
				"mn": "Гарын үсгийн лог", "ar": "سجلات التوقيع", "zh": "签名日志", "fr": "Journaux de signature", "ru": "Журналы подписей", "es": "Registros de firma"}},
			{ID: "batch", EN: "Batch signing", Icon: "layers", Labels: map[string]string{
				"mn": "Багц баталгаажуулалт", "ar": "التوقيع المجمع", "zh": "批量签名", "fr": "Signature par lot", "ru": "Пакетное подписание", "es": "Firma por lotes"}},
		},
		Settings: []futureMenu{
			{ID: "placement", EN: "Stamp placement", Icon: "move", Labels: map[string]string{
				"mn": "Тамганы байрлал", "ar": "موضع الختم", "zh": "印章位置", "fr": "Position du cachet", "ru": "Расположение печати", "es": "Posición del sello"}},
			{ID: "hsm", EN: "HSM connection", Icon: "server-cog", Labels: map[string]string{
				"mn": "HSM холболт", "ar": "اتصال HSM", "zh": "HSM 连接", "fr": "Connexion HSM", "ru": "Подключение HSM", "es": "Conexión HSM"}},
			{ID: "policies", EN: "Signing policies", Icon: "shield-check", Labels: map[string]string{
				"mn": "Гарын үсгийн бодлого", "ar": "سياسات التوقيع", "zh": "签署策略", "fr": "Politiques de signature", "ru": "Политики подписания", "es": "Políticas de firma"}},
		}},

	"io.example.developer_portal": {Slug: "developer",
		Modules: []futureMenu{
			// API, OAuth and Webhook are product vocabulary, not prose. They stay
			// Latin even in the scripts that would otherwise transliterate them.
			{ID: "api-keys", EN: "API keys", Icon: "key-round", Labels: map[string]string{
				"mn": "API түлхүүр", "ar": "مفاتيح API", "zh": "API 密钥", "fr": "Clés API", "ru": "Ключи API", "es": "Claves API"}},
			{ID: "webhooks", EN: "Webhooks", Icon: "webhook", Labels: map[string]string{
				"mn": "Webhook", "ar": "Webhooks", "zh": "Webhook", "fr": "Webhooks", "ru": "Webhooks", "es": "Webhooks"}},
		},
		Settings: []futureMenu{
			{ID: "scopes", EN: "OAuth scopes", Icon: "shield-check", Labels: map[string]string{
				"mn": "OAuth scope", "ar": "نطاقات OAuth", "zh": "OAuth 权限范围", "fr": "Portées OAuth", "ru": "Области OAuth", "es": "Ámbitos OAuth"}},
			{ID: "redirects", EN: "Redirect policies", Icon: "route", Labels: map[string]string{
				"mn": "Redirect бодлого", "ar": "سياسات إعادة التوجيه", "zh": "重定向策略", "fr": "Politiques de redirection", "ru": "Политики перенаправления", "es": "Políticas de redirección"}},
			{ID: "audit", EN: "Access audit", Icon: "scroll-text", Labels: map[string]string{
				"mn": "Хандалтын аудит", "ar": "تدقيق الوصول", "zh": "访问审计", "fr": "Audit des accès", "ru": "Аудит доступа", "es": "Auditoría de acceso"}},
		}},

	"io.example.gov_services": {Slug: "gov-services",
		Modules: []futureMenu{
			{ID: "requests", EN: "Service requests", Icon: "inbox", Labels: map[string]string{
				"mn": "Үйлчилгээний хүсэлт", "ar": "طلبات الخدمة", "zh": "服务申请", "fr": "Demandes de service", "ru": "Заявки на услуги", "es": "Solicitudes de servicio"}},
			{ID: "appointments", EN: "Appointments", Icon: "calendar-clock", Labels: map[string]string{
				"mn": "Цаг захиалга", "ar": "المواعيد", "zh": "预约", "fr": "Rendez-vous", "ru": "Записи на приём", "es": "Citas"}},
		},
		Settings: []futureMenu{
			{ID: "catalog", EN: "Service catalog", Icon: "landmark", Labels: map[string]string{
				"mn": "Үйлчилгээний каталог", "ar": "كتالوج الخدمات", "zh": "服务目录", "fr": "Catalogue de services", "ru": "Каталог услуг", "es": "Catálogo de servicios"}},
			{ID: "workflow", EN: "Decision workflow", Icon: "workflow", Labels: map[string]string{
				"mn": "Шийдвэрлэх урсгал", "ar": "سير اتخاذ القرار", "zh": "决策流程", "fr": "Flux de décision", "ru": "Процесс принятия решений", "es": "Flujo de decisión"}},
			{ID: "sla", EN: "Service levels", Icon: "timer", Labels: map[string]string{
				"mn": "Үйлчилгээний түвшин", "ar": "مستويات الخدمة", "zh": "服务级别", "fr": "Niveaux de service", "ru": "Уровни обслуживания", "es": "Niveles de servicio"}},
		}},
}

// The two group headers every app's menu hangs under.
var (
	groupModules = map[string]string{
		"mn": "Модуль", "ar": "الوحدات", "zh": "模块", "fr": "Modules", "ru": "Модули", "es": "Módulos"}
	groupSettings = map[string]string{
		"mn": "Тохиргоо", "ar": "الإعدادات", "zh": "设置", "fr": "Paramètres", "ru": "Настройки", "es": "Configuración"}
)
