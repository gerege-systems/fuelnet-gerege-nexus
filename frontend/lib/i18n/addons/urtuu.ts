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
    mn: "Дээд шатны болон харьяа байгууллагатай холбогдох суваг: холбоос, хүсэлтийн кодууд.",
    en: "The channel to the superior and subordinate organisations: links and request codes.",
  },
  "urtuu.view.identity": { mn: "Энэ суулгацын мөр", en: "This installation's identity" },
  "urtuu.view.identity_hint": {
    mn: "Харьяа байгууллага энэ түлхүүрээр таны илгээсэн дугтуйг шалгана. Түлхүүр солигдвол одоо байгаа холбоосууд дахин байгуулагдана.",
    en: "A subordinate organisation verifies your envelopes with this key. Changing it means every existing link has to be established again.",
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
    mn: "Энэ кодыг харьяа байгууллагын админд дамжуул. 24 цаг хүчинтэй, нэг л удаа ажиллана, дахин харагдахгүй.",
    en: "Pass this code to the other organisation's administrator. It lasts 24 hours, works once, and is never shown again.",
  },
  "urtuu.message.joined": { mn: "Холбоос бүртгэгдлээ. Дээд шатны байгууллага баталгаажуулахыг хүлээж байна.", en: "The link is recorded. It is waiting for the superior organisation to confirm." },
  "urtuu.message.confirmed": { mn: "Холбоос идэвхжлээ", en: "The link is open" },
  "urtuu.message.revoked": { mn: "Холбоос цуцлагдлаа", en: "The link is closed" },
  "urtuu.message.confirm_revoke": {
    mn: "{name} холбоосыг цуцлах уу? Хүргэгдээгүй дугтуйнууд зогсоно.",
    en: "Close the link to {name}? Anything undelivered stops.",
  },
  "urtuu.message.codes_saved": { mn: "Кодын жагсаалт хадгалагдаж, харьяа байгууллага руу зарлагдлаа", en: "The vocabulary is saved and announced to the subordinate" },
  "urtuu.message.code_created": { mn: "Код бүртгэгдлээ", en: "The code is registered" },
  "urtuu.message.code_updated": { mn: "Код шинэчлэгдлээ", en: "The code is updated" },
  "urtuu.message.imported": { mn: "{count} код ring.dgov.mn-ээс импортлогдлоо", en: "{count} codes imported from ring.dgov.mn" },
  "urtuu.message.ring_unchanged": {
    mn: "ring.dgov.mn дээр шинэ зүйл алга — кодууд хэвээрээ.",
    en: "ring.dgov.mn has published nothing new; the codes are unchanged.",
  },
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
    mn: "Харьяа тал л холбогдоно: харьяа байгууллага дээд шат руугаа өөрөө хандаж даалгавраа татаж, биелэлтээ түлхэнэ.",
    en: "Only the subordinate dials. It reaches up for its work and pushes back what it has done.",
  },
  "urtuu.hint.codes": {
    mn: "Даалгавар чөлөөт текстээр үүсэхгүй — кодоор үүснэ. Код нь юу бөглөхийг, хэдий хугацаанд хийхийг өөрөө хэлнэ.",
    en: "A task is never free text. It is raised under a code, and the code says what has to be filled in and how long the work is allowed.",
  },

  "urtuu.action.invite": { mn: "Харьяа байгууллага урих", en: "Invite a subordinate organisation" },
  "urtuu.action.join": { mn: "Дээд шатны байгууллагад холбогдох", en: "Connect to the superior organisation" },
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
  "urtuu.field.base_url": { mn: "Дээд шатны байгууллагын хаяг", en: "The superior organisation's address" },
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

  "urtuu.role.parent": { mn: "Бид — дээд шат", en: "We are the superior" },
  "urtuu.role.child": { mn: "Бид — харьяа", en: "We are the subordinate" },

  "urtuu.status.pending": { mn: "Хүлээгдэж буй", en: "Waiting" },
  "urtuu.status.active": { mn: "Идэвхтэй", en: "Open" },
  "urtuu.status.revoked": { mn: "Цуцлагдсан", en: "Closed" },

  "urtuu.source.ring": { mn: "ring.dgov.mn", en: "ring.dgov.mn" },
  "urtuu.source.link": { mn: "Холбоосоор ирсэн", en: "Announced upstream" },
  "urtuu.source.local": { mn: "Энд зохиосон", en: "Authored here" },

  "urtuu.modal.invite": { mn: "Харьяа байгууллага урих", en: "Invite a subordinate organisation" },
  "urtuu.modal.join": { mn: "Дээд шатны байгууллагад холбогдох", en: "Connect to the superior organisation" },
  "urtuu.modal.code": { mn: "Локал код бүртгэх", en: "Register a local code" },
  "urtuu.modal.open_codes": { mn: "{name} холбоос дээр нээх кодууд", en: "Codes open on the link to {name}" },
  // ---- The task board (io.gerege.nexus.urtuu). The statuses carry the names
  // the proposal uses; the values behind them are Latin and live in
  // pkg/urtuu/status.go, where the reason for that is written down.
  "urtuu.board.title": { mn: "Өртөө — самбар", en: "Urtuu board" },
  "urtuu.board.subtitle": {
    mn: "Ирсэн, илгээсэн даалгаврын тойм, хугацаа хэтэрсэн ажлын улаан бүс.",
    en: "What is queued in both directions, and the red zone of work that is late.",
  },
  "urtuu.incoming.title": { mn: "Ирсэн даалгавар", en: "Incoming tasks" },
  "urtuu.incoming.subtitle": {
    mn: "Дээд шатны байгууллагаас ирсэн ажил. Хүлээн авах, хариуцагч оноох, харьяалалдаа задлах, эсвэл шалтгаантай буцаах.",
    en: "Work given to this organisation. Take it on, assign somebody, split it further, or return it with a reason.",
  },
  "urtuu.outgoing.title": { mn: "Илгээсэн даалгавар", en: "Sent tasks" },
  "urtuu.outgoing.subtitle": {
    mn: "Харьяа байгууллага руу илгээсэн ажил. Төлөв нь тэдний мэдэгдлээр хөдөлнө.",
    en: "Work given to subordinate installations. Each moves when they say it has.",
  },
  "urtuu.links.title": { mn: "Холбоосууд", en: "Links" },
  "urtuu.links.subtitle": {
    mn: "Дээд шатны болон харьяа байгууллагатай холбогдсон суваг бүрийн эрүүл мэнд. Холбоос үүсгэх нь Тохиргоо → Өртөө дээр.",
    en: "The health of every channel to an installation above or below. Establishing one is in Settings → Өртөө.",
  },

  "urtuu.status.RECEIVED": { mn: "Ирсэн", en: "Received" },
  "urtuu.status.ACCEPTED": { mn: "Хүлээн авсан", en: "Accepted" },
  "urtuu.status.IN_PROGRESS": { mn: "Хийгдэж байгаа", en: "In progress" },
  "urtuu.status.DELEGATED": { mn: "Задалсан", en: "Delegated" },
  "urtuu.status.COMPLETED": { mn: "Биелсэн", en: "Completed" },
  "urtuu.status.RETURNED": { mn: "Буцаасан", en: "Returned" },
  "urtuu.status.CLOSED": { mn: "Хаагдсан", en: "Closed" },

  "urtuu.action.new_task": { mn: "Даалгавар үүсгэх", en: "Raise a task" },
  "urtuu.action.accept": { mn: "Хүлээн авах", en: "Accept" },
  "urtuu.action.return": { mn: "Буцаах", en: "Return" },
  "urtuu.action.complete": { mn: "Биелүүлсэн", en: "Complete" },
  "urtuu.action.close": { mn: "Хаах", en: "Close" },
  "urtuu.action.delegate": { mn: "Доошоо задлах", en: "Delegate" },
  "urtuu.action.send": { mn: "Илгээх", en: "Send" },

  "urtuu.field.title": { mn: "Гарчиг", en: "Title" },
  "urtuu.field.deadline": { mn: "Хугацаа", en: "Due" },
  "urtuu.field.from": { mn: "Хаанаас", en: "From" },
  "urtuu.field.to": { mn: "Хаашаа", en: "To" },
  "urtuu.field.note": { mn: "Тайлбар", en: "Note" },
  "urtuu.field.reason": { mn: "Буцаах шалтгаан", en: "Reason for returning" },
  "urtuu.field.targets": { mn: "Зорьсон харьяа байгууллагууд", en: "Subordinate organisations" },
  "urtuu.field.payload": { mn: "Агуулга", en: "Body" },

  "urtuu.filter.all": { mn: "Бүгд", en: "All" },
  "urtuu.filter.overdue": { mn: "Зөвхөн хоцорсон", en: "Late only" },

  "urtuu.section.timeline": { mn: "Түүх", en: "History" },
  "urtuu.section.branches": { mn: "Задаргаа", en: "Branches" },
  "urtuu.section.chain": { mn: "Дамжсан зам", en: "Where it has been" },
  "urtuu.section.overdue": { mn: "Хугацаа хэтэрсэн", en: "Past due" },

  "urtuu.message.overdue": { mn: "Хоцорсон", en: "Late" },
  "urtuu.message.no_tasks": { mn: "Даалгавар алга", en: "No tasks" },
  "urtuu.message.no_overdue": { mn: "Хугацаа хэтэрсэн ажил алга", en: "Nothing is late" },
  "urtuu.message.task_created": { mn: "Даалгавар үүсч, илгээгдлээ", en: "The task is raised and on its way" },
  "urtuu.message.moved": { mn: "Даалгаврын төлөв өөрчлөгдлөө", en: "The task has moved" },
  "urtuu.message.no_deadline": { mn: "Хугацаагүй", en: "No deadline" },
  "urtuu.message.pick_code": { mn: "Хүсэлтийн код сонгоно уу", en: "Choose a request code" },
  // The two lines. They are two promises, not two filters — see
  // db/migrations/00065 and pkg/urtuu/status.go.
  "urtuu.line.service": { mn: "Үйлчилгээний хүсэлт", en: "Service request" },
  "urtuu.line.assignment": { mn: "Албан даалгавар", en: "Official assignment" },
  "urtuu.line.service_hint": {
    mn: "Иргэн, байгууллагаас ирсэн хүсэлт доошоо явж, ХАРИУ нь заавал буцаж ирнэ. Хүсэгч платформын гадна байгаа тул хариугүй хаагдвал тэр хүний асуулт зүгээр л алга болно.",
    en: "A request from a citizen or an organisation travels down and an ANSWER must come back. The person who asked is outside the platform, so a request closed without one is their question thrown away.",
  },
  "urtuu.line.assignment_hint": {
    mn: "Дээд шатны байгууллагаас харьяа байгууллагад өгсөн ажил. Хүсэгч гэж байхгүй — эхлүүлсэн байгууллага өөрөө үр дүнг нь хүлээж авна.",
    en: "Work a superior organisation gave a subordinate. There is no applicant: the organisation that raised it is the one waiting for the outcome.",
  },
  "urtuu.field.number": { mn: "Бүртгэлийн дугаар", en: "Register number" },
  "urtuu.field.line": { mn: "Шугам", en: "Line" },
  "urtuu.field.applicant": { mn: "Хүсэгч", en: "Applicant" },
  "urtuu.field.applicant_kind": { mn: "Хэн", en: "Who" },
  "urtuu.field.applicant_citizen": { mn: "Иргэн", en: "Citizen" },
  "urtuu.field.applicant_organisation": { mn: "Байгууллага", en: "Organisation" },
  "urtuu.field.registry_number": { mn: "Регистрийн дугаар", en: "Registration number" },
  "urtuu.field.contact": { mn: "Холбоо барих", en: "Contact" },
  "urtuu.field.answer": { mn: "Хүсэгчид өгөх хариу", en: "Answer for the applicant" },
  "urtuu.section.answer": { mn: "Хариу", en: "Answer" },
  "urtuu.message.answer_required": {
    mn: "Үйлчилгээний хүсэлтийг хариугүйгээр биелсэн гэж хаах боломжгүй.",
    en: "A service request cannot be completed without an answer for the applicant.",
  },
  "urtuu.message.no_answer_yet": { mn: "Хариу хараахан алга", en: "No answer yet" },

  "urtuu.section.evidence": { mn: "Албан бичиг", en: "Official document" },
  "urtuu.field.document_title": { mn: "Албан бичгийн гарчиг", en: "Document title" },
  "urtuu.field.document_type": { mn: "Төрөл", en: "Type" },
  "urtuu.message.signed": { mn: "Гарын үсэг {count}/{required}", en: "Signed {count} of {required}" },
  "urtuu.message.filed_elsewhere": {
    mn: "Илгээгч тал дээр бүртгэлтэй. Баримт өөрөө дамждаггүй — зөвхөн лавлагаа.",
    en: "Filed at the sending installation. The document itself never travels — only the reference.",
  },
  "urtuu.hint.document": {
    mn: "Заавал биш. Бөглөвөл Баримт бичиг апп дээр бүртгэгдэж, тэндээ eID-ээр гарын үсэг зурагдана; даалгаварт зөвхөн лавлагаа нь хавсрагдана.",
    en: "Optional. Filling it files a document in the Documents app, where it is signed with eID; the task carries only the reference.",
  },
  "urtuu.message.no_open_links": {
    mn: "Идэвхтэй харьяа холбоос алга. Тохиргоо → Өртөө дээр эхлээд холбоос үүсгэнэ.",
    en: "No live subordinate links. Establish one in Settings → Өртөө first.",
  },
} as const;
