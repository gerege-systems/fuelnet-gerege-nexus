/**
 * esign — PDF digital signing through the Gerege eSign HSM service.
 */
export const esign = {
  "esign.view.title": { mn: "PDF тоон гарын үсэг", en: "PDF e-signature" },
  "esign.view.subtitle": {
    mn: "PDF баримт бичгийг Gerege eSign HSM үйлчилгээгээр тоон гарын үсгээр баталгаажуулах",
    en: "Sign PDF documents with the Gerege eSign HSM service",
  },
  "esign.view.upload_title": { mn: "PDF баримт бичиг оруулах", en: "Upload a PDF document" },
  "esign.view.sign_title": { mn: "Тоон гарын үсгээр баталгаажуулах", en: "Sign with a digital signature" },
  "esign.view.sign_placement": {
    mn: "{title} — гарын үсэг сүүлийн хуудсанд ({page}-р хуудас) байрлана",
    en: "{title} — the signature is placed on the last page (page {page})",
  },
  "esign.view.step_certificate": { mn: "1. Сертификат шалгах", en: "1. Check the certificate" },
  "esign.view.step_signature": { mn: "2. Гарын үсгээ зурна уу", en: "2. Draw your signature" },

  "esign.field.document": { mn: "Баримт бичиг", en: "Document" },
  "esign.field.title": { mn: "Баримт бичгийн нэр", en: "Document title" },
  "esign.field.title_placeholder": { mn: "ж: Хамтран ажиллах гэрээ 2026", en: "e.g. Partnership agreement 2026" },
  "esign.field.file": { mn: "PDF файл (макс 15MB)", en: "PDF file (15MB max)" },
  "esign.field.pages": { mn: "Хуудас", en: "Pages" },
  "esign.field.signer": { mn: "Гарын үсэг зурсан", en: "Signed by" },
  "esign.field.phone": { mn: "Утасны дугаар", en: "Phone number" },
  "esign.field.civil_id": { mn: "Регистр / Civil ID", en: "Registration / Civil ID" },

  "esign.state.signed": { mn: "БАТАЛГААЖСАН", en: "SIGNED" },
  "esign.state.pending": { mn: "ХҮЛЭЭГДЭЖ БУЙ", en: "PENDING" },

  "esign.action.upload": { mn: "PDF оруулах", en: "Upload a PDF" },
  "esign.action.submit_upload": { mn: "Оруулах", en: "Upload" },
  "esign.action.sign": { mn: "Гарын үсэг зурах", en: "Sign" },
  "esign.action.check_certificate": { mn: "Сертификат шалгах", en: "Check the certificate" },
  "esign.action.clear": { mn: "Арилгах", en: "Clear" },
  "esign.action.download_original": { mn: "Эх хувь татах", en: "Download the original" },
  "esign.action.download_signed": { mn: "Баталгаажсан хувь татах", en: "Download the signed copy" },
  "esign.action.original_short": { mn: "Эх", en: "Original" },
  "esign.action.signed_short": { mn: "Баталгаажсан", en: "Signed" },

  "esign.message.loading": { mn: "Баримт бичгүүдийг ачаалж байна...", en: "Loading documents..." },
  "esign.message.empty": { mn: "Баримт бичиг алга. “PDF оруулах” товчоор эхлүүлнэ үү.", en: "No documents yet. Start with “Upload a PDF”." },
  "esign.message.uploading": { mn: "Оруулж байна...", en: "Uploading..." },
  "esign.message.checking": { mn: "Шалгаж байна...", en: "Checking..." },
  "esign.message.signing": { mn: "Баталгаажуулж байна...", en: "Signing..." },
  "esign.message.certificate_valid": { mn: "Сертификат хүчинтэй: {name}", en: "Certificate valid: {name}" },
  "esign.message.error_download": { mn: "Татаж чадсангүй: {error}", en: "Could not download: {error}" },
  "esign.message.error_upload": { mn: "Оруулж чадсангүй: {error}", en: "Could not upload: {error}" },
  "esign.message.error_certificate": { mn: "Сертификат шалгалт амжилтгүй: {error}", en: "The certificate check failed: {error}" },
  "esign.message.error_sign": { mn: "Гарын үсэг зурж чадсангүй: {error}", en: "Could not sign: {error}" },
} as const;
