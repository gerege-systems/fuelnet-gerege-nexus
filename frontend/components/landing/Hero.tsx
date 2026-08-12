"use client";

import {ArrowRight} from "lucide-react";

import EIDLogin from "@/components/EIDLogin";
import {useI18n} from "@/lib/i18n";

/**
 * The first screen: what the platform is, and the eID panel to act on it.
 *
 * The headline is about the platform rather than the sign-in, because that is
 * the question a visitor arrives with. The sign-in panel still sits beside it
 * rather than behind the header button: the shortest path from landing to
 * signed-in is worth keeping even when it is no longer the argument.
 */
export default function Hero() {
  const {t} = useI18n();

  return (
    <section className="gp-hero">
      <div className="gp-pattern" />
      <div className="gp-hero__inner">
        <div className="gp-copy">
          <span className="gp-eyebrow">
            <i /> OPEN SOURCE · APACHE 2.0 · GO
          </span>
          <h1>
            {t("website.view.hero_title_lead")} <em>{t("website.view.hero_title_highlight")}</em>{" "}
            {t("website.view.hero_title_tail")}
          </h1>
          <p>{t("website.view.hero_lede")}</p>
          <div className="gp-cta">
            <a href="#eid-login" className="gp-gold gp-gold--large">
              {t("website.action.eid_sign_in")} <ArrowRight />
            </a>
            <a href="#architecture" className="gp-outline">
              {t("website.action.see_features")}
            </a>
          </div>
          <div className="gp-stats">
            <span>
              <b>9</b>
              {t("website.stat.apps")}
            </span>
            <span>
              <b>7</b>
              {t("website.stat.languages")}
            </span>
            <span>
              <b>1</b>
              {t("website.stat.binary")}
            </span>
          </div>
        </div>
        <div id="eid-login" className="gp-login-slot">
          <EIDLogin compact />
        </div>
      </div>
    </section>
  );
}
