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
import { FileSignature, GitBranch, History, Route } from "lucide-react";

import { api, type UrtuuEvidence, type UrtuuTask, type UrtuuTaskEvent } from "@/lib/api";
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
  const [evidence, setEvidence] = useState<UrtuuEvidence[]>([]);
  const [next, setNext] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");
  const [reason, setReason] = useState("");
  const [answer, setAnswer] = useState("");

  const load = useCallback(async () => {
    try {
      const answer = await api.getUrtuuTask(id);
      setTask(answer.task);
      setEvents(answer.events || []);
      setBranches(answer.branches || []);
      setEvidence(answer.evidence || []);
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
      await api.moveUrtuuTask(id, action, { note: reason, answer });
      setReason("");
      setAnswer("");
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
      subtitle={[task.number, task.code].filter(Boolean).join(" · ")}
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
          {/* The sender's number is read together with the sender's name.
              Neither half means anything alone — which is why the number
              itself carries no platform prefix. */}
          {task.origin_number && (
            <span className="font-mono text-xs text-slate-500"> · {task.origin_number}</span>
          )}
        </Fact>
        {/* Who asked. Only the service line has one, and there it is the
            first thing anybody working the task needs. */}
        {task.applicant?.name && (
          <Fact label={t("urtuu.field.applicant")}>
            {task.applicant.name}
            {task.applicant.registry_number ? ` · ${task.applicant.registry_number}` : ""}
            {task.applicant.contact ? ` · ${task.applicant.contact}` : ""}
          </Fact>
        )}
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

      {/* The order behind the work. A reference, not the document: it stays in
          the documents app of whoever filed it, and is signed there. */}
      {evidence.length > 0 && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 flex items-center gap-2 mb-2">
            <FileSignature className="w-4 h-4 text-indigo-500" />
            {t("urtuu.section.evidence")}
          </h2>
          <ul className="space-y-2 text-sm">
            {evidence.map((item) => (
              <li
                key={`${item.installation}:${item.ref}`}
                className="flex flex-wrap items-baseline justify-between gap-2"
              >
                <span className="text-slate-800">{item.title}</span>
                <span className="flex items-center gap-2">
                  <Chip tone={item.signed ? "emerald" : "amber"}>
                    {t("urtuu.message.signed", {
                      count: item.signatures,
                      required: item.required_signatures,
                    })}
                  </Chip>
                  {/* Filed here: the link opens it. Filed elsewhere: there is
                      nothing here to open, and saying so is the honest answer. */}
                  {task.direction === "incoming" ? (
                    <span className="text-xs text-slate-500">{t("urtuu.message.filed_elsewhere")}</span>
                  ) : (
                    <Link href={`/module/documents/${item.ref}`} className="text-xs text-indigo-700 hover:underline">
                      {t("base.action.open")}
                    </Link>
                  )}
                </span>
              </li>
            ))}
          </ul>
        </Panel>
      )}

      {/* The answer, on the service line. It is what the whole line exists
          for: the person who asked is outside the platform, and a request
          closed without one is their question thrown away. */}
      {task.line === "service" && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 mb-1">{t("urtuu.section.answer")}</h2>
          <p className="text-sm text-slate-700 whitespace-pre-wrap">
            {task.answer || <span className="text-slate-400">{t("urtuu.message.no_answer_yet")}</span>}
          </p>
        </Panel>
      )}

      {next.length > 0 && (
        <Panel className="p-4 space-y-3">
          {/* Offered wherever this side is the one doing the work, because
              completing is one of the moves on offer and it cannot be made
              without this. */}
          {task.line === "service" && !task.target_peer_id && (
            <textarea
              rows={3}
              value={answer}
              onChange={(event) => setAnswer(event.target.value)}
              placeholder={t("urtuu.field.answer")}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg"
            />
          )}
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
                  {branch.number && (
                    <span className="font-mono text-xs text-slate-400"> · {branch.number}</span>
                  )}
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
