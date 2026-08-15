"use client";

/**
 * Ирсэн даалгавар — the work this organisation has been given.
 *
 * The filter that matters is "late only": everything else on this screen can
 * wait, and a queue that cannot be narrowed to what is overdue is a queue
 * somebody reads top to bottom every morning.
 */

import React, { useCallback, useEffect, useState } from "react";
import { Inbox } from "lucide-react";

import { api, type UrtuuTask } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { ErrorNote, Loading, Screen } from "@/components/module/kit";
import { TaskQueue, useLiveRefresh } from "../shared";

export default function IncomingTasksPage() {
  const { t } = useI18n();
  const [tasks, setTasks] = useState<UrtuuTask[]>([]);
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");

  // Not setLoading(true) here: this runs every fifteen seconds as well as on
  // the first paint, and a table that blanks itself on a poll is a table
  // nobody can read while it is refreshing.
  const load = useCallback(async () => {
    try {
      const answer = await api.getUrtuuTasks({ direction: "incoming", overdue: overdueOnly });
      setTasks(answer.tasks || []);
      setFailure("");
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [overdueOnly]);

  useEffect(() => {
    load();
  }, [load]);
  useLiveRefresh(load);

  return (
    <Screen
      icon={<Inbox className="w-5 h-5" />}
      title={t("urtuu.incoming.title")}
      subtitle={t("urtuu.incoming.subtitle")}
      action={
        <label className="flex items-center gap-2 text-xs font-semibold text-slate-600">
          <input
            type="checkbox"
            checked={overdueOnly}
            onChange={(event) => setOverdueOnly(event.target.checked)}
            className="w-4 h-4 accent-rose-600"
          />
          {t("urtuu.filter.overdue")}
        </label>
      }
    >
      {failure && <ErrorNote>{failure}</ErrorNote>}
      {loading ? (
        <Loading label={t("base.message.loading")} />
      ) : (
        <TaskQueue tasks={tasks} empty={t("urtuu.message.no_tasks")} />
      )}
    </Screen>
  );
}
