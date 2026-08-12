/**
 * deviceLine — төхөөрөмжийн domain шугам.
 *
 * Backend цор ганц. Гэхдээ төхөөрөмж бүр өөрийн host-оор ханддаг:
 *
 *   nexus.gerege.mn          → хөтөч / PWA — web app өөрөө бүрэн апп
 *   mac.nexus.gerege.mn      → macOS native хүрээн доторх ажлын муж
 *   win.nexus.gerege.mn      → Windows
 *   ios.nexus.gerege.mn      → iOS / iPadOS
 *   android.nexus.gerege.mn  → Android
 *   kiosk.nexus.gerege.mn    → Kiosk
 *   pos.nexus.gerege.mn      → POS
 *
 * Шугам бүр өөрийн host дээрээ `/api/v1`-ээ мөн үйлчилдэг (nginx нь бүгдийг
 * НЭГ ижил API upstream руу дамжуулна), тиймээс webview доторх дуудлага үргэлж
 * same-origin — session cookie нь `SameSite=Strict` хэвээр ажиллаж, CORS
 * preflight огт үүсэхгүй.
 *
 * Эх сурвалж: `native-apps/shared/device_lines.json`. Тэр файл нь native
 * талын, энэ нь web талын хуулбар — шугам нэмэхэд ХОЁУЛАНГ нь өөрчилнө.
 */

import type { ShellFormFactor, ShellPlatform } from "@/lib/shell";

export interface DeviceLine {
  platform: ShellPlatform;
  formFactor: ShellFormFactor;
  /**
   * Native бүрхүүл нэвтэрсний дараа нээх эхний зам — шугамын үндэс.
   *
   * Тэнд тухайн шугамын өөрийн нүүр дэлгэц рендерлэгдэнэ: `proxy.ts` нь
   * `/`-ыг `/line/<platform>` рүү rewrite хийнэ (redirect биш — хаяг нь `/`
   * хэвээр үлдэнэ). Тиймээс шугам бүр өөрийн нүүрээ өөрөө хөгжүүлнэ.
   */
  startRoute: string;
}

/**
 * Host-ын хамгийн зүүн шошго → шугам.
 *
 * Домэйн бүтнээр нь биш зөвхөн эхний шошгоор тааруулж байгаа нь санаатай:
 * `mac.nexus.gerege.mn`, `mac.nexus.staging.gerege.mn`, `mac.localhost` гурвуул
 * ижил шугам. Ингэснээр staging/preview орчинд энэ файлыг хөндөх шаардлагагүй.
 */
const LINES_BY_LABEL: Record<string, DeviceLine> = {
  mac: { platform: "macos", formFactor: "desktop", startRoute: "/" },
  win: { platform: "windows", formFactor: "desktop", startRoute: "/" },
  ios: { platform: "ios", formFactor: "mobile", startRoute: "/" },
  android: { platform: "android", formFactor: "mobile", startRoute: "/" },
  kiosk: { platform: "kiosk", formFactor: "kiosk", startRoute: "/" },
  pos: { platform: "pos", formFactor: "pos", startRoute: "/" },
};

/** Шугамын нүүр дэлгэцийн бодит зам. `proxy.ts` `/`-ыг үүн рүү rewrite хийнэ. */
export function lineHomePath(line: DeviceLine): string {
  return `/line/${line.platform}`;
}

/** Middleware-ээс доош дамжуулах толгой. Хөтчийн шугам дээр огт тавигдахгүй. */
export const DEVICE_LINE_HEADER = "x-gerege-device-line";

/**
 * Host-оос шугамыг тодорхойлно. Танихгүй бол `null` — өөрөөр хэлбэл хөтчийн
 * шугам, тэнд юу ч өөрчлөгдөхгүй.
 *
 * `Host` толгой нь порт агуулж болно (`mac.localhost:3000`), мөн IPv6 хаяг нь
 * хаалтанд байдаг — хоёуланг нь тайрч байж шошгыг уншина.
 */
export function deviceLineFromHost(host: string | null | undefined): DeviceLine | null {
  if (!host) return null;
  const hostname = host.trim().toLowerCase().replace(/^\[.*\]/, "").replace(/:\d+$/, "");
  const label = hostname.split(".")[0];
  return LINES_BY_LABEL[label] ?? null;
}

/** Хөтөч дээр ажиллаж байгаа шугам. SSR-д үргэлж `null`. */
export function currentDeviceLine(): DeviceLine | null {
  if (typeof window === "undefined") return null;
  return deviceLineFromHost(window.location.host);
}
