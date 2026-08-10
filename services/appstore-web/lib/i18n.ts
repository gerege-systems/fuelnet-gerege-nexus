/**
 * The storefront's own words, in the seven languages the platform supports.
 *
 * App names, descriptions and categories are not here: those belong to whoever
 * published the app, and the registry resolves them per locale before
 * answering. This file is only the shop window — headings, labels, the sentence
 * that explains how to install something.
 *
 * Mongolian first because it is the source language, and the fallback is
 * English rather than the key: an untranslated storefront should read as an
 * English one, not as plumbing.
 */

export const LOCALES = [
  { code: "mn", label: "Монгол" },
  { code: "ar", label: "العربية", rtl: true },
  { code: "zh", label: "中文" },
  { code: "en", label: "English" },
  { code: "fr", label: "Français" },
  { code: "ru", label: "Русский" },
  { code: "es", label: "Español" },
] as const;

export type Locale = (typeof LOCALES)[number]["code"];

const DICTIONARY: Record<string, Record<Locale, string>> = {
  "store.title": {
    mn: "Gerege Апп Дэлгүүр",
    en: "Gerege App Store",
    ar: "متجر تطبيقات Gerege",
    zh: "Gerege 应用商店",
    fr: "Boutique d'applications Gerege",
    ru: "Магазин приложений Gerege",
    es: "Tienda de aplicaciones Gerege",
  },
  "store.tagline": {
    mn: "Gerege Nexus платформ дээр суулгах боломжтой аппликейшнууд.",
    en: "Applications you can install on a Gerege Nexus platform.",
    ar: "تطبيقات يمكنك تثبيتها على منصة Gerege Nexus.",
    zh: "可安装到 Gerege Nexus 平台的应用。",
    fr: "Applications installables sur une plateforme Gerege Nexus.",
    ru: "Приложения, которые можно установить на платформу Gerege Nexus.",
    es: "Aplicaciones que puede instalar en una plataforma Gerege Nexus.",
  },
  "store.search": {
    mn: "Апп хайх...",
    en: "Search apps...",
    ar: "البحث عن تطبيقات...",
    zh: "搜索应用...",
    fr: "Rechercher des applications...",
    ru: "Поиск приложений...",
    es: "Buscar aplicaciones...",
  },
  "store.all": { mn: "Бүгд", en: "All", ar: "الكل", zh: "全部", fr: "Tout", ru: "Все", es: "Todas" },
  "store.empty": {
    mn: "Одоогоор нийтлэгдсэн апп алга.",
    en: "Nothing has been published yet.",
    ar: "لم يتم نشر أي شيء بعد.",
    zh: "尚未发布任何应用。",
    fr: "Rien n'a encore été publié.",
    ru: "Пока ничего не опубликовано.",
    es: "Todavía no se ha publicado nada.",
  },
  "store.unavailable": {
    mn: "Каталогийг ачаалж чадсангүй. Түр хүлээгээд дахин оролдоно уу.",
    en: "The catalogue could not be loaded. Please try again shortly.",
    ar: "تعذّر تحميل الكتالوج. حاول مرة أخرى بعد قليل.",
    zh: "无法加载目录，请稍后重试。",
    fr: "Le catalogue n'a pas pu être chargé. Réessayez dans un instant.",
    ru: "Не удалось загрузить каталог. Повторите попытку позже.",
    es: "No se pudo cargar el catálogo. Inténtelo de nuevo en breve.",
  },
  "store.publisher": {
    mn: "Нийтлэгч",
    en: "Publisher",
    ar: "الناشر",
    zh: "发布者",
    fr: "Éditeur",
    ru: "Издатель",
    es: "Editor",
  },
  "store.version": {
    mn: "Хувилбар",
    en: "Version",
    ar: "الإصدار",
    zh: "版本",
    fr: "Version",
    ru: "Версия",
    es: "Versión",
  },
  "store.type.module": {
    mn: "Платформын модуль",
    en: "Platform module",
    ar: "وحدة المنصة",
    zh: "平台模块",
    fr: "Module de la plateforme",
    ru: "Модуль платформы",
    es: "Módulo de la plataforma",
  },
  "store.type.external": {
    mn: "Гадаад платформ",
    en: "External platform",
    ar: "منصة خارجية",
    zh: "外部平台",
    fr: "Plateforme externe",
    ru: "Внешняя платформа",
    es: "Plataforma externa",
  },
  "store.permissions": {
    mn: "Шаардах эрхүүд",
    en: "Permissions requested",
    ar: "الأذونات المطلوبة",
    zh: "所需权限",
    fr: "Autorisations demandées",
    ru: "Запрашиваемые права",
    es: "Permisos solicitados",
  },
  "store.dependencies": {
    mn: "Хамаарал",
    en: "Depends on",
    ar: "يعتمد على",
    zh: "依赖于",
    fr: "Dépend de",
    ru: "Зависит от",
    es: "Depende de",
  },
  "store.history": {
    mn: "Хувилбарын түүх",
    en: "Version history",
    ar: "سجل الإصدارات",
    zh: "版本历史",
    fr: "Historique des versions",
    ru: "История версий",
    es: "Historial de versiones",
  },
  "store.requires_platform": {
    mn: "Шаардах платформ",
    en: "Requires platform",
    ar: "يتطلب منصة",
    zh: "所需平台",
    fr: "Plateforme requise",
    ru: "Требуется платформа",
    es: "Plataforma requerida",
  },
  "store.install.title": {
    mn: "Хэрхэн суулгах вэ?",
    en: "How to install",
    ar: "كيفية التثبيت",
    zh: "如何安装",
    fr: "Comment installer",
    ru: "Как установить",
    es: "Cómo instalar",
  },
  // Deliberately not an "Install" button. Installing happens inside the
  // organisation's own Nexus, by an administrator who is signed in there — a
  // button here could only ever pretend to do that.
  "store.install.body": {
    mn: "Өөрийн байгууллагын Gerege Nexus руу нэвтэрч, Апп Дэлгүүр хэсгээс энэ аппыг сонгоод «Суулгах» дарна. Суулгах эрх нь байгууллагын админд байна.",
    en: "Sign in to your organisation's Gerege Nexus, open the App Store and choose Install on this app. Installing is a tenant administrator's action.",
    ar: "سجّل الدخول إلى Gerege Nexus الخاص بمؤسستك، وافتح متجر التطبيقات واختر «تثبيت». التثبيت من صلاحيات مسؤول المؤسسة.",
    zh: "登录贵组织的 Gerege Nexus，打开应用商店并点击「安装」。安装需由组织管理员执行。",
    fr: "Connectez-vous au Gerege Nexus de votre organisation, ouvrez la boutique et choisissez Installer. L'installation est une action de l'administrateur.",
    ru: "Войдите в Gerege Nexus вашей организации, откройте магазин приложений и нажмите «Установить». Установку выполняет администратор организации.",
    es: "Inicie sesión en el Gerege Nexus de su organización, abra la tienda y pulse Instalar. La instalación la realiza un administrador.",
  },
  "store.external.note": {
    mn: "Энэ бол гуравдагч талын платформ. Суулгасны дараа хэрэглэгчид Gerege SSO-гоор нэвтэрч ажиллана; суулгаагүй байгууллагын хэрэглэгч нэвтрэх боломжгүй.",
    en: "This is a third-party platform. Once installed, your people sign in to it with Gerege SSO; users of organisations that have not installed it cannot sign in at all.",
    ar: "هذه منصة تابعة لجهة خارجية. بعد التثبيت يسجّل موظفوك الدخول عبر Gerege SSO؛ ولا يمكن لمستخدمي المؤسسات غير المثبِّتة الدخول.",
    zh: "这是第三方平台。安装后，贵方人员可通过 Gerege SSO 登录；未安装的组织用户无法登录。",
    fr: "Il s'agit d'une plateforme tierce. Une fois installée, vos collaborateurs s'y connectent via Gerege SSO ; les utilisateurs des organisations qui ne l'ont pas installée ne peuvent pas se connecter.",
    ru: "Это сторонняя платформа. После установки сотрудники входят в неё через Gerege SSO; пользователи организаций без установки войти не смогут.",
    es: "Es una plataforma de terceros. Tras la instalación, su personal accede mediante Gerege SSO; los usuarios de organizaciones que no la hayan instalado no pueden acceder.",
  },
  "store.back": {
    mn: "← Каталог руу буцах",
    en: "← Back to the catalogue",
    ar: "← العودة إلى الكتالوج",
    zh: "← 返回目录",
    fr: "← Retour au catalogue",
    ru: "← Назад к каталогу",
    es: "← Volver al catálogo",
  },
  "store.notfound": {
    mn: "Ийм апп олдсонгүй.",
    en: "No such app.",
    ar: "لا يوجد تطبيق بهذا الاسم.",
    zh: "找不到该应用。",
    fr: "Application introuvable.",
    ru: "Приложение не найдено.",
    es: "No se ha encontrado la aplicación.",
  },
  "store.developers": {
    mn: "Хөгжүүлэгчдэд",
    en: "For developers",
    ar: "للمطورين",
    zh: "面向开发者",
    fr: "Pour les développeurs",
    ru: "Разработчикам",
    es: "Para desarrolladores",
  },
};

export function isLocale(value: string | undefined): value is Locale {
  return !!value && LOCALES.some((entry) => entry.code === value);
}

export function translator(locale: Locale) {
  return (key: keyof typeof DICTIONARY) => DICTIONARY[key]?.[locale] ?? DICTIONARY[key]?.en ?? key;
}

export function isRTL(locale: Locale) {
  return LOCALES.some((entry) => entry.code === locale && "rtl" in entry && entry.rtl);
}
