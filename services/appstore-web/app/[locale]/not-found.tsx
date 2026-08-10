import Link from "next/link";
import { translator } from "@/lib/i18n";

/**
 * The 404 page.
 *
 * Next renders this without route params, so it cannot know which language the
 * visitor was reading. Rather than guess, it says the same thing twice — the
 * source language and the fallback one — which is what every other Gerege
 * surface falls back to when a translation is missing.
 */
export default function NotFound() {
  const mn = translator("mn");
  const en = translator("en");
  return (
    <div style={{ paddingTop: 48 }}>
      <h1 style={{ margin: "0 0 8px", fontSize: 26 }}>404</h1>
      <p style={{ margin: "0 0 4px" }}>{mn("store.notfound")}</p>
      <p className="muted" style={{ marginTop: 0 }}>
        {en("store.notfound")}
      </p>
      <p style={{ marginTop: 20 }}>
        <Link href="/mn">{mn("store.back")}</Link>
      </p>
    </div>
  );
}
