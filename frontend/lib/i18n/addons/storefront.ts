/**
 * storefront — The public landing page of a deployment that is an app store.
 *
 * Separate from `website` because it answers a different question. The platform
 * page argues what Gerege Nexus is; a store's visitor already knows and is
 * asking what is in it. Same image, same components, different first screen —
 * which deployment gets which is decided at run time, not at build time.
 */
export const storefront = {
  "storefront.view.eyebrow": { mn: "GEREGE APP STORE · НЭЭЛТТЭЙ КАТАЛОГ", en: "GEREGE APP STORE · OPEN CATALOGUE" },
  "storefront.view.title_lead": { mn: "Байгууллагадаа хэрэгтэй", en: "Everything an organisation runs," },
  "storefront.view.title_highlight": { mn: "аппуудаа сонгоно уу", en: "one app at a time" },
  "storefront.view.lede": {
    mn: "{brand} дээр ажилладаг аппуудын албан ёсны каталог. Апп бүр гарын үсэгтэй manifest-тэй ирдэг тул суулгац бүр юуг суулгаж байгаагаа шалгаж чадна.",
    en: "The official catalogue of applications that run on {brand}. Every app arrives with a signed manifest, so an instance can verify what it is installing.",
  },
  "storefront.action.browse": { mn: "Каталогийг үзэх", en: "Browse the catalogue" },
  "storefront.action.publish": { mn: "Апп нийтлэх", en: "Publish an app" },

  "storefront.stat.apps": { mn: "апп", en: "apps" },
  "storefront.stat.categories": { mn: "ангилал", en: "categories" },
  "storefront.stat.publishers": { mn: "нийтлэгч", en: "publishers" },

  "storefront.view.catalogue_eyebrow": { mn: "КАТАЛОГ", en: "CATALOGUE" },
  "storefront.view.catalogue_title": { mn: "Нийтлэгдсэн аппууд", en: "Published applications" },
  "storefront.view.catalogue_lede": {
    mn: "Суулгацын администратор эдгээрийг Апп дэлгүүрээсээ шууд суулгана. Хувилбар бүр гарын үсэгтэй каталогоор тараагддаг.",
    en: "An instance administrator installs these from their own App Store screen. Every version is distributed through the signed catalogue.",
  },
  "storefront.category.other": { mn: "Бусад", en: "Other" },
};
