import Link from "next/link";
import { getMe, listMyApps } from "@/lib/registry";
import { PublisherForm, AppForm } from "./forms";

/**
 * The console's front page: sign in, then whatever the next thing to do is.
 *
 * Three states, in order of how far along somebody is — not signed in, signed
 * in with no publisher account, and publishing. Each shows only its own step,
 * because a form for something you cannot do yet is noise.
 */
export default async function ConsolePage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  const me = await getMe();

  if ("error" in me) {
    return (
      <section className="hero">
        <h1>Gerege хөгжүүлэгчийн консол</h1>
        <p>
          Апп бүртгэж, хувилбар нийтэлж, appstore.gerege.mn дээр гаргахын тулд Gerege бүртгэлээрээ
          нэвтэрнэ үү.
        </p>
        {error && <div className="bad">{error}</div>}
        {me.status !== 401 && <div className="bad">{me.error}</div>}
        <p style={{ marginTop: 20 }}>
          {/* A link, not a form: signing in is a navigation to another origin,
              and the platform decides everything that happens next. */}
          <a className="chip on" href="/auth/login" style={{ padding: "10px 18px" }}>
            Gerege-ээр нэвтрэх
          </a>
        </p>
      </section>
    );
  }

  const { publisher, admin, name, email } = me.data;

  if (!publisher) {
    return (
      <>
        <section className="hero">
          <h1>Publisher бүртгэл</h1>
          <p>
            {name || email} нэрээр нэвтэрлээ. Апп нийтлэхийн өмнө байгууллагаа publisher болгон
            бүртгүүлнэ. Бүртгэлийг Gerege баталгаажуулсны дараа таны аппууд дэлгүүрт харагдана.
          </p>
        </section>
        <PublisherForm />
      </>
    );
  }

  const apps = await listMyApps();

  return (
    <>
      <section className="hero">
        <h1>{publisher.name}</h1>
        <p className="split">
          <span className="tag">{publisher.slug}</span>
          <span className={`status ${publisher.verified ? "published" : "in_review"}`}>
            {publisher.verified ? "Баталгаажсан" : "Баталгаажаагүй"}
          </span>
          {admin && (
            <Link className="tag blue" href="/review">
              Хяналтын дараалал
            </Link>
          )}
        </p>
      </section>

      {!publisher.verified && (
        <div className="notice">
          Баталгаажаагүй publisher-ийн хувилбарууд хяналтын дараалалд орж, зөвшөөрөгдсөний дараа л
          дэлгүүрт гарна.
        </div>
      )}

      <h2 style={{ marginTop: 28 }}>Миний аппууд</h2>
      {"error" in apps ? (
        <div className="bad">{apps.error}</div>
      ) : apps.data.length === 0 ? (
        <div className="notice">Одоогоор апп бүртгээгүй байна. Доорх маягтаар эхлүүлнэ үү.</div>
      ) : (
        <div className="grid">
          {apps.data.map((app) => (
            <Link key={app.id} className="card" href={`/apps/${app.slug}`}>
              <div className="row">
                <span className="icon" aria-hidden>
                  {app.name.trim().charAt(0).toUpperCase()}
                </span>
                <div>
                  <h2 style={{ fontSize: 16 }}>{app.name}</h2>
                  <div className="meta">
                    <span>{app.category}</span>
                    {app.latest_version ? (
                      <span>v{app.latest_version}</span>
                    ) : (
                      <span>нийтлэгдээгүй</span>
                    )}
                  </div>
                </div>
              </div>
              <div className="meta">
                <span className={`tag ${app.type === "external" ? "blue" : ""}`}>{app.type}</span>
                <span className="tag">{app.id}</span>
              </div>
            </Link>
          ))}
        </div>
      )}

      <h2 style={{ marginTop: 32 }}>Апп нэмэх, засах</h2>
      <p className="muted" style={{ maxWidth: "62ch" }}>
        Байгаа аппын ID-г оруулбал мэдээлэл нь шинэчлэгдэнэ. Апп өөрөө бүртгэгдсэн ч хувилбар
        нийтлэх хүртэл дэлгүүрт харагдахгүй.
      </p>
      <AppForm />
    </>
  );
}
