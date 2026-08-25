"use client";

/**
 * Fuel network — the stations this organisation operates.
 *
 * A skeleton, matching the module behind it: it calls the one route that
 * exists and renders what comes back, which is nothing. It is here because
 * internal/apps.TestEveryMenuEntryHasARealPage asserts that a menu entry has
 * a screen — a rule worth keeping, since a sidebar link to a 404 is the kind
 * of drift no single-language test can see.
 *
 * What it does prove: the menu came from the server, this route is reachable
 * only when the tenant has the app installed, and the read gate is the one the
 * module declared.
 */

import { useEffect, useState } from "react";
import { Fuel, Loader2 } from "lucide-react";

import { api, type FuelStation } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

export default function FuelPage() {
  const { t } = useI18n();
  const [stations, setStations] = useState<FuelStation[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .listFuelStations()
      .then((result) => alive && setStations(result.stations))
      .catch((err) => alive && setError(err instanceof Error ? err.message : String(err)));
    return () => {
      alive = false;
    };
  }, []);

  return (
    <div className="p-6 max-w-5xl">
      <header className="mb-8">
        <h1 className="text-2xl font-semibold text-slate-900 flex items-center gap-3">
          <Fuel className="w-6 h-6 text-[var(--gerege-blue)]" />
          {t("fuel.view.title")}
        </h1>
        <p className="mt-1 text-slate-500">{t("fuel.view.subtitle")}</p>
      </header>

      {error ? (
        <p className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</p>
      ) : stations === null ? (
        <Loader2 className="w-5 h-5 animate-spin text-slate-400" />
      ) : stations.length === 0 ? (
        <section className="rounded-2xl border border-slate-200 bg-white p-10 text-center">
          <div className="mx-auto mb-4 grid h-14 w-14 place-items-center rounded-2xl bg-[var(--gerege-blue-soft)] text-[var(--gerege-blue)]">
            <Fuel className="h-7 w-7" />
          </div>
          <h2 className="text-lg font-semibold text-slate-900">{t("fuel.view.empty_title")}</h2>
          <p className="mt-2 text-sm text-slate-500">{t("fuel.view.empty_body")}</p>
        </section>
      ) : (
        <>
          <p className="mb-3 text-sm text-slate-500">
            {t("fuel.view.count")}: <b className="text-slate-900">{stations.length}</b>
          </p>
          <ul className="divide-y divide-slate-200 rounded-2xl border border-slate-200 bg-white">
            {stations.map((station) => (
              <li key={station.id} className="flex items-baseline gap-4 px-5 py-3">
                <span className="min-w-0 flex-1 truncate font-medium text-slate-900">
                  {station.name}
                </span>
                <span className="w-36 shrink-0 truncate text-sm text-slate-500">
                  {station.brand_label || station.brand}
                </span>
                <span className="w-40 shrink-0 truncate text-sm text-slate-500">
                  {station.aimag}
                </span>
                <span className="w-16 shrink-0 text-right text-sm text-slate-400">
                  {station.fuel_type_count > 0 ? `${station.fuel_type_count} түлш` : "—"}
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}
