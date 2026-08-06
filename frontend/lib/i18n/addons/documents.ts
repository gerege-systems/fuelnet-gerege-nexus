/**
 * documents — Document routing, categories and e-signatures.
 */
export const documents = {
  "documents.view.title": { mn: "Цахим баримт ба тоон гарын үсэг", en: "Digital Documents & E-Signatures" },
  "documents.view.create_title": { mn: "Цахим баримт үүсгэх", en: "Create Digital Document" },
  "documents.view.sign_title": { mn: "Баримтад гарын үсэг зурах", en: "Sign Document" },

  "documents.field.title": { mn: "Баримтын гарчиг", en: "Document Title" },
  "documents.field.title_placeholder": { mn: "e.g. Хамтран ажиллах гэрээ 2026", en: "e.g. Partnership agreement 2026" },
  "documents.field.category": { mn: "Баримтын ангилал", en: "Document Category" },
  "documents.field.signature": { mn: "Тоон гарын үсэг (E-ID / ДАН)", en: "Digital Signature (E-ID / DAN)" },
  "documents.field.created": { mn: "Үүсгэсэн огноо", en: "Created Date" },
  "documents.field.signature_method": { mn: "Гарын үсгийн арга", en: "Signature Method" },
  "documents.field.reg_number": { mn: "Регистрийн дугаар", en: "Registration Number" },
  "documents.field.otp_code": { mn: "Нэг удаагийн нууц код (OTP)", en: "One-Time Code (OTP)" },

  "documents.category.legal_contract": { mn: "Гэрээ", en: "Legal Contract" },
  "documents.category.official_request": { mn: "Албан хүсэлт", en: "Official Request" },
  "documents.category.internal_approval": { mn: "Дотоод батламж", en: "Internal Approval" },

  "documents.state.pending_signature": { mn: "Гарын үсэг хүлээж буй", en: "Pending Signature" },
  "documents.state.draft": { mn: "Ноорог", en: "Draft" },
  "documents.state.pending": { mn: "Хүлээгдэж буй", en: "Pending" },
  "documents.state.approved": { mn: "Баталсан", en: "Approved" },
  "documents.state.rejected": { mn: "Татгалзсан", en: "Rejected" },

  "documents.action.create": { mn: "Баримт үүсгэх", en: "Create Document" },
  "documents.action.sign": { mn: "Гарын үсэг зурах", en: "Sign" },
  "documents.action.reject": { mn: "Татгалзах", en: "Reject" },

  "documents.message.loading": { mn: "Баримтуудыг ачаалж байна...", en: "Loading documents..." },
  "documents.message.empty": {
    mn: "Одоогоор баримт байхгүй. Баримт үүсгээд E-ID / ДАН гарын үсэгт илгээнэ үү.",
    en: "No documents yet. Create one and route it for an E-ID / DAN signature.",
  },
  "documents.message.signing": { mn: "Гарын үсэг зурж байна...", en: "Signing..." },
  "documents.message.rejected_not_signed": {
    mn: "Татгалзсан — гарын үсэг зураагүй",
    en: "Rejected — not signed",
  },
  "documents.message.sign_success": {
    mn: "\"{title}\"-д {method}-ээр гарын үсэг зурлаа.",
    en: "\"{title}\" was successfully signed via {method}.",
  },
  "documents.message.sign_failed": { mn: "Гарын үсэг зурж чадсангүй", en: "Signature failed" },
  "documents.message.reject_confirm": {
    mn: "\"{title}\"-г татгалзах уу? Үүнийг буцаах боломжгүй.",
    en: "Reject \"{title}\"? This cannot be undone.",
  },
  "documents.message.reject_success": { mn: "\"{title}\"-г татгалзлаа.", en: "\"{title}\" was rejected." },
  "documents.message.reject_failed": { mn: "Татгалзаж чадсангүй", en: "Reject failed" },
  "documents.message.create_failed": { mn: "Баримт үүсгэж чадсангүй", en: "Failed to create document" },
} as const;
