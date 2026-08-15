"use client";

/**
 * Самбар — the app's front page.
 *
 * Two questions, in the order they are asked: what is late, and how much of
 * everything else there is. The red zone is first because it is the only part
 * of this screen that is asking somebody to do something.
 */

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Route } from "lucide-react";

import { api, type UrtuuLinkHealth, type UrtuuTally, type UrtuuTask, type UrtuuTreeProgress } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";
import { TaskQueue, useLiveRefresh, useStatusLabel } from "./shared";

export default function UrtuuBoardPage() {
  const { t } = useI18n();
  const statusLabel = useStatusLabel();

  const [counts, setCounts] = useState<UrtuuTally[]>([]);
  const [overdue, setOverdue] = useState<UrtuuTask[]>([]);
  const [links, setLinks] = useState<UrtuuLinkHealth[]>([]);
  const [trees, setTrees] = useState<UrtuuTreeProgress[]>([]);
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    try {
      const board = await api.getUrtuuBoard();
      setCounts(board.counts || []);
      setOverdue(board.overdue || []);
      setLinks(board.links || []);
      setTrees(board.trees || []);
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
  useLiveRefresh(load);

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

      {/* How far each fan-out has got. The question a ministry actually has is
          not "how many tasks" but "of the twenty-one provinces, who is done". */}
      {trees.length > 0 && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 mb-2">{t("urtuu.section.branches")}</h2>
          <ul className="space-y-2">
            {trees.map((tree) => (
              <li key={tree.id} className="space-y-1">
                <div className="flex items-baseline justify-between gap-2 text-sm">
                  <Link href={`/module/urtuu/tasks/${tree.id}`} className="text-indigo-700 hover:underline">
                    {tree.title || tree.code}
                  </Link>
                  <span className="text-xs tabular-nums text-slate-600">
                    {tree.done}/{tree.total}
                    {tree.late > 0 && (
                      <span className="ml-2 text-rose-600">
                        {tree.late} {t("urtuu.message.overdue")}
                      </span>
                    )}
                  </span>
                </div>
                <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
                  <div
                    className={`h-full ${tree.late > 0 ? "bg-rose-400" : "bg-emerald-500"}`}
                    style={{ width: `${tree.total ? Math.round((tree.done / tree.total) * 100) : 0}%` }}
                  />
                </div>
              </li>
            ))}
          </ul>
        </Panel>
      )}

      {/* The channel underneath. A queue that has stopped moving is usually a
          link that has stopped talking, and this is where that shows. */}
      {links.length > 0 && (
        <Panel className="p-4">
          <h2 className="text-sm font-semibold text-slate-800 mb-2">{t("urtuu.section.links")}</h2>
          <ul className="space-y-1 text-sm">
            {links.map((link) => (
              <li key={link.id} className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-slate-700">
                  {link.name || link.id.slice(0, 8)}
                  <span className="text-xs text-slate-400 ml-2">
                    {link.role === "parent" ? t("urtuu.role.parent") : t("urtuu.role.child")}
                  </span>
                </span>
                <span className="flex items-center gap-2 text-xs">
                  {link.undelivered > 0 && (
                    <Chip tone="amber">
                      {t("urtuu.message.undelivered", { count: link.undelivered })}
                    </Chip>
                  )}
                  <span className="text-slate-500">
                    {link.last_seen_at
                      ? new Date(link.last_seen_at).toLocaleString()
                      : t("urtuu.message.never")}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        </Panel>
      )}
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
