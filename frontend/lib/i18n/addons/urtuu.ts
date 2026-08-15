/**
 * urtuu — Өртөө: the channel this installation has to the ones above and below
 * it, and the request codes work may be raised under.
 *
 * Mongolian is the source language here more than anywhere else on the
 * platform: the subject is the relay-post network, the codes come from the
 * state's own register, and the words for a link, an invitation and a request
 * code are the words the people using this screen already say.
 */
export const urtuu = {
  "urtuu.view.title": { mn: "Өртөө", en: "Urtuu Relay" },
  "urtuu.view.subtitle": {
    mn: "Дээд, доод платформтой холбогдох суваг: холбоос, хүсэлтийн кодууд.",
    en: "The channel to the installations above and below this one: links and request codes.",
  },
  "urtuu.view.identity": { mn: "Энэ суулгацын мөр", en: "This installation's identity" },
  "urtuu.view.identity_hint": {
    mn: "Доод платформ энэ түлхүүрээр таны илгээсэн дугтуйг шалгана. Түлхүүр солигдвол одоо байгаа холбоосууд дахин байгуулагдана.",
    en: "A subordinate installation verifies your envelopes with this key. Changing it means every existing link has to be established again.",
  },
  "urtuu.view.installation_id": { mn: "Суулгацын ID", en: "Installation id" },
  "urtuu.view.public_key": { mn: "Нийтийн түлхүүр", en: "Public key" },

  "urtuu.message.disabled": {
    mn: "Энэ суулгац дээр Өртөө тохируулагдаагүй байна: URTUU_SIGNING_KEY тавигдаагүй тул холбоос үүсгэх боломжгүй.",
    en: "Өртөө is not configured on this installation: URTUU_SIGNING_KEY is unset, so no link can be established.",
  },
  "urtuu.message.no_links": { mn: "Холбоос алга", en: "No links yet" },
  "urtuu.message.no_codes": { mn: "Хүсэлтийн код алга", en: "No request codes yet" },
  "urtuu.message.never": { mn: "Хэзээ ч", en: "Never" },
  "urtuu.message.invite_hint": {
    mn: "Энэ кодыг доод байгууллагын админд дамжуул. 24 цаг хүчинтэй, нэг л удаа ажиллана, дахин харагдахгүй.",
    en: "Pass this code to the other organisation's administrator. It lasts 24 hours, works once, and is never shown again.",
  },
  "urtuu.message.joined": { mn: "Холбоос бүртгэгдлээ. Дээд тал баталгаажуулахыг хүлээж байна.", en: "The link is recorded. It is waiting for the parent to confirm." },
  "urtuu.message.confirmed": { mn: "Холбоос идэвхжлээ", en: "The link is open" },
  "urtuu.message.revoked": { mn: "Холбоос цуцлагдлаа", en: "The link is closed" },
  "urtuu.message.confirm_revoke": {
    mn: "{name} холбоосыг цуцлах уу? Хүргэгдээгүй дугтуйнууд зогсоно.",
    en: "Close the link to {name}? Anything undelivered stops.",
  },
  "urtuu.message.codes_saved": { mn: "Кодын жагсаалт хадгалагдаж, доод тал руу зарлагдлаа", en: "The vocabulary is saved and announced downstream" },
  "urtuu.message.code_created": { mn: "Код бүртгэгдлээ", en: "The code is registered" },
  "urtuu.message.code_updated": { mn: "Код шинэчлэгдлээ", en: "The code is updated" },
  "urtuu.message.imported": { mn: "{count} код ring.dgov.mn-ээс импортлогдлоо", en: "{count} codes imported from ring.dgov.mn" },
  "urtuu.message.ring_off": {
    mn: "ring.dgov.mn тохируулагдаагүй (RING_BASE_URL). Кодуудыг гараар local. угтвартай үүсгэж болно.",
    en: "ring.dgov.mn is not configured (RING_BASE_URL). Codes can still be authored by hand under the local. prefix.",
  },
  "urtuu.message.undelivered": { mn: "{count} хүргэгдээгүй", en: "{count} undelivered" },
  "urtuu.message.clock_skew": { mn: "Цагийн зөрүү {seconds} сек", en: "Clock differs by {seconds}s" },
  "urtuu.message.local_prefix": {
    mn: "Энд зохиосон код заавал local. угтвартай байна — угтваргүй нэрийн орон зай ring.dgov.mn-ийнх.",
    en: "A code authored here must start with local. — the unprefixed namespace belongs to ring.dgov.mn.",
  },

  "urtuu.section.links": { mn: "Холбоосууд", en: "Links" },
  "urtuu.section.codes": { mn: "Хүсэлтийн кодууд", en: "Request codes" },
  "urtuu.hint.links": {
    mn: "Доод тал л холбогдоно: доод платформ дээд рүүгээ өөрөө хандаж даалгавраа татаж, биелэлтээ түлхэнэ.",
    en: "Only the child dials. A subordinate installation reaches up for its work and pushes back what it has done.",
  },
  "urtuu.hint.codes": {
    mn: "Даалгавар чөлөөт текстээр үүсэхгүй — кодоор үүснэ. Код нь юу бөглөхийг, хэдий хугацаанд хийхийг өөрөө хэлнэ.",
    en: "A task is never free text. It is raised under a code, and the code says what has to be filled in and how long the work is allowed.",
  },

  "urtuu.action.invite": { mn: "Доод платформ урих", en: "Invite a subordinate" },
  "urtuu.action.join": { mn: "Дээд платформд холбогдох", en: "Join a parent" },
  "urtuu.action.confirm": { mn: "Баталгаажуулах", en: "Confirm" },
  "urtuu.action.revoke": { mn: "Цуцлах", en: "Close" },
  "urtuu.action.open_codes": { mn: "Кодуудыг нээх", en: "Open codes" },
  "urtuu.action.create_code": { mn: "Локал код", en: "Local code" },
  "urtuu.action.ring_sync": { mn: "ring.dgov.mn-ээс татах", en: "Import from ring.dgov.mn" },
  "urtuu.action.save": { mn: "Хадгалах", en: "Save" },
  "urtuu.action.copy": { mn: "Хуулах", en: "Copy" },

  "urtuu.field.name": { mn: "Нэр", en: "Name" },
  "urtuu.field.role": { mn: "Үүрэг", en: "Role" },
  "urtuu.field.status": { mn: "Төлөв", en: "Status" },
  "urtuu.field.last_seen": { mn: "Сүүлд холбогдсон", en: "Last seen" },
  "urtuu.field.base_url": { mn: "Дээд платформын хаяг", en: "The parent's address" },
  "urtuu.field.invite_code": { mn: "Урилгын код", en: "Invitation code" },
  "urtuu.field.code": { mn: "Код", en: "Code" },
  "urtuu.field.source": { mn: "Эх сурвалж", en: "Source" },
  "urtuu.field.sla": { mn: "Хугацааны норм", en: "Time allowed" },
  "urtuu.field.sla_days": { mn: "{days} хоног", en: "{days} days" },
  "urtuu.field.sla_none": { mn: "Норм заагаагүй", en: "No norm" },
  "urtuu.field.mn_name": { mn: "Нэр (монгол)", en: "Name (Mongolian)" },
  "urtuu.field.en_name": { mn: "Нэр (англи)", en: "Name (English)" },
  "urtuu.field.schema": { mn: "Талбаруудын schema (JSON)", en: "Field schema (JSON)" },
  "urtuu.field.active": { mn: "Ашиглана", en: "In use" },

  "urtuu.role.parent": { mn: "Бид дээд", en: "We are the parent" },
  "urtuu.role.child": { mn: "Бид доод", en: "We are the child" },

  "urtuu.status.pending": { mn: "Хүлээгдэж буй", en: "Waiting" },
  "urtuu.status.active": { mn: "Идэвхтэй", en: "Open" },
  "urtuu.status.revoked": { mn: "Цуцлагдсан", en: "Closed" },

  "urtuu.source.ring": { mn: "ring.dgov.mn", en: "ring.dgov.mn" },
  "urtuu.source.link": { mn: "Холбоосоор ирсэн", en: "Announced upstream" },
  "urtuu.source.local": { mn: "Энд зохиосон", en: "Authored here" },

  "urtuu.modal.invite": { mn: "Доод платформ урих", en: "Invite a subordinate" },
  "urtuu.modal.join": { mn: "Дээд платформд холбогдох", en: "Join a parent" },
  "urtuu.modal.code": { mn: "Локал код бүртгэх", en: "Register a local code" },
  "urtuu.modal.open_codes": { mn: "{name} холбоос дээр нээх кодууд", en: "Codes open on the link to {name}" },
} as const;
