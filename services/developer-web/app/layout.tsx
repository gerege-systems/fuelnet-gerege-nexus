import "./globals.css";
import type { Metadata } from "next";
import Link from "next/link";
import { readSession } from "@/lib/session";
import { unverifiedClaims } from "@/lib/oidc";

export const metadata: Metadata = {
  title: "Gerege хөгжүүлэгчийн консол",
  description: "Апп бүртгэх, хувилбар нийтлэх, хяналтын дараалал.",
  // A console is not a shop window. Nothing here should be indexed.
  robots: { index: false, follow: false },
};

/**
 * The console shell.
 *
 * Mongolian, like the platform it publishes to. The storefront is the surface
 * that speaks seven languages because it is read by the public; this is read by
 * publishers who already work in Gerege Nexus.
 */
export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const session = await readSession();
  const claims = session ? unverifiedClaims(session) : {};
  const who = (claims.name as string) || (claims.email as string) || "";
  const storefront = process.env.NEXT_PUBLIC_STOREFRONT_URL || "https://appstore.gerege.mn";

  return (
    <html lang="mn">
      <body>
        <header className="top">
          <div className="inner">
            <Link href="/" className="brand">
              <span className="mark">G</span>
              <span>Хөгжүүлэгчийн консол</span>
            </Link>
            <span className="spacer" />
            <nav className="nav">
              <Link href="/">Миний аппууд</Link>
              <a href={storefront}>Дэлгүүр</a>
            </nav>
            {who && (
              <form action="/auth/logout" method="post" className="who">
                <span className="muted">{who}</span>
                <button type="submit" className="linklike">
                  Гарах
                </button>
              </form>
            )}
          </div>
        </header>
        <main className="shell">{children}</main>
        <footer className="bottom">
          <div className="inner">
            <span>Gerege Systems · developer.gerege.mn</span>
          </div>
        </footer>
      </body>
    </html>
  );
}
