"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Banner } from "@/components/ui";
import { useAccess } from "@/lib/permissions";
import { Building2 } from "lucide-react";

/**
 * The organisation as it is, rather than as it is labelled.
 *
 * The name in the sidebar has always existed; none of the rest has, and every
 * app that has to print who this organisation is — a signed document, an
 * invoice, a request to a ministry — has been going without it.
 */
type Organisation = Awaited<ReturnType<typeof api.getOrganisation>>;

// Grouped the way somebody fills them in, not the way they are stored.
const GROUPS: Array<{ title: string; fields: Array<{ key: keyof Organisation; label: string; placeholder?: string }> }> = [
  {
    title: "core.group.identity",
    fields: [
      { key: "name", label: "core.field.name" },
      { key: "legal_name", label: "core.field.legal_name" },
      { key: "registration_number", label: "core.field.registration_number", placeholder: "1234567" },
      { key: "tax_number", label: "core.field.tax_number" },
    ],
  },
  {
    title: "core.group.address",
    fields: [
      { key: "province", label: "core.field.province" },
      { key: "district", label: "core.field.district" },
      { key: "khoroo", label: "core.field.khoroo" },
      { key: "address_line", label: "core.field.address_line" },
      { key: "postal_code", label: "core.field.postal_code" },
    ],
  },
  {
    title: "core.group.contact",
    fields: [
      { key: "phone", label: "base.field.phone" },
      { key: "email", label: "base.field.email" },
      { key: "website", label: "core.field.website" },
      { key: "logo_url", label: "core.field.logo_url" },
    ],
  },
  {
    title: "core.group.defaults",
    fields: [
      { key: "timezone", label: "core.field.timezone", placeholder: "Asia/Ulaanbaatar" },
      { key: "locale", label: "base.field.language", placeholder: "mn" },
      { key: "currency", label: "core.field.currency", placeholder: "MNT" },
    ],
  },
];

export default function OrganisationPage() {
  const { t } = useI18n();
  const { allowed: canManage } = useAccess("core.manage");
  const [organisation, setOrganisation] = useState<Organisation | null>(null);
  const [draft, setDraft] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    try {
      setOrganisation(await api.getOrganisation());
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  // Only what was touched is sent. The server merges field by field, so an
  // edit to a phone number cannot blank a registration number.
  const save = async () => {
    setSaving(true);
    setMessage(null);
    try {
      await api.updateOrganisation(draft);
      setDraft({});
      await load();
      setMessage({ type: "success", text: t("core.message.saved") });
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setSaving(false);
    }
  };

  if (!organisation) {
    return <div className="py-12 text-center text-slate-500 text-sm">{t("base.message.loading")}</div>;
  }

  const value = (key: keyof Organisation) => draft[key] ?? (organisation[key] as string) ?? "";

  return (
    <div className="space-y-6">
      <div className="border-b border-slate-200 pb-4 flex items-center gap-3">
        <Building2 className="w-6 h-6 text-slate-600" />
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{t("core.view.organisation_title")}</h1>
          <p className="text-sm text-slate-500">{t("core.view.organisation_subtitle")}</p>
        </div>
      </div>

      {message && <Banner tone={message.type} message={message.text} onDismiss={() => setMessage(null)} />}
      {!canManage && <Banner tone="info" message={t("base.message.read_only") + " core.manage"} />}

      <div className="grid gap-6 md:grid-cols-2">
        {GROUPS.map((group) => (
          <section key={group.title} className="bg-white border border-slate-200 rounded-xl p-5 space-y-3">
            <h2 className="font-semibold text-slate-800">{t(group.title as any)}</h2>
            {group.fields.map((field) => (
              <label key={field.key} className="block">
                <span className="block text-xs font-medium text-slate-500 mb-1">{t(field.label as any)}</span>
                <input
                  value={value(field.key)}
                  placeholder={field.placeholder}
                  disabled={!canManage}
                  onChange={(e) => setDraft((d) => ({ ...d, [field.key]: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-slate-50 disabled:text-slate-500"
                />
              </label>
            ))}
          </section>
        ))}
      </div>

      {canManage && (
        <div className="flex items-center gap-3">
          <button
            onClick={save}
            disabled={saving || Object.keys(draft).length === 0}
            className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium text-sm py-2 px-5 rounded-lg"
          >
            {saving ? t("base.message.saving") : t("base.action.save")}
          </button>
          {Object.keys(draft).length > 0 && (
            <button onClick={() => setDraft({})} className="text-sm text-slate-500 hover:text-slate-700">
              {t("base.action.cancel")}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
