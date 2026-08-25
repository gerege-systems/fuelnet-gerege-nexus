"use client";

import {PLATFORM_DEPTH} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";

/**
 * Six shared capabilities implemented by the base runtime: resilience, state
 * connectors, security controls, AI endpoints, locale fallback and
 * instrumentation. External integrations still require deployment config.
 *
 * These are the sections a reader evaluating the platform against building it
 * themselves actually weighs, which is why they come last — by this point the
 * question has moved from "does it work" to "what would I not have to write".
 */
export default function PlatformDepth() {
  const {t} = useI18n();

  return (
    <section className="gp-section" id="platform">
      <div className="gp-heading">
        <span>{t("website.depth.eyebrow")}</span>
        <h2>{t("website.depth.title")}</h2>
        <p>{t("website.depth.lede")}</p>
      </div>
      <div className="gp-grid">
        {PLATFORM_DEPTH.map(({icon: Icon, title, body}, index) => (
          <article key={title} className={index === 3 ? "gp-feature gp-feature--dark" : "gp-feature"}>
            <Icon />
            <h3>{t(title)}</h3>
            <p>{t(body)}</p>
          </article>
        ))}
      </div>
    </section>
  );
}
