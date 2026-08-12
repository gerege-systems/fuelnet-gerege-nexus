"use client";

import {ArrowRight} from "lucide-react";

import EIDLogin from "@/components/EIDLogin";
import {useI18n} from "@/lib/i18n";

/**
 * The first screen: what this is, and the eID panel to act on it.
 *
 * The sign-in panel sits in the hero rather than behind the header button
 * because the shortest path from landing to signed-in is the point of the page.
 */
export default function Hero() {
  const {t} = useI18n();

  return (
    <section className="gp-hero">
      <div className="gp-pattern" />
      <div className="gp-hero__inner">
        <div className="gp-copy">
          <span className="gp-eyebrow">
            <i /> NEXUS · eID · SSO
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
            <a href="#features" className="gp-outline">
              {t("website.action.see_features")}
            </a>
          </div>
          <div className="gp-stats">
            <span>
              <b>eID</b>
              {t("website.stat.eid")}
            </span>
            <span>
              <b>OAuth2 · OIDC</b>
              {t("website.stat.standards")}
            </span>
            <span>
              <b>SSO</b>
              {t("website.stat.sso")}
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
