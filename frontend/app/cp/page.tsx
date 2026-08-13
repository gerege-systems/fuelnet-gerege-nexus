"use client";

/**
 * The console's front page: every organisation on the deployment.
 *
 * Read-only, and it says so. CP-1 is the foundation — accounts, sessions,
 * audit, isolation — and the buttons that change anything arrive in CP-2 on top
 * of it. A console that grew its actions before its audit trail would have to
 * be trusted rather than checked.
 */

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Building2, Search } from "lucide-react";

import Console from "@/components/cp/Console";
import { cp, type TenantSummary } from "@/lib/cp";
import { useI18n } from "@/lib/i18n";

export default function ControlPlanePage() {
  return (
    <Console>
      <Tenants />
    </Console>
  );
}

function Tenants() {
  const { t, locale } = useI18n();
  const [search, setSearch] = useState("");
  const [tenants, setTenants] = useState<TenantSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");

  const load = useCallback(async (query: string) => {
    setLoading(true);
    try {
      const result = await cp.tenants(query);
      setTenants(result.tenants);
      setFailure("");
    } catch (error) {
      setFailure(error instanceof Error ? error.message : String(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Debounced, so typing a registration number is one query rather than
    // eleven.
    const timer = setTimeout(() => void load(search), 250);
    return () => clearTimeout(timer);
  }, [search, load]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-slate-900">{t("cp.section.tenants")}</h1>
        <p className="mt-1 text-sm text-slate-500">{t("cp.view.subtitle")}</p>
      </div>

      <p className="text-sm rounded-xl bg-amber-50 border border-amber-200 text-amber-900 px-4 py-3">
        {t("cp.message.read_only")}
      </p>

      <div className="relative">
        <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
        <input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("cp.field.search")}
          className="w-full rounded-xl border border-slate-300 bg-white pl-9 pr-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
        />
      </div>

      {failure && (
        <p className="text-sm rounded-lg bg-red-50 text-red-700 border border-red-200 px-3 py-2">
          {t("cp.message.load_failed")}
        </p>
      )}

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="text-left font-medium px-4 py-3">{t("cp.field.organisation")}</th>
                <th className="text-left font-medium px-4 py-3">{t("cp.field.registration")}</th>
                <th className="text-right font-medium px-4 py-3">{t("cp.field.users")}</th>
                <th className="text-right font-medium px-4 py-3">{t("cp.field.apps")}</th>
                <th className="text-left font-medium px-4 py-3">{t("cp.field.last_activity")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {tenants.map((tenant) => (
                <tr key={tenant.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <Link
                      href={`/cp/tenants/${tenant.id}`}
                      className="flex items-center gap-2 font-medium text-slate-900 hover:underline"
                    >
                      <Building2 className="w-4 h-4 text-slate-400" />
                      {tenant.name}
                    </Link>
                    <span className="text-xs text-slate-400">{tenant.slug}</span>
                  </td>
                  <td className="px-4 py-3 text-slate-600">{tenant.registration_number || "—"}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{tenant.user_count}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{tenant.app_count}</td>
                  <td className="px-4 py-3 text-slate-600">
                    {formatMoment(tenant.last_activity_at, locale) || t("cp.message.never")}
                  </td>
                </tr>
              ))}
              {!loading && tenants.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-10 text-center text-slate-500">
                    {t("cp.message.no_tenants")}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

/**
 * Timestamps are rendered in the reader's own locale and time zone.
 *
 * The API sends RFC 3339 with an offset, and the browser is the only party that
 * knows where the person reading it is sitting — the same reasoning the
 * monitoring alerts were changed to follow.
 */
export function formatMoment(value: string | null | undefined, locale: string): string {
  if (!value) return "";
  const moment = new Date(value);
  if (Number.isNaN(moment.getTime())) return "";
  return moment.toLocaleString(locale === "mn" ? "mn-MN" : locale, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}
