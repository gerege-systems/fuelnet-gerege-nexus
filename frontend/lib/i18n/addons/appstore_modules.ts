/**
 * appstore_modules — the three App Store screens, on the instance that is one.
 *
 * Publisher Studio and the review queue replace what used to be a separate
 * product at developer.gerege.mn. The registry screen is the two questions that
 * are about the registry itself rather than about any app in it.
 */
export const appstore_modules = {
  // --- Publisher Studio ---
  "mod.publisher.title": { mn: "Нийтлэгчийн студи", en: "Publisher Studio" },
  "mod.publisher.subtitle": {
    mn: "Байгууллагынхаа аппуудыг бүртгэж, хувилбарыг хяналтад илгээнэ.",
    en: "Register your organisation's apps and submit versions for review.",
  },
  "mod.publisher.profile": { mn: "Нийтлэгчийн мэдээлэл", en: "Publishing profile" },
  "mod.publisher.create_profile": { mn: "Профайл үүсгэх", en: "Create a profile" },
  "mod.publisher.edit_profile": { mn: "Профайл засах", en: "Edit the profile" },
  "mod.publisher.no_profile": {
    mn: "Энэ байгууллага хараахан нийтлэгчээр бүртгүүлээгүй байна.",
    en: "This organisation has not registered as a publisher yet.",
  },
  "mod.publisher.handle": { mn: "Богино нэр", en: "Handle" },
  "mod.publisher.handle_note": {
    mn: "Дэлгүүрийн хаяганд энэ нэр орно — хэн нэгэн холбоос тавьсны дараа солих нь тэдгээрийг эвдэнэ.",
    en: "This appears in storefront URLs — changing it after anybody has linked to you breaks those links.",
  },
  "mod.publisher.contact": { mn: "Холбоо барих и-мэйл", en: "Contact e-mail" },
  "mod.publisher.verified": { mn: "Баталгаажсан", en: "Verified" },
  "mod.publisher.unverified": { mn: "Баталгаажаагүй", en: "Not verified" },
  "mod.publisher.no_apps": { mn: "Одоогоор апп бүртгээгүй байна.", en: "No apps registered yet." },
  "mod.publisher.no_versions": { mn: "Хувилбар илгээгээгүй", en: "No versions submitted" },
  "mod.publisher.submit_version": { mn: "Хувилбар илгээх", en: "Submit a version" },
  "mod.publisher.submit_note": {
    mn: "Manifest-ийг хяналтын дараалалд илгээнэ. Нийтлэх шийдвэрийг өөр хүн гаргана.",
    en: "The manifest goes to the review queue. Publishing is somebody else's decision.",
  },
  "mod.publisher.manifest_invalid": { mn: "Manifest нь зөв JSON биш байна.", en: "The manifest is not valid JSON." },
  "mod.publisher.submitted": {
    mn: "{version} хувилбар хяналтад илгээгдлээ.",
    en: "Version {version} is in the review queue.",
  },
  "mod.publisher.status.draft": { mn: "ноорог", en: "draft" },
  "mod.publisher.status.in_review": { mn: "хянагдаж байна", en: "in review" },
  "mod.publisher.status.published": { mn: "нийтлэгдсэн", en: "published" },
  "mod.publisher.status.rejected": { mn: "татгалзсан", en: "rejected" },
  "mod.publisher.status.yanked": { mn: "буцаасан", en: "withdrawn" },

  // --- Review queue ---
  "mod.store_review.title": { mn: "Хяналтын дараалал", en: "Review queue" },
  "mod.store_review.subtitle": {
    mn: "Илгээсэн хувилбарууд — ирсэн дарааллаараа.",
    en: "Submitted versions, in the order they arrived.",
  },
  "mod.store_review.empty": { mn: "Хүлээгдэж буй зүйл алга.", en: "Nothing is waiting." },
  "mod.store_review.publish": { mn: "Нийтлэх", en: "Publish" },
  "mod.store_review.reject": { mn: "Татгалзах", en: "Reject" },
  "mod.store_review.note": { mn: "Тайлбар", en: "Note" },
  "mod.store_review.note_required": {
    mn: "Татгалзахад шалтгаанаа бичнэ — нийтлэгч засах боломжтой байх ёстой.",
    en: "A rejection needs a reason its publisher can act on.",
  },
  "mod.store_review.submitted_by": { mn: "Илгээсэн", en: "Submitted by" },
  "mod.store_review.requires_platform": { mn: "Шаардах платформ:", en: "Requires platform" },
  "mod.store_review.publishers": { mn: "Нийтлэгчид", en: "Publishers" },
  "mod.store_review.no_publishers": { mn: "Нийтлэгч бүртгэгдээгүй байна.", en: "No publishers registered." },
  "mod.store_review.verify": { mn: "Баталгаажуулах", en: "Verify" },
  "mod.store_review.verified": { mn: "Баталгаажсан", en: "Verified" },

  // --- Registry ---
  "mod.appstore_registry.title": { mn: "Аппын бүртгэл", en: "App Registry" },
  "mod.appstore_registry.subtitle": {
    mn: "Энэ инстанс ямар каталог нийтэлж, юугаар гарын үсэг зурж байна.",
    en: "What this instance publishes, and what it signs with.",
  },
  "mod.appstore_registry.revision": { mn: "Хувилбарын дугаар", en: "Revision" },
  "mod.appstore_registry.revision_note": {
    mn: "Каталог өөрчлөгдөх бүрд нэмэгдэнэ. Хуучин дугаараар угсарсан хуулбарыг дахин угсарна.",
    en: "It moves whenever the catalogue changes; a snapshot built under an older one is rebuilt.",
  },
  "mod.appstore_registry.key": { mn: "Гарын үсгийн түлхүүр", en: "Signing key" },
  "mod.appstore_registry.key_note": {
    mn: "Түлхүүр солиход дугаар хөдөлдөггүй тул хадгалагдсан каталогууд хүчинтэй мэт харагдсаар байдаг. Дахин угсрах нь үүнээс гарах зам.",
    en: "Rotating a key moves no revision, so cached catalogues keep looking valid. Rebuilding is the way out.",
  },
  "mod.appstore_registry.rebuild": { mn: "Каталогийг дахин угсрах", en: "Rebuild the catalogue" },
  "mod.appstore_registry.rebuilt": {
    mn: "{count} хуулбар устгагдлаа — дараагийн хүсэлтэд дахин угсарна.",
    en: "{count} snapshots discarded; the next request rebuilds them.",
  },
} as const;
