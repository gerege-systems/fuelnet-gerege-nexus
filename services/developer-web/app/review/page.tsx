import Link from "next/link";
import { reviewQueue } from "@/lib/registry";
import { ReviewForm } from "../forms";

/**
 * The review queue.
 *
 * Oldest first, because a queue is answered in the order people joined it. The
 * whole manifest is shown rather than a summary: what is being approved is
 * every permission and every URL in it, and a reviewer who cannot see those is
 * not reviewing anything.
 *
 * Whether this page is allowed at all is the registry's decision — it answers
 * 403 to anyone who is not a reviewer. The console renders what it is given.
 */
export default async function ReviewPage() {
  const queue = await reviewQueue();

  if ("error" in queue) {
    return (
      <>
        <p style={{ paddingTop: 24 }}>
          <Link href="/">← Буцах</Link>
        </p>
        <div className="bad">{queue.error}</div>
      </>
    );
  }

  return (
    <>
      <p style={{ paddingTop: 24 }}>
        <Link href="/">← Миний аппууд</Link>
      </p>

      <section className="hero" style={{ paddingTop: 12 }}>
        <h1>Хяналтын дараалал</h1>
        <p>Хүлээгдэж буй {queue.data.length} хувилбар.</p>
      </section>

      {queue.data.length === 0 ? (
        <div className="notice">Хүлээгдэж буй хувилбар алга.</div>
      ) : (
        queue.data.map((version) => (
          <div className="panel" key={version.id} style={{ marginBottom: 16 }}>
            <div className="split">
              <strong>
                {version.app?.name || version.app_id} v{version.version}
              </strong>
              <span className="status in_review">{version.channel}</span>
              <span className="muted">{version.min_platform}</span>
              <span className="muted">{version.submitted_by}</span>
            </div>

            <details style={{ marginTop: 12 }}>
              <summary className="muted">Manifest</summary>
              <pre
                style={{
                  overflowX: "auto",
                  fontSize: 12.5,
                  background: "var(--ground)",
                  padding: 12,
                  borderRadius: 10,
                }}
              >
                {JSON.stringify(version.manifest, null, 2)}
              </pre>
            </details>

            <div style={{ marginTop: 12 }}>
              <ReviewForm id={version.id} />
            </div>
          </div>
        ))
      )}
    </>
  );
}
