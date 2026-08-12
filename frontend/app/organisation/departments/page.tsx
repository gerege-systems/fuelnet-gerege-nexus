"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Banner } from "@/components/ui";
import { useAccess } from "@/lib/permissions";
import { Network, Archive, ArchiveRestore, Plus, Pencil, Trash2, Check, X } from "lucide-react";

type Department = Awaited<ReturnType<typeof api.getDepartments>>[number];
type Person = Awaited<ReturnType<typeof api.getPeople>>[number];

/** A unit with its children, which is the shape the screen actually draws. */
type Node = Department & { children: Node[] };

/**
 * How the organisation is arranged.
 *
 * Drawn as a tree because that is what a department structure is. It used to be
 * a flat list indented by depth, which reads as a tree only if you already know
 * the answer: nothing said which unit a row hung from, and two siblings looked
 * exactly like a parent and its child.
 */
export default function DepartmentsPage() {
  const { t } = useI18n();
  const { allowed: canManage } = useAccess("core.manage");
  const [departments, setDepartments] = useState<Department[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [draft, setDraft] = useState({ code: "", name: "", parent_id: "" });
  const [editing, setEditing] = useState<{ id: string; name: string; parent_id: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [currentTenant, setCurrentTenant] = useState("");
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    try {
      const [units, staff, me] = await Promise.all([
        api.getDepartments(),
        api.getPeople(),
        api.getTenants().catch(() => null),
      ]);
      setCurrentTenant(me?.current || "");
      setDepartments(units || []);
      setPeople((staff || []).filter((p) => p.active));
    } catch (err: any) {
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const active = useMemo(() => departments.filter((d) => d.active), [departments]);
  const archived = useMemo(() => departments.filter((d) => !d.active), [departments]);

  // Built once per change rather than per row: the old screen walked up the
  // parent chain for every unit it drew, which is the same tree rebuilt as many
  // times as there are branches on it.
  // One tree per organisation when the session reads across more than one.
  // Merging them would be wrong rather than merely confusing: two organisations
  // have two structures, and a unit in one never reports to a unit in the other.
  const spanning = useMemo(() => new Set(departments.map((d) => d.tenant_id)).size > 1, [departments]);

  const roots = useMemo(() => {
    const byID = new Map<string, Node>(active.map((d) => [d.id, { ...d, children: [] }]));
    const top: Node[] = [];
    for (const node of byID.values()) {
      // A unit whose parent is archived hangs at the top rather than vanishing
      // with it — otherwise archiving one unit would hide a whole branch.
      const parent = node.parent_id ? byID.get(node.parent_id) : undefined;
      (parent ? parent.children : top).push(node);
    }
    const sort = (nodes: Node[]) => {
      nodes.sort((a, b) => a.name.localeCompare(b.name));
      nodes.forEach((n) => sort(n.children));
    };
    sort(top);
    return top;
  }, [active]);

  // Everything below a unit, so the parent selector cannot offer a move that
  // would tie the tree in a knot. The server refuses it too; this is what stops
  // the screen offering it in the first place.
  const descendantsOf = useCallback(
    (id: string) => {
      const out = new Set<string>([id]);
      let grew = true;
      while (grew) {
        grew = false;
        for (const d of active) {
          if (d.parent_id && out.has(d.parent_id) && !out.has(d.id)) {
            out.add(d.id);
            grew = true;
          }
        }
      }
      return out;
    },
    [active],
  );

  const run = async (action: () => Promise<unknown>, success?: string) => {
    setBusy(true);
    setMessage(null);
    try {
      await action();
      await load();
      if (success) setMessage({ type: "success", text: success });
    } catch (err: any) {
      // The server refuses three things in sentences — a foreign parent, a loop,
      // a unit that still holds people — and each sentence is the explanation.
      setMessage({ type: "error", text: err.message || t("base.message.error") });
    } finally {
      setBusy(false);
    }
  };

  const create = () =>
    run(async () => {
      await api.createDepartment({
        code: draft.code.trim(),
        name: draft.name.trim(),
        parent_id: draft.parent_id || undefined,
      });
      setDraft({ code: "", name: "", parent_id: "" });
    });

  const saveEdit = () =>
    run(async () => {
      if (!editing) return;
      const unit = active.find((d) => d.id === editing.id);
      await api.updateDepartment(editing.id, {
        name: editing.name.trim(),
        parent_id: editing.parent_id || undefined,
        manager_membership_id: unit?.manager_membership_id || undefined,
      });
      setEditing(null);
    });

  const remove = (unit: Node) => {
    if (!window.confirm(t("core.message.confirm_delete", { name: unit.name }))) return;
    return run(() => api.deleteDepartment(unit.id));
  };

  const row = (node: Node, depth: number, isLast: boolean, guides: boolean[]) => {
    const isEditing = editing?.id === node.id;
    const blocked = descendantsOf(node.id);
    return (
      <React.Fragment key={node.id}>
        <div className="flex flex-wrap items-center gap-3 py-3 px-4 hover:bg-slate-50/70">
          {/* The tree guides. Drawn as boxes rather than characters so they line
              up at any font, and marked aria-hidden because the nesting is
              already carried by the parent selector on each row. */}
          <div className="flex items-stretch self-stretch shrink-0" aria-hidden>
            {guides.map((drawn, i) => (
              <span key={i} className={`w-5 ${drawn ? "border-s border-slate-200" : ""}`} />
            ))}
            {depth > 0 && (
              <span className="relative w-5">
                <span className={`absolute inset-y-0 start-0 border-s border-slate-200 ${isLast ? "h-1/2" : ""}`} />
                <span className="absolute top-1/2 start-0 w-3 border-t border-slate-200" />
              </span>
            )}
          </div>

          {isEditing ? (
            <input
              autoFocus
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              onKeyDown={(e) => {
                if (e.key === "Enter") void saveEdit();
                if (e.key === "Escape") setEditing(null);
              }}
              className="flex-1 min-w-40 px-2 py-1 text-sm border border-indigo-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          ) : (
            <div className="flex-1 min-w-40">
              <div className="font-semibold text-slate-900">{node.name}</div>
              <div className="text-xs text-slate-500">
                <code>{node.code}</code> · {t("core.field.people_count", { count: node.people_count })}
              </div>
            </div>
          )}

          {isEditing ? (
            <label className="w-52">
              <span className="block text-[11px] font-medium text-slate-500">{t("core.field.parent")}</span>
              <select
                value={editing.parent_id}
                onChange={(e) => setEditing({ ...editing, parent_id: e.target.value })}
                className="w-full px-2 py-1.5 text-sm border border-slate-200 rounded bg-white"
              >
                <option value="">—</option>
                {/* Itself and everything below it are left out: moving a unit
                    under its own child is the one move that would make the tree
                    unreadable, and offering it only to refuse it is a worse way
                    to say so. */}
                {active
                  .filter((d) => !blocked.has(d.id))
                  .map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name}
                    </option>
                  ))}
              </select>
            </label>
          ) : (
            <label className="w-52">
              <span className="block text-[11px] font-medium text-slate-500">{t("core.field.manager")}</span>
              <select
                value={node.manager_membership_id || ""}
                disabled={!canManage || busy || node.tenant_id !== currentTenant}
                onChange={(e) =>
                  run(() =>
                    api.updateDepartment(node.id, {
                      name: node.name,
                      parent_id: node.parent_id || undefined,
                      manager_membership_id: e.target.value || undefined,
                    }),
                  )
                }
                className="w-full px-2 py-1.5 text-sm border border-slate-200 rounded bg-white disabled:bg-transparent disabled:border-transparent"
              >
                <option value="">—</option>
                {people.map((p) => (
                  <option key={p.membership_id} value={p.membership_id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </label>
          )}

          {/* Only the organisation being acted in can be edited: a write lands
              there and the policies refuse anything else, so the controls are
              absent rather than present and failing. */}
          {canManage && node.tenant_id === currentTenant && (
            <div className="flex items-center gap-1.5 shrink-0">
              {isEditing ? (
                <>
                  <button
                    onClick={saveEdit}
                    disabled={busy || !editing.name.trim()}
                    title={t("base.action.save")}
                    className="p-1.5 rounded-lg border border-emerald-600 bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-50"
                  >
                    <Check className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => setEditing(null)}
                    title={t("base.action.cancel")}
                    className="p-1.5 rounded-lg border border-slate-200 text-slate-500 hover:bg-white"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                </>
              ) : (
                <>
                  <button
                    onClick={() => setEditing({ id: node.id, name: node.name, parent_id: node.parent_id || "" })}
                    disabled={busy}
                    title={t("base.action.edit")}
                    className="p-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-white"
                  >
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => run(() => api.archiveDepartment(node.id))}
                    disabled={busy}
                    title={t("core.action.archive")}
                    className="p-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-white"
                  >
                    <Archive className="w-3.5 h-3.5" />
                  </button>
                  {/* Delete is last and is the only red control on the row.
                      The server refuses it while anything points at the unit,
                      so what is left here is the one it is for: a unit created
                      by mistake. */}
                  <button
                    onClick={() => remove(node)}
                    disabled={busy}
                    title={t("base.action.delete")}
                    className="p-1.5 rounded-lg border border-red-200 text-red-600 hover:bg-red-50"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </>
              )}
            </div>
          )}
        </div>
        {node.children.map((child, i) =>
          row(child, depth + 1, i === node.children.length - 1, [...guides, depth > 0 ? !isLast : false]),
        )}
      </React.Fragment>
    );
  };

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

      {spanning ? (
        Array.from(new Set(roots.map((r) => r.tenant_id))).map((tenantID) => {
          const owned = roots.filter((r) => r.tenant_id === tenantID);
          return (
            <div key={tenantID} className="space-y-2">
              <h2 className="text-sm font-semibold text-slate-700">{owned[0]?.tenant_name}</h2>
              <div className="bg-white border border-slate-200 rounded-xl divide-y divide-slate-100">
                {owned.map((node, i) => row(node, 0, i === owned.length - 1, []))}
              </div>
            </div>
          );
        })
      ) : (
        <div className="bg-white border border-slate-200 rounded-xl divide-y divide-slate-100">
          {roots.map((node, i) => row(node, 0, i === roots.length - 1, []))}
          {roots.length === 0 && (
            <div className="py-10 text-center text-sm text-slate-500">{t("base.message.no_data")}</div>
          )}
        </div>
      )}

      {archived.length > 0 && (
        <details className="text-sm">
          {/* Archived rather than deleted, so the people and the history that
              point at them stay readable. */}
          <summary className="cursor-pointer text-slate-500">
            {t("core.view.archived", { count: archived.length })}
          </summary>
          <ul className="mt-2 space-y-1 text-slate-500">
            {archived.map((unit) => (
              <li key={unit.id} className="flex items-center gap-2 py-0.5">
                <span className="flex-1">
                  {unit.name} <code className="text-xs">{unit.code}</code>
                </span>
                {canManage && (
                  <button
                    onClick={() => run(() => api.restoreDepartment(unit.id))}
                    disabled={busy}
                    className="px-2.5 py-1 rounded-lg text-xs font-semibold border border-slate-200 text-slate-600 hover:bg-white inline-flex items-center gap-1.5 disabled:opacity-50"
                  >
                    <ArchiveRestore className="w-3.5 h-3.5" />
                    {t("core.action.restore")}
                  </button>
                )}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}
