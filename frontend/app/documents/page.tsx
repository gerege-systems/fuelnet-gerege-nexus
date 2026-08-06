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

  const loadData = async () => {
    setLoading(true);
    try {
      const data = await api.getDocuments();
      setDocuments(data || []);
    } catch (err) {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  const { busyId, message, setMessage, sign, reject } = useDocumentActions(loadData);

  useEffect(() => {
    loadData();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.createDocument(form);
      setShowModal(false);
      setForm({ title: "", doc_type: "CONTRACT" });
      loadData();
    } catch (err: any) {
      setMessage({ type: "error", text: `${t("documents.message.create_failed")}: ${err.message}` });
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
          <p className="text-sm text-slate-500 mt-1">
            Enterprise document routing, digital signatures, and approval workflows
          </p>
        </div>
        <button
          onClick={() => setShowModal(true)}
          className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center space-x-2 shadow-sm transition"
        >
          <Plus className="w-4 h-4" />
          <span>{t("documents.action.create")}</span>
        </button>
      </div>

      {message && <Banner message={message} onDismiss={() => setMessage(null)} />}

      {loading ? (
        <div className="py-12 text-center text-slate-400">{t("documents.message.loading")}</div>
      ) : documents.length === 0 ? (
        <div className="bg-white border border-slate-200 rounded-xl p-8 text-center text-slate-500 text-sm">
          {t("documents.message.empty")}
        </div>
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
                    <SignatureCell doc={doc} />
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
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

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-xl font-bold text-slate-900 mb-4">{t("documents.view.create_title")}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Document Title *</label>
                <input
                  type="text"
                  placeholder={t("documents.field.title_placeholder")}
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
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
                  onClick={() => setShowModal(false)}
                  className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="w-1/2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 rounded-lg text-xs"
                >{t("documents.action.create")}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {signTarget && (
        <SignatureDialog
          doc={signTarget}
          busy={busyId === signTarget.id}
          onCancel={() => setSignTarget(null)}
          onSubmit={async (form) => {
            const ok = await sign(signTarget, form);
            if (ok) setSignTarget(null);
          }}
        />
      )}
    </div>
  );
}
