"use client";

import {useI18n} from "@/lib/i18n";

export default function SiteFooter() {
  const {t} = useI18n();

  return (
    <footer className="gp-footer">
      <span>© 2026 Gerege Systems · Gerege Nexus</span>
      <span>{t("website.message.footer_note")}</span>
    </footer>
  );
}
