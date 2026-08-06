/**
 * auth — Sign-in, including the national E-ID and DAN identity channels.
 */
export const auth = {
  "auth.view.subtitle": { mn: "Модуль бүтэцтэй байгууллагын платформ", en: "Modular Enterprise Application Platform" },
  "auth.view.eid_title": { mn: "E-ID Mongolia танилт (eidmongolia.mn)", en: "E-ID Mongolia Identity (eidmongolia.mn)" },
  "auth.view.dan_title": { mn: "Үндэсний ДАН Танилт Нэвтрэх Суваг", en: "National DAN identity channel" },

  "auth.field.email": { mn: "И-мэйл хаяг", en: "Email Address" },
  "auth.field.password": { mn: "Нууц үг", en: "Password" },
  "auth.field.reg_number": { mn: "Иргэний Регистрийн Дугаар *", en: "Registration number *" },
  "auth.field.otp": { mn: "Баталгаажуулах Код (OTP / Pin)", en: "Verification code (OTP / PIN)" },
  "auth.field.identity_channel": { mn: "Танилт Нэвтрэх Суваг", en: "Identity channel" },

  "auth.method.eid": { mn: "E-ID Mongolia (Танилт Нэвтрэлт)", en: "E-ID Mongolia (National Identity)" },
  "auth.method.pki": { mn: "Тоон Гарын Үсэг (PKI)", en: "PKI digital signature" },
  "auth.method.otp": { mn: "Нэг удаагийн код (Mobile OTP)", en: "Mobile OTP" },
  "auth.method.bank": { mn: "Банкны системээр нэвтрэх", en: "Bank SSO" },
  "auth.method.biometric": { mn: "Царай танилт (Biometric Face)", en: "Biometric face verification" },

  "auth.action.sign_in": { mn: "Платформ руу нэвтрэх", en: "Sign In to Platform" },
  "auth.action.verify_sign_in": { mn: "Баталгаажуулж Нэвтрэх", en: "Verify and sign in" },

  "auth.label.or": { mn: "ЭСВЭЛ", en: "OR" },

  "auth.message.signing_in": { mn: "Нэвтэрч байна...", en: "Signing in..." },
  "auth.message.demo_credentials": { mn: "Туршилтын эрх:", en: "Demo credentials:" },
} as const;
