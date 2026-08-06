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

  const loadData = async () => {
    setLoading(true);
    try {
      const data = await api.getDocuments();
      setDocuments(data || []);
    } catch (err) {
      // Layout redirects to /login when the session is the problem.
    } finally {
      setLoading(false);
    }
  };

  const { busyId, message, setMessage, succeed, fail, reject } = useDocumentActions(loadData);

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

  // The oldest document in the queue is the one an approver should worry about.
  const waitingSince = useMemo(() => {
    if (pending.length === 0) return null;
    return pending.reduce((oldest, doc) => (doc.created_at < oldest.created_at ? doc : oldest));
  }, [pending]);

  const days = (iso: string) => Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={<ListChecks className="w-7 h-7 text-indigo-600" />}
        title={t("documents.menu.approvals")}
        subtitle={t("documents.view.approvals_hint")}
      />

      {message && <Banner message={message} onDismiss={() => setMessage(null)} />}

      <section className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="p-4 bg-white border border-slate-200 rounded-xl">
          <div className="text-2xl font-bold text-amber-600">{pending.length}</div>
          <div className="text-[11px] text-slate-500 leading-snug mt-1">{t("documents.stat.awaiting")}</div>
        </div>
        <div className="p-4 bg-white border border-slate-200 rounded-xl">
          <div className="text-2xl font-bold text-slate-700">
            {waitingSince ? days(waitingSince.created_at) : 0}
          </div>
          <div className="text-[11px] text-slate-500 leading-snug mt-1">{t("documents.stat.oldest_days")}</div>
        </div>
        {byType.slice(0, 2).map(([docType, count]) => (
          <div key={docType} className="p-4 bg-white border border-slate-200 rounded-xl">
            <div className="text-2xl font-bold text-indigo-600">{count}</div>
            <div className="text-[11px] text-slate-500 leading-snug mt-1 font-mono">{docType}</div>
          </div>
        ))}
      </section>

      {loading ? (
        <div className="py-12 text-center text-slate-400">{t("documents.message.loading")}</div>
      ) : pending.length === 0 ? (
        <div className="bg-white border border-slate-200 rounded-xl p-8 text-center text-slate-500 text-sm">
          {t("documents.message.no_pending")}
        </div>
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
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-slate-500">{days(doc.created_at)}</td>
                  <td className="px-4 py-3 text-right">
                    <RowActions
                      doc={doc}
                      busy={busyId === doc.id}
                      canSign={can("documents.sign")}
                      onSign={setSignTarget}
                      onReject={reject}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {ready && !can("documents.sign") && pending.length > 0 && (
        <p className="text-xs text-slate-500">{t("documents.message.sign_not_granted")}</p>
      )}

      {signTarget && (
        <SignatureDialog
          doc={signTarget}
          onClose={() => setSignTarget(null)}
          onDone={async (text) => {
            setSignTarget(null);
            await succeed(text);
          }}
          onError={fail}
        />
      )}
    </div>
  );
}
