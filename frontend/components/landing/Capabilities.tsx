"use client";

import {CAPABILITIES} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";

/**
 * The four capability cards.
 *
 * The second card is painted dark. That is not decoration: the grid is two by
 * two and an unbroken field of white cards gives the eye nowhere to land, so
 * one card carries the weight.
 */
export default function Capabilities() {
  const {t} = useI18n();

  return (
    <section className="gp-section" id="features">
      <div className="gp-heading">
        <span>{t("website.view.features_eyebrow")}</span>
        <h2>{t("website.view.features_title")}</h2>
        <p>{t("website.view.features_lede")}</p>
      </div>
      <div className="gp-grid">
        {CAPABILITIES.map(({icon: Icon, title, body}, index) => (
          <article key={title} className={index === 1 ? "gp-feature gp-feature--dark" : "gp-feature"}>
            <Icon />
            <h3>{t(title)}</h3>
            <p>{t(body)}</p>
          </article>
        ))}
      </div>
    </section>
  );
}
