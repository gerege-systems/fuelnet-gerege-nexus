"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  ACTION_REQUIRES_COMMENT,
  Dashboard,
  GovApiError,
  GovService,
  OrgUnit,
  Page,
  RequestDetail,
  Task,
  TaskAction,
  TaskStatus,
  Workflow,
  WorkflowTemplate,
  availableActions,
  gov,
} from "@/lib/gov";
import { useI18n } from "@/lib/i18n";
import {
  AlertTriangle,
  Building2,
  CheckCircle2,
  ChevronRight,
  Clock,
  Landmark,
  Loader2,
  RefreshCw,
  Settings2,
  Undo2,
  Workflow as WorkflowIcon,
  X,
} from "lucide-react";

type Tab = "queue" | "configuration";

const STATUS_STYLE: Record<TaskStatus, string> = {
  RECEIVED: "bg-slate-100 text-slate-700",
  ASSIGNED: "bg-sky-100 text-sky-700",
  IN_PROGRESS: "bg-blue-100 text-blue-700",
  FORWARDED: "bg-violet-100 text-violet-700",
  INFO_REQUESTED: "bg-amber-100 text-amber-700",
  AWAITING_VERIFICATION: "bg-orange-100 text-orange-700",
  RETURNED: "bg-rose-100 text-rose-700",
  COMPLETED: "bg-emerald-100 text-emerald-700",
  CLOSED: "bg-indigo-100 text-indigo-700",
  REJECTED: "bg-red-100 text-red-700",
  CANCELLED: "bg-slate-100 text-slate-500",
};

const PAGE_SIZE = 20;

export default function GovServicesPage() {
  const { t, locale } = useI18n();

  const [tab, setTab] = useState<Tab>("queue");
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [queue, setQueue] = useState<Page<Task> | null>(null);
  const [services, setServices] = useState<GovService[]>([]);
  const [units, setUnits] = useState<OrgUnit[]>([]);
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [templates, setTemplates] = useState<WorkflowTemplate[]>([]);
  const [detail, setDetail] = useState<RequestDetail | null>(null);

  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [statusFilter, setStatusFilter] = useState("");
  const [unitFilter, setUnitFilter] = useState("");
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [page, setPage] = useState(1);

  const report = useCallback(
    (err: unknown) => {
      // The backend's machine code travels with the message so an operator can
      // report exactly what happened.
      if (err instanceof GovApiError) {
        setError(`${err.message} (${err.code})`);
      } else {
        setError(t("common.error"));
      }
    },
    [t],
  );

  const loadOperational = useCallback(async () => {
    setError(null);
    try {
      const [summary, tasks] = await Promise.all([
        gov.dashboard(),
        gov.tasks({
          status: statusFilter || undefined,
          unit_id: unitFilter || undefined,
          overdue: overdueOnly || undefined,
          page,
          page_size: PAGE_SIZE,
        }),
      ]);
      setDashboard(summary);
      setQueue(tasks);
    } catch (err) {
      report(err);
    }
  }, [statusFilter, unitFilter, overdueOnly, page, report]);

  const loadReference = useCallback(async () => {
    try {
      const [svc, unitList, flows, tpl] = await Promise.all([
        gov.services(),
        gov.units(),
        gov.workflows(),
        gov.templates(),
      ]);
      setServices(svc);
      setUnits(unitList);
      setWorkflows(flows);
      setTemplates(tpl);
    } catch (err) {
      report(err);
    }
  }, [report]);

  useEffect(() => {
    (async () => {
      setLoading(true);
      await Promise.all([loadOperational(), loadReference()]);
      setLoading(false);
    })();
    // Reference data is reloaded explicitly after a configuration change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!loading) loadOperational();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter, unitFilter, overdueOnly, page]);

  const unitName = useMemo(() => {
    const map = new Map(units.map((u) => [u.id, u.name]));
    return (id?: string | null) => (id ? map.get(id) || id : "—");
  }, [units]);

  const act = async (task: Task, action: TaskAction) => {
    let comment = "";
    if (ACTION_REQUIRES_COMMENT.includes(action)) {
      comment = window.prompt(t("gov.commentPrompt")) || "";
      if (!comment) return;
    }

    let targetUnit: string | undefined;
    if (action === "delegate") {
      const children = units.filter((u) => u.parent_id === task.unit_id && u.active);
      if (children.length === 0) {
        setError(t("gov.noChildUnit"));
        return;
      }
      targetUnit =
        children.length === 1
          ? children[0].id
          : window.prompt(`${t("gov.delegateTo")}\n${children.map((c) => `${c.code} = ${c.id}`).join("\n")}`) || "";
      if (!targetUnit) return;
    }

    setBusy(task.id);
    setError(null);
    try {
      const updated = await gov.act(task.id, {
        action,
        row_version: task.row_version,
        comment: comment || undefined,
        target_unit_id: targetUnit,
      });
      setNotice(`${task.reference || ""} → ${updated.status}`);
      await loadOperational();
      if (detail?.request.id === task.application_id) {
        setDetail(await gov.requestDetail(task.application_id));
      }
    } catch (err) {
      report(err);
    } finally {
      setBusy(null);
    }
  };

  const openDetail = async (applicationID: string) => {
    setError(null);
    try {
      setDetail(await gov.requestDetail(applicationID));
    } catch (err) {
      report(err);
    }
  };

  const statusLabel = (status: TaskStatus) => {
    const label = t(`gov.status.${status}` as never);
    return label === `gov.status.${status}` ? status : label;
  };

  const statusBadge = (status: TaskStatus) => (
    <span className={`text-xs font-semibold px-2 py-0.5 rounded ${STATUS_STYLE[status] || "bg-slate-100"}`}>
      {statusLabel(status)}
    </span>
  );

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-slate-500 text-sm">
        <Loader2 className="w-4 h-4 animate-spin" />
        {t("gov.loading")}
      </div>
    );
  }

  const cards = dashboard
    ? [
        { key: "received", label: t("gov.card.received"), value: dashboard.received, tone: "text-slate-700" },
        { key: "in_progress", label: t("gov.card.inProgress"), value: dashboard.in_progress, tone: "text-blue-600" },
        { key: "delegated", label: t("gov.card.delegated"), value: dashboard.delegated, tone: "text-violet-600" },
        {
          key: "verify",
          label: t("gov.card.awaitingVerification"),
          value: dashboard.awaiting_verification,
          tone: "text-orange-600",
        },
        { key: "returned", label: t("gov.card.returned"), value: dashboard.returned, tone: "text-rose-600" },
        {
          key: "completed",
          label: t("gov.card.completed"),
          value: dashboard.completed + dashboard.closed,
          tone: "text-emerald-600",
        },
        { key: "overdue", label: t("gov.card.overdue"), value: dashboard.overdue, tone: "text-red-600" },
      ]
    : [];

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Landmark className="w-7 h-7 text-[var(--gerege-blue)]" />
            {t("gov.title")}
          </h1>
          <p className="text-sm text-slate-500 mt-1">{t("gov.subtitle")}</p>
        </div>
        <button
          onClick={() => loadOperational()}
          className="flex items-center gap-2 px-3 py-2 text-sm border border-slate-200 rounded-lg bg-white hover:bg-slate-50 text-slate-600"
        >
          <RefreshCw className="w-4 h-4" />
          {t("gov.refresh")}
        </button>
      </header>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
          <span className="flex-1">{error}</span>
          <button onClick={() => setError(null)} aria-label={t("common.close")}>
            <X className="w-4 h-4" />
          </button>
        </div>
      )}
      {notice && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 text-emerald-700 text-sm rounded-lg flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4" />
          <span className="flex-1">{notice}</span>
          <button onClick={() => setNotice(null)} aria-label={t("common.close")}>
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      <section className="grid grid-cols-2 md:grid-cols-4 xl:grid-cols-7 gap-3">
        {cards.map((card) => (
          <div key={card.key} className="p-4 bg-white border border-slate-200 rounded-xl">
            <div className={`text-2xl font-bold ${card.tone}`}>{card.value}</div>
            <div className="text-[11px] text-slate-500 leading-snug mt-1">{card.label}</div>
          </div>
        ))}
      </section>

      <nav className="flex gap-1 border-b border-slate-200">
        {(["queue", "configuration"] as Tab[]).map((key) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition ${
              tab === key
                ? "border-[var(--gerege-blue)] text-[var(--gerege-blue)]"
                : "border-transparent text-slate-500 hover:text-slate-700"
            }`}
          >
            {key === "queue" ? t("gov.tab.queue") : t("gov.tab.configuration")}
          </button>
        ))}
      </nav>

      {tab === "queue" && (
        <>
          <div className="flex flex-wrap gap-2 items-center">
            <select
              value={statusFilter}
              onChange={(e) => {
                setPage(1);
                setStatusFilter(e.target.value);
              }}
              className="px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white"
            >
              <option value="">{t("gov.filter.allStatuses")}</option>
              {(Object.keys(STATUS_STYLE) as TaskStatus[]).map((status) => (
                <option key={status} value={status}>
                  {statusLabel(status)}
                </option>
              ))}
            </select>

            <select
              value={unitFilter}
              onChange={(e) => {
                setPage(1);
                setUnitFilter(e.target.value);
              }}
              className="px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white"
            >
              <option value="">{t("gov.filter.allUnits")}</option>
              {units.map((unit) => (
                <option key={unit.id} value={unit.id}>
                  {locale === "en" && unit.name_en ? unit.name_en : unit.name}
                </option>
              ))}
            </select>

            <label className="flex items-center gap-2 text-sm text-slate-600">
              <input
                type="checkbox"
                checked={overdueOnly}
                onChange={(e) => {
                  setPage(1);
                  setOverdueOnly(e.target.checked);
                }}
              />
              {t("gov.filter.overdueOnly")}
            </label>
          </div>

          <div className="bg-white border border-slate-200 rounded-xl overflow-hidden">
            {!queue || queue.items.length === 0 ? (
              <p className="p-6 text-sm text-slate-500">{t("gov.queueEmpty")}</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm min-w-[880px]">
                  <thead className="bg-slate-50 text-left text-xs uppercase text-slate-500">
                    <tr>
                      <th className="px-4 py-3">{t("gov.reference")}</th>
                      <th className="px-4 py-3">{t("gov.service")}</th>
                      <th className="px-4 py-3">{t("gov.unit")}</th>
                      <th className="px-4 py-3">{t("common.status")}</th>
                      <th className="px-4 py-3">{t("gov.due")}</th>
                      <th className="px-4 py-3 text-right">{t("common.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {queue.items.map((task) => (
                      <tr key={task.id} className={task.overdue ? "bg-red-50/40" : undefined}>
                        <td className="px-4 py-3">
                          <button
                            onClick={() => openDetail(task.application_id)}
                            className="font-mono text-xs text-[var(--gerege-blue)] hover:underline"
                          >
                            {task.reference}
                          </button>
                        </td>
                        <td className="px-4 py-3">{task.service_name}</td>
                        <td className="px-4 py-3">{task.unit_name || unitName(task.unit_id)}</td>
                        <td className="px-4 py-3">{statusBadge(task.status)}</td>
                        <td className="px-4 py-3 text-xs">
                          {task.due_at ? (
                            <span className={task.overdue ? "text-red-600 font-semibold" : "text-slate-500"}>
                              {new Date(task.due_at).toLocaleDateString(locale === "en" ? "en-GB" : "mn-MN")}
                              {task.overdue ? ` · ${t("gov.overdue")}` : ""}
                            </span>
                          ) : (
                            "—"
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-wrap gap-1.5 justify-end">
                            {busy === task.id ? (
                              <Loader2 className="w-4 h-4 animate-spin text-slate-400" />
                            ) : (
                              availableActions(task.status).map((action) => (
                                <button
                                  key={action}
                                  onClick={() => act(task, action)}
                                  className="text-xs px-2 py-1 rounded border border-slate-200 hover:bg-slate-50 text-slate-600"
                                >
                                  {t(`gov.action.${action}` as never)}
                                </button>
                              ))
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {queue && queue.total > PAGE_SIZE && (
            <div className="flex items-center justify-between text-sm text-slate-500">
              <span>
                {t("gov.pageOf")
                  .replace("{page}", String(queue.page))
                  .replace("{total}", String(Math.ceil(queue.total / queue.page_size)))}
              </span>
              <div className="flex gap-2">
                <button
                  disabled={queue.page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  className="px-3 py-1.5 border border-slate-200 rounded-lg disabled:opacity-40"
                >
                  {t("gov.previous")}
                </button>
                <button
                  disabled={queue.page * queue.page_size >= queue.total}
                  onClick={() => setPage((p) => p + 1)}
                  className="px-3 py-1.5 border border-slate-200 rounded-lg disabled:opacity-40"
                >
                  {t("gov.next")}
                </button>
              </div>
            </div>
          )}
        </>
      )}

      {tab === "configuration" && (
        <ConfigurationPanel
          services={services}
          units={units}
          workflows={workflows}
          templates={templates}
          onChanged={async () => {
            await loadReference();
            await loadOperational();
          }}
          onError={report}
        />
      )}

      {detail && (
        <RequestDrawer detail={detail} onClose={() => setDetail(null)} unitName={unitName} statusBadge={statusBadge} />
      )}
    </div>
  );
}

// ─── Request detail ──────────────────────────────────────────────────────────

function RequestDrawer({
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
          <button onClick={onClose} aria-label={t("common.close")}>
            <X className="w-5 h-5 text-slate-400" />
          </button>
        </div>

        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <dt className="text-xs text-slate-500">{t("gov.mode")}</dt>
            <dd className="font-medium">{detail.request.fulfillment_mode}</dd>
          </div>
          <div>
            <dt className="text-xs text-slate-500">{t("gov.currentUnit")}</dt>
            <dd className="font-medium">{unitName(detail.request.current_unit_id)}</dd>
          </div>
        </dl>

        <section className="space-y-2">
          <h3 className="text-sm font-semibold text-slate-700 flex items-center gap-2">
            <WorkflowIcon className="w-4 h-4" />
            {t("gov.tasks")}
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
          <h3 className="text-sm font-semibold text-slate-700">{t("gov.timeline")}</h3>
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

// ─── Configuration ───────────────────────────────────────────────────────────

function ConfigurationPanel({
  services,
  units,
  workflows,
  templates,
  onChanged,
  onError,
}: {
  services: GovService[];
  units: OrgUnit[];
  workflows: Workflow[];
  templates: WorkflowTemplate[];
  onChanged: () => Promise<void>;
  onError: (err: unknown) => void;
}) {
  const { t, locale } = useI18n();
  const [unitForm, setUnitForm] = useState({ code: "", name: "", parent_id: "", region_code: "" });
  const [template, setTemplate] = useState("");
  const [saving, setSaving] = useState(false);

  const publishedVersions = workflows.flatMap((wf) =>
    (wf.versions || [])
      .filter((v) => v.status === "PUBLISHED")
      .map((v) => ({ id: v.id, label: `${wf.code} v${v.version}` })),
  );

  const submitUnit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await gov.createUnit({
        code: unitForm.code,
        name: unitForm.name,
        parent_id: unitForm.parent_id || null,
        region_code: unitForm.region_code || undefined,
      });
      setUnitForm({ code: "", name: "", parent_id: "", region_code: "" });
      await onChanged();
    } catch (err) {
      onError(err);
    } finally {
      setSaving(false);
    }
  };

  // Instantiating a template publishes it, because a service can only point at
  // a published version.
  const instantiate = async () => {
    if (!template) return;
    setSaving(true);
    try {
      const version = await gov.createWorkflow({ template });
      await gov.publishVersion(version.id);
      setTemplate("");
      await onChanged();
    } catch (err) {
      onError(err);
    } finally {
      setSaving(false);
    }
  };

  const configure = async (
    service: GovService,
    patch: Partial<Pick<GovService, "fulfillment_mode" | "workflow_version_id" | "owner_unit_id">>,
  ) => {
    setSaving(true);
    try {
      await gov.configureService(service.id, {
        fulfillment_mode: patch.fulfillment_mode ?? service.fulfillment_mode,
        workflow_version_id: patch.workflow_version_id ?? service.workflow_version_id ?? null,
        owner_unit_id: patch.owner_unit_id ?? service.owner_unit_id ?? null,
      });
      await onChanged();
    } catch (err) {
      onError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <section className="bg-white border border-slate-200 rounded-xl p-5 space-y-4">
        <h2 className="font-semibold flex items-center gap-2">
          <Building2 className="w-5 h-5 text-[var(--gerege-blue)]" />
          {t("gov.config.units")}
        </h2>
        <ul className="text-sm divide-y divide-slate-100">
          {units.length === 0 && <li className="py-2 text-slate-500">{t("gov.config.noUnits")}</li>}
          {units.map((unit) => (
            <li key={unit.id} className="py-2 flex items-center gap-2">
              <span className="font-mono text-xs text-slate-400 w-16">{unit.code}</span>
              <span className="flex-1">{locale === "en" && unit.name_en ? unit.name_en : unit.name}</span>
              <span className="text-xs text-slate-400">{unit.unit_type}</span>
              {unit.parent_id && <Undo2 className="w-3.5 h-3.5 text-slate-300" />}
            </li>
          ))}
        </ul>

        <form onSubmit={submitUnit} className="grid sm:grid-cols-4 gap-2">
          <input
            required
            placeholder={t("gov.config.unitCode")}
            value={unitForm.code}
            onChange={(e) => setUnitForm({ ...unitForm, code: e.target.value })}
            className="px-3 py-2 text-sm border border-slate-300 rounded-lg"
          />
          <input
            required
            placeholder={t("gov.config.unitName")}
            value={unitForm.name}
            onChange={(e) => setUnitForm({ ...unitForm, name: e.target.value })}
            className="px-3 py-2 text-sm border border-slate-300 rounded-lg"
          />
          <select
            value={unitForm.parent_id}
            onChange={(e) => setUnitForm({ ...unitForm, parent_id: e.target.value })}
            className="px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white"
          >
            <option value="">{t("gov.config.noParent")}</option>
            {units.map((unit) => (
              <option key={unit.id} value={unit.id}>
                {unit.code}
              </option>
            ))}
          </select>
          <button
            disabled={saving}
            className="px-3 py-2 text-sm font-semibold text-white bg-[var(--gerege-blue)] rounded-lg disabled:opacity-50"
          >
            {t("common.create")}
          </button>
        </form>
      </section>

      <section className="bg-white border border-slate-200 rounded-xl p-5 space-y-4">
        <h2 className="font-semibold flex items-center gap-2">
          <WorkflowIcon className="w-5 h-5 text-[var(--gerege-blue)]" />
          {t("gov.config.workflows")}
        </h2>
        <ul className="text-sm divide-y divide-slate-100">
          {workflows.length === 0 && <li className="py-2 text-slate-500">{t("gov.config.noWorkflows")}</li>}
          {workflows.map((wf) => (
            <li key={wf.id} className="py-2">
              <span className="font-medium">{locale === "en" && wf.name_en ? wf.name_en : wf.name}</span>
              <span className="ml-2 text-xs text-slate-400">{wf.code}</span>
              <div className="flex flex-wrap gap-1.5 mt-1">
                {(wf.versions || []).map((version) => (
                  <span
                    key={version.id}
                    className={`text-xs px-2 py-0.5 rounded ${
                      version.status === "PUBLISHED" ? "bg-emerald-100 text-emerald-700" : "bg-slate-100 text-slate-600"
                    }`}
                  >
                    v{version.version} · {version.status}
                  </span>
                ))}
              </div>
            </li>
          ))}
        </ul>

        <div className="flex flex-wrap gap-2">
          <select
            value={template}
            onChange={(e) => setTemplate(e.target.value)}
            className="px-3 py-2 text-sm border border-slate-300 rounded-lg bg-white flex-1 min-w-52"
          >
            <option value="">{t("gov.config.pickTemplate")}</option>
            {templates.map((tpl) => (
              <option key={tpl.Code} value={tpl.Code}>
                {locale === "en" ? tpl.NameEN : tpl.Name}
              </option>
            ))}
          </select>
          <button
            onClick={instantiate}
            disabled={!template || saving}
            className="px-3 py-2 text-sm font-semibold text-white bg-[var(--gerege-blue)] rounded-lg disabled:opacity-50"
          >
            {t("gov.config.publish")}
          </button>
        </div>
      </section>

      <section className="bg-white border border-slate-200 rounded-xl p-5 space-y-4">
        <h2 className="font-semibold flex items-center gap-2">
          <Settings2 className="w-5 h-5 text-[var(--gerege-blue)]" />
          {t("gov.config.services")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm min-w-[720px]">
            <thead className="bg-slate-50 text-left text-xs uppercase text-slate-500">
              <tr>
                <th className="px-3 py-2">{t("gov.service")}</th>
                <th className="px-3 py-2">{t("gov.mode")}</th>
                <th className="px-3 py-2">{t("gov.config.workflowVersion")}</th>
                <th className="px-3 py-2">{t("gov.config.ownerUnit")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {services.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-3 py-4 text-slate-500">
                    {t("gov.noServices")}
                  </td>
                </tr>
              )}
              {services.map((service) => (
                <tr key={service.id}>
                  <td className="px-3 py-2">{locale === "en" && service.name_en ? service.name_en : service.name}</td>
                  <td className="px-3 py-2">
                    <select
                      value={service.fulfillment_mode}
                      onChange={(e) =>
                        configure(service, { fulfillment_mode: e.target.value as GovService["fulfillment_mode"] })
                      }
                      className="px-2 py-1 text-xs border border-slate-300 rounded bg-white"
                    >
                      {(["LOCAL", "DELEGATE", "HYBRID"] as const).map((mode) => (
                        <option key={mode} value={mode}>
                          {mode}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <select
                      value={service.workflow_version_id || ""}
                      onChange={(e) => configure(service, { workflow_version_id: e.target.value || null })}
                      className="px-2 py-1 text-xs border border-slate-300 rounded bg-white"
                    >
                      <option value="">—</option>
                      {publishedVersions.map((version) => (
                        <option key={version.id} value={version.id}>
                          {version.label}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <select
                      value={service.owner_unit_id || ""}
                      onChange={(e) => configure(service, { owner_unit_id: e.target.value || null })}
                      className="px-2 py-1 text-xs border border-slate-300 rounded bg-white"
                    >
                      <option value="">—</option>
                      {units.map((unit) => (
                        <option key={unit.id} value={unit.id}>
                          {unit.code}
                        </option>
                      ))}
                    </select>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
