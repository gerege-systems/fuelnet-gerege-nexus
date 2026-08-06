/**
 * access — Tenant access control — roles, rights and who holds them.
 */
export const access = {
  "access.view.eyebrow": { mn: "RBAC · Байгууллагын аюулгүй байдал", en: "RBAC · Tenant security" },
  "access.view.title": { mn: "Эрхийн тохиргоо", en: "Access rights" },
  "access.view.subtitle": { mn: "Role бүрийн нэмэгдэх эрх болон хэрэглэгчийн role оноолтыг удирдана.", en: "Manage the additive rights of each role and who holds them." },
  "access.view.tab_roles": { mn: "Role ба эрх", en: "Roles and rights" },
  "access.view.tab_members": { mn: "Хэрэглэгчид", en: "Members" },
  "access.view.create_role": { mn: "Шинэ role", en: "New role" },
  "access.view.members_title": { mn: "Хэрэглэгчийн role оноолт", en: "Role assignment" },
  "access.view.members_hint": { mn: "Олон role-ийн эрх нийлж үйлчилнэ. Өөрчлөлт шууд хадгалагдана.", en: "Rights from several roles add up. Changes save immediately." },

  "access.field.code_placeholder": { mn: "sales_manager", en: "sales_manager" },
  "access.field.name_placeholder": { mn: "Борлуулалтын менежер", en: "Sales manager" },
  "access.field.description_placeholder": { mn: "Тайлбар", en: "Description" },

  "access.action.create_role": { mn: "Role нэмэх", en: "Add role" },

  "access.message.loading": { mn: "Эрхийн тохиргоо ачаалж байна…", en: "Loading access settings…" },
  "access.message.role_summary": { mn: "{count} эрх", en: "{count} rights" },
  "access.message.no_description": { mn: "Тайлбаргүй", en: "No description" },
  "access.message.admin_note": { mn: "Administrator role бүх эрхийг автоматаар эзэмшинэ.", en: "The administrator role always holds every right." },
  "access.message.confirm_delete": { mn: "{name} role-ийг устгах уу?", en: "Delete the {name} role?" },
  "access.message.saved": { mn: "Эрхүүд хадгалагдлаа", en: "Rights saved" },
  "access.message.role_created": { mn: "Role үүслээ", en: "Role created" },
  "access.message.role_deleted": { mn: "Role устлаа", en: "Role deleted" },
  "access.message.member_updated": { mn: "Хэрэглэгчийн role шинэчлэгдлээ", en: "Member roles updated" },
  "access.message.error_load": { mn: "Эрхийн мэдээлэл ачаалж чадсангүй", en: "Could not load access settings" },
  "access.message.error_save": { mn: "Хадгалж чадсангүй", en: "Could not save" },
  "access.message.error_create": { mn: "Role үүсгэж чадсангүй", en: "Could not create the role" },
  "access.message.error_delete": { mn: "Устгаж чадсангүй", en: "Could not delete" },
  "access.message.error_assign": { mn: "Role оноож чадсангүй", en: "Could not assign the role" },
} as const;
