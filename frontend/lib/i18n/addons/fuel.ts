/**
 * fuel — the fuel distribution network: the stations an organisation operates,
 * and (from Ү2) the citizen-facing map, entitlement and voucher.
 *
 * Mongolian and English only at this commit. The other five fall back to
 * English, which is the documented behaviour for a term nobody has translated
 * yet; they arrive with the screens that need them rather than as empty
 * columns nobody can review. The *menu* labels are a different matter and are
 * complete in all seven — see internal/apps/fuel/fuel.go.
 */
export const fuel = {
  "fuel.view.title": { mn: "Шатахууны сүлжээ", en: "Fuel network" },
  "fuel.view.subtitle": {
    mn: "Энэ байгууллагын ШТС, нөөц, дараалал",
    en: "The stations this organisation operates, their stock and their queues",
  },
  "fuel.view.empty_title": { mn: "Бүртгэлтэй ШТС алга", en: "No stations yet" },
  "fuel.view.empty_body": {
    mn: "ШТС-ын бүртгэл дараагийн үе шатанд нэмэгдэнэ.",
    en: "The station register arrives in the next phase.",
  },
  "fuel.view.count": { mn: "Бүртгэлтэй ШТС", en: "Stations registered" },

  "fuel.map.title": { mn: "Шатахуун хаана байна", en: "Where to buy fuel" },
  "fuel.map.no_prices": { mn: "Үнэ мэдэгдээгүй", en: "No price reported" },

  "fuel.sheet.stock_title": { mn: "Шатахууны үлдэгдэл", en: "What is in the tanks" },
  "fuel.sheet.today_remaining": { mn: "Өнөөдрийн үлдэгдэл эрх", en: "Left of today's entitlement" },
  "fuel.sheet.take_voucher": { mn: "Ваучер авах", en: "Take a voucher" },
  "fuel.sheet.voucher_ready": { mn: "Ваучер бэлэн", en: "Your voucher" },
  "fuel.sheet.any_pump": {
    mn: "Дурын шатахуун түгээх станц дээр хүчинтэй",
    en: "Good at any filling station",
  },
  "fuel.sheet.valid_until": { mn: "Хүчинтэй:", en: "Valid until" },
  "fuel.sheet.sign_in_to_take": { mn: "eID-ээр нэвтэрч ваучер авах", en: "Sign in with eID to take one" },
  "fuel.sheet.spent": {
    mn: "Өнөөдрийн эрхээ бүрэн ашигласан байна. Маргааш шинэчлэгдэнэ.",
    en: "Today's entitlement is spent. It resets tomorrow.",
  },
  "fuel.rail.sign_in": { mn: "eID-ээр нэвтрэх", en: "Sign in with eID" },
  "fuel.rail.why_sign_in": {
    mn: "Өдрийн 50,000₮-ийн эрхээ авахын тулд eID-ээр нэвтэрнэ үү.",
    en: "Sign in with eID to draw your 50,000₮ daily entitlement.",
  },
  "fuel.rail.eid_note": {
    mn: "eID Mongolia апп эсвэл регистрийн дугаараар.",
    en: "With the eID Mongolia app, or your registry number.",
  },
  "fuel.rail.my_vouchers": { mn: "Миний ваучер", en: "My vouchers" },
  "fuel.rail.no_vouchers": {
    mn: "Идэвхтэй ваучер алга. Газрын зураг дээрх ШТС дээр дарж авна уу.",
    en: "No active voucher. Tap a station on the map to take one.",
  },
  "fuel.rail.how_to": {
    mn: "Ваучерыг дурын шатахуун түгээх станц дээр QR-аар уншуулна.",
    en: "Show the QR at any filling station.",
  },
  "fuel.rail.refresh": { mn: "Шинэчлэх", en: "Refresh" },
  "fuel.rail.tap_to_close": { mn: "Хаахын тулд дарна уу", en: "Tap to close" },
} as const;
