"use client";

import React from "react";
import { RequestDetail, TaskStatus } from "@/lib/gov";
import { useI18n } from "@/lib/i18n";
import { ChevronRight, Clock, Workflow as WorkflowIcon, X } from "lucide-react";

/** Side panel showing one request: its task tree and full timeline. */
export default function RequestDrawer({
  detail,
  onClose,
  unitName,
  statusBadge,
}: {
  detail: RequestDetail;
  onClose: () => void;
  unitName: (id?: string | null) => string;
  statusBadge: (status: TaskStatus) => React.ReactNode;
}) {
  const { t, locale } = useI18n();
  const format = (value: string) => new Date(value).toLocaleString(locale === "en" ? "en-GB" : "mn-MN");

  return (
    <div className="fixed inset-0 bg-slate-950/40 flex justify-end z-50" onClick={onClose}>
      <aside
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-xl bg-white h-full overflow-y-auto p-6 space-y-6 shadow-xl"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="font-mono text-xs text-[var(--gerege-blue)]">{detail.request.reference}</p>
            <h2 className="text-lg font-bold text-slate-900">{detail.request.service_name}</h2>
            <p className="text-sm text-slate-500">
              {detail.request.applicant_name} · {detail.request.source_system}
            </p>
          </div>
          <button onClick={onClose} aria-label={t("base.action.close")}>
            <X className="w-5 h-5 text-slate-400" />
          </button>
        </div>

        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <dt className="text-xs text-slate-500">{t("gov.field.mode")}</dt>
            <dd className="font-medium">{detail.request.fulfillment_mode}</dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500">{t("gov.field.current_unit")}</dt>
            <dd className="font-medium">{unitName(detail.request.current_unit_id)}</dd>
          </div>
        </dl>

        <section className="space-y-2">
          <h3 className="text-sm font-semibold text-slate-700 flex items-center gap-2">
            <WorkflowIcon className="w-4 h-4" />
            {t("gov.view.tasks")}
          </h3>
          {detail.tasks.map((task) => (
            <div key={task.id} className="p-3 border border-slate-200 rounded-lg text-sm flex items-center gap-3">
              {task.parent_task_id && <ChevronRight className="w-4 h-4 text-slate-300 shrink-0" />}
              <div className="flex-1">
                <p className="font-medium text-slate-800">{task.unit_name || unitName(task.unit_id)}</p>
                <p className="text-xs text-slate-500">
                  {task.step_code}
                  {task.due_at ? ` · ${format(task.due_at)}` : ""}
                </p>
              </div>
              {task.overdue && <Clock className="w-4 h-4 text-red-500" />}
              {statusBadge(task.status)}
            </div>
          ))}
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-slate-700">{t("gov.view.timeline")}</h3>
          <ol className="space-y-3">
            {detail.timeline.map((event) => (
              <li key={event.id} className="flex gap-3 text-sm">
                <span className="mt-1.5 w-2 h-2 rounded-full bg-[var(--gerege-blue)] shrink-0" />
                <div>
                  <p className="text-slate-800">
                    {event.message || event.event_type}
                    {event.from_status && event.to_status ? (
                      <span className="text-xs text-slate-400">
                        {" "}
                        · {event.from_status} → {event.to_status}
                      </span>
                    ) : null}
                  </p>
                  <p className="text-xs text-slate-400">
                    {format(event.created_at)}
                    {event.unit_name ? ` · ${event.unit_name}` : ""}
                    {event.actor ? ` · ${event.actor}` : ""}
                  </p>
                </div>
              </li>
            ))}
          </ol>
        </section>
      </aside>
    </div>
  );
}
