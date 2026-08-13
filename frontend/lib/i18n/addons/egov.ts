/**
 * egov — this organisation's connection to the state's systems.
 *
 * Three screens: what it can ask (registry lookups), how it is connected
 * (rails), and what it has asked (history).
 */
export const egov = {
  "egov.view.title": { mn: "Лавлагаа", en: "Registry lookups" },
  "egov.view.subtitle": { mn: "ХУР-аас иргэн, хуулийн этгээдийн албан ёсны мэдээлэл авах", en: "Authoritative citizen and legal-entity data from ХУР" },
  "egov.view.connections_title": { mn: "Холболтууд", en: "Connections" },
  "egov.view.connections_subtitle": { mn: "Энэ суулгац төрийн системүүдтэй хэрхэн холбогдсон бэ", en: "How this deployment reaches the state's systems" },
  "egov.view.history_title": { mn: "Лавлагааны түүх", en: "Lookup history" },
  "egov.view.history_subtitle": { mn: "Энэ байгууллага төрөөс юу асуусан, хэн асуусан бэ", en: "What this organisation asked the state, and who asked it" },

  "egov.tab.citizen": { mn: "Иргэн", en: "Citizen" },
  "egov.tab.company": { mn: "Хуулийн этгээд", en: "Legal entity" },

  "egov.field.reg_number": { mn: "Регистрийн дугаар", en: "Registration number" },
  "egov.field.company_reg": { mn: "Байгууллагын регистр", en: "Company registration number" },
  "egov.action.lookup": { mn: "Лавлах", en: "Look up" },

  "egov.field.civil_id": { mn: "Иргэний үнэмлэхийн дугаар", en: "Civil ID" },
  "egov.field.full_name": { mn: "Овог, нэр", en: "Name" },
  "egov.field.gender": { mn: "Хүйс", en: "Gender" },
  "egov.field.address": { mn: "Хаяг", en: "Address" },
  "egov.field.passport_status": { mn: "Иргэний үнэмлэхийн төлөв", en: "Passport status" },
  "egov.field.verified": { mn: "Баталгаажсан", en: "Verified" },
  "egov.field.company_name": { mn: "Нэр", en: "Name" },
  "egov.field.executive": { mn: "Гүйцэтгэх удирдлага", en: "Executive" },
  "egov.field.vat_payer": { mn: "НӨАТ төлөгч", en: "VAT payer" },
  "egov.field.status": { mn: "Төлөв", en: "Status" },
  "egov.field.founding_date": { mn: "Байгуулагдсан огноо", en: "Founded" },

  "egov.field.rail": { mn: "Суваг", en: "Rail" },
  "egov.field.mode": { mn: "Горим", en: "Mode" },
  "egov.field.endpoint": { mn: "Хаяг", en: "Endpoint" },
  "egov.mode.live": { mn: "Ажиллаж байна", en: "Live" },
  "egov.mode.mock": { mn: "Туршилтын өгөгдөл", en: "Mock data" },
  "egov.mode.unconfigured": { mn: "Тохируулаагүй", en: "Not configured" },
  "egov.message.mock_warning": {
    mn: "Туршилтын горимд байгаа суваг жинхэнэ хариу өгөхгүй. Буцаж ирсэн өгөгдлийг албан ёсны баримтад ашиглаж болохгүй.",
    en: "A rail in mock mode answers from fixtures. Nothing it returns is authoritative, and none of it belongs on an official document.",
  },
  "egov.message.identities_hint": {
    mn: "Өөрийн eID / ДАН холболтоо профайл хэсгээсээ харна. Тэр нь хүний өөрийнх тул энэ аппаас хамаардаггүй.",
    en: "Your own eID and ДАН links live in your profile. They belong to you rather than to this organisation, so they do not depend on this app.",
  },
  "egov.action.open_profile": { mn: "Профайл руу очих", en: "Open profile" },

  "egov.field.when": { mn: "Хэзээ", en: "When" },
  "egov.field.who": { mn: "Хэн", en: "Who" },
  "egov.field.what": { mn: "Юу", en: "What" },
  "egov.action.citizen_queried": { mn: "Иргэний лавлагаа", en: "Citizen lookup" },
  "egov.action.company_queried": { mn: "Хуулийн этгээдийн лавлагаа", en: "Legal-entity lookup" },

  "egov.message.empty_result": { mn: "Регистрийн дугаараа оруулаад лавлана уу.", en: "Enter a registration number to look one up." },
  "egov.message.empty_history": { mn: "Энэ байгууллага одоогоор лавлагаа аваагүй байна.", en: "This organisation has not looked anything up yet." },
  "egov.message.lookup_failed": { mn: "Лавлагаа авч чадсангүй", en: "The lookup failed" },
};
