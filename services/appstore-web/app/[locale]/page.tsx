import Link from "next/link";
import { notFound } from "next/navigation";
import { listApps } from "@/lib/registry";
import { isLocale, translator, type Locale } from "@/lib/i18n";

/**
 * The catalogue.
 *
 * Search and category are query parameters handled on the server, so the page
 * works with JavaScript switched off and every filtered view has a URL somebody
 * can send to a colleague. There is no client component on this screen at all.
 */
export default async function CatalogPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ q?: string; category?: string }>;
}) {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();
  const { q = "", category = "" } = await searchParams;

  const t = translator(locale as Locale);
  const result = await listApps(locale);

  if ("error" in result) {
    return (
      <>
        <section className="hero">
          <h1>{t("store.title")}</h1>
          <p>{t("store.tagline")}</p>
        </section>
        <div className="notice">{t("store.unavailable")}</div>
      </>
    );
  }

  const apps = result.data;
  const categories = Array.from(new Set(apps.map((app) => app.category).filter(Boolean))).sort();
  const needle = q.trim().toLocaleLowerCase();
  const shown = apps.filter((app) => {
    const matchesText =
      !needle ||
      app.name.toLocaleLowerCase().includes(needle) ||
      app.description.toLocaleLowerCase().includes(needle);
    return matchesText && (!category || app.category === category);
  });

  return (
    <>
      <section className="hero">
        <h1>{t("store.title")}</h1>
        <p>{t("store.tagline")}</p>
      </section>

      <form className="filters" action={`/${locale}`} method="get">
        <input type="search" name="q" defaultValue={q} placeholder={t("store.search")} aria-label={t("store.search")} />
        <Link className={`chip ${category ? "" : "on"}`} href={`/${locale}`}>
          {t("store.all")}
        </Link>
        {categories.map((entry) => (
          <Link
            key={entry}
            className={`chip ${category === entry ? "on" : ""}`}
            href={`/${locale}?category=${encodeURIComponent(entry)}`}
          >
            {entry}
          </Link>
        ))}
      </form>

      {shown.length === 0 ? (
        <div className="notice">{t("store.empty")}</div>
      ) : (
        <div className="grid">
          {shown.map((app) => (
            <Link key={app.id} className="card" href={`/${locale}/apps/${app.slug}`}>
              <div className="row">
                <span className="icon" aria-hidden>
                  {app.name.trim().charAt(0).toUpperCase()}
                </span>
                <div>
                  <h2>{app.name}</h2>
                  <div className="meta">
                    <span>{app.category}</span>
                    {app.latest_version && <span>v{app.latest_version}</span>}
                  </div>
                </div>
              </div>
              <p>{app.description}</p>
              <div className="meta">
                <span className={`tag ${app.type === "external" ? "blue" : ""}`}>
                  {app.type === "external" ? t("store.type.external") : t("store.type.module")}
                </span>
                <span className="tag">{app.publisher}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </>
  );
}
