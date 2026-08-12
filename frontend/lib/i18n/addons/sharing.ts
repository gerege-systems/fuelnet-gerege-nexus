/**
 * sharing — Cross-tenant report sharing (§3.5 of the proposal).
 *
 * Its own block rather than more `reports.*` keys: the screen lives under
 * Settings, is used by administrators rather than by whoever reads a report,
 * and the vocabulary is about agreements rather than about numbers.
 */
export const reportSharing = {
  "sharing.view.title": { mn: "Тайлан хуваалцах", en: "Report sharing" },
  "sharing.view.subtitle": {
    mn: "Өөр байгууллагад ямар тайлан харуулахаа шийдэх, өөрт өгөгдсөн эрхийг харах, хэн юу татсаныг мөрдөх.",
    en: "Decide which reports other organisations may see, review what you have been given, and follow who read what.",
  },

  "sharing.section.pending": { mn: "Хүлээгдэж буй хүсэлт", en: "Requests waiting for you" },
  "sharing.section.given": { mn: "Бидний өгсөн эрх", en: "Access we have given" },
  "sharing.section.received": { mn: "Бидэнд өгөгдсөн эрх", en: "Access we have been given" },
  "sharing.section.history": { mn: "Хандалтын түүх", en: "Access history" },
  "sharing.section.closed": { mn: "Хаагдсан", en: "Closed" },

  "sharing.action.request": { mn: "Тайлан хүсэх", en: "Ask for a report" },
  "sharing.action.send_request": { mn: "Хүсэлт илгээх", en: "Send the request" },
  "sharing.action.accept": { mn: "Зөвшөөрөх", en: "Agree" },
  "sharing.action.refuse": { mn: "Татгалзах", en: "Refuse" },
  "sharing.action.revoke": { mn: "Цуцлах", en: "Revoke" },

  "sharing.field.registration": { mn: "Улсын бүртгэлийн дугаар", en: "Registration number" },
  "sharing.field.report": { mn: "Тайлан", en: "Report" },
  "sharing.field.scope": { mn: "Хүрээ", en: "Scope" },
  "sharing.field.valid_until": { mn: "Дуусах хугацаа", en: "Valid until" },
  "sharing.field.note": { mn: "Тайлбар", en: "Note" },
  "sharing.field.when": { mn: "Хэзээ", en: "When" },
  "sharing.field.who": { mn: "Хэн", en: "Who" },
  "sharing.field.rows": { mn: "Мөр", en: "Rows" },

  "sharing.scope.counterparty": {
    mn: "Зөвхөн бидэнтэй холбоотой мөрүүд",
    en: "Only the rows that concern us",
  },
  "sharing.scope.full": { mn: "Тайлан бүхэлдээ", en: "The whole report" },

  "sharing.hint.pending": {
    mn: "Өөр байгууллага бидний тайланг харах хүсэлт гаргасан. Зөвшөөрөх хүртэл юу ч харагдахгүй.",
    en: "Another organisation has asked to see one of our reports. Nothing is visible to them until this is agreed.",
  },
  "sharing.hint.given": {
    mn: "Бидний өгөгдлийг харах эрх. Цуцлахад тэр дороо хаагдана.",
    en: "Permission over our data. Revoking closes it at once.",
  },
  "sharing.hint.received": {
    mn: "Бусад байгууллагын бидэнд өгсөн эрх. Нэгдсэн тайланд эдгээр л орно.",
    en: "What other organisations have agreed to show us. A consolidated report covers exactly these.",
  },
  "sharing.hint.history": {
    mn: "Бидний өгөгдлийг хэн, хэзээ, ямар тайлангаар уншсан бэ.",
    en: "Who read our data, when, and through which report.",
  },
  "sharing.hint.request": {
    mn: "Хүсэлт нь эрх биш. Өгөгдөл эзэмшигч талын админ зөвшөөрөх хүртэл юу ч харагдахгүй.",
    en: "A request is not a permission. Nothing is visible until the owning organisation's administrator agrees.",
  },
  "sharing.hint.registration": {
    mn: "Өгөгдөл эзэмшигч байгууллагын дугаар. Нэрээр биш дугаараар: нэр давхардаж, өөрчлөгддөг.",
    en: "The owning organisation's number. By number rather than by name: names repeat and change.",
  },
  "sharing.hint.scope": {
    mn: "«Зөвхөн бидэнтэй холбоотой» нь гэрээт талын хүрээ. «Бүхэлдээ» нь толгой байгууллага охин компанийг нэгтгэх тохиолдол.",
    en: "“Only the rows that concern us” is the contracted-parties scope. “The whole report” is a parent consolidating a subsidiary.",
  },
  "sharing.hint.note": {
    mn: "Яагаад хэрэгтэй байгаагаа бич — зөвшөөрөх эсэхээ шийдэх хүн үүнийг уншина.",
    en: "Say why. The person deciding whether to agree reads this.",
  },

  "sharing.message.no_pending": { mn: "Хүлээгдэж буй хүсэлт алга.", en: "No requests are waiting." },
  "sharing.message.no_given": { mn: "Бид хэнд ч эрх өгөөгүй байна.", en: "We have given nobody access." },
  "sharing.message.no_received": { mn: "Бидэнд эрх өгөгдөөгүй байна.", en: "Nobody has given us access." },
  "sharing.message.no_history": { mn: "Бидний өгөгдлийг хэн ч уншаагүй байна.", en: "Nobody has read our data." },
  "sharing.message.accepted": { mn: "Зөвшөөрөл олгогдлоо.", en: "Access has been granted." },
  "sharing.message.refused": { mn: "Хүсэлт татгалзагдлаа.", en: "The request was refused." },
  "sharing.message.revoked": { mn: "Эрх цуцлагдлаа.", en: "The access was revoked." },
  "sharing.message.withdrawn": { mn: "Эрхээс татгалзлаа.", en: "The access was given up." },
  "sharing.message.requested": { mn: "Хүсэлт илгээгдлээ.", en: "The request was sent." },
} as const;
