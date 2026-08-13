/**
 * cp — the operator console.
 *
 * Its own file rather than lines in `web`, because none of these terms belong
 * to the product tenants use: nobody outside the operating team ever sees a
 * string from here, and mixing them in would put the console's vocabulary into
 * every translator's queue for the tenant application.
 */
export const cp = {
  "cp.view.title": { mn: "Удирдлагын самбар", en: "Control Plane" },
  "cp.view.subtitle": {
    mn: "Платформын операторын консол. Байгууллагууд, тэдгээрийн төлөв, операторын үйлдлийн бүртгэл.",
    en: "The platform operator's console: organisations, their state, and what operators have done.",
  },

  "cp.login.title": { mn: "Операторын нэвтрэлт", en: "Operator sign-in" },
  "cp.login.hint": {
    mn: "И-мэйл, нууц үг, баталгаажуулагчийн код гурвуулаа шаардлагатай.",
    en: "Your e-mail, password and authenticator code are all required.",
  },
  "cp.login.failed": {
    mn: "И-мэйл, нууц үг эсвэл код таарсангүй.",
    en: "The e-mail address, password or code was not right.",
  },

  "cp.field.email": { mn: "И-мэйл", en: "E-mail" },
  "cp.field.password": { mn: "Нууц үг", en: "Password" },
  "cp.field.code": { mn: "Баталгаажуулагчийн код", en: "Authenticator code" },
  "cp.field.search": { mn: "Нэр, slug, регистрээр хайх", en: "Search by name, slug or registration number" },
  "cp.field.organisation": { mn: "Байгууллага", en: "Organisation" },
  "cp.field.slug": { mn: "Slug", en: "Slug" },
  "cp.field.registration": { mn: "Регистр", en: "Registration" },
  "cp.field.users": { mn: "Хэрэглэгч", en: "People" },
  "cp.field.apps": { mn: "Апп", en: "Apps" },
  "cp.field.created": { mn: "Үүссэн", en: "Created" },
  "cp.field.last_activity": { mn: "Сүүлийн идэвх", en: "Last activity" },
  "cp.field.legal_name": { mn: "Албан ёсны нэр", en: "Legal name" },
  "cp.field.tax_number": { mn: "Татварын дугаар", en: "Tax number" },
  "cp.field.version": { mn: "Хувилбар", en: "Version" },
  "cp.field.status": { mn: "Төлөв", en: "Status" },
  "cp.field.installed": { mn: "Суусан", en: "Installed" },
  "cp.field.roles": { mn: "Үүрэг", en: "Roles" },
  "cp.field.action": { mn: "Үйлдэл", en: "Action" },
  "cp.field.operator": { mn: "Оператор", en: "Operator" },
  "cp.field.reason": { mn: "Шалтгаан", en: "Reason" },
  "cp.field.when": { mn: "Хэзээ", en: "When" },
  "cp.field.resource": { mn: "Обьект", en: "Resource" },

  "cp.action.sign_in": { mn: "Нэвтрэх", en: "Sign in" },
  "cp.action.sign_out": { mn: "Гарах", en: "Sign out" },
  "cp.action.back": { mn: "Жагсаалт руу", en: "Back to the list" },
  "cp.action.search": { mn: "Хайх", en: "Search" },

  "cp.section.tenants": { mn: "Байгууллагууд", en: "Organisations" },
  "cp.section.apps": { mn: "Суусан аппууд", en: "Installed apps" },
  "cp.section.members": { mn: "Хэрэглэгчид", en: "People" },
  "cp.section.activity": { mn: "Сүүлийн идэвх", en: "Recent activity" },
  "cp.section.operator_actions": { mn: "Операторын үйлдэл", en: "Operator actions" },
  "cp.section.audit": { mn: "Операторын бүртгэл", en: "Operator audit" },

  "cp.role.superadmin": { mn: "Ерөнхий админ", en: "Superadmin" },
  "cp.role.operator": { mn: "Оператор", en: "Operator" },
  "cp.role.support": { mn: "Дэмжлэг", en: "Support" },
  "cp.role.auditor": { mn: "Аудитор", en: "Auditor" },

  "cp.message.read_only": {
    mn: "Энэ үе шатанд консол зөвхөн уншина. Түдгэлзүүлэх, устгах, дэмжлэгийн үйлдлүүд дараагийн үе шатанд нэмэгдэнэ.",
    en: "The console is read-only at this stage. Suspension, deletion and support actions arrive in the next phase.",
  },
  "cp.message.no_tenants": { mn: "Байгууллага олдсонгүй.", en: "No organisations found." },
  "cp.message.no_activity": { mn: "Бүртгэл алга.", en: "Nothing recorded." },
  "cp.message.load_failed": {
    mn: "Мэдээллийг уншиж чадсангүй.",
    en: "That could not be loaded.",
  },
  "cp.message.never": { mn: "Хэзээ ч", en: "Never" },
};
