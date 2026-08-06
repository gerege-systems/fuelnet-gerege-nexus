"use client";

import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAccess } from "@/lib/access";
import { useI18n } from "@/lib/i18n";
import { ActionMessage, Banner, SectionHeader } from "@/components/documents/shared";
import { Files, Plus, Trash2, Wand2 } from "lucide-react";

interface Template {
  id: string;
  name: string;
  doc_type: string;
  title_pattern: string;
  active: boolean;
  created_at: string;
}

const DOC_TYPES = ["CONTRACT", "REQUEST", "APPROVAL"] as const;

/**
 * Document templates: the presets a document is started from. A document is a
 * title and a type, so that is what a template carries — the title may hold
 * {year}, {month} or {date}, which are filled in when the template is used.
 */
export default function DocumentTemplatesPage() {
  const { t } = useI18n();
  const { can } = useAccess();
  const mayManage = can("documents.manage");

  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState<ActionMessage | null>(null);
  const [form, setForm] = useState({ name: "", doc_type: "CONTRACT", title_pattern: "" });

  const loadData = async () => {
    setLoading(true);
    try {
      setTemplates((await api.getDocumentTemplates()) || []);
    } catch (err: any) {
      setMessage({ type: "error", text: err?.message || t("documents.message.templates_failed") });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy("create");
    setMessage(null);
    try {
      await api.createDocumentTemplate(form);
      setForm({ name: "", doc_type: "CONTRACT", title_pattern: "" });
      setMessage({ type: "success", text: t("documents.message.template_saved") });
      await loadData();
    } catch (err: any) {
      setMessage({ type: "error", text: err?.message || t("documents.message.templates_failed") });
    } finally {
      setBusy(null);
    }
  };

  const handleUse = async (tpl: Template) => {
    setBusy(tpl.id);
    setMessage(null);
    try {
      const doc: any = await api.useDocumentTemplate(tpl.id);
      setMessage({
        type: "success",
        text: t("documents.message.template_used", { title: doc?.title || tpl.name }),
      });
    } catch (err: any) {
      setMessage({ type: "error", text: err?.message || t("documents.message.templates_failed") });
    } finally {
      setBusy(null);
    }
  };

  const handleDelete = async (tpl: Template) => {
    if (!confirm(t("documents.message.template_delete_confirm", { name: tpl.name }))) return;
    setBusy(tpl.id);
    setMessage(null);
    try {
      await api.deleteDocumentTemplate(tpl.id);
      setMessage({ type: "success", text: t("documents.message.template_deleted", { name: tpl.name }) });
      await loadData();
    } catch (err: any) {
      setMessage({ type: "error", text: err?.message || t("documents.message.templates_failed") });
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={<Files className="w-7 h-7 text-indigo-600" />}
        title={t("documents.menu.templates")}
        subtitle={t("documents.view.templates_hint")}
      />

      {message && <Banner message={message} onDismiss={() => setMessage(null)} />}

      {mayManage && (
        <form onSubmit={handleCreate} className="bg-white border border-slate-200 rounded-xl p-4 grid gap-3 md:grid-cols-4">
          <div className="md:col-span-1">
            <label className="block text-xs font-semibold text-slate-700 mb-1">{t("documents.field.template_name")}</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>
          <div className="md:col-span-1">
            <label className="block text-xs font-semibold text-slate-700 mb-1">{t("documents.field.category")}</label>
            <select
              value={form.doc_type}
              onChange={(e) => setForm({ ...form, doc_type: e.target.value })}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
            >
              {DOC_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
          </div>
          <div className="md:col-span-1">
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              {t("documents.field.title_pattern")}
            </label>
            <input
              type="text"
              placeholder="Хамтран ажиллах гэрээ {year}"
              value={form.title_pattern}
              onChange={(e) => setForm({ ...form, title_pattern: e.target.value })}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>
          <div className="md:col-span-1 flex items-end">
            <button
              type="submit"
              disabled={busy === "create"}
              className="w-full bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center justify-center space-x-2 disabled:opacity-50"
            >
              <Plus className="w-4 h-4" />
              <span>{t("documents.action.add_template")}</span>
            </button>
          </div>
          <p className="md:col-span-4 text-[11px] text-slate-500">{t("documents.message.title_pattern_hint")}</p>
        </form>
      )}

      {loading ? (
        <div className="py-12 text-center text-slate-400">{t("documents.message.loading")}</div>
      ) : templates.length === 0 ? (
        <div className="bg-white border border-slate-200 rounded-xl p-8 text-center text-slate-500 text-sm">
          {t("documents.message.no_templates")}
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <table className="w-full text-left text-xs text-slate-600">
            <thead className="bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase">
              <tr>
                <th className="px-4 py-3">{t("documents.field.template_name")}</th>
                <th className="px-4 py-3">{t("base.field.type")}</th>
                <th className="px-4 py-3">{t("documents.field.title_pattern")}</th>
                <th className="px-4 py-3 text-right">{t("base.field.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {templates.map((tpl) => (
                <tr key={tpl.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 font-semibold text-slate-900">{tpl.name}</td>
                  <td className="px-4 py-3 font-mono text-slate-600">{tpl.doc_type}</td>
                  <td className="px-4 py-3 font-mono text-slate-500">{tpl.title_pattern}</td>
                  <td className="px-4 py-3 text-right">
                    {mayManage ? (
                      <div className="flex items-center justify-end space-x-2">
                        <button
                          onClick={() => handleUse(tpl)}
                          disabled={busy === tpl.id}
                          className="inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold border border-indigo-200 text-indigo-600 hover:bg-indigo-50 disabled:opacity-50"
                        >
                          <Wand2 className="w-3.5 h-3.5" />
                          <span>{t("documents.action.use_template")}</span>
                        </button>
                        <button
                          onClick={() => handleDelete(tpl)}
                          disabled={busy === tpl.id}
                          className="inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold border border-red-200 text-red-600 hover:bg-red-50 disabled:opacity-50"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                          <span>{t("base.action.delete")}</span>
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
    </div>
  );
}
