"use client";

/**
 * Илгээсэн даалгавар — this side's mirror of work given to subordinates, and
 * the form that raises it.
 *
 * The form is here rather than on its own page because raising work and
 * watching it are the same job: somebody sends a task to twenty-one provinces
 * and then wants to see the twenty-one rows appear.
 *
 * The body is rendered from the code's own JSON Schema — one text field per
 * declared property, which is as much of a schema form as this needs. A code
 * with a nested schema still works; its body is typed as JSON.
 */

import React, { useCallback, useEffect, useState } from "react";
import { Plus, Send } from "lucide-react";

import { api, type UrtuuCode, type UrtuuPeer, type UrtuuTask } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { ErrorNote, Loading, Modal, Screen } from "@/components/module/kit";
import { TaskQueue, useLiveRefresh } from "../shared";

/** The properties a code's schema declares, as a flat list of field names. */
function fieldsOf(schema: unknown): { name: string; title: string }[] {
  const properties = (schema as { properties?: Record<string, { title?: string }> } | undefined)
    ?.properties;
  if (!properties) return [];
  return Object.entries(properties).map(([name, spec]) => ({ name, title: spec?.title || name }));
}

export default function OutgoingTasksPage() {
  const { t, locale } = useI18n();
  const [tasks, setTasks] = useState<UrtuuTask[]>([]);
  const [codes, setCodes] = useState<UrtuuCode[]>([]);
  const [peers, setPeers] = useState<UrtuuPeer[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");
  const [notice, setNotice] = useState("");
  const [raising, setRaising] = useState(false);

  const load = useCallback(async () => {
    try {
      const [queue, vocabulary, links] = await Promise.all([
        api.getUrtuuTasks({ direction: "outgoing" }),
        api.getUrtuuCodes(),
        api.getUrtuuPeers(),
      ]);
      setTasks(queue.tasks || []);
      setCodes((vocabulary.codes || []).filter((code) => code.active));
      // Only the links this organisation is the parent on and that are open:
      // a child does not decide what it may be asked to do.
      setPeers((links.peers || []).filter((peer) => peer.role === "parent" && peer.status === "active"));
      setFailure("");
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);
  useLiveRefresh(load);

  const codeName = (code: UrtuuCode) =>
    code.names?.[locale] || code.names?.mn || code.names?.en || code.code;

  return (
    <Screen
      icon={<Send className="w-5 h-5" />}
      title={t("urtuu.outgoing.title")}
      subtitle={t("urtuu.outgoing.subtitle")}
      action={
        <button
          onClick={() => setRaising(true)}
          className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          {t("urtuu.action.new_task")}
        </button>
      }
    >
      {failure && <ErrorNote>{failure}</ErrorNote>}
      {notice && (
        <p className="text-sm text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2">
          {notice}
        </p>
      )}

      {loading ? (
        <Loading label={t("base.message.loading")} />
      ) : (
        <TaskQueue tasks={tasks} empty={t("urtuu.message.no_tasks")} />
      )}

      {raising && (
        <RaiseDialog
          codes={codes}
          peers={peers}
          codeName={codeName}
          onClose={() => setRaising(false)}
          onRaise={async (input) => {
            setFailure("");
            try {
              await api.createUrtuuTask(input);
              setNotice(t("urtuu.message.task_created"));
              setRaising(false);
              await load();
            } catch (err) {
              setFailure(err instanceof Error ? err.message : String(err));
            }
          }}
        />
      )}
    </Screen>
  );
}

function RaiseDialog({
  codes,
  peers,
  codeName,
  onRaise,
  onClose,
}: {
  codes: UrtuuCode[];
  peers: UrtuuPeer[];
  codeName: (code: UrtuuCode) => string;
  onRaise: (input: {
    code: string;
    title?: string;
    payload?: unknown;
    deadline?: string | null;
    peer_ids?: string[];
  }) => Promise<void>;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [code, setCode] = useState("");
  const [title, setTitle] = useState("");
  const [deadline, setDeadline] = useState("");
  const [values, setValues] = useState<Record<string, string>>({});
  const [targets, setTargets] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);

  const chosen = codes.find((entry) => entry.code === code);
  const fields = fieldsOf(chosen?.schema);

  return (
    <Modal onClose={onClose}>
      <form
        className="space-y-3"
        onSubmit={async (event) => {
          event.preventDefault();
          setBusy(true);
          try {
            await onRaise({
              code,
              title: title.trim() || undefined,
              payload: values,
              // A date the person typed, at the end of that day in their own
              // zone. Sent absolute, because the two installations' clocks are
              // not the same one.
              deadline: deadline ? new Date(deadline + "T23:59:59").toISOString() : null,
              peer_ids: targets,
            });
          } finally {
            setBusy(false);
          }
        }}
      >
        <h2 className="text-lg font-bold text-slate-900">{t("urtuu.action.new_task")}</h2>

        <label className="block text-xs font-semibold text-slate-600">
          {t("urtuu.field.code")}
          <select
            required
            value={code}
            onChange={(event) => {
              setCode(event.target.value);
              setValues({});
            }}
            className="w-full mt-1 px-3 py-2 text-sm border border-slate-300 rounded-lg"
          >
            <option value="">{t("urtuu.message.pick_code")}</option>
            {codes.map((entry) => (
              <option key={entry.id} value={entry.code}>
                {entry.code} — {codeName(entry)}
              </option>
            ))}
          </select>
        </label>

        <label className="block text-xs font-semibold text-slate-600">
          {t("urtuu.field.title")}
          <input
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder={chosen ? codeName(chosen) : ""}
            className="w-full mt-1 px-3 py-2 text-sm border border-slate-300 rounded-lg"
          />
        </label>

        {/* The form the code declares. A code with no schema simply has none. */}
        {fields.map((field) => (
          <label key={field.name} className="block text-xs font-semibold text-slate-600">
            {field.title}
            <input
              value={values[field.name] || ""}
              onChange={(event) => setValues({ ...values, [field.name]: event.target.value })}
              className="w-full mt-1 px-3 py-2 text-sm border border-slate-300 rounded-lg"
            />
          </label>
        ))}

        <label className="block text-xs font-semibold text-slate-600">
          {t("urtuu.field.deadline")}
          <input
            type="date"
            value={deadline}
            onChange={(event) => setDeadline(event.target.value)}
            className="w-full mt-1 px-3 py-2 text-sm border border-slate-300 rounded-lg"
          />
        </label>

        <fieldset className="block text-xs font-semibold text-slate-600">
          <legend>{t("urtuu.field.targets")}</legend>
          {peers.length === 0 ? (
            <p className="mt-1 font-normal text-slate-500">{t("urtuu.message.no_open_links")}</p>
          ) : (
            <ul className="mt-1 space-y-1">
              {peers.map((peer) => (
                <li key={peer.id}>
                  <label className="flex items-center gap-2 font-normal text-slate-700">
                    <input
                      type="checkbox"
                      checked={targets.includes(peer.id)}
                      onChange={(event) =>
                        setTargets((current) =>
                          event.target.checked
                            ? [...current, peer.id]
                            : current.filter((id) => id !== peer.id),
                        )
                      }
                      className="w-4 h-4 accent-indigo-600"
                    />
                    {peer.name || peer.id.slice(0, 8)}
                  </label>
                </li>
              ))}
            </ul>
          )}
        </fieldset>

        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-slate-600 rounded-lg">
            {t("base.action.cancel")}
          </button>
          <button
            type="submit"
            disabled={busy || !code}
            className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-40 text-white text-sm font-semibold px-4 py-2 rounded-lg"
          >
            {t("urtuu.action.send")}
          </button>
        </div>
      </form>
    </Modal>
  );
}
