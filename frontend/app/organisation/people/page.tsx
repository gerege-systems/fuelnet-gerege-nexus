"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Banner } from "@/components/ui";
import { useAccess } from "@/lib/permissions";
import { Users, UserCheck, UserX } from "lucide-react";

type Person = Awaited<ReturnType<typeof api.getPeople>>[number];
type Department = Awaited<ReturnType<typeof api.getDepartments>>[number];

/**
 * Who works here.
 *
 * The name and the email belong to the person and are the same in every
 * organisation they belong to, so they are shown and not editable here. The job
 * title and the department belong to this organisation's view of them, and are.
 */
export default function PeoplePage() {
  const { t } = useI18n();
  const { allowed: canManage } = useAccess("organisation.manage");
  const [people, setPeople] = useState<Person[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  // Rows belonging to another organisation are shown but not editable: a write
  // lands in the organisation being acted in, and the policies refuse anything
  // else — so the controls say so rather than failing on save.
  const [currentTenant, setCurrentTenant] = useState("");
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    try {
      const [staff, units, me] = await Promise.all([
        api.getPeople(),
        api.getDepartments(),
        api.getTenants().catch(() => null),
      ]);
      setCurrentTenant(me?.current || "");
      setPeople(staff || []);
      setDepartments((units || []).filter((d) => d.active));
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const patch = async (person: Person, body: { job_title?: string; department_id?: string }) => {
    setBusy(person.membership_id);
    setMessage(null);
    try {
      await api.updatePerson(person.membership_id, body);
      await load();
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setBusy(null);
    }
  };

  const setActive = async (person: Person, active: boolean) => {
    setBusy(person.membership_id);
    setMessage(null);
    try {
      await api.setPersonActive(person.membership_id, active);
      await load();
    } catch (err: any) {
      // The server refuses two things here — deactivating yourself, and
      // deactivating the last administrator — and says which. Passing the
      // message through is the whole point of it being a sentence.
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setBusy(null);
    }
  };

  // Shown only when the session is actually reading across more than one: a
  // column repeating the same organisation on every row is noise.
  const spanning = new Set(people.map((p) => p.tenant_id)).size > 1;

  return (
    <div className="space-y-6">
      <div className="border-b border-slate-200 pb-4 flex items-center gap-3">
        <Users className="w-6 h-6 text-slate-600" />
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{t("core.view.people_title")}</h1>
          <p className="text-sm text-slate-500">{t("core.view.people_subtitle")}</p>
        </div>
      </div>

      {message && <Banner tone={message.type} message={message.text} onDismiss={() => setMessage(null)} />}

      <div className="bg-white border border-slate-200 rounded-xl overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="bg-slate-50 text-xs uppercase tracking-wider text-slate-600">
            <tr>
              <th className="py-3 px-4">{t("base.field.name")}</th>
              {spanning && <th className="py-3 px-4">{t("core.field.organisation")}</th>}
              <th className="py-3 px-4">{t("core.field.job_title")}</th>
              <th className="py-3 px-4">{t("core.field.department")}</th>
              <th className="py-3 px-4">{t("core.field.roles")}</th>
              <th className="py-3 px-4 text-right">{t("base.field.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {people.map((person) => (
              <tr key={person.membership_id} className={person.active ? "" : "opacity-60"}>
                <td className="py-3 px-4">
                  <div className="font-semibold text-slate-900">{person.name}</div>
                  <div className="text-xs text-slate-500">{person.email}</div>
                </td>
                {spanning && <td className="py-3 px-4 text-slate-600">{person.tenant_name}</td>}
                <td className="py-3 px-4">
                  <input
                    defaultValue={person.job_title}
                    disabled={!canManage || busy === person.membership_id || person.tenant_id !== currentTenant}
                    onBlur={(e) =>
                      e.target.value !== person.job_title && patch(person, { job_title: e.target.value })
                    }
                    placeholder="—"
                    className="w-40 px-2 py-1 text-sm border border-transparent hover:border-slate-300 focus:border-indigo-500 rounded focus:outline-none disabled:bg-transparent"
                  />
                </td>
                <td className="py-3 px-4">
                  <select
                    value={person.department_id || ""}
                    disabled={!canManage || busy === person.membership_id || person.tenant_id !== currentTenant}
                    onChange={(e) => patch(person, { department_id: e.target.value })}
                    className="px-2 py-1 text-sm border border-slate-200 rounded bg-white disabled:bg-transparent disabled:border-transparent"
                  >
                    <option value="">—</option>
                    {departments.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.name}
                      </option>
                    ))}
                    {/* Somebody left in a department that has since been
                        archived still belongs to it. Without this option the
                        select would fall back to "—" and read as though they
                        had already been moved out. */}
                    {person.department_id && !departments.some((d) => d.id === person.department_id) && (
                      <option value={person.department_id}>{person.department_name}</option>
                    )}
                  </select>
                </td>
                <td className="py-3 px-4">
                  <div className="flex flex-wrap gap-1">
                    {person.is_admin && (
                      <span className="text-xs px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 border border-indigo-200">
                        {t("core.label.admin")}
                      </span>
                    )}
                    {person.roles.map((role) => (
                      <span key={role} className="text-xs px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
                        {role}
                      </span>
                    ))}
                  </div>
                </td>
                <td className="py-3 px-4 text-right">
                  {canManage && (
                    <button
                      onClick={() => setActive(person, !person.active)}
                      disabled={busy === person.membership_id}
                      className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold border transition ${
                        person.active
                          ? "border-slate-200 text-slate-600 hover:bg-slate-50"
                          : "border-emerald-600 bg-emerald-600 text-white hover:bg-emerald-700"
                      }`}
                    >
                      {person.active ? <UserX className="w-3.5 h-3.5" /> : <UserCheck className="w-3.5 h-3.5" />}
                      {person.active ? t("core.action.deactivate") : t("core.action.reactivate")}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {people.length === 0 && (
          <div className="py-10 text-center text-sm text-slate-500">{t("base.message.no_data")}</div>
        )}
      </div>

      {/* Adding somebody is not here on purpose: a person joins an organisation
          by being invited to an account, which is Access control's business and
          already has a screen. This one is about who is already here. */}
      <p className="text-xs text-slate-500">{t("core.message.people_hint")}</p>
    </div>
  );
}
