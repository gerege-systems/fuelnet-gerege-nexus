"use client";

import {CheckCircle2} from "lucide-react";

import {TRUST_POINTS} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";

/** The chain from identity to permission, stated as a claim and its checklist. */
export default function Trust() {
  const {t} = useI18n();

  return (
    <section className="gp-trust" id="trust">
      <div>
        <span className="gp-eyebrow gp-eyebrow--blue">
          <i /> {t("website.view.trust_eyebrow")}
        </span>
        <h2>{t("website.view.trust_title")}</h2>
        <p>{t("website.view.trust_lede")}</p>
      </div>
      <ul>
        {TRUST_POINTS.map((point) => (
          <li key={point}>
            <CheckCircle2 />
            {t(point)}
          </li>
        ))}
      </ul>
    </section>
  );
}
