"use client";

import {ARCHITECTURE} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";

/**
 * How the platform is built, for the reader who has stopped asking whether the
 * sign-in works and started asking what they would be adopting.
 *
 * Built from the same section, heading and card classes as the capability grid
 * above it — no new styling was introduced for any of the sections on this
 * page, only new arrangements of what the page already had.
 */
export default function Architecture() {
  const {t} = useI18n();

  return (
    <section className="gp-section" id="architecture">
      <div className="gp-heading">
        <span>{t("website.arch.eyebrow")}</span>
        <h2>{t("website.arch.title")}</h2>
        <p>{t("website.arch.lede")}</p>
      </div>
      <div className="gp-grid">
        {ARCHITECTURE.map(({icon: Icon, title, body}, index) => (
          <article key={title} className={index === 2 ? "gp-feature gp-feature--dark" : "gp-feature"}>
            <Icon />
            <h3>{t(title)}</h3>
            <p>{t(body)}</p>
          </article>
        ))}
      </div>
    </section>
  );
}
