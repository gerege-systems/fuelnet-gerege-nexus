import Link from "next/link";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { getApp } from "@/lib/registry";
import { isLocale, translator, type Locale } from "@/lib/i18n";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; slug: string }>;
}): Promise<Metadata> {
  const { locale, slug } = await params;
  const result = isLocale(locale) ? await getApp(slug, locale) : { error: "missing" as const };
  if ("error" in result) return { title: "Gerege App Store" };
  return {
    title: `${result.data.name} — Gerege App Store`,
    description: result.data.description,
  };
}

/**
 * One app: what it is, what it asks for, and what it has released.
 *
 * There is deliberately no install button. Installing happens inside an
 * organisation's own Nexus, decided by an administrator signed in there; a
 * button here could only pretend, and a storefront that pretends is worse than
 * one that explains.
 */
export default async function AppPage({
  params,
}: {
  params: Promise<{ locale: string; slug: string }>;
}) {
  const { locale, slug } = await params;
  if (!isLocale(locale)) notFound();

  const t = translator(locale as Locale);
  const result = await getApp(slug, locale);
  // An app that does not exist is a 404, not a page that says 404: this site is
  // indexed, and a soft 404 teaches a crawler that missing pages are fine.
  if ("error" in result && result.error === "missing") notFound();
  if ("error" in result) {
    return (
      <>
        <p style={{ paddingTop: 28 }}>
          <Link href={`/${locale}`}>{t("store.back")}</Link>
        </p>
        <div className="notice">{t("store.unavailable")}</div>
      </>
    );
  }

  const app = result.data;

  const manifest = app.manifest || app.versions[0]?.manifest;
  const permissions = manifest?.permissions || [];
  const dependencies = manifest?.dependencies || [];

  return (
    <>
      <p style={{ paddingTop: 24 }}>
        <Link href={`/${locale}`}>{t("store.back")}</Link>
      </p>

      <div className="detail">
        <div>
          <div className="row" style={{ gap: 16 }}>
            <span className="icon" style={{ width: 60, height: 60, fontSize: 24 }} aria-hidden>
              {app.name.trim().charAt(0).toUpperCase()}
            </span>
            <div>
              <h1 style={{ margin: 0, fontSize: 26 }}>{app.name}</h1>
              <div className="meta" style={{ marginTop: 6 }}>
                <span className={`tag ${app.type === "external" ? "blue" : ""}`}>
                  {app.type === "external" ? t("store.type.external") : t("store.type.module")}
                </span>
                <span className="tag">{app.category}</span>
                {app.latest_version && <span className="tag">v{app.latest_version}</span>}
              </div>
            </div>
          </div>

          <p style={{ marginTop: 18, maxWidth: "68ch" }}>{app.description}</p>

          {app.type === "external" && <div className="notice">{t("store.external.note")}</div>}

          {permissions.length > 0 && (
            <div className="panel" style={{ marginTop: 20 }}>
              <h3>{t("store.permissions")}</h3>
              <ul className="list">
                {permissions.map((permission) => (
                  <li key={permission.code}>
                    <strong>{permission.name}</strong> <code>{permission.code}</code>
                    {permission.description && (
                      <div className="muted">{permission.description}</div>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {dependencies.length > 0 && (
            <div className="panel">
              <h3>{t("store.dependencies")}</h3>
              <ul className="list">
                {dependencies.map((dependency) => (
                  <li key={dependency.id}>
                    <code>{dependency.id}</code>{" "}
                    <span className="muted">{dependency.version_constraint}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="panel">
            <h3>{t("store.history")}</h3>
            <ul className="list">
              {app.versions.map((version) => (
                <li key={version.version}>
                  <strong>v{version.version}</strong>{" "}
                  <span className="muted">
                    {version.published_at
                      ? new Date(version.published_at).toISOString().slice(0, 10)
                      : ""}
                    {version.min_platform ? ` · ${t("store.requires_platform")} ${version.min_platform}` : ""}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <aside>
          <div className="panel">
            <h3>{t("store.install.title")}</h3>
            <p style={{ margin: 0, fontSize: 14 }} className="muted">
              {t("store.install.body")}
            </p>
          </div>

          <div className="panel">
            <h3>{t("store.publisher")}</h3>
            <p style={{ margin: 0 }}>{app.publisher}</p>
            <p style={{ marginBottom: 0, marginTop: 12 }} className="muted">
              <code>{app.id}</code>
            </p>
          </div>

          {app.external && (
            <div className="panel">
              <h3>{t("store.type.external")}</h3>
              <ul className="list">
                <li>
                  <span className="muted">launch_url</span>
                  <div>
                    <code>{app.external.launch_url}</code>
                  </div>
                </li>
                {app.external.scopes?.length > 0 && (
                  <li>
                    <span className="muted">scopes</span>
                    <div>
                      <code>{app.external.scopes.join(" ")}</code>
                    </div>
                  </li>
                )}
              </ul>
            </div>
          )}
        </aside>
      </div>
    </>
  );
}
