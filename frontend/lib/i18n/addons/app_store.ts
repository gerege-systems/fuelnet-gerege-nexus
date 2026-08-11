/**
 * app_store — The application catalogue and what a tenant has installed.
 */
export const app_store = {
  "app_store.view.title": { mn: "Платформын Апп Дэлгүүр", en: "Platform App Store" },
  "app_store.view.subtitle": { mn: "Компиляцын үеийн бизнес модулиудыг суулгаж, идэвхжүүлж, удирдана", en: "Install, enable, and manage compile-time business modules" },
  "app_store.view.installed_title": { mn: "Суулгасан аппуудын тохиргоо", en: "Installed Apps Settings" },
  "app_store.view.search_placeholder": { mn: "Апп хайх...", en: "Search apps..." },

  "app_store.field.requires": { mn: "Шаардлага: ", en: "Requires:" },
  "app_store.field.application_name": { mn: "Аппликейшны нэр", en: "Application Name" },
  "app_store.field.module_id": { mn: "Модулийн ID", en: "Module ID" },
  "app_store.field.installed_version": { mn: "Суулгасан хувилбар", en: "Installed Version" },
  "app_store.field.installed_date": { mn: "Суулгасан огноо", en: "Installed Date" },

  "app_store.field.latest_version": { mn: "Сүүлийн хувилбар", en: "Latest version" },
  "app_store.field.updates": { mn: "Шинэчлэлт", en: "Updates" },

  "app_store.action.install": { mn: "Суулгах", en: "Install App" },
  "app_store.action.enable": { mn: "Идэвхжүүлэх", en: "Enable App" },
  "app_store.action.disable": { mn: "Идэвхгүй болгох", en: "Disable App" },
  "app_store.action.update": { mn: "Шинэчлэх", en: "Update" },
  "app_store.action.approve_update": { mn: "Зөвшөөрч шинэчлэх", en: "Approve and update" },

  "app_store.state.installed": { mn: "Суулгасан ба идэвхтэй", en: "Installed & Enabled" },
  "app_store.state.disabled": { mn: "Идэвхгүй", en: "Disabled" },
  "app_store.state.update_available": { mn: "Шинэчлэлт бэлэн", en: "Update available" },
  "app_store.state.auto_update_on": { mn: "Автоматаар шинэчилнэ", en: "Updates automatically" },
  "app_store.state.auto_update_off": { mn: "Гараар шинэчилнэ", en: "Updated by hand" },
  "app_store.state.pinned": { mn: "{version} хувилбар дээр тогтоосон", en: "Held at {version}" },

  "app_store.filter.all": { mn: "Бүгд", en: "All" },

  "app_store.message.loading": { mn: "Апп каталог ачаалж байна...", en: "Loading apps catalog..." },
  "app_store.message.loading_installed": { mn: "Суулгасан аппуудыг ачаалж байна...", en: "Loading installed apps..." },
  "app_store.message.no_match": { mn: "Хайлтад тохирох апп олдсонгүй.", en: "No apps found matching your query." },

  "app_store.view.installed_subtitle": { mn: "Суулгасан модулиудыг удирдаж, төлөвийг хянаж, идэвхжүүлэх буюу идэвхгүй болгоно", en: "Manage installed tenant modules, check operational status, and enable or disable features" },

  "app_store.action.browse_store": { mn: "Апп Дэлгүүр рүү очих", en: "Go to the App Store" },

  // Two self-contained sentences rather than one split around a link. A
  // sentence cut in half translates badly: word order moves in Chinese and
  // reverses in Arabic, so the fragments no longer join up.
  "app_store.message.none_installed": { mn: "Энэ тенантад одоогоор апп суулгаагүй байна.", en: "No apps installed for this tenant yet." },
  "app_store.message.load_failed": { mn: "Аппын каталогийг ачаалж чадсангүй", en: "Failed to load the app catalog" },
  "app_store.message.action_failed": { mn: "Үйлдэл амжилтгүй боллоо", en: "Action failed" },

  // What the store says while it is working and after it has finished. The app
  // name is a variable rather than part of the sentence: it is catalogue
  // content, already translated by the API, and splicing a translated name into
  // a hand-written half-sentence is what leaves a screen half in one language.
  "app_store.message.installing": { mn: "Суулгаж байна...", en: "Installing..." },
  "app_store.message.updating": { mn: "Шинэчилж байна...", en: "Updating..." },
  "app_store.message.install_succeeded": {
    mn: "{app} болон түүний шаардлагатай аппуудыг суулгалаа.",
    en: "Installed {app} and the apps it depends on.",
  },
  "app_store.message.install_failed": { mn: "{app}-ыг суулгаж чадсангүй.", en: "Could not install {app}." },
  "app_store.message.update_succeeded": {
    mn: "{app} {version} хувилбар руу шинэчлэгдлээ.",
    en: "Updated {app} to {version}.",
  },
  "app_store.message.update_failed": { mn: "{app}-ыг шинэчилж чадсангүй.", en: "Could not update {app}." },
  "app_store.message.enabled": { mn: "{app} идэвхжлээ.", en: "Enabled {app}." },
  "app_store.message.disabled": { mn: "{app} идэвхгүй боллоо.", en: "Disabled {app}." },

  // Said where the decision is made. An app whose new version asks for more
  // than the installed one is not updated on its own — the administrator is
  // shown what it added and decides.
  "app_store.message.held_for_approval": {
    mn: "Шинэ хувилбар нэмэлт эрх шаардаж байна:",
    en: "The new version asks for more:",
  },
} as const;
