"use client";

import {useI18n} from "@/lib/i18n";
import {useBrand} from "@/lib/brandContext";

export default function SiteFooter() {
  const {t} = useI18n();
  const brand = useBrand();

  return (
    <footer className="gp-footer">
      <span>© 2026 Gerege Systems · {brand.name}</span>
      <span>{t("website.message.footer_note")}</span>
    </footer>
  );
}
