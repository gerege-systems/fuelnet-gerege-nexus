import "../globals.css";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { LOCALES, isLocale, isRTL, translator, type Locale } from "@/lib/i18n";

export const metadata: Metadata = {
  title: "Gerege App Store",
  description: "Applications you can install on a Gerege Nexus platform.",
  // A storefront is meant to be found; the platform itself is not (Nexus sends
  // noindex). This is the one Gerege surface that wants a search engine.
  robots: { index: true, follow: true },
};

/**
 * The root layout, inside the language segment.
 *
 * It lives under [locale] rather than above it because <html lang> and the text
 * direction are decided by the language, and a layout above the segment cannot
 * read which one was asked for. Arabic reverses the whole page, so this is not
 * cosmetic.
 */
export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();

  const t = translator(locale as Locale);
  const developerConsole = process.env.NEXT_PUBLIC_DEVELOPER_URL || "https://developer.gerege.mn";

  return (
    <html lang={locale} dir={isRTL(locale as Locale) ? "rtl" : "ltr"}>
      <body>
        <header className="top">
          <div className="inner">
            <Link href={`/${locale}`} className="brand">
              <span className="mark">G</span>
              <span>{t("store.title")}</span>
            </Link>
            <span className="spacer" />
            <nav className="langs" aria-label="Language">
              {LOCALES.map((entry) => (
                <Link
                  key={entry.code}
                  href={`/${entry.code}`}
                  className={entry.code === locale ? "on" : ""}
                  hrefLang={entry.code}
                >
                  {entry.label}
                </Link>
              ))}
            </nav>
          </div>
        </header>

        <main className="shell">{children}</main>

        <footer className="bottom">
          <div className="inner">
            <span>Gerege Systems · appstore.gerege.mn</span>
            <span className="spacer" />
            <a href={developerConsole}>{t("store.developers")}</a>
          </div>
        </footer>
      </body>
    </html>
  );
}

// The seven languages are known at build time, so every storefront route can be
// rendered once rather than per request.
export function generateStaticParams() {
  return LOCALES.map((entry) => ({ locale: entry.code }));
}
