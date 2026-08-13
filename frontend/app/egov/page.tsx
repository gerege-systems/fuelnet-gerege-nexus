"use client";

/**
 * Registry lookups — asking the state about a person or a company.
 *
 * The answer is deliberately not stored. This screen exists so somebody can
 * check a registration number against the authoritative record; keeping a copy
 * of the reply would make this app a second register of citizens, which is the
 * one thing a lookup surface must not quietly become. What is kept is the fact
 * that the question was asked, and that lives in the history screen.
 */

import { useState } from "react";
import { Landmark, Search } from "lucide-react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, Empty, ErrorNote, Panel, Screen } from "@/components/module/kit";

type Row = { label: string; value: string };

export default function EgovLookupsPage() {
  const { t } = useI18n();
  const [kind, setKind] = useState<"citizen" | "company">("citizen");
  const [query, setQuery] = useState("");
  const [rows, setRows] = useState<Row[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const lookup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    setBusy(true);
    setError("");
    setRows(null);
    try {
      if (kind === "citizen") {
        const info = await api.queryXYPCitizen(query.trim());
        setRows([
          { label: t("egov.field.reg_number"), value: info.reg_number },
          { label: t("egov.field.civil_id"), value: info.civil_id },
          { label: t("egov.field.full_name"), value: `${info.last_name} ${info.first_name}`.trim() },
          { label: t("egov.field.gender"), value: info.gender },
          { label: t("egov.field.address"), value: info.address },
          { label: t("egov.field.passport_status"), value: info.passport_status },
          { label: t("egov.field.verified"), value: String(info.verified) },
        ]);
      } else {
        const info = await api.queryXYPCompany(query.trim());
        setRows([
          { label: t("egov.field.company_reg"), value: info.company_reg },
          { label: t("egov.field.company_name"), value: info.name },
          { label: t("egov.field.executive"), value: info.executive },
          { label: t("egov.field.address"), value: info.address },
          { label: t("egov.field.vat_payer"), value: String(info.vat_payer) },
          { label: t("egov.field.status"), value: info.status },
          { label: t("egov.field.founding_date"), value: info.founding_date },
        ]);
      }
    } catch (err) {
      setError(`${t("egov.message.lookup_failed")}: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen
      icon={<Landmark className="w-6 h-6" />}
      title={t("egov.view.title")}
      subtitle={t("egov.view.subtitle")}
    >
      <div className="flex gap-2">
        {(["citizen", "company"] as const).map((option) => (
          <button
            key={option}
            type="button"
            onClick={() => { setKind(option); setRows(null); setError(""); }}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold border transition ${
              kind === option
                ? "bg-slate-900 text-white border-slate-900"
                : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50"
            }`}
          >
            {t(option === "citizen" ? "egov.tab.citizen" : "egov.tab.company")}
          </button>
        ))}
      </div>

      <form onSubmit={lookup} className="flex flex-wrap gap-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t(kind === "citizen" ? "egov.field.reg_number" : "egov.field.company_reg")}
          className="flex-1 min-w-[16rem] px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-slate-900/10"
        />
        <button
          type="submit"
          disabled={busy || !query.trim()}
          className="px-4 py-2 text-sm font-semibold bg-slate-900 hover:bg-slate-800 disabled:opacity-40 text-white rounded-lg inline-flex items-center gap-2"
        >
          <Search className="w-4 h-4" /> {t("egov.action.lookup")}
        </button>
      </form>

      {error && <ErrorNote>{error}</ErrorNote>}

      {rows ? (
        <Panel>
          <dl className="divide-y divide-slate-100">
            {rows.map((row) => (
              <div key={row.label} className="flex flex-wrap gap-2 px-4 py-3">
                <dt className="text-xs font-semibold text-slate-500 w-56 shrink-0">{row.label}</dt>
                <dd className="text-sm text-slate-900 break-all">{row.value || "—"}</dd>
              </div>
            ))}
          </dl>
        </Panel>
      ) : (
        !error && (
          <Empty icon={<Landmark className="w-10 h-10" />}>
            {t("egov.message.empty_result")}
          </Empty>
        )
      )}

      <p className="text-xs text-slate-400">
        <Chip>{t(kind === "citizen" ? "egov.tab.citizen" : "egov.tab.company")}</Chip>
      </p>
    </Screen>
  );
}
