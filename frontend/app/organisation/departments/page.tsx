"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Banner } from "@/components/ui";
import { useAccess } from "@/lib/permissions";
import { Network, Archive, Plus } from "lucide-react";

type Department = Awaited<ReturnType<typeof api.getDepartments>>[number];
type Person = Awaited<ReturnType<typeof api.getPeople>>[number];

/**
 * How the organisation is arranged.
 *
 * Drawn as a tree rather than a list, because that is what a department
 * structure is and a flat table makes the reader rebuild it in their head. The
 * parent selector deliberately never offers a descendant: the server refuses a
 * department that reports to itself, and the screen is what keeps the deeper
 * cycles from being offered in the first place.
 */
export default function DepartmentsPage() {
  const { t } = useI18n();
  const { allowed: canManage } = useAccess("core.manage");
  const [departments, setDepartments] = useState<Department[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [draft, setDraft] = useState({ code: "", name: "", parent_id: "" });
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    try {
      const [units, staff] = await Promise.all([api.getDepartments(), api.getPeople()]);
      setDepartments(units || []);
      setPeople((staff || []).filter((p) => p.active));
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    setBusy(true);
    setMessage(null);
    try {
      await api.createDepartment({
        code: draft.code.trim(),
        name: draft.name.trim(),
        parent_id: draft.parent_id || undefined,
      });
      setDraft({ code: "", name: "", parent_id: "" });
      await load();
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setBusy(false);
    }
  };

  const assignManager = async (unit: Department, membershipID: string) => {
    setBusy(true);
    try {
      await api.updateDepartment(unit.id, {
        name: unit.name,
        parent_id: unit.parent_id || undefined,
        manager_membership_id: membershipID || undefined,
      });
      await load();
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setBusy(false);
    }
  };

  const archive = async (unit: Department) => {
    setBusy(true);
    try {
      await api.archiveDepartment(unit.id);
      await load();
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setBusy(false);
    }
  };

  // Depth by walking up: the list is small and this keeps the tree honest
  // without asking the server for a nested shape it would have to flatten again.
  const depthOf = (unit: Department): number => {
    let depth = 0;
    let current = unit;
    while (current.parent_id) {
      const parent = departments.find((d) => d.id === current.parent_id);
      if (!parent || depth > 8) break;
      current = parent;
      depth += 1;
    }
    return depth;
  };

  const active = departments.filter((d) => d.active);
  const archived = departments.filter((d) => !d.active);

  return (
    <div className="space-y-6">
      <div className="border-b border-slate-200 pb-4 flex items-center gap-3">
        <Network className="w-6 h-6 text-slate-600" />
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{t("core.view.departments_title")}</h1>
          <p className="text-sm text-slate-500">{t("core.view.departments_subtitle")}</p>
        </div>
      </div>

      {message && <Banner tone={message.type} message={message.text} onDismiss={() => setMessage(null)} />}

      {canManage && (
        <div className="bg-white border border-slate-200 rounded-xl p-4 flex flex-wrap items-end gap-3">
          <label className="flex-1 min-w-40">
            <span className="block text-xs font-medium text-slate-500 mb-1">{t("base.field.name")}</span>
            <input
              value={draft.name}
              onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </label>
          <label className="w-40">
            <span className="block text-xs font-medium text-slate-500 mb-1">{t("core.field.code")}</span>
            <input
              value={draft.code}
              onChange={(e) => setDraft((d) => ({ ...d, code: e.target.value }))}
              placeholder="sales"
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </label>
          <label className="w-48">
            <span className="block text-xs font-medium text-slate-500 mb-1">{t("core.field.parent")}</span>
            <select
              value={draft.parent_id}
              onChange={(e) => setDraft((d) => ({ ...d, parent_id: e.target.value }))}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white"
            >
              <option value="">—</option>
              {active.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
          <button
            onClick={create}
            disabled={busy || !draft.name.trim() || !draft.code.trim()}
            className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium text-sm py-2 px-4 rounded-lg inline-flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            {t("base.action.create")}
          </button>
        </div>
      )}

      <div className="bg-white border border-slate-200 rounded-xl divide-y divide-slate-100">
        {active.map((unit) => (
          <div key={unit.id} className="p-4 flex flex-wrap items-center gap-3">
            <div className="flex-1 min-w-48" style={{ paddingInlineStart: depthOf(unit) * 20 }}>
              <div className="font-semibold text-slate-900">{unit.name}</div>
              <div className="text-xs text-slate-500">
                <code>{unit.code}</code> · {t("core.field.people_count", { count: unit.people_count })}
              </div>
            </div>
            <label className="w-56">
              <span className="block text-xs font-medium text-slate-500 mb-1">{t("core.field.manager")}</span>
              <select
                value={unit.manager_membership_id || ""}
                disabled={!canManage || busy}
                onChange={(e) => assignManager(unit, e.target.value)}
                className="w-full px-2 py-1.5 text-sm border border-slate-200 rounded bg-white disabled:bg-transparent"
              >
                <option value="">—</option>
                {people.map((p) => (
                  <option key={p.membership_id} value={p.membership_id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </label>
            {canManage && (
              <button
                onClick={() => archive(unit)}
                disabled={busy}
                title={t("core.action.archive")}
                className="px-3 py-1.5 rounded-lg text-xs font-semibold border border-slate-200 text-slate-600 hover:bg-slate-50 inline-flex items-center gap-1.5"
              >
                <Archive className="w-3.5 h-3.5" />
                {t("core.action.archive")}
              </button>
            )}
          </div>
        ))}
        {active.length === 0 && (
          <div className="py-10 text-center text-sm text-slate-500">{t("base.message.no_data")}</div>
        )}
      </div>

      {archived.length > 0 && (
        <details className="text-sm">
          {/* Archived rather than deleted, so the people and the history that
              point at them stay readable. */}
          <summary className="cursor-pointer text-slate-500">
            {t("core.view.archived", { count: archived.length })}
          </summary>
          <ul className="mt-2 space-y-1 text-slate-500">
            {archived.map((unit) => (
              <li key={unit.id}>
                {unit.name} <code className="text-xs">{unit.code}</code>
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}
