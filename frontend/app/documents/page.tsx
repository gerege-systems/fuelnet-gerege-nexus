"use client";

import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAccess } from "@/lib/access";
import { useI18n } from "@/lib/i18n";
import {
  Banner,
  DocumentRecord,
  RowActions,
  SignatureCell,
  SignatureDialog,
  SignatureHistoryButton,
  SignatureHistoryDialog,
  SignatureProgress,
  StatusBadge,
  useDocumentActions,
} from "@/components/documents/shared";
import { FileText, Plus } from "lucide-react";

export default function DocumentsPage() {
  const { t } = useI18n();
  const { can } = useAccess();
  const [documents, setDocuments] = useState<DocumentRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState({ title: "", doc_type: "CONTRACT" });
  const [signTarget, setSignTarget] = useState<DocumentRecord | null>(null);
  const [historyTarget, setHistoryTarget] = useState<DocumentRecord | null>(null);

  // A failed load must never be reported as an empty list: "No documents yet" is a
  // claim about the tenant, and twelve contracts may be sitting in the queue.
  const [loadFailed, setLoadFailed] = useState(false);

  // How many rows this screen has asked for. The list is answered one page at a time —
  // each row counts its own signatures and outstanding steps — so "Load more" raises
  // this rather than fetching everything.
  const [limit, setLimit] = useState(200);
  const [total, setTotal] = useState(0);

  const loadData = async (want = limit) => {
    setLoading(true);
    try {
      const page = await api.getDocuments({ limit: want });
      setDocuments(page?.documents || []);
      setTotal(page?.total ?? 0);
      setLoadFailed(false);
    } catch (err: any) {
      setLoadFailed(true);
      setMessage({ type: "error", text: err?.message || t("documents.message.load_failed") });
    } finally {
      setLoading(false);
    }
  };

  const { busyId, message, setMessage, succeed, fail, route, reject } = useDocumentActions(loadData);

  useEffect(() => {
    loadData();
  }, []);

  // Nothing between the click and the POST changed any state, so a second click —
  // or Enter held down — created a second document, each one routed for approval.
  const [creating, setCreating] = useState(false);
  // The modal's own overlay covers the page banner, so a refusal reported only there
  // is a refusal the operator cannot read: the dialog just sat there with the button
  // re-enabled and no reason given. This is shown inside the modal.
  const [createFailure, setCreateFailure] = useState<string | null>(null);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (creating) return;
    setCreating(true);
    // Cleared before the attempt, and replaced after it. A refused create left its
    // red banner up; when the operator corrected the title and succeeded, that
    // banner was still the only thing the screen said about the operation — and a
    // document read as "failed" is submitted again, which lands a duplicate in the
    // approval queue, since a new document is routed for approval on creation.
    setMessage(null);
    setCreateFailure(null);
    const title = form.title;
    try {
      await api.createDocument(form);
      setShowModal(false);
      setForm({ title: "", doc_type: "CONTRACT" });
      await succeed(t("documents.message.create_success", { title }));
      return;
    } catch (err: any) {
      // Both: inside the modal, where it is readable now, and on the page banner for
      // after the operator gives up and closes it.
      const text = `${t("documents.message.create_failed")}: ${err.message}`;
      setCreateFailure(text);
      setMessage({ type: "error", text });
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
            <FileText className="w-7 h-7 text-indigo-600" />
            <span>{t("documents.view.title")}</span>
          </h1>
          <p className="text-sm text-slate-500 mt-1">{t("documents.view.subtitle")}</p>
        </div>
        {can("documents.manage") && (
          <button
            onClick={() => setShowModal(true)}
            className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center space-x-2 shadow-sm transition"
          >
            <Plus className="w-4 h-4" />
            <span>{t("documents.action.create")}</span>
          </button>
        )}
      </div>

      {message && <Banner message={message} onDismiss={() => setMessage(null)} />}

      {loading ? (
        <div className="py-12 text-center text-slate-400">{t("documents.message.loading")}</div>
      ) : documents.length === 0 ? (
        // Only a load that succeeded may claim the tenant has no documents; a
        // failed one says so in the banner and shows whatever it already had.
        loadFailed ? null : (
          <div className="bg-white border border-slate-200 rounded-xl p-8 text-center text-slate-500 text-sm">
            {t("documents.message.empty")}
          </div>
        )
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase">
              <tr>
                <th className="px-4 py-3">{t("documents.field.title")}</th>
                <th className="px-4 py-3">{t("base.field.type")}</th>
                <th className="px-4 py-3">{t("base.field.status")}</th>
                <th className="px-4 py-3">{t("documents.field.signature")}</th>
                <th className="px-4 py-3">{t("documents.field.created")}</th>
                <th className="px-4 py-3 text-right">{t("base.field.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {documents.map((doc) => (
                <tr key={doc.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 font-semibold text-slate-900">{doc.title}</td>
                  <td className="px-4 py-3 font-mono text-slate-600">{doc.doc_type}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <StatusBadge status={doc.status} />
                      <SignatureProgress doc={doc} />
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-start gap-2">
                      <SignatureCell doc={doc} />
                      <SignatureHistoryButton doc={doc} onOpen={setHistoryTarget} />
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-right">
                    <RowActions
                      doc={doc}
                      busy={busyId === doc.id}
                      canSign={can("documents.sign")}
                      canManage={can("documents.manage")}
                      onSign={setSignTarget}
                      onReject={reject}
                      onRoute={route}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {/* A partial list says so. Silence here is what makes an operator search for
              a contract that is simply on the next page. */}
          {total > documents.length && (
            <div className="flex items-center justify-between gap-3 px-4 py-3 border-t border-slate-200 bg-slate-50">
              <p className="text-[11px] text-slate-500">
                {t("documents.message.showing_some", { shown: documents.length, total })}
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

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-xl font-bold text-slate-900 mb-4">{t("documents.view.create_title")}</h2>

            {createFailure && (
              <p className="mb-4 text-xs text-red-700 bg-red-50 border border-red-200 rounded-lg p-3">
                {createFailure}
              </p>
            )}

            <form onSubmit={handleCreate} className="space-y-4">
              {/* maxLength is what document_records.title holds, in the characters
                  Postgres counts. The server refuses more, so there is no reason to
                  let it be typed and then refused. */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("documents.field.title")} *
                </label>
                <input
                  type="text"
                  placeholder={t("documents.field.title_placeholder")}
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  maxLength={255}
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">{t("documents.field.category")}</label>
                <select
                  value={form.doc_type}
                  onChange={(e) => setForm({ ...form, doc_type: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="CONTRACT">{t("documents.category.legal_contract")}</option>
                  <option value="REQUEST">{t("documents.category.official_request")}</option>
                  <option value="APPROVAL">{t("documents.category.internal_approval")}</option>
                </select>
              </div>

              <div className="flex items-center space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowModal(false);
                    setCreateFailure(null);
                  }}
                  className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
                >
                  {t("base.action.cancel")}
                </button>
                <button
                  type="submit"
                  disabled={creating || !form.title.trim()}
                  className="w-1/2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 rounded-lg text-xs disabled:opacity-50"
                >{t("documents.action.create")}</button>
              </div>
            </form>
          </div>
        </div>
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
