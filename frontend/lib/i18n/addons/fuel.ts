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

  // The company's own half of the chain: what crossed the border, and the
  // bases it was unloaded into.
  "fuel.tab.stations": { mn: "ШТС", en: "Stations" },
  "fuel.tab.shipments": { mn: "Гаалийн мэдүүлэг", en: "Customs" },
  "fuel.tab.depots": { mn: "Нефть бааз", en: "Depots" },

  "fuel.ship.title": { mn: "Гаалийн мэдүүлэг", en: "Customs declarations" },
  "fuel.ship.subtitle": {
    mn: "Хилээр орж ирсэн ачаа. Гаалиар цэвэрлэгдэхэд партийн дугаар үүсэж, тэр дугаар хошуу хүртэл дагана.",
    en: "What crossed the border. Clearing a consignment mints the batch number that travels with it to the pump.",
  },
  "fuel.ship.empty": {
    mn: "Бүртгэсэн мэдүүлэг алга.",
    en: "No declarations recorded yet.",
  },
  "fuel.ship.declare": { mn: "Мэдүүлэг бүртгэх", en: "Declare a consignment" },
  "fuel.ship.declaration_no": { mn: "Мэдүүлгийн дугаар", en: "Declaration number" },
  "fuel.ship.border_port": { mn: "Боомт", en: "Border port" },
  "fuel.ship.origin": { mn: "Гарал үүсэл", en: "Origin" },
  "fuel.ship.exporter": { mn: "Экспортлогч", en: "Exporter" },
  "fuel.ship.fuel_type": { mn: "Түлшний төрөл", en: "Grade" },
  "fuel.ship.liters": { mn: "Мэдүүлсэн литр", en: "Declared litres" },
  "fuel.ship.tons": { mn: "Мэдүүлсэн тонн", en: "Declared tonnes" },
  "fuel.ship.tons_note": {
    mn: "Литр нь тооцооны үндэс. Тонн нь мэдүүлэгт бичигдсэн зүйл — нягт температураас хамаарч хөвдөг тул хоёр нь яг таарахгүй, тулгагдахгүй.",
    en: "Litres are what everything downstream works in. Tonnes are what the document says; density moves with temperature, so the two never agree exactly and are not reconciled.",
  },
  "fuel.ship.wagons": { mn: "Вагон", en: "Wagons" },
  "fuel.ship.convoy": { mn: "Цувааны дугаар", en: "Convoy" },
  "fuel.ship.batch": { mn: "Партийн дугаар", en: "Batch" },
  "fuel.ship.clear": { mn: "Гаалиар цэвэрлэх", en: "Clear customs" },
  "fuel.ship.inspect": { mn: "Шалгалтад авах", en: "Send to inspection" },
  "fuel.ship.cert_no": { mn: "Чанарын гэрчилгээ", en: "Quality certificate" },
  "fuel.ship.octane": { mn: "Октан", en: "Octane" },
  "fuel.ship.sulfur": { mn: "Хүхэр (ppm)", en: "Sulfur (ppm)" },
  "fuel.ship.status.border_arrived": { mn: "Хилд ирсэн", en: "At the border" },
  "fuel.ship.status.inspecting": { mn: "Шалгалтад", en: "Under inspection" },
  "fuel.ship.status.cleared": { mn: "Цэвэрлэгдсэн", en: "Cleared" },
  "fuel.ship.status.in_transit": { mn: "Замд", en: "In transit" },
  "fuel.ship.status.at_depot": { mn: "Баазад", en: "At a depot" },

  "fuel.depot.title": { mn: "Нефть бааз", en: "Depots" },
  "fuel.depot.subtitle": {
    mn: "Сав бүрийн үлдэгдэл. Түвшинг гараар тавьж болохгүй — орсон ба гарсан баримтын нийлбэр.",
    en: "What is in each tank. The level cannot be typed in: it is the sum of what went in and what came out.",
  },
  "fuel.depot.empty": { mn: "Бүртгэсэн бааз алга.", en: "No depots registered yet." },
  "fuel.depot.add": { mn: "Бааз бүртгэх", en: "Register a depot" },
  "fuel.depot.name": { mn: "Баазын нэр", en: "Depot name" },
  "fuel.depot.aimag": { mn: "Аймаг / хот", en: "Province or city" },
  "fuel.depot.rail": { mn: "Төмөр замын татах замтай", en: "Has a rail siding" },
  "fuel.depot.rail_code": { mn: "Өртөөний код", en: "Rail station code" },
  "fuel.depot.no_tanks": { mn: "Сав бүртгэгдээгүй", en: "No tanks registered" },
  "fuel.depot.add_tank": { mn: "Сав нэмэх", en: "Add a tank" },
  "fuel.depot.tank_no": { mn: "Савны дугаар", en: "Tank number" },
  "fuel.depot.capacity": { mn: "Багтаамж (литр)", en: "Capacity (litres)" },
  "fuel.depot.receive": { mn: "Ачаа хүлээж авах", en: "Receive a consignment" },
  "fuel.depot.receive_into": { mn: "Аль сав руу", en: "Into which tank" },
  "fuel.depot.which_shipment": { mn: "Аль мэдүүлэг", en: "Which consignment" },
  "fuel.depot.no_cleared": {
    mn: "Гаалиар цэвэрлэгдсэн, хүлээж аваагүй ачаа алга.",
    en: "No cleared consignment is waiting to be unloaded.",
  },
  "fuel.depot.measured": { mn: "Хэмжсэн литр", en: "Measured litres" },
  "fuel.depot.manifest": { mn: "Дагалдах бичгийн литр", en: "Manifest litres" },
  "fuel.depot.manifest_note": {
    mn: "Хоёр нь зөрвөл зөрүү нь үлдэнэ. Систем зөрүүг тайлбарлахгүй — алдагдал, хулгай, хэмжлийн алдааны аль нь болохыг хэлж чадахгүй, гэхдээ зөрүү байгааг хэлнэ.",
    en: "A gap between them survives as a gap. Nothing here can tell loss from theft from a badly calibrated gauge — but a system storing only one figure could not say a gap existed.",
  },
  "fuel.depot.of_capacity": { mn: "багтаамжийн", en: "of capacity" },
  "fuel.depot.saving": { mn: "Хадгалж байна…", en: "Saving…" },
  "fuel.common.cancel": { mn: "Болих", en: "Cancel" },
  "fuel.common.save": { mn: "Хадгалах", en: "Save" },
} as const;
