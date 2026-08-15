"use client";

/**
 * What the four Өртөө screens share.
 *
 * A queue is a queue whichever way it points, so the list, the status chip and
 * the deadline live here once and the incoming and outgoing screens differ only
 * in which end of the link they name and what they let you do.
 */

import Link from "next/link";
import { AlertTriangle } from "lucide-react";

import type { UrtuuTask } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, Empty, Panel } from "@/components/module/kit";

/** The seven statuses, coloured by what they mean rather than by order. */
const STATUS_TONE: Record<string, "slate" | "blue" | "amber" | "emerald" | "rose"> = {
  RECEIVED: "amber",
  ACCEPTED: "blue",
  IN_PROGRESS: "blue",
  DELEGATED: "blue",
  COMPLETED: "emerald",
  RETURNED: "rose",
  CLOSED: "slate",
};

/**
 * useStatusLabel keeps the dynamic key out of t().
 *
 * The dictionary key type is a union of literals, so a template string would
 * defeat the one check that catches a renamed key. The lookup table is written
 * out instead — seven lines that fail to compile if a status is added to the
 * contract and not to the dictionary.
 */
export function useStatusLabel() {
  const { t } = useI18n();
  const labels: Record<string, string> = {
    RECEIVED: t("urtuu.status.RECEIVED"),
    ACCEPTED: t("urtuu.status.ACCEPTED"),
    IN_PROGRESS: t("urtuu.status.IN_PROGRESS"),
    DELEGATED: t("urtuu.status.DELEGATED"),
    COMPLETED: t("urtuu.status.COMPLETED"),
    RETURNED: t("urtuu.status.RETURNED"),
    CLOSED: t("urtuu.status.CLOSED"),
  };
  return (status: string) => labels[status] || status;
}

export function StatusChip({ status }: { status: string }) {
  const label = useStatusLabel();
  return <Chip tone={STATUS_TONE[status] || "slate"}>{label(status)}</Chip>;
}

/** The late flag. A flag and not a status — a task can be in progress and late. */
export function OverdueMark() {
  const { t } = useI18n();
  return (
    <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-600">
      <AlertTriangle className="w-3 h-3" />
      {t("urtuu.message.overdue")}
    </span>
  );
}

export function TaskQueue({ tasks, empty }: { tasks: UrtuuTask[]; empty: React.ReactNode }) {
  const { t } = useI18n();
  if (tasks.length === 0) return <Empty icon={<AlertTriangle className="w-8 h-8" />}>{empty}</Empty>;

  return (
    <Panel className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase text-[11px]">
          <tr>
            <th className="px-3 py-2 text-left">{t("urtuu.field.code")}</th>
            <th className="px-3 py-2 text-left">{t("urtuu.field.title")}</th>
            <th className="px-3 py-2 text-left">{t("urtuu.field.status")}</th>
            <th className="px-3 py-2 text-left">{t("urtuu.field.deadline")}</th>
            <th className="px-3 py-2 text-left">{t("urtuu.field.from")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {tasks.map((task) => (
            <tr key={task.id} className="hover:bg-slate-50">
              <td className="px-3 py-2 font-mono text-xs text-slate-600 align-top">{task.code}</td>
              <td className="px-3 py-2 align-top">
                <Link
                  href={`/module/urtuu/tasks/${task.id}`}
                  className="font-semibold text-indigo-700 hover:underline"
                >
                  {task.title || task.code}
                </Link>
                {task.note && <p className="text-xs text-slate-500 mt-0.5">{task.note}</p>}
              </td>
              <td className="px-3 py-2 align-top">
                <div className="flex flex-col gap-1 items-start">
                  <StatusChip status={task.status} />
                  {task.overdue && <OverdueMark />}
                </div>
              </td>
              <td className="px-3 py-2 align-top text-xs text-slate-600">
                {task.deadline
                  ? new Date(task.deadline).toLocaleDateString()
                  : t("urtuu.message.no_deadline")}
              </td>
              <td className="px-3 py-2 align-top text-xs text-slate-600">
                {task.origin_peer_name || task.target_peer_name || "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Panel>
  );
}
