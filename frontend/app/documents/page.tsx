"use client";

import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { FileText, Plus, CheckCircle, ShieldCheck, Clock } from "lucide-react";

export default function DocumentsPage() {
  const [documents, setDocuments] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState({ title: "", doc_type: "CONTRACT" });

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
      alert("Failed to create document: " + err.message);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
            <FileText className="w-7 h-7 text-indigo-600" />
            <span>Digital Documents & E-Signatures</span>
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
          <span>Create Document</span>
        </button>
      </div>

      {loading ? (
        <div className="py-12 text-center text-slate-400">Loading documents...</div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase">
              <tr>
                <th className="px-4 py-3">Document Title</th>
                <th className="px-4 py-3">Type</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Digital Signature (E-ID / DAN)</th>
                <th className="px-4 py-3">Created Date</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {documents.map((doc) => (
                <tr key={doc.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 font-semibold text-slate-900">{doc.title}</td>
                  <td className="px-4 py-3 font-mono text-slate-600">{doc.doc_type}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full ${
                        doc.status === "APPROVED"
                          ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                          : "bg-amber-50 text-amber-700 border border-amber-200"
                      }`}
                    >
                      {doc.status === "APPROVED" ? (
                        <CheckCircle className="w-3 h-3 text-emerald-500" />
                      ) : (
                        <Clock className="w-3 h-3 text-amber-500" />
                      )}
                      <span>{doc.status}</span>
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {doc.signed_by ? (
                      <span className="bg-blue-50 text-blue-700 text-[11px] font-semibold px-2.5 py-0.5 rounded-full border border-blue-200 flex items-center w-max space-x-1">
                        <ShieldCheck className="w-3 h-3 text-blue-500" />
                        <span>{doc.signed_by}</span>
                      </span>
                    ) : (
                      <span className="text-slate-400 italic">Pending Signature</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-400">{new Date(doc.created_at).toLocaleDateString()}</td>
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
            <h2 className="text-xl font-bold text-slate-900 mb-4">Create Digital Document</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Document Title *</label>
                <input
                  type="text"
                  placeholder="e.g. Хамтран ажиллах гэрээ 2026"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Document Category</label>
                <select
                  value={form.doc_type}
                  onChange={(e) => setForm({ ...form, doc_type: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="CONTRACT">Legal Contract</option>
                  <option value="REQUEST">Official Request</option>
                  <option value="APPROVAL">Internal Approval</option>
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
                >
                  Create Document
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
