"use client";

/**
 * Lookup history — what this organisation asked the state, and who asked it.
 *
 * Read from the audit trail rather than from a table of its own, so there is
 * one record of the act and no way for this screen to disagree with the audit
 * log. It is also, deliberately, not deletable from here: "who read whose
 * registry record" is exactly the question an organisation should not be able
 * to tidy away.
 */

import { useEffect, useState } from "react";
import { ScrollText } from "lucide-react";
import { api, type EgovHistoryEntry } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, Empty, ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";

// The action names the audit log carries. `xyp.*` are the same acts under the
// name they had before the e-Government app existed; the server returns both.
const ACTION_LABELS = {
  "egov.citizen_queried": "egov.action.citizen_queried",
  "xyp.citizen_queried": "egov.action.citizen_queried",
  "egov.company_queried": "egov.action.company_queried",
  "xyp.company_queried": "egov.action.company_queried",
} as const;

export default function EgovHistoryPage() {
  const { t, locale } = useI18n();
  const [entries, setEntries] = useState<EgovHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void (async () => {
      try {
        setEntries((await api.getEgovHistory()) || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const subject = (entry: EgovHistoryEntry) =>
    String(entry.details?.reg_number || entry.details?.company_reg || "—");

  return (
    <Screen
      icon={<ScrollText className="w-6 h-6" />}
      title={t("egov.view.history_title")}
      subtitle={t("egov.view.history_subtitle")}
    >
      {error && <ErrorNote>{error}</ErrorNote>}
      {loading ? (
        <Loading label={t("egov.view.history_title")} />
      ) : entries.length === 0 ? (
        <Empty icon={<ScrollText className="w-10 h-10" />}>{t("egov.message.empty_history")}</Empty>
      ) : (
        <Panel>
          <table className="w-full text-sm">
            <thead className="text-left text-xs font-semibold text-slate-500 border-b border-slate-100">
              <tr>
                <th className="px-4 py-3">{t("egov.field.when")}</th>
                <th className="px-4 py-3">{t("egov.field.what")}</th>
                <th className="px-4 py-3">{t("egov.field.who")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {entries.map((entry, index) => (
                <tr key={`${entry.created_at}-${index}`}>
                  <td className="px-4 py-3 text-slate-500 whitespace-nowrap">
                    {new Date(entry.created_at).toLocaleString(locale)}
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-slate-900">
                      {entry.action in ACTION_LABELS
                        ? t(ACTION_LABELS[entry.action as keyof typeof ACTION_LABELS])
                        : entry.action}
                    </span>{" "}
                    <Chip mono>{subject(entry)}</Chip>
                  </td>
                  <td className="px-4 py-3 text-slate-500 font-mono text-xs break-all">
                    {entry.user_id || "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      )}
    </Screen>
  );
}
