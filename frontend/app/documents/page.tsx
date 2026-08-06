"use client";

import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { FileText, Plus, CheckCircle, ShieldCheck, Clock, XCircle, PenLine, Ban } from "lucide-react";

interface Document {
  id: string;
  title: string;
  doc_type: string;
  status: string;
  signed_by?: string;
  signature_hash?: string;
  signer_reg_number?: string;
  signer_method?: string;
  signed_at?: string;
  created_at: string;
}

type SignMethod = "EID" | "DAN";

export default function DocumentsPage() {
  const { t } = useI18n();
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState({ title: "", doc_type: "CONTRACT" });

  // E-signature modal state
  const [signTarget, setSignTarget] = useState<Document | null>(null);
  const [signForm, setSignForm] = useState<{ method: SignMethod; reg_number: string; otp_code: string }>({
    method: "EID",
    reg_number: "",
    otp_code: "",
  });
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

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

  const openSignModal = (doc: Document) => {
    setSignForm({ method: "EID", reg_number: "", otp_code: "" });
    setMessage(null);
    setSignTarget(doc);
  };

  const handleSign = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!signTarget) return;
    setActionLoading(signTarget.id);
    setMessage(null);
    try {
      await api.signDocument(signTarget.id, signForm);
      setMessage({
        type: "success",
        text: t("documents.message.sign_success", {
          title: signTarget.title,
          method: signForm.method === "EID" ? "E-ID" : "DAN",
        }),
      });
      setSignTarget(null);
      await loadData();
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("documents.message.sign_failed") });
    } finally {
      setActionLoading(null);
    }
  };

  const handleReject = async (doc: Document) => {
    if (!confirm(t("documents.message.reject_confirm", { title: doc.title }))) return;
    setActionLoading(doc.id);
    setMessage(null);
    try {
      await api.rejectDocument(doc.id);
      setMessage({ type: "success", text: t("documents.message.reject_success", { title: doc.title }) });
      await loadData();
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("documents.message.reject_failed") });
    } finally {
      setActionLoading(null);
    }
  };

  // Only the four states the module defines get a translated badge. Anything
  // else — the DRAFT the table still defaults to, or a state a later workflow
  // introduces — is shown verbatim rather than dressed up as "Pending", so the
  // screen never claims a document is awaiting signature when it is not.
  const statusBadge = (status: string) => {
    if (status === "APPROVED") {
      return (
        <span className="inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200">
          <CheckCircle className="w-3 h-3 text-emerald-500" />
          <span>{t("documents.state.approved")}</span>
        </span>
      );
    }
    if (status === "REJECTED") {
      return (
        <span className="inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-red-50 text-red-700 border border-red-200">
          <XCircle className="w-3 h-3 text-red-500" />
          <span>{t("documents.state.rejected")}</span>
        </span>
      );
    }
    if (status === "PENDING_APPROVAL") {
      return (
        <span className="inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-amber-50 text-amber-700 border border-amber-200">
          <Clock className="w-3 h-3 text-amber-500" />
          <span>{t("documents.state.pending")}</span>
        </span>
      );
    }
    if (status === "DRAFT") {
      return (
        <span className="inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">
          <FileText className="w-3 h-3 text-slate-400" />
          <span>{t("documents.state.draft")}</span>
        </span>
      );
    }
    return (
      <span className="inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">
        <span>{status}</span>
      </span>
    );
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

      {message && (
        <div
          className={`p-4 rounded-lg flex items-center space-x-3 text-sm font-medium border ${
            message.type === "success"
              ? "bg-emerald-50 border-emerald-200 text-emerald-800"
              : "bg-red-50 border-red-200 text-red-800"
          }`}
        >
          {message.type === "success" ? (
            <CheckCircle className="w-5 h-5 text-emerald-600 shrink-0" />
          ) : (
            <XCircle className="w-5 h-5 text-red-600 shrink-0" />
          )}
          <span>{message.text}</span>
        </div>
      )}

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
                  <td className="px-4 py-3">{statusBadge(doc.status)}</td>
                  <td className="px-4 py-3">
                    {doc.signed_by ? (
                      <div className="space-y-1">
                        <span className="bg-blue-50 text-blue-700 text-[11px] font-semibold px-2.5 py-0.5 rounded-full border border-blue-200 inline-flex items-center w-max space-x-1">
                          <ShieldCheck className="w-3 h-3 text-blue-500" />
                          <span>{doc.signed_by}</span>
                        </span>
                        {doc.signature_hash && (
                          <div
                            className="font-mono text-[10px] text-slate-400 truncate max-w-[220px]"
                            title={doc.signature_hash}
                          >
                            {doc.signer_reg_number ? `${doc.signer_reg_number} · ` : ""}
                            {doc.signature_hash}
                          </div>
                        )}
                      </div>
                    ) : doc.status === "REJECTED" ? (
                      <span className="text-red-400 italic">{t("documents.message.rejected_not_signed")}</span>
                    ) : (
                      <span className="text-slate-400 italic">{t("documents.state.pending_signature")}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-right">
                    {doc.status === "PENDING_APPROVAL" ? (
                      <div className="flex items-center justify-end space-x-2">
                        <button
                          onClick={() => openSignModal(doc)}
                          disabled={actionLoading === doc.id}
                          className="inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold border border-indigo-200 text-indigo-600 hover:bg-indigo-50 transition disabled:opacity-50"
                        >
                          <PenLine className="w-3.5 h-3.5" />
                          <span>{t("documents.action.sign")}</span>
                        </button>
                        <button
                          onClick={() => handleReject(doc)}
                          disabled={actionLoading === doc.id}
                          className="inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold border border-red-200 text-red-600 hover:bg-red-50 transition disabled:opacity-50"
                        >
                          <Ban className="w-3.5 h-3.5" />
                          <span>{t("documents.action.reject")}</span>
                        </button>
                      </div>
                    ) : (
                      <span className="text-slate-300 text-[11px]">—</span>
                    )}
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

      {/* E-Signature Modal */}
      {signTarget && (
        <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-xl font-bold text-slate-900 mb-1 flex items-center space-x-2">
              <PenLine className="w-5 h-5 text-indigo-600" />
              <span>{t("documents.view.sign_title")}</span>
            </h2>
            <p className="text-xs text-slate-500 mb-4 truncate">{signTarget.title}</p>

            <form onSubmit={handleSign} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                  {t("documents.field.signature_method")}
                </label>
                <div className="grid grid-cols-2 gap-2">
                  {(["EID", "DAN"] as SignMethod[]).map((m) => (
                    <button
                      key={m}
                      type="button"
                      onClick={() => setSignForm({ ...signForm, method: m })}
                      className={`py-2 rounded-lg text-xs font-semibold border transition ${
                        signForm.method === m
                          ? "bg-indigo-600 text-white border-indigo-600"
                          : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"
                      }`}
                    >
                      {m === "EID" ? "E-ID (eidmongolia.mn)" : "DAN (dan.gerege.mn)"}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("documents.field.reg_number")} *
                </label>
                <input
                  type="text"
                  placeholder="e.g. AA90010111"
                  value={signForm.reg_number}
                  onChange={(e) => setSignForm({ ...signForm, reg_number: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("documents.field.otp_code")}
                </label>
                <input
                  type="text"
                  placeholder="e.g. 123456"
                  value={signForm.otp_code}
                  onChange={(e) => setSignForm({ ...signForm, otp_code: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 font-mono"
                />
              </div>

              <div className="flex items-center space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => setSignTarget(null)}
                  className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
                >
                  {t("base.action.cancel")}
                </button>
                <button
                  type="submit"
                  disabled={actionLoading === signTarget.id}
                  className="w-1/2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 rounded-lg text-xs disabled:opacity-50"
                >
                  {actionLoading === signTarget.id ? t("documents.message.signing") : t("documents.action.sign")}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
