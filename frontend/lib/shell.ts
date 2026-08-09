"use client";

/**
 * shell — "Native Shell + Web Work Area" архитектурын web талын гэрээ.
 *
 * Native бүрхүүл (одоо Swift/AppKit, дараа Tauri) нь нэвтрэлт, толгой хэсэг,
 * цэс, хөл, төхөөрөмжийн хандалтыг өөртөө авдаг. Тэр үед web app нь өөрийн
 * chrome-оо нуугаад зөвхөн ажлын муж болж рендерлэгдэнэ. Хөтөч дээр shell
 * байхгүй тул энэ файлын бүх зүйл `null`/`false` буцаана — өөрөөр хэлбэл
 * хөтчийн горим ямар ч нөхцөлд өөрчлөгдөхгүй.
 *
 * Гэрээний бүрэн бүртгэл: `docs/SHELL_CONTRACT.md`.
 */

import { useSyncExternalStore } from "react";

export type ShellPlatform = "macos" | "windows" | "linux" | "ios" | "android" | "kiosk" | "pos";

export interface GeregeShell {
  /** Гэрээний semver. Одоо "1.0". */
  version: string;
  platform: ShellPlatform;
  /** Тухайн бүрхүүлд ҮНЭХЭЭР хэрэгжсэн чадварууд. Зарлагдаагүй method дуудвал
   *  invoke() reject хийнэ. */
  capabilities: string[];
  invoke<T>(method: string, params?: Record<string, unknown>): Promise<T>;
  /** Буцаах утга нь бүртгэлээ цуцлах функц. */
  on(event: string, handler: (payload: unknown) => void): () => void;
}

declare global {
  interface Window {
    GeregeShell?: GeregeShell;
  }
}

/** Гэрээгээр тогтсон method нэрс. Хаана ч мөрөн дотор бичихгүй — нэр солигдвол
 *  энэ нэг газраас л өөрчлөгдөнө. */
export const SHELL_METHODS = {
  /** Session дуусахад бүрхүүлийн нэвтрэлтийн урсгалыг эхлүүлнэ. */
  AUTH_RE_LOGIN: "auth.reLogin",
  NOTIFY_SHOW: "notify.show",
  BADGE_SET: "badge.set",
  BIOMETRIC_AUTHENTICATE: "biometric.authenticate",
  EXTERNAL_OPEN: "external.open",
  PRINT_SYSTEM: "print.system",
  FS_SAVE_AS: "fs.saveAs",
  /** Тенантын цэс өөрчлөгдсөнийг мэдэгдэнэ — бүрхүүл өөрийн native цэсээ дахин
   *  татах боломжтой болно. */
  MENU_CHANGED: "menu.changed",
} as const;

/** Гэрээгээр тогтсон event нэрс. Бүрхүүлээс web рүү чиглэнэ. */
export const SHELL_EVENTS = {
  NAVIGATE: "shell:navigate",
  SEARCH: "shell:search",
  MENU_REFRESH: "shell:menu-refresh",
} as const;

/** Гэрээгээр тогтсон capability нэрс. */
export const SHELL_CAPABILITIES = {
  BIOMETRIC: "biometric",
  PRINT_SYSTEM: "print.system",
  /** `fs.saveAs` method-ын чадвар. Чадвар нь боломж, method нь дуудлага. */
  FS_SAVE: "fs.save",
  NOTIFY: "notify",
  BADGE: "badge",
  EXTERNAL_OPEN: "external.open",
  SECURE_STORE: "secure-store",
  MENU_NATIVE: "menu.native",
} as const;

export interface ShellNavigatePayload {
  path: string;
}

export interface ShellSearchPayload {
  query: string;
}

/**
 * Одоогийн бүрхүүл. SSR-д үргэлж `null` — `window` байхгүй.
 *
 * Бүрхүүл объектыг document start дээр inject хийдэг тул hydration-ы үед аль
 * хэдийн байрандаа байдаг; ижил объектын лавлагаа буцдаг нь
 * `useSyncExternalStore`-ын snapshot-ыг тогтвортой байлгана.
 */
export function getShell(): GeregeShell | null {
  if (typeof window === "undefined") return null;
  return window.GeregeShell ?? null;
}

export function hasCapability(cap: string): boolean {
  return getShell()?.capabilities?.includes(cap) ?? false;
}

// Бүрхүүл ажиллах хугацаандаа солигддоггүй (inject хийгээд л дуусдаг), тул
// захиалга нь юу ч хийхгүй — snapshot нэг л удаа уншигдана.
const subscribeToShell = () => () => {};
const getServerShell = () => null;

/**
 * React хук: бүрхүүл ба түүн дотор ажиллаж байгаа эсэх.
 *
 * Сервер дээр `inShell` нь үргэлж `false` тул анхны markup хөтчийнхтэй адил
 * гарч, hydration-ы дараа л бүрхүүлийн харагдац руу шилжинэ.
 */
export function useShell(): { shell: GeregeShell | null; inShell: boolean } {
  const shell = useSyncExternalStore(subscribeToShell, getShell, getServerShell);
  return { shell, inShell: shell !== null };
}

export type ShellInvokeResult<T> =
  | { ok: true; value: T }
  | { ok: false; reason: "no-shell" | "error" | "timeout"; error?: Error };

/**
 * Хэзээ ч throw хийхгүй, хэзээ ч мөнхөд өлгөөстэй үлдэхгүй invoke.
 *
 * Дуудагч тал ихэвчлэн "бүрхүүл үүнийг барьж авсан уу, үгүй юу" гэдгийг л
 * мэдэх шаардлагатай байдаг — бүрхүүл дэмжихгүй бол, алдвал, эсвэл огт хариу
 * буцаахгүй бол бүгд адилхан "web талын fallback-аа ажиллуул" гэсэн утгатай.
 */
export async function invokeShell<T>(
  method: string,
  params?: Record<string, unknown>,
  timeoutMs = 20000,
): Promise<ShellInvokeResult<T>> {
  const shell = getShell();
  if (!shell) return { ok: false, reason: "no-shell" };

  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<{ ok: false; reason: "timeout" }>((resolve) => {
    timer = setTimeout(() => resolve({ ok: false, reason: "timeout" }), timeoutMs);
  });

  try {
    const invocation = shell
      .invoke<T>(method, params)
      .then((value) => ({ ok: true, value }) as const)
      .catch((error: unknown) => ({
        ok: false,
        reason: "error",
        error: error instanceof Error ? error : new Error(String(error)),
      }) as const);
    return await Promise.race([invocation, timeout]);
  } finally {
    clearTimeout(timer);
  }
}
