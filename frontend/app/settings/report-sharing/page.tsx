"use client";

/**
 * Report sharing — the screen behind §3.5 of the monitoring and reporting
 * proposal.
 *
 * Four things, in the order the people using them think about them: requests
 * waiting for this organisation's answer, agreements it has given, agreements
 * it has received, and a log of who has actually read its data.
 *
 * The one that matters most is the last. A transport company will only agree to
 * show a mine its trips if it can see afterwards what the mine looked at, and a
 * trail nobody can read is not a control.
 */

import React, { useCallback, useEffect, useState } from "react";
import { Check, Handshake, History, Plus, Share2, X } from "lucide-react";

import {
  api,
  type ReportAccessEntry,
  type ReportGrant,
  type ReportGroup,
} from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  Banner,
  EmptyState,
  LoadingBlock,
  Modal,
  PageHeader,
  cardClass,
  fieldClass,
  tableHeadClass,
} from "@/components/ui";

/** scopeLabel keeps the dynamic key out of t(). The dictionary key type is a
 *  union of literals, and a template string would defeat the check that a key
 *  exists at all — which is the one thing that catches a renamed key. */
function useScopeLabel() {
  const { t } = useI18n();
  return (scope: "counterparty" | "full") =>
    scope === "full" ? t("sharing.scope.full") : t("sharing.scope.counterparty");
}

export default function ReportSharingPage() {
  const { t, locale } = useI18n();
  const scopeLabel = useScopeLabel();

  const [grants, setGrants] = useState<ReportGrant[]>([]);
  const [history, setHistory] = useState<ReportAccessEntry[]>([]);
  const [groups, setGroups] = useState<ReportGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");
  const [notice, setNotice] = useState("");
  const [asking, setAsking] = useState(false);

  const label = useCallback(
    (titles: Record<string, string> | undefined, fallback: string) =>
      titles?.[locale] || titles?.mn || titles?.en || fallback,
    [locale],
  );

  const load = useCallback(async () => {
    try {
      const [grantList, accessLog, reportList] = await Promise.all([
        api.getReportGrants(),
        api.getReportAccessHistory(),
        api.getReports(),
      ]);
      setGrants(grantList.grants || []);
      setHistory(accessLog.history || []);
      setGroups(reportList.groups || []);
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const act = async (action: () => Promise<unknown>, message: string) => {
    setFailure("");
    try {
      await action();
      setNotice(message);
      await load();
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    }
  };

  if (loading) return <LoadingBlock label={t("base.message.loading")} />;

  const live = (grant: ReportGrant) => !grant.revoked_at;
  // A request waiting on *this* organisation: we own the data, and nobody has
  // answered yet. These are the ones that need somebody to decide.
  const pending = grants.filter((g) => live(g) && g.direction === "given" && !g.accepted_at);
  const given = grants.filter((g) => live(g) && g.direction === "given" && g.accepted_at);
  const received = grants.filter((g) => live(g) && g.direction === "received");
  const closed = grants.filter((g) => !live(g));

  return (
    <div className="space-y-6">
      <PageHeader
        icon={<Share2 className="w-7 h-7 text-indigo-600" />}
        title={t("sharing.view.title")}
        subtitle={t("sharing.view.subtitle")}
        actions={
          <button
            onClick={() => setAsking(true)}
            className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center gap-2 shadow-sm transition"
          >
            <Plus className="w-4 h-4" />
            {t("sharing.action.request")}
          </button>
        }
      />

      {failure && <Banner tone="error" message={failure} onDismiss={() => setFailure("")} />}
      {notice && <Banner tone="success" message={notice} onDismiss={() => setNotice("")} />}

      {/* Requests on us. First, because they are the only thing on this page
          that is waiting for a person. */}
      <section className={`${cardClass} p-4`}>
        <h2 className="text-sm font-semibold text-slate-800 flex items-center gap-2 mb-1">
          <Handshake className="w-4 h-4 text-amber-500" />
          {t("sharing.section.pending")}
        </h2>
        <p className="text-xs text-slate-500 mb-3">{t("sharing.hint.pending")}</p>

        {pending.length === 0 ? (
          <p className="text-sm text-slate-500">{t("sharing.message.no_pending")}</p>
        ) : (
          <div className="space-y-2">
            {pending.map((grant) => (
              <div
                key={grant.id}
                className="flex flex-wrap items-center justify-between gap-3 border border-amber-200 bg-amber-50 rounded-lg px-3 py-2.5"
              >
                <div className="text-sm">
                  <p className="font-semibold text-slate-800">
                    {grant.grantee_name} — {label(grant.titles, grant.report_key)}
                  </p>
                  <p className="text-xs text-slate-600 mt-0.5">
                    {scopeLabel(grant.scope)}
                    {grant.note ? ` · ${grant.note}` : ""}
                  </p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() =>
                      act(() => api.acceptReportGrant(grant.id), t("sharing.message.accepted"))
                    }
                    className="bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1"
                  >
                    <Check className="w-3.5 h-3.5" />
                    {t("sharing.action.accept")}
                  </button>
                  <button
                    onClick={() =>
                      act(() => api.revokeReportGrant(grant.id), t("sharing.message.refused"))
                    }
                    className="bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1"
                  >
                    <X className="w-3.5 h-3.5" />
                    {t("sharing.action.refuse")}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <GrantList
          title={t("sharing.section.given")}
          hint={t("sharing.hint.given")}
          empty={t("sharing.message.no_given")}
          grants={given}
          other={(g) => g.grantee_name}
          onRevoke={(id) => act(() => api.revokeReportGrant(id), t("sharing.message.revoked"))}
          label={label}
        />
        <GrantList
          title={t("sharing.section.received")}
          hint={t("sharing.hint.received")}
          empty={t("sharing.message.no_received")}
          grants={received}
          other={(g) => g.grantor_name}
          onRevoke={(id) => act(() => api.revokeReportGrant(id), t("sharing.message.withdrawn"))}
          label={label}
        />
      </div>

      {/* Who has read our data. The half of the agreement the data owner
          gets. */}
      <section className={`${cardClass} p-4`}>
        <h2 className="text-sm font-semibold text-slate-800 flex items-center gap-2 mb-1">
          <History className="w-4 h-4 text-indigo-500" />
          {t("sharing.section.history")}
        </h2>
        <p className="text-xs text-slate-500 mb-3">{t("sharing.hint.history")}</p>

        {history.length === 0 ? (
          <p className="text-sm text-slate-500">{t("sharing.message.no_history")}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className={tableHeadClass}>
                <tr>
                  <th className="px-3 py-2 text-left">{t("sharing.field.when")}</th>
                  <th className="px-3 py-2 text-left">{t("sharing.field.who")}</th>
                  <th className="px-3 py-2 text-left">{t("sharing.field.report")}</th>
                  <th className="px-3 py-2 text-right">{t("sharing.field.rows")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {history.map((entry, index) => (
                  <tr key={index} className="hover:bg-slate-50">
                    <td className="px-3 py-2 text-slate-600 text-xs">
                      {new Date(entry.at).toLocaleString()}
                    </td>
                    <td className="px-3 py-2 text-slate-800">{entry.by}</td>
                    <td className="px-3 py-2 font-mono text-xs text-slate-600">
                      {entry.report_key}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums text-slate-700">
                      {String(entry.details?.rows ?? "—")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {closed.length > 0 && (
        <section className={`${cardClass} p-4`}>
          <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-3">
            {t("sharing.section.closed")}
          </h2>
          <ul className="space-y-1 text-sm text-slate-500">
            {closed.map((grant) => (
              <li key={grant.id}>
                {grant.direction === "given" ? grant.grantee_name : grant.grantor_name} —{" "}
                {label(grant.titles, grant.report_key)} ·{" "}
                {grant.revoked_at ? new Date(grant.revoked_at).toLocaleDateString() : ""}
              </li>
            ))}
          </ul>
        </section>
      )}

      {asking && (
        <RequestModal
          groups={groups}
          label={label}
          onClose={() => setAsking(false)}
          onSent={async () => {
            setAsking(false);
            setNotice(t("sharing.message.requested"));
            await load();
          }}
        />
      )}
    </div>
  );
}

function GrantList({
  title,
  hint,
  empty,
  grants,
  other,
  onRevoke,
  label,
}: {
  title: string;
  hint: string;
  empty: string;
  grants: ReportGrant[];
  other: (grant: ReportGrant) => string;
  onRevoke: (id: string) => void;
  label: (titles: Record<string, string> | undefined, fallback: string) => string;
}) {
  const { t } = useI18n();
  const scopeLabel = useScopeLabel();

  return (
    <section className={`${cardClass} p-4`}>
      <h2 className="text-sm font-semibold text-slate-800 mb-1">{title}</h2>
      <p className="text-xs text-slate-500 mb-3">{hint}</p>

      {grants.length === 0 ? (
        <EmptyState message={empty} />
      ) : (
        <ul className="space-y-2">
          {grants.map((grant) => (
            <li
              key={grant.id}
              className="flex items-center justify-between gap-3 border border-slate-200 rounded-lg px-3 py-2.5"
            >
              <div className="text-sm min-w-0">
                <p className="font-semibold text-slate-800 truncate">{other(grant)}</p>
                <p className="text-xs text-slate-500 truncate">
                  {label(grant.titles, grant.report_key)} · {scopeLabel(grant.scope)}
                  {grant.valid_until
                    ? ` · ${new Date(grant.valid_until).toLocaleDateString()}`
                    : ""}
                </p>
              </div>
              <button
                onClick={() => onRevoke(grant.id)}
                className="text-xs font-semibold text-red-600 hover:text-red-700 whitespace-nowrap"
              >
                {t("sharing.action.revoke")}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function RequestModal({
  groups,
  label,
  onClose,
  onSent,
}: {
  groups: ReportGroup[];
  label: (titles: Record<string, string> | undefined, fallback: string) => string;
  onClose: () => void;
  onSent: () => void;
}) {
  const { t } = useI18n();
  const [registration, setRegistration] = useState("");
  const [reportKey, setReportKey] = useState("");
  const [scope, setScope] = useState<"counterparty" | "full">("counterparty");
  const [validUntil, setValidUntil] = useState("");
  const [note, setNote] = useState("");
  const [failure, setFailure] = useState("");
  const [sending, setSending] = useState(false);

  const send = async (event: React.FormEvent) => {
    event.preventDefault();
    setSending(true);
    setFailure("");
    try {
      await api.requestReportGrant({
        grantor_registration_number: registration.trim(),
        report_key: reportKey,
        scope,
        valid_until: validUntil || undefined,
        note: note.trim() || undefined,
      });
      onSent();
    } catch (err) {
      setFailure(err instanceof Error ? err.message : String(err));
    } finally {
      setSending(false);
    }
  };

  return (
    <Modal label={t("sharing.action.request")}>
      <h2 className="text-xl font-bold text-slate-900 mb-1">{t("sharing.action.request")}</h2>
      <p className="text-xs text-slate-500 mb-4">{t("sharing.hint.request")}</p>

      <form onSubmit={send} className="space-y-4">
        {failure && <Banner tone="error" message={failure} />}

        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            {t("sharing.field.registration")}
          </label>
          <input
            className={fieldClass}
            value={registration}
            onChange={(e) => setRegistration(e.target.value)}
            required
          />
          <p className="text-[11px] text-slate-400 mt-1">{t("sharing.hint.registration")}</p>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            {t("sharing.field.report")}
          </label>
          <select
            className={fieldClass}
            value={reportKey}
            onChange={(e) => setReportKey(e.target.value)}
            required
          >
            <option value="">—</option>
            {groups.flatMap((group) =>
              group.reports.map((report) => (
                <option key={report.key} value={report.key}>
                  {label(report.titles, report.key)}
                </option>
              )),
            )}
          </select>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            {t("sharing.field.scope")}
          </label>
          <select
            className={fieldClass}
            value={scope}
            onChange={(e) => setScope(e.target.value as "counterparty" | "full")}
          >
            <option value="counterparty">{t("sharing.scope.counterparty")}</option>
            <option value="full">{t("sharing.scope.full")}</option>
          </select>
          <p className="text-[11px] text-slate-400 mt-1">{t("sharing.hint.scope")}</p>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            {t("sharing.field.valid_until")}
          </label>
          <input
            type="date"
            className={fieldClass}
            value={validUntil}
            onChange={(e) => setValidUntil(e.target.value)}
          />
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            {t("sharing.field.note")}
          </label>
          <input className={fieldClass} value={note} onChange={(e) => setNote(e.target.value)} />
          <p className="text-[11px] text-slate-400 mt-1">{t("sharing.hint.note")}</p>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-xs font-semibold text-slate-600 hover:text-slate-800"
          >
            {t("base.action.cancel")}
          </button>
          <button
            type="submit"
            disabled={sending}
            className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-xs font-semibold px-4 py-2 rounded-lg"
          >
            {t("sharing.action.send_request")}
          </button>
        </div>
      </form>
    </Modal>
  );
}
