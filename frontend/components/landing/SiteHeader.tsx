"use client";

import Link from "next/link";

import LanguageSwitcher from "@/components/LanguageSwitcher";
import {DOCS_URL} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";

/**
 * The public header.
 *
 * The first three items scroll to a section of this page and are listed in the
 * order those sections appear, so the first item is never the furthest away.
 * The fourth leaves for the published documentation, so it opens in a new tab
 * and carries `rel="noopener"` rather than silently replacing the page someone
 * is reading.
 */
export default function SiteHeader() {
  const {t} = useI18n();

  return (
    <header className="gp-nav">
      <a href="#top" className="gp-brand">
        <img src="/brand.webp" alt="" />
        <span>Gerege Nexus</span>
      </a>
      <nav>
        <a href="#architecture">{t("website.menu.architecture")}</a>
        <a href="#applications">{t("website.menu.applications")}</a>
        <a href="#trust">{t("website.menu.trust")}</a>
        <a href={DOCS_URL} target="_blank" rel="noopener noreferrer">
          {t("website.menu.docs")}
        </a>
      </nav>
      <div className="gp-actions">
        <LanguageSwitcher variant="dark" />
        <Link href="/login" className="gp-gold">
          {t("website.action.sign_in")}
        </Link>
      </div>
    </header>
  );
}
