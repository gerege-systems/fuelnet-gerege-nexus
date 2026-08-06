/**
 * documents — Document routing, categories and e-signatures.
 */
export const documents = {
  "documents.view.title": { mn: "Цахим баримт ба тоон гарын үсэг", en: "Digital Documents & E-Signatures" },
  "documents.view.create_title": { mn: "Цахим баримт үүсгэх", en: "Create Digital Document" },
  "documents.view.sign_title": { mn: "Баримтад гарын үсэг зурах", en: "Sign Document" },
  "documents.view.approvals_hint": {
    mn: "Гарын үсэг хүлээж байгаа баримтууд — E-ID / ДАН-аар батлах эсвэл татгалзах.",
    en: "Documents awaiting a decision — approve with an E-ID / DAN signature, or reject.",
  },

  // Screen title for the queue. The sidebar entry itself is labelled by the
  // menu blueprint on the server; this is the heading the page draws.
  "documents.menu.approvals": { mn: "Батлах дараалал", en: "Approval queue" },

  "documents.field.title": { mn: "Баримтын гарчиг", en: "Document Title" },
  "documents.field.title_placeholder": { mn: "e.g. Хамтран ажиллах гэрээ 2026", en: "e.g. Partnership agreement 2026" },
  "documents.field.category": { mn: "Баримтын ангилал", en: "Document Category" },
  "documents.field.signature": { mn: "Тоон гарын үсэг (E-ID / ДАН)", en: "Digital Signature (E-ID / DAN)" },
  "documents.field.created": { mn: "Үүсгэсэн огноо", en: "Created Date" },
  "documents.field.signature_method": { mn: "Гарын үсгийн арга", en: "Signature Method" },
  "documents.field.reg_number": { mn: "Регистрийн дугаар", en: "Registration Number" },
  "documents.field.otp_code": { mn: "Нэг удаагийн нууц код (OTP)", en: "One-Time Code (OTP)" },
  "documents.field.waiting_days": { mn: "Хүлээсэн хоног", en: "Days waiting" },

  "documents.stat.awaiting": { mn: "Гарын үсэг хүлээж буй", en: "Awaiting signature" },
  "documents.stat.oldest_days": { mn: "Хамгийн урт хүлээлт (хоног)", en: "Longest wait (days)" },

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
  "documents.message.no_pending": {
    mn: "Батлах хүлээж байгаа баримт байхгүй.",
    en: "Nothing is waiting for approval.",
  },
  "documents.message.sign_not_granted": {
    mn: "Танд гарын үсэг зурах эрх (documents.sign) байхгүй тул үйлдлүүд харагдахгүй.",
    en: "The actions are hidden because you do not hold the documents.sign permission.",
  },
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
