"use client";

import {CheckCircle2} from "lucide-react";

import {APPLICATIONS} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";

/**
 * The nine applications that ship with the platform.
 *
 * This reuses the trust panel's layout — a claim on one side, a checked list on
 * the other — because the claim being made is the same shape: here is a set of
 * things you get, and each one is already true rather than promised.
 */
export default function Applications() {
  const {t} = useI18n();

  return (
    <section className="gp-trust" id="applications">
      <div>
        <span className="gp-eyebrow gp-eyebrow--blue">
          <i /> {t("website.apps.eyebrow")}
        </span>
        <h2>{t("website.apps.title")}</h2>
        <p>{t("website.apps.lede")}</p>
      </div>
      <ul>
        {APPLICATIONS.map((app) => (
          <li key={app}>
            <CheckCircle2 />
            {t(app)}
          </li>
        ))}
      </ul>
    </section>
  );
}
