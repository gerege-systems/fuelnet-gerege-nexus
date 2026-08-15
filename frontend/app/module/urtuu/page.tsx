"use client";

/**
 * Самбар — the app's front page.
 *
 * Two questions, in the order they are asked: what is late, and how much of
 * everything else there is. The red zone is first because it is the only part
 * of this screen that is asking somebody to do something.
 */

import React, { useCallback, useEffect, useState } from "react";
import { Route } from "lucide-react";

import { api, type UrtuuTally, type UrtuuTask } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";
import { TaskQueue, useStatusLabel } from "./shared";

export default function UrtuuBoardPage() {
  const { t } = useI18n();
  const statusLabel = useStatusLabel();

  const [counts, setCounts] = useState<UrtuuTally[]>([]);
  const [overdue, setOverdue] = useState<UrtuuTask[]>([]);
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    try {
      const board = await api.getUrtuuBoard();
      setCounts(board.counts || []);
      setOverdue(board.overdue || []);
      setEnabled(board.enabled);
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

  if (loading) return <Loading label={t("base.message.loading")} />;

  const incoming = counts.filter((tally) => tally.direction === "incoming");
  const outgoing = counts.filter((tally) => tally.direction === "outgoing");

  return (
    <Screen
      icon={<Route className="w-5 h-5" />}
      title={t("urtuu.board.title")}
      subtitle={t("urtuu.board.subtitle")}
    >
      {failure && <ErrorNote>{failure}</ErrorNote>}
      {!enabled && <ErrorNote>{t("urtuu.message.disabled")}</ErrorNote>}

      <section className="space-y-2">
        <h2 className="text-sm font-semibold text-slate-800">{t("urtuu.section.overdue")}</h2>
        <TaskQueue tasks={overdue} empty={t("urtuu.message.no_overdue")} />
      </section>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Tally title={t("urtuu.incoming.title")} rows={incoming} label={statusLabel} />
        <Tally title={t("urtuu.outgoing.title")} rows={outgoing} label={statusLabel} />
      </div>
    </Screen>
  );
}

function Tally({
  title,
  rows,
  label,
}: {
  title: string;
  rows: UrtuuTally[];
  label: (status: string) => string;
}) {
  const { t } = useI18n();
  return (
    <Panel className="p-4">
      <h2 className="text-sm font-semibold text-slate-800 mb-2">{title}</h2>
      {rows.length === 0 ? (
        <p className="text-sm text-slate-500">{t("urtuu.message.no_tasks")}</p>
      ) : (
        <ul className="space-y-1 text-sm">
          {rows.map((row) => (
            <li key={row.status} className="flex items-center justify-between gap-2">
              <span className="text-slate-700">{label(row.status)}</span>
              <span className="tabular-nums text-slate-800 font-semibold">
                {row.count}
                {row.overdue > 0 && (
                  <span className="ml-2 text-xs font-normal text-rose-600">
                    {row.overdue} {t("urtuu.message.overdue")}
                  </span>
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
