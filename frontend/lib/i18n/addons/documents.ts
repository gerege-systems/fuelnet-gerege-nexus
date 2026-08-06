/**
 * documents — Document routing, categories and e-signatures.
 */
export const documents = {
  "documents.view.title": { mn: "Цахим баримт ба тоон гарын үсэг", en: "Digital Documents & E-Signatures" },
  "documents.view.create_title": { mn: "Цахим баримт үүсгэх", en: "Create Digital Document" },

  "documents.field.title": { mn: "Баримтын гарчиг", en: "Document Title" },
  "documents.field.title_placeholder": { mn: "e.g. Хамтран ажиллах гэрээ 2026", en: "e.g. Partnership agreement 2026" },
  "documents.field.category": { mn: "Баримтын ангилал", en: "Document Category" },
  "documents.field.signature": { mn: "Тоон гарын үсэг (E-ID / ДАН)", en: "Digital Signature (E-ID / DAN)" },
  "documents.field.created": { mn: "Үүсгэсэн огноо", en: "Created Date" },

  "documents.category.legal_contract": { mn: "Гэрээ", en: "Legal Contract" },
  "documents.category.official_request": { mn: "Албан хүсэлт", en: "Official Request" },
  "documents.category.internal_approval": { mn: "Дотоод батламж", en: "Internal Approval" },

  "documents.state.pending_signature": { mn: "Гарын үсэг хүлээж буй", en: "Pending Signature" },

  "documents.action.create": { mn: "Баримт үүсгэх", en: "Create Document" },

  "documents.message.loading": { mn: "Баримтуудыг ачаалж байна...", en: "Loading documents..." },
} as const;
