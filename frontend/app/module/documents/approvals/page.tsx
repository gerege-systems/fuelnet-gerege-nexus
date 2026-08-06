"use client";

import React, { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useAccess } from "@/lib/access";
import { useI18n } from "@/lib/i18n";
import {
  Banner,
  DocumentRecord,
  PENDING,
  RowActions,
  SectionHeader,
  SignatureDialog,
  SignatureHistoryButton,
  SignatureHistoryDialog,
  SignatureProgress,
  useDocumentActions,
} from "@/components/documents/shared";
import { ListChecks } from "lucide-react";

/**
 * Approval queue: the documents list narrowed to what is actually waiting for a
 * decision. It reads the same /documents collection and drives the same sign
 * and reject endpoints — the queue is a lens on that data, not a second store.
 */
export default function DocumentApprovalsPage() {
  const { t } = useI18n();
  const { can, ready } = useAccess();
  const [documents, setDocuments] = useState<DocumentRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [signTarget, setSignTarget] = useState<DocumentRecord | null>(null);
  const [historyTarget, setHistoryTarget] = useState<DocumentRecord | null>(null);

  // "Nothing is waiting for approval" is a claim, and an approver who reads it
  // closes the tab. A load that failed says so instead.
  const [loadFailed, setLoadFailed] = useState(false);

  // How many the queue has asked for, and how many are waiting in total.
  const [limit, setLimit] = useState(200);
  const [total, setTotal] = useState(0);

  // The queue asks the SERVER for what is waiting. Filtering a capped page in the
  // browser would let a document waiting for a signature fall off the end of a page
  // full of approved ones — the one document this screen exists to show.
  const loadData = async (want = limit) => {
    setLoading(true);
    try {
      const page = await api.getDocuments({ status: PENDING, order: "oldest", limit: want });
      setDocuments(page?.documents || []);
      setTotal(page?.total ?? 0);
      setLoadFailed(false);
    } catch (err: any) {
      setLoadFailed(true);
      // Only when there is nothing on screen to carry the news. With rows showing, the
      // footer says the list is stale and stays saying it — and a banner here would
      // overwrite what the action that triggered this refresh had just reported, so a
      // completed signature read as a failure.
      if (documents.length === 0) {
        setMessage({ type: "error", text: err?.message || t("documents.message.load_failed") });
      }
    } finally {
      setLoading(false);
    }
  };

  const { isBusy, message, setMessage, succeed, fail, reject } = useDocumentActions(loadData);

  useEffect(() => {
    loadData();
  }, []);

  const pending = useMemo(() => documents.filter((doc) => doc.status === PENDING), [documents]);

  const byType = useMemo(() => {
    const counts = new Map<string, number>();
    for (const doc of pending) {
      counts.set(doc.doc_type, (counts.get(doc.doc_type) || 0) + 1);
    }
    return [...counts.entries()].sort((a, b) => b[1] - a[1]);
  }, [pending]);

  // The oldest document in the queue is the one an approver should worry about — and
  // the server sends this page oldest first, so it is the first row rather than the
  // oldest of whatever this page happens to hold.
  const waitingSince = useMemo(() => pending[0] ?? null, [pending]);

  const days = (iso: string) => Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={<ListChecks className="w-7 h-7 text-indigo-600" />}
        title={t("documents.menu.approvals")}
        subtitle={t("documents.view.approvals_hint")}
      />

      {message && <Banner message={message} onDismiss={() => setMessage(null)} />}

      {/* A tile is as much a claim as a sentence. "Awaiting signature: 0" over a
          queue the page could not read is the same falsehood the prose below is
          careful not to tell, so a failed load shows a dash in both places. */}
      <section className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="p-4 bg-white border border-slate-200 rounded-xl">
          {/* The queue's real size, not this page's — the server counted it. */}
          <div className="text-2xl font-bold text-amber-600">{loadFailed ? "—" : total}</div>
          <div className="text-[11px] text-slate-500 leading-snug mt-1">{t("documents.stat.awaiting")}</div>
        </div>
        <div className="p-4 bg-white border border-slate-200 rounded-xl">
          <div className="text-2xl font-bold text-slate-700">
            {loadFailed ? "—" : waitingSince ? days(waitingSince.created_at) : 0}
          </div>
          <div className="text-[11px] text-slate-500 leading-snug mt-1">{t("documents.stat.oldest_days")}</div>
        </div>
        {/* Counted over the rows this page holds, so a partial queue is marked: the
            server counts the queue as a whole but not its breakdown by type. */}
        {byType.slice(0, 2).map(([docType, count]) => (
          <div key={docType} className="p-4 bg-white border border-slate-200 rounded-xl">
            <div className="text-2xl font-bold text-indigo-600">
              {total > pending.length ? `≥${count}` : count}
            </div>
            <div className="text-[11px] text-slate-500 leading-snug mt-1 font-mono">{docType}</div>
          </div>
        ))}
      </section>

      {loading ? (
        <div className="py-12 text-center text-slate-400">{t("documents.message.loading")}</div>
      ) : pending.length === 0 ? (
        // "Nothing is waiting" is a claim about the queue, so only a load that
        // succeeded may make it.
        loadFailed ? null : (
          <div className="bg-white border border-slate-200 rounded-xl p-8 text-center text-slate-500 text-sm">
            {t("documents.message.no_pending")}
          </div>
        )
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase">
              <tr>
                <th className="px-4 py-3">{t("documents.field.title")}</th>
                <th className="px-4 py-3">{t("base.field.type")}</th>
                <th className="px-4 py-3">{t("documents.field.created")}</th>
                <th className="px-4 py-3">{t("documents.field.waiting_days")}</th>
                <th className="px-4 py-3 text-right">{t("base.field.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {pending.map((doc) => (
                <tr key={doc.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 font-semibold text-slate-900">{doc.title}</td>
                  <td className="px-4 py-3 font-mono text-slate-600">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span>{doc.doc_type}</span>
                      <SignatureProgress doc={doc} />
                      <SignatureHistoryButton doc={doc} onOpen={setHistoryTarget} />
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-slate-500">{days(doc.created_at)}</td>
                  <td className="px-4 py-3 text-right">
                    <RowActions
                      doc={doc}
                      busy={isBusy(doc.id)}
                      canSign={can("documents.sign")}
                      onSign={setSignTarget}
                      onReject={reject}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {/* A stale list says so for as long as it is stale — the banner can be
              dismissed, and a refresh that failed after an action must not be the only
              thing that says the rows are old. */}
          {loadFailed && (
            <div className="flex items-center justify-between gap-3 px-4 py-3 border-t border-amber-200 bg-amber-50">
              <p className="text-[11px] text-amber-800">{t("documents.message.stale_rows")}</p>
              <button
                type="button"
                disabled={loading}
                onClick={() => loadData()}
                className="text-[11px] font-semibold px-3 py-1.5 rounded-lg bg-white border border-amber-300 text-amber-800 hover:bg-amber-100 disabled:opacity-50"
              >
                {t("documents.action.retry")}
              </button>
            </div>
          )}

          {/* A queue shown in part says so, and can be read to the end. */}
          {total > pending.length && (
            <div className="flex items-center justify-between gap-3 px-4 py-3 border-t border-slate-200 bg-slate-50">
              <p className="text-[11px] text-slate-500">
                {t("documents.message.showing_some", { shown: pending.length, total })}
              </p>
              <button
                type="button"
                disabled={loading}
                onClick={() => {
                  const want = limit + 200;
                  setLimit(want);
                  loadData(want);
                }}
                className="text-[11px] font-semibold px-3 py-1.5 rounded-lg bg-white border border-slate-300 text-slate-700 hover:bg-slate-100 disabled:opacity-50"
              >
                {t("documents.action.load_more")}
              </button>
            </div>
          )}

        </div>
      )}

      {ready && !can("documents.sign") && pending.length > 0 && (
        <p className="text-xs text-slate-500">{t("documents.message.sign_not_granted")}</p>
      )}

      {signTarget && (
        <SignatureDialog
          key={signTarget.id}
          doc={signTarget}
          onClose={() => setSignTarget(null)}
          onDone={async (text) => {
            setSignTarget(null);
            await succeed(text);
          }}
          onError={fail}
        />
      )}

      {historyTarget && (
        <SignatureHistoryDialog doc={historyTarget} onClose={() => setHistoryTarget(null)} />
      )}
    </div>
  );
}
