"use client";

/**
 * One task: what it is, where it has been, what it was split into, and what may
 * be done to it next.
 *
 * The buttons are built from the server's own answer (`next`), not from a copy
 * of the state machine here. A screen that decided for itself which moves were
 * legal would be a second version of the rules, and the two would disagree the
 * first time one of them changed.
 */

import React, { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { GitBranch, History, Route } from "lucide-react";

import { api, type UrtuuTask, type UrtuuTaskEvent } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";
import { OverdueMark, StatusChip, useStatusLabel } from "../../shared";

/** Which endpoint each offered status is reached through. */
const MOVES: Record<string, "accept" | "return" | "complete" | "close"> = {
  ACCEPTED: "accept",
  RETURNED: "return",
  COMPLETED: "complete",
  CLOSED: "close",
};

export default function UrtuuTaskPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id as string;
  const { t } = useI18n();
  const statusLabel = useStatusLabel();

  const [task, setTask] = useState<UrtuuTask | null>(null);
  const [events, setEvents] = useState<UrtuuTaskEvent[]>([]);
  const [branches, setBranches] = useState<UrtuuTask[]>([]);
  const [next, setNext] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");
  const [reason, setReason] = useState("");

  const load = useCallback(async () => {
    try {
      const answer = await api.getUrtuuTask(id);
      setTask(answer.task);
      setEvents(answer.events || []);
      setBranches(answer.branches || []);
      setNext(answer.next || []);
      setFailure("");
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const move = async (status: string) => {
    const action = MOVES[status];
    if (!action) return;
    setFailure("");
    try {
      await api.moveUrtuuTask(id, action, { note: reason });
      setReason("");
      await load();
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    }
  };

  if (loading) return <Loading label={t("base.message.loading")} />;
  if (!task) return <ErrorNote>{failure || t("urtuu.message.no_tasks")}</ErrorNote>;

  return (
    <Screen
      icon={<Route className="w-5 h-5" />}
      title={task.title || task.code}
      subtitle={task.code}
      action={
        <div className="flex flex-col items-end gap-1">
          <StatusChip status={task.status} />
          {task.overdue && <OverdueMark />}
        </div>
      }
    >
      {failure && <ErrorNote>{failure}</ErrorNote>}

      <Panel className="p-4 grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
        <Fact label={t("urtuu.field.deadline")}>
          {task.deadline
            ? new Date(task.deadline).toLocaleString()
            : t("urtuu.message.no_deadline")}
        </Fact>
        <Fact label={task.origin_peer_name ? t("urtuu.field.from") : t("urtuu.field.to")}>
          {task.origin_peer_name || task.target_peer_name || "—"}
        </Fact>
        {task.assigned_name && (
          <Fact label={t("urtuu.field.name")}>{task.assigned_name}</Fact>
        )}
        {task.note && <Fact label={t("urtuu.field.note")}>{task.note}</Fact>}
      </Panel>

      {/* The body, as it was filled in. Rendered as its own JSON rather than
          re-typed here: the fields are the code's, not this screen's. */}
      {task.payload != null && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 mb-2">{t("urtuu.field.payload")}</h2>
          <pre className="text-xs font-mono text-slate-600 whitespace-pre-wrap break-all">
            {JSON.stringify(task.payload, null, 2)}
          </pre>
        </Panel>
      )}

      {next.length > 0 && (
        <Panel className="p-4 space-y-3">
          <input
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder={t("urtuu.field.reason")}
            className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg"
          />
          <div className="flex flex-wrap gap-2">
            {next
              .filter((status) => MOVES[status])
              .map((status) => (
                <button
                  key={status}
                  onClick={() => move(status)}
                  className="bg-white border border-indigo-200 text-indigo-700 hover:bg-indigo-50 text-xs font-semibold px-3 py-1.5 rounded-lg"
                >
                  {statusLabel(status)}
                </button>
              ))}
          </div>
        </Panel>
      )}

      {branches.length > 0 && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 flex items-center gap-2 mb-2">
            <GitBranch className="w-4 h-4 text-indigo-500" />
            {t("urtuu.section.branches")}
          </h2>
          <ul className="space-y-1 text-sm">
            {branches.map((branch) => (
              <li key={branch.id} className="flex items-center justify-between gap-2">
                <Link
                  href={`/module/urtuu/tasks/${branch.id}`}
                  className="text-indigo-700 hover:underline"
                >
                  {branch.target_peer_name || branch.title || branch.id.slice(0, 8)}
                </Link>
                <span className="flex items-center gap-2">
                  {branch.overdue && <OverdueMark />}
                  <StatusChip status={branch.status} />
                </span>
              </li>
            ))}
          </ul>
        </Panel>
      )}

      <Panel className="p-4">
        <h2 className="text-sm font-semibold text-slate-800 flex items-center gap-2 mb-2">
          <History className="w-4 h-4 text-indigo-500" />
          {t("urtuu.section.timeline")}
        </h2>
        <ol className="space-y-2 text-sm">
          {events.map((event, index) => (
            <li key={index} className="flex flex-wrap items-baseline gap-2">
              <span className="text-xs text-slate-400 tabular-nums">
                {new Date(event.created_at).toLocaleString()}
              </span>
              <StatusChip status={event.to_status} />
              <span className="text-slate-600 text-xs">
                {event.actor_name || event.peer_name || ""}
                {event.note ? ` · ${event.note}` : ""}
              </span>
            </li>
          ))}
        </ol>
      </Panel>

      {/* Where it has been. This is the list the cycle guard checks against, so
          it is worth being able to read. */}
      {task.origin_chain?.length > 0 && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 mb-2">{t("urtuu.section.chain")}</h2>
          <div className="flex flex-wrap gap-1">
            {task.origin_chain.map((installation) => (
              <Chip key={installation} mono>
                {installation.slice(0, 12)}
              </Chip>
            ))}
          </div>
        </Panel>
      )}
    </Screen>
  );
}

function Fact({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-xs uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="text-slate-800">{children}</dd>
    </div>
  );
}
