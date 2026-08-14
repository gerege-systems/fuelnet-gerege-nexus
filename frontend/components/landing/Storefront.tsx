"use client";

import { ArrowRight, PackageSearch } from "lucide-react";

import { useI18n } from "@/lib/i18n";
import { localizedApp, type StoreApp } from "@/lib/storefront";

/**
 * Апп сторын нүүр хуудас.
 *
 * Платформын нүүр нь "энэ юу вэ" гэсэн асуултад хариулдаг маркетингийн хуудас.
 * Сторын зочин өөр асуулттай ирдэг — "энд юу байна вэ" — тиймээс хамгийн
 * дээрээ маргаан биш, каталог байх ёстой.
 *
 * Аппууд нь build үед бичигдсэн жагсаалт биш, registry-ийн нийтийн endpoint-ээс
 * ирнэ. Өөрөөр байж ч болохгүй: гуравдагч тал апп нийтлэх бүрд frontend-ийг
 * дахин барих стор бол стор биш, брошур юм.
 */
export default function Storefront({ apps }: { apps: StoreApp[] }) {
  const { t, locale } = useI18n();
  const listed = apps.map((app) => localizedApp(app, locale));

  // Ангиллаар нь бүлэглэнэ: есөн картыг ялгаагүй нэг тор болгож тавихаар
  // зочин юу хайхаа мэдэхгүй болно.
  const byCategory = new Map<string, StoreApp[]>();
  for (const app of listed) {
    const key = app.category || t("storefront.category.other");
    byCategory.set(key, [...(byCategory.get(key) || []), app]);
  }
  const categories = [...byCategory.entries()].sort((a, b) => a[0].localeCompare(b[0], locale));

  return (
    <>
      <section className="gp-hero">
        <div className="gp-pattern" />
        <div className="gp-hero__inner">
          <div className="gp-copy">
            <span className="gp-eyebrow">
              <i /> {t("storefront.view.eyebrow")}
            </span>
            <h1>
              {t("storefront.view.title_lead")} <em>{t("storefront.view.title_highlight")}</em>
            </h1>
            <p>{t("storefront.view.lede")}</p>
            <div className="gp-cta">
              <a href="#catalogue" className="gp-gold gp-gold--large">
                {t("storefront.action.browse")} <ArrowRight />
              </a>
              <a href="/login" className="gp-outline">
                {t("storefront.action.publish")}
              </a>
            </div>
            <div className="gp-stats">
              <span>
                <b>{listed.length}</b>
                {t("storefront.stat.apps")}
              </span>
              <span>
                <b>{categories.length}</b>
                {t("storefront.stat.categories")}
              </span>
              <span>
                <b>{new Set(listed.map((a) => a.publisher).filter(Boolean)).size}</b>
                {t("storefront.stat.publishers")}
              </span>
            </div>
          </div>
        </div>
      </section>

      <section className="gp-section" id="catalogue">
        <div className="gp-heading">
          <span>{t("storefront.view.catalogue_eyebrow")}</span>
          <h2>{t("storefront.view.catalogue_title")}</h2>
          <p>{t("storefront.view.catalogue_lede")}</p>
        </div>

        {categories.map(([category, group]) => (
          <div key={category} className="gp-store-group">
            <h3 className="gp-store-group__name">{category}</h3>
            <div className="gp-grid">
              {group.map((app) => (
                <article key={app.id} className="gp-feature gp-store-card">
                  <PackageSearch />
                  <div className="gp-store-card__head">
                    <h3>{app.name}</h3>
                    <span className="gp-store-card__version">v{app.latest_version}</span>
                  </div>
                  <p>{app.description}</p>
                  <footer className="gp-store-card__foot">
                    {/* The id, not the name: it is what an administrator types
                        to install, and what every installation on every
                        instance is keyed by. */}
                    <code>{app.id}</code>
                    {app.publisher && <span>{app.publisher}</span>}
                  </footer>
                </article>
              ))}
            </div>
          </div>
        ))}
      </section>
    </>
  );
}
