import Link from "next/link";
import { notFound } from "next/navigation";
import { listMyApps, listVersions } from "@/lib/registry";
import { AppForm, VersionForm } from "../../forms";

/**
 * One of the publisher's apps: its releases and the form that adds another.
 *
 * The manifest is pre-filled with a template built from the app itself, because
 * the fields that must agree with the catalogue entry — the id, the type — are
 * the ones a publisher would otherwise get wrong and have rejected.
 */
export default async function AppPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;

  const apps = await listMyApps();
  if ("error" in apps) {
    return (
      <>
        <p style={{ paddingTop: 24 }}>
          <Link href="/">← Буцах</Link>
        </p>
        <div className="bad">{apps.error}</div>
      </>
    );
  }

  const app = apps.data.find((candidate) => candidate.slug === slug);
  if (!app) notFound();

  const versions = await listVersions(slug);
  const template = JSON.stringify(
    app.type === "external"
      ? {
          id: app.id,
          type: "external",
          name: app.name,
          version: "1.0.0",
          platform: ">=1.0.0",
          dependencies: [],
          external: {
            launch_url: "https://example.mn/sso/gerege",
            sso_client_id: "",
            scopes: ["openid", "profile", "email"],
            embed: "new_tab",
            health_url: "",
          },
          permissions: [
            {
              code: `${app.slug.replace(/-/g, "_")}.read`,
              name: `Open ${app.name}`,
              description: "",
            },
          ],
          menus: [
            {
              id: `${app.slug.replace(/-/g, "_")}_home`,
              label: app.name,
              external_url: "https://example.mn/sso/gerege",
              icon: "share-2",
              order: 10,
            },
          ],
        }
      : {
          id: app.id,
          name: app.name,
          version: "1.0.0",
          platform: ">=1.0.0",
          dependencies: [],
          permissions: [],
          menus: [],
        },
    null,
    2,
  );

  return (
    <>
      <p style={{ paddingTop: 24 }}>
        <Link href="/">← Миний аппууд</Link>
      </p>

      <section className="hero" style={{ paddingTop: 12 }}>
        <h1>{app.name}</h1>
        <p className="split">
          <span className="tag">{app.id}</span>
          <span className={`tag ${app.type === "external" ? "blue" : ""}`}>{app.type}</span>
          <span className="tag">{app.category}</span>
        </p>
      </section>

      <h2>Хувилбарууд</h2>
      {"error" in versions ? (
        <div className="bad">{versions.error}</div>
      ) : versions.data.length === 0 ? (
        <div className="notice">Хувилбар илгээгээгүй байна.</div>
      ) : (
        <div className="panel">
          <ul className="list">
            {versions.data.map((version) => (
              <li key={version.id} className="split">
                <strong>v{version.version}</strong>
                <span className={`status ${version.status}`}>{version.status}</span>
                <span className="muted">{version.channel}</span>
                <span className="muted">{version.min_platform}</span>
                {version.review_note && <span className="muted">· {version.review_note}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      <h2 style={{ marginTop: 32 }}>Шинэ хувилбар илгээх</h2>
      <p className="muted" style={{ maxWidth: "62ch" }}>
        Нийтлэгдсэн хувилбар өөрчлөгдөхгүй — засвар хийхдээ шинэ дугаар өгнө. Manifest-ийг
        registry нь Nexus-ийн ашигладаг яг тэр шалгалтаар шалгана.
      </p>
      <VersionForm slug={slug} template={template} />

      <h2 style={{ marginTop: 32 }}>Аппын мэдээлэл</h2>
      <AppForm app={app} />
    </>
  );
}
