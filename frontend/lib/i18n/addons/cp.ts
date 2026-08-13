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
    mn: "Үйлдэл бүр шалтгаан шаардана, audit-д бичигдэнэ. Устгал нь хоёр дахь superadmin-ий зөвшөөрөл ба 30 хоногийн хугацаатай.",
    en: "Every action needs a reason and is recorded. Deletion needs a second superadmin and takes thirty days.",
  },
  "cp.message.no_tenants": { mn: "Байгууллага олдсонгүй.", en: "No organisations found." },
  "cp.message.no_activity": { mn: "Бүртгэл алга.", en: "Nothing recorded." },
  "cp.message.load_failed": {
    mn: "Мэдээллийг уншиж чадсангүй.",
    en: "That could not be loaded.",
  },
  "cp.message.never": { mn: "Хэзээ ч", en: "Never" },

  "cp.action.cancel": { mn: "Болих", en: "Cancel" },
  "cp.action.confirm": { mn: "Гүйцэтгэх", en: "Confirm" },
  "cp.action.suspend": { mn: "Түдгэлзүүлэх", en: "Suspend" },
  "cp.action.resume": { mn: "Сэргээх", en: "Resume" },
  "cp.action.delete": { mn: "Устгах хүсэлт", en: "Ask to delete" },
  "cp.action.cancel_deletion": { mn: "Устгалыг цуцлах", en: "Cancel the deletion" },
  "cp.action.export": { mn: "Өгөгдлийг татах", en: "Export the data" },
  "cp.action.quota": { mn: "Хязгаар", en: "Limits" },
  "cp.action.impersonate": { mn: "Дотор нь орж харах", en: "Look inside" },
  "cp.action.new_tenant": { mn: "Шинэ байгууллага", en: "New organisation" },
  "cp.action.create": { mn: "Үүсгэх", en: "Create" },
  "cp.action.approve": { mn: "Зөвшөөрөх", en: "Approve" },
  "cp.action.reject": { mn: "Татгалзах", en: "Reject" },
  "cp.action.unlock": { mn: "Түгжээ тайлах", en: "Unlock" },
  "cp.action.revoke_sessions": { mn: "Бүх session хаах", en: "End every session" },
  "cp.action.send_reset": { mn: "Нууц үг сэргээх холбоос", en: "Send a reset link" },

  "cp.field.max_users": { mn: "Хэрэглэгчийн дээд тоо", en: "Maximum people" },
  "cp.field.max_storage": { mn: "Хадгалалт (MB)", en: "Storage (MB)" },
  "cp.field.max_ai": { mn: "AI дуудлага / сар", en: "AI calls / month" },
  "cp.field.enforcement": { mn: "Горим", en: "Mode" },
  "cp.field.name": { mn: "Нэр", en: "Name" },
  "cp.field.admin_email": { mn: "Эхний админы и-мэйл", en: "First administrator's e-mail" },
  "cp.field.install_apps": { mn: "Суулгах аппууд", en: "Apps to install" },
  "cp.field.person": { mn: "Хүн", en: "Person" },
  "cp.field.state": { mn: "Төлөв", en: "State" },
  "cp.field.requested_by": { mn: "Хүссэн", en: "Asked by" },

  "cp.state.active": { mn: "Идэвхтэй", en: "Active" },
  "cp.state.suspended": { mn: "Түдгэлзсэн", en: "Suspended" },
  "cp.state.deleting": { mn: "Устгал хүлээж буй", en: "Awaiting deletion" },
  "cp.state.soft": { mn: "Зөөлөн (анхааруулна)", en: "Soft (warns)" },
  "cp.state.hard": { mn: "Хатуу (хориглоно)", en: "Hard (refuses)" },
  "cp.state.locked": { mn: "Түгжигдсэн", en: "Locked" },

  "cp.section.support": { mn: "Дэмжлэг", en: "Support" },
  "cp.section.approvals": { mn: "Хүлээгдэж буй зөвшөөрөл", en: "Waiting for approval" },
  "cp.section.impersonations": { mn: "Дотор нь орсон түүх", en: "Who has looked inside" },
  "cp.section.limits": { mn: "Хязгаарууд", en: "Limits" },
  "cp.section.actions": { mn: "Үйлдэл", en: "Actions" },

  "cp.hint.reason": {
    mn: "Энэ бичиг audit-д үлдэж, зарим тохиолдолд тухайн байгууллагад ч харагдана.",
    en: "This is recorded in the audit trail, and for some actions the organisation sees it too.",
  },
  "cp.hint.search_people": { mn: "И-мэйл эсвэл нэрээр (3-аас дээш тэмдэгт)", en: "By e-mail or name (three characters or more)" },
  "cp.hint.not_enforced": {
    mn: "Бүртгэгдэнэ, гэхдээ хараахан хэрэгжихгүй — CP-5-ын хэрэглээний хэмжилт ирэхэд ажиллана.",
    en: "Recorded but not yet enforced — it starts working when CP-5's usage metering lands.",
  },
  "cp.message.step_up": {
    mn: "Баталгаажуулагчийн кодоо дахин оруулна уу.",
    en: "Enter your authenticator code again.",
  },
  "cp.message.deletion_requested": {
    mn: "Хоёр дахь superadmin зөвшөөрөх хүртэл юу ч болохгүй. Зөвшөөрсний дараа 30 хоногийн дотор буцаах боломжтой.",
    en: "Nothing happens until a second superadmin agrees. After that, thirty days in which it can be undone.",
  },
  "cp.message.no_people": { mn: "Хэн ч олдсонгүй.", en: "Nobody found." },
  "cp.message.no_approvals": { mn: "Хүлээгдэж буй зүйл алга.", en: "Nothing is waiting." },
};
