/**
 * emailverify — The platform's shared "prove this address" service: the keys
 * issued to outside callers and the record of what has been sent.
 */
export const emailverify = {
  "emailverify.view.title": { mn: "И-мэйл баталгаажуулалт", en: "Email verification" },
  "emailverify.view.subtitle": {
    mn: "Платформын бүх апп болон гадаад системд зориулсан нэгдсэн и-мэйл баталгаажуулалт",
    en: "One address-proving flow, shared by every app on the platform and by outside systems",
  },
  "emailverify.view.clients_title": { mn: "API клиентүүд", en: "API clients" },
  "emailverify.view.recent_title": { mn: "Сүүлийн баталгаажуулалтууд", en: "Recent verifications" },
  "emailverify.view.create_title": { mn: "Шинэ клиент үүсгэх", en: "New API client" },
  "emailverify.view.usage_title": { mn: "Хэрхэн ашиглах вэ", en: "How to call it" },
  "emailverify.view.test_title": { mn: "Туршилтын илгээлт", en: "Test send" },

  "emailverify.stat.total": { mn: "Нийт", en: "Total" },
  "emailverify.stat.verified": { mn: "Баталгаажсан", en: "Verified" },
  "emailverify.stat.pending": { mn: "Хүлээгдэж буй", en: "Pending" },
  "emailverify.stat.expired": { mn: "Хугацаа дууссан", en: "Expired" },
  "emailverify.stat.last_24h": { mn: "Сүүлийн 24 цаг", en: "Last 24 hours" },
  "emailverify.stat.verified_pct": { mn: "Баталгаажсан хувь", en: "Verified rate" },

  "emailverify.field.client_name": { mn: "Клиентийн нэр", en: "Client name" },
  "emailverify.field.client_name_placeholder": { mn: "жишээ: Мобайл апп", en: "e.g. Mobile app" },
  "emailverify.field.hourly_limit": { mn: "Цагийн хязгаар", en: "Hourly limit" },
  "emailverify.field.allowed_hosts": { mn: "Зөвшөөрөгдсөн буцах хаягууд", en: "Allowed redirect hosts" },
  "emailverify.field.allowed_hosts_placeholder": { mn: "theirapp.com, portal.theirapp.com", en: "theirapp.com, portal.theirapp.com" },
  "emailverify.field.source": { mn: "Илгээсэн", en: "Sent by" },
  "emailverify.field.purpose": { mn: "Зорилго", en: "Purpose" },
  "emailverify.field.last_used": { mn: "Сүүлд ашигласан", en: "Last used" },
  "emailverify.field.redirect_url": { mn: "Буцах хаяг", en: "Redirect URL" },

  "emailverify.state.pending": { mn: "Хүлээгдэж буй", en: "Pending" },
  "emailverify.state.verified": { mn: "Баталгаажсан", en: "Verified" },
  "emailverify.state.expired": { mn: "Хугацаа дууссан", en: "Expired" },

  "emailverify.action.create_client": { mn: "Клиент нэмэх", en: "Add client" },
  "emailverify.action.issue": { mn: "Түлхүүр олгох", en: "Issue key" },
  "emailverify.action.enable": { mn: "Идэвхжүүлэх", en: "Enable" },
  "emailverify.action.disable": { mn: "Идэвхгүй болгох", en: "Disable" },
  "emailverify.action.copy": { mn: "Хуулах", en: "Copy" },
  "emailverify.action.send_test": { mn: "Туршилт илгээх", en: "Send test" },

  "emailverify.message.loading": { mn: "Ачаалж байна...", en: "Loading…" },
  "emailverify.message.no_clients": {
    mn: "Одоогоор клиент үүсгээгүй байна. Гадаад систем бүрт тусдаа түлхүүр олгоно уу.",
    en: "No clients yet. Issue a separate key to each system that calls this.",
  },
  "emailverify.message.no_verifications": {
    mn: "Одоогоор баталгаажуулалт байхгүй байна.",
    en: "Nothing has been sent yet.",
  },
  // Shown once, above the key. The sentence is the whole reason the box exists.
  "emailverify.message.secret_once": {
    mn: "Энэ түлхүүр дахин харагдахгүй. Одоо хуулж, аюулгүй газар хадгална уу.",
    en: "This key is shown once and never again. Copy it now and store it somewhere safe.",
  },
  "emailverify.message.copied": { mn: "Хуулагдлаа", en: "Copied" },
  "emailverify.message.confirm_delete": {
    mn: "{name} клиентийг устгах уу? Түлхүүр нь тэр дороо хүчингүй болно.",
    en: "Delete the client {name}? Its key stops working immediately.",
  },
  "emailverify.message.disabled_note": {
    mn: "Идэвхгүй болгосон клиентийн түлхүүр дараагийн хүсэлтээс эхлэн 401 хариу авна.",
    en: "A disabled client's key is refused from its very next request.",
  },
  "emailverify.message.mail_not_configured": {
    mn: "SMTP тохируулаагүй байна — баталгаажуулах захидал илгээгдэхгүй, зөвхөн логт бичигдэнэ.",
    en: "SMTP is not configured, so verification mail is written to the log instead of being sent.",
  },
  "emailverify.message.usage": {
    mn: "Клиент бүрт өөрийн түлхүүр олгоно. Хэрэглэгч захидал дахь холбоос дээр дарахад хаяг нь баталгаажаад redirect_url руу шилжинэ. Холбоос нэг л удаа ажиллана.",
    en: "Give every caller its own key. When the recipient follows the link in the mail, the address is proven and they are sent on to redirect_url. Each link works exactly once.",
  },
  "emailverify.message.in_app_usage": {
    mn: "Платформ доторх апп модулиуд үүнийг Go дуудлагаар шууд ашиглана: emailverify.Service.Send.",
    en: "App modules inside the platform call the same service in process, through emailverify.Service.Send.",
  },
  "emailverify.message.test_sent": { mn: "{email} рүү баталгаажуулах захидал илгээлээ.", en: "A verification mail is on its way to {email}." },
  "emailverify.message.load_failed": { mn: "Ачаалж чадсангүй", en: "Could not load this screen" },
  "emailverify.message.create_failed": { mn: "Клиент үүсгэж чадсангүй", en: "Could not create the client" },
  "emailverify.message.update_failed": { mn: "Клиентийг өөрчилж чадсангүй", en: "Could not update the client" },
  "emailverify.message.delete_failed": { mn: "Клиентийг устгаж чадсангүй", en: "Could not delete the client" },
  "emailverify.message.send_failed": { mn: "Захидал илгээж чадсангүй", en: "Could not send the verification" },
  "emailverify.message.hosts_hint": {
    mn: "Хоосон бол дурын HTTPS хаяг руу буцаана. Жагсаалт бичвэл зөвхөн тэдгээр хаяг зөвшөөрөгдөнө.",
    en: "Empty allows any HTTPS destination. List hosts to allow only those.",
  },
  "emailverify.message.limit_hint": {
    mn: "Нэг цагт илгээх дээд хэмжээ. Хэтэрсэн үед 429 буцаана.",
    en: "How many links this client may send in an hour. Over that, it is answered with 429.",
  },
} as const;
