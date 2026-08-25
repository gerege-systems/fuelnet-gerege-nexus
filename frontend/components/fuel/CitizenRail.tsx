"use client";

/**
 * The rail down the left of the citizen's map.
 *
 * Everything a person needs that is not a place on the map: who they are, what
 * is left of today's ration, the vouchers they are holding, and — before any of
 * that — the way in.
 *
 * It is not the platform's shell. That sidebar lists an organisation's apps and
 * belongs to somebody administering a company; this one belongs to a driver.
 * The map is a public route, so the shell is not rendered around it at all
 * (components/Layout.tsx) and this fills the space instead.
 */

import { useCallback, useEffect, useState } from "react";
import { Fuel, LogIn, QrCode, RefreshCw, Ticket } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";

import { api, type Entitlement, type Voucher } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

const money = (value: number) => Math.round(value).toLocaleString("mn-MN");

export default function CitizenRail({ stationCount, tripCount }: {
  stationCount: number | null;
  tripCount: number;
}) {
  const { t } = useI18n();
  const [entitlement, setEntitlement] = useState<Entitlement | null>(null);
  const [vouchers, setVouchers] = useState<Voucher[]>([]);
  // null while unknown; false once the server has said there is no session.
  const [signedIn, setSignedIn] = useState<boolean | null>(null);
  const [showing, setShowing] = useState<Voucher | null>(null);

  const load = useCallback(async () => {
    try {
      const [ration, mine] = await Promise.all([
        api.myFuelEntitlement(),
        api.myFuelVouchers(),
      ]);
      setEntitlement(ration);
      setVouchers(mine.vouchers);
      setSignedIn(true);
    } catch {
      // A 401 is the ordinary case: most visitors have not signed in. It is a
      // state, not a failure, and it must not read as one.
      setSignedIn(false);
    }
  }, []);

  useEffect(() => {
    void load();
    // The sheet issues vouchers, and this rail has to notice. An event rather
    // than a shared store: the two are siblings on a map, and threading a
    // setter through the map component to reach them would make the map know
    // about vouchers.
    const onIssued = () => void load();
    window.addEventListener("fuel:voucher-issued", onIssued);
    return () => window.removeEventListener("fuel:voucher-issued", onIssued);
  }, [load]);

  const active = vouchers.filter((v) => v.status === "active");

  return (
    <aside className="pointer-events-auto absolute left-0 top-0 z-20 flex h-full w-[300px] max-w-[85vw] flex-col border-r border-slate-200 bg-white/95 backdrop-blur">
      <header className="border-b border-slate-200 px-5 py-4">
        <div className="flex items-center gap-2">
          <Fuel className="h-5 w-5 text-[var(--gerege-blue)]" />
          <h1 className="text-base font-semibold text-slate-900">{t("fuel.map.title")}</h1>
        </div>
        <p className="mt-1 text-xs text-slate-500">
          {stationCount === null ? "…" : `${stationCount} ШТС`}
          {tripCount > 0 ? ` · ${tripCount} цистерн замд` : ""}
        </p>
      </header>

      <div className="flex-1 overflow-y-auto px-5 py-4">
        {signedIn === false ? (
          <>
            <p className="mb-3 text-sm text-slate-600">{t("fuel.rail.why_sign_in")}</p>
            <a
              href="/login"
              className="flex w-full items-center justify-center gap-2 rounded-xl bg-[var(--gerege-blue)] px-4 py-3 font-semibold text-white"
            >
              <LogIn className="h-5 w-5" />
              {t("fuel.rail.sign_in")}
            </a>
            <p className="mt-3 text-xs text-slate-400">{t("fuel.rail.eid_note")}</p>
          </>
        ) : signedIn === null ? (
          <div className="h-20 animate-pulse rounded-xl bg-slate-100" />
        ) : (
          <>
            <section className="rounded-2xl bg-slate-900 px-4 py-4 text-white">
              <p className="text-xs text-slate-300">{t("fuel.sheet.today_remaining")}</p>
              <p className="mt-1 text-3xl font-bold">
                {money(entitlement?.remaining_mnt ?? 0)} <span className="text-lg">₮</span>
              </p>
              <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/20">
                <div
                  className="h-full bg-white"
                  style={{
                    width: `${
                      entitlement && entitlement.granted_mnt > 0
                        ? (entitlement.remaining_mnt / entitlement.granted_mnt) * 100
                        : 0
                    }%`,
                  }}
                />
              </div>
              <p className="mt-2 text-xs text-slate-400">
                {money(entitlement?.used_mnt ?? 0)} / {money(entitlement?.granted_mnt ?? 0)} ₮
              </p>
            </section>

            <div className="mt-5 flex items-center justify-between">
              <h2 className="flex items-center gap-1.5 text-sm font-semibold text-slate-900">
                <Ticket className="h-4 w-4" />
                {t("fuel.rail.my_vouchers")}
              </h2>
              <button
                type="button"
                onClick={() => void load()}
                aria-label={t("fuel.rail.refresh")}
                className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100"
              >
                <RefreshCw className="h-4 w-4" />
              </button>
            </div>

            {active.length === 0 ? (
              <p className="mt-2 rounded-xl border border-dashed border-slate-200 px-4 py-6 text-center text-sm text-slate-400">
                {t("fuel.rail.no_vouchers")}
              </p>
            ) : (
              <ul className="mt-2 space-y-2">
                {active.map((voucher) => (
                  <li key={voucher.id}>
                    <button
                      type="button"
                      onClick={() => setShowing(voucher)}
                      className="flex w-full items-center gap-3 rounded-xl border border-slate-200 px-3 py-2.5 text-left hover:border-slate-300"
                    >
                      <QrCode className="h-5 w-5 shrink-0 text-slate-400" />
                      <span className="min-w-0 flex-1">
                        <span className="block font-semibold text-slate-900">
                          {money(voucher.amount_mnt)} ₮
                        </span>
                        <span className="block truncate text-xs text-slate-500">
                          {voucher.fuel_label} ·{" "}
                          {new Date(voucher.expires_at).toLocaleTimeString("mn-MN", {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                          {" "}хүртэл
                        </span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}

            <p className="mt-4 text-xs text-slate-400">{t("fuel.rail.how_to")}</p>
          </>
        )}
      </div>

      {showing ? (
        // Full-screen, because this is held up to a scanner on a forecourt in
        // daylight. A code in a corner of a sidebar is a code nobody can read.
        <div
          className="fixed inset-0 z-30 flex flex-col items-center justify-center bg-white p-6"
          onClick={() => setShowing(null)}
          role="button"
          tabIndex={0}
          onKeyDown={(event) => event.key === "Escape" && setShowing(null)}
        >
          <p className="text-3xl font-bold text-slate-900">{money(showing.amount_mnt)} ₮</p>
          <p className="mt-1 text-slate-500">{showing.fuel_label}</p>
          <div className="mt-6 rounded-2xl p-4 ring-1 ring-slate-200">
            <QRCodeSVG value={showing.qr_token} size={260} level="M" />
          </div>
          <p className="mt-5 text-sm text-slate-500">{t("fuel.sheet.any_pump")}</p>
          <p className="mt-1 text-xs text-slate-400">{t("fuel.rail.tap_to_close")}</p>
        </div>
      ) : null}
    </aside>
  );
}
