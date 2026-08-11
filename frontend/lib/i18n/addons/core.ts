/**
 * core — the organisation and the people in it.
 *
 * Terms here name what an organisation *is* rather than what any one app does
 * with it, so they are deliberately the plainest words available: a
 * registration number is called a registration number, not a "company code".
 */
export const core = {
  "core.view.organisation_title": { mn: "Байгууллага", en: "Organisation" },
  "core.view.organisation_subtitle": {
    mn: "Байгууллагын албан ёсны мэдээлэл — баримт бичиг, тайланд хэвлэгдэнэ",
    en: "The organisation's official details — printed on documents and reports",
  },
  "core.view.people_title": { mn: "Ажилтнууд", en: "People" },
  "core.view.people_subtitle": {
    mn: "Энэ байгууллагад ажиллаж буй хүмүүс, тэдний албан тушаал, харьяалал",
    en: "Who works in this organisation, their job title and where they belong",
  },
  "core.view.departments_title": { mn: "Хэлтэс, нэгж", en: "Departments" },
  "core.view.departments_subtitle": {
    mn: "Байгууллагын бүтэц — аль нэгж хэнд харьяалагдахыг тодорхойлно",
    en: "How the organisation is arranged — which unit reports to which",
  },
  "core.view.archived": { mn: "Архивласан нэгж ({count})", en: "Archived units ({count})" },

  "core.group.identity": { mn: "Албан ёсны нэр, бүртгэл", en: "Legal identity" },
  "core.group.address": { mn: "Хаяг", en: "Address" },
  "core.group.contact": { mn: "Холбоо барих", en: "Contact" },
  // Not "settings": these are what the organisation uses when nobody has said
  // otherwise, and a person may still choose their own language over them.
  "core.group.defaults": { mn: "Үндсэн тохиргоо", en: "Defaults" },

  "core.field.name": { mn: "Дэлгэцэнд харагдах нэр", en: "Display name" },
  "core.field.legal_name": { mn: "Албан ёсны нэр", en: "Legal name" },
  "core.field.registration_number": { mn: "Улсын бүртгэлийн дугаар", en: "Registration number" },
  "core.field.tax_number": { mn: "Татвар төлөгчийн дугаар", en: "Tax number" },
  "core.field.province": { mn: "Аймаг / нийслэл", en: "Province / city" },
  "core.field.district": { mn: "Сум / дүүрэг", en: "District" },
  "core.field.khoroo": { mn: "Баг / хороо", en: "Khoroo" },
  "core.field.address_line": { mn: "Дэлгэрэнгүй хаяг", en: "Address" },
  "core.field.postal_code": { mn: "Шуудангийн код", en: "Postal code" },
  "core.field.website": { mn: "Вэб хуудас", en: "Website" },
  "core.field.logo_url": { mn: "Лого (URL)", en: "Logo URL" },
  "core.field.timezone": { mn: "Цагийн бүс", en: "Time zone" },
  "core.field.currency": { mn: "Валют", en: "Currency" },

  "core.field.code": { mn: "Код", en: "Code" },
  "core.field.parent": { mn: "Харьяалагдах нэгж", en: "Reports to" },
  "core.field.manager": { mn: "Хариуцсан ажилтан", en: "Manager" },
  "core.field.people_count": { mn: "{count} ажилтан", en: "{count} people" },
  "core.field.job_title": { mn: "Албан тушаал", en: "Job title" },
  "core.field.department": { mn: "Хэлтэс, нэгж", en: "Department" },
  "core.field.roles": { mn: "Эрх", en: "Roles" },
  "core.label.admin": { mn: "Админ", en: "Admin" },

  "core.action.deactivate": { mn: "Идэвхгүй болгох", en: "Deactivate" },
  "core.action.reactivate": { mn: "Сэргээх", en: "Reactivate" },
  "core.action.archive": { mn: "Архивлах", en: "Archive" },
  // Its own term rather than reusing the people one: in Mongolian both are
  // "Сэргээх", but a person is reactivated and a unit is restored, and the
  // languages that distinguish the two should be allowed to.
  "core.action.restore": { mn: "Сэргээх", en: "Restore" },

  "core.message.saved": { mn: "Байгууллагын мэдээлэл хадгалагдлаа", en: "The organisation's details were saved" },
  // Said once, at the bottom, rather than as a disabled "Add person" button:
  // the button would suggest the screen could do it and is merely refusing.
  "core.message.people_hint": {
    mn: "Шинэ ажилтныг Тохиргоо → Хандалтын эрх хэсгээс урина. Ажилтныг устгахгүй, идэвхгүй болгодог — түүний хийсэн ажлын түүх хэвээр үлдэнэ.",
    en: "Invite somebody from Settings → Access control. People are deactivated rather than deleted, so what they did stays readable.",
  },
} as const;
