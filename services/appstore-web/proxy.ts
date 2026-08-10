import { NextResponse, type NextRequest } from "next/server";
import { LOCALES } from "@/lib/i18n";

/**
 * Everything the storefront serves lives under a language: /mn, /en/apps/esign.
 *
 * A visitor who arrives without one is sent to the language their browser asks
 * for, falling back to Mongolian — this is a Mongolian platform, and defaulting
 * to English would be a decision nobody made.
 *
 * (Next 16 calls this Proxy; it is the file that used to be called middleware.)
 */
const codes = LOCALES.map((entry) => entry.code) as string[];
const DEFAULT_LOCALE = "mn";

function preferredLocale(header: string | null): string {
  if (!header) return DEFAULT_LOCALE;
  // Accept-Language is a weighted list; the highest-weighted language we
  // actually publish in wins. Region subtags are dropped — the catalogue is
  // translated per language, not per country.
  const ranked = header
    .split(",")
    .map((part) => {
      const [tag, quality] = part.trim().split(";q=");
      return { tag: tag.split("-")[0].toLowerCase(), weight: quality ? Number(quality) : 1 };
    })
    .sort((a, b) => b.weight - a.weight);

  for (const { tag } of ranked) {
    if (codes.includes(tag)) return tag;
  }
  return DEFAULT_LOCALE;
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  if (codes.some((code) => pathname === `/${code}` || pathname.startsWith(`/${code}/`))) {
    return;
  }

  const locale = preferredLocale(request.headers.get("accept-language"));
  const target = request.nextUrl.clone();
  target.pathname = `/${locale}${pathname === "/" ? "" : pathname}`;
  return NextResponse.redirect(target);
}

export const config = {
  // Static assets and the health probe are not translated content and must not
  // be redirected: a load balancer asking /health should get an answer, not a
  // 307 into Mongolian.
  matcher: ["/((?!_next|favicon.ico|health).*)"],
};
