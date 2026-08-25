"use client";

/**
 * The panel that opens when somebody taps a filling station.
 *
 * A React panel rather than a MapLibre popup. The popup was fine while it only
 * showed prices — it takes an HTML string, and a string is enough to render a
 * list. It stops being enough the moment the panel has to do something: taking
 * a voucher is a form, a request, a failure to report and a QR code to draw,
 * and none of that survives being serialised into `setHTML`.
 */

import { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { Fuel, Loader2, LogIn, X, Zap } from "lucide-react";

import { api, type Entitlement, type PublicStation, type Voucher } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

/** The amounts offered. A citizen may take less than the day's whole ration. */
const AMOUNTS = [10_000, 20_000, 30_000, 50_000];

const money = (value: number) => Math.round(value).toLocaleString("mn-MN");

/** Green above 40% of tank, amber above 15, red below. Grey when unreported. */
function levelColour(percent: number | null): string {
  if (percent === null) return "bg-slate-300";
  if (percent < 15) return "bg-red-600";
  if (percent < 40) return "bg-amber-500";
  return "bg-green-600";
}

export default function StationSheet({
  station,
  onClose,
}: {
  station: PublicStation;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [entitlement, setEntitlement] = useState<Entitlement | null>(null);
  // null while unknown, false once the server has said there is no session.
  const [signedIn, setSignedIn] = useState<boolean | null>(null);
  const [fuelType, setFuelType] = useState<string>(station.fuels[0]?.type ?? "ai92");
  const [amount, setAmount] = useState<number>(20_000);
  const [issuing, setIssuing] = useState(false);
  const [voucher, setVoucher] = useState<Voucher | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Ask once, on open. A 401 is the ordinary case here — most visitors are not
  // signed in — so it sets state rather than surfacing as an error.
  useEffect(() => {
    let alive = true;
    api
      .myFuelEntitlement()
      .then((result) => {
        if (!alive) return;
        setEntitlement(result);
        setSignedIn(true);
      })
      .catch(() => alive && setSignedIn(false));
    return () => {
      alive = false;
    };
  }, []);

  async function takeVoucher() {
    setIssuing(true);
    setError(null);
    try {
      const result = await api.issueFuelVoucher({
        amount_mnt: amount,
        fuel_type: fuelType,
        intended_station_id: station.id,
      });
      setVoucher(result.voucher);
      setEntitlement(result.entitlement);
      // The rail holds the same balance and the same list. It listens rather
      // than being handed a setter through the map, which has no business
      // knowing what a voucher is.
      window.dispatchEvent(new CustomEvent("fuel:voucher-issued"));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setIssuing(false);
    }
  }

  const remaining = entitlement?.remaining_mnt ?? 0;
  const affordable = AMOUNTS.filter((value) => value <= remaining);

  return (
    <aside className="pointer-events-auto absolute inset-x-0 bottom-0 z-20 max-h-[75dvh] overflow-y-auto rounded-t-2xl bg-white shadow-2xl ring-1 ring-black/10 sm:inset-y-0 sm:right-0 sm:left-auto sm:w-[380px] sm:max-h-none sm:rounded-none sm:rounded-l-2xl">
      <header className="sticky top-0 flex items-start gap-3 border-b border-slate-200 bg-white px-5 py-4">
        <div className="min-w-0 flex-1">
          <span className="inline-block rounded-md bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-800">
            {station.brand_label || station.brand}
          </span>
          <h2 className="mt-1.5 truncate text-lg font-semibold text-slate-900">{station.name}</h2>
          <p className="truncate text-sm text-slate-500">
            {[station.district, station.aimag].filter(Boolean).join(", ") || "—"}
          </p>
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label={t("web.action.close")}
          className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
        >
          <X className="h-5 w-5" />
        </button>
      </header>

      <div className="px-5 py-4">
        {station.address ? (
          <p className="mb-4 text-sm text-slate-600">{station.address}</p>
        ) : null}

        <h3 className="mb-2 text-sm font-semibold text-slate-900">
          {t("fuel.sheet.stock_title")}
        </h3>
        <ul className="grid gap-2 sm:grid-cols-1">
          {station.fuels.length === 0 ? (
            <li className="rounded-xl border border-slate-200 px-3 py-2 text-sm text-slate-400">
              {t("fuel.map.no_prices")}
            </li>
          ) : (
            station.fuels.map((fuel) => (
              <li key={fuel.type} className="rounded-xl border border-slate-200 px-3 py-2">
                <div className="flex items-baseline justify-between gap-3">
                  <span className="font-medium text-slate-900">{fuel.label}</span>
                  <b className="text-slate-900">{money(fuel.price_mnt)} ₮</b>
                </div>
                <div className="mt-1.5 flex items-center gap-2">
                  <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-slate-100">
                    <div
                      className={`h-full ${levelColour(fuel.stock_percent)}`}
                      style={{ width: `${Math.max(fuel.stock_percent ?? 100, 2)}%`,
                               opacity: fuel.stock_percent === null ? 0.35 : 1 }}
                    />
                  </div>
                  <span className="w-10 text-right text-xs text-slate-500">
                    {fuel.stock_percent === null ? "—" : `${fuel.stock_percent}%`}
                  </span>
                </div>
              </li>
            ))
          )}
        </ul>
      </div>

      <div className="border-t border-slate-200 px-5 py-4">
        {voucher ? (
          // The finished article. Shown here rather than on a page of its own:
          // somebody who has just taken one is standing on a forecourt looking
          // for something to hold up to a scanner.
          <div className="text-center">
            <p className="text-sm text-slate-500">{t("fuel.sheet.voucher_ready")}</p>
            <p className="mt-1 text-2xl font-bold text-slate-900">
              {money(voucher.amount_mnt)} ₮ · {voucher.fuel_label}
            </p>
            <div className="mx-auto mt-4 w-fit rounded-2xl bg-white p-3 ring-1 ring-slate-200">
              <QRCodeSVG value={voucher.qr_token} size={168} level="M" />
            </div>
            <p className="mt-3 text-xs text-slate-500">{t("fuel.sheet.any_pump")}</p>
            <p className="mt-1 text-xs text-slate-400">
              {t("fuel.sheet.valid_until")}{" "}
              {new Date(voucher.expires_at).toLocaleTimeString("mn-MN", {
                hour: "2-digit",
                minute: "2-digit",
              })}
            </p>
          </div>
        ) : signedIn === false ? (
          <a
            href="/login"
            className="flex w-full items-center justify-center gap-2 rounded-xl bg-[var(--gerege-blue)] px-4 py-3 font-semibold text-white"
          >
            <LogIn className="h-5 w-5" />
            {t("fuel.sheet.sign_in_to_take")}
          </a>
        ) : signedIn === null ? (
          <div className="flex justify-center py-2">
            <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
          </div>
        ) : (
          <>
            <div className="mb-3 flex items-baseline justify-between text-sm">
              <span className="text-slate-500">{t("fuel.sheet.today_remaining")}</span>
              <b className="text-slate-900">{money(remaining)} ₮</b>
            </div>

            {remaining <= 0 ? (
              <p className="rounded-xl bg-slate-100 px-4 py-3 text-center text-sm text-slate-500">
                {t("fuel.sheet.spent")}
              </p>
            ) : (
              <>
                <div className="mb-3 flex flex-wrap gap-2">
                  {station.fuels.map((fuel) => (
                    <button
                      key={fuel.type}
                      type="button"
                      onClick={() => setFuelType(fuel.type)}
                      className={`rounded-lg px-3 py-1.5 text-sm ring-1 ${
                        fuelType === fuel.type
                          ? "bg-slate-900 text-white ring-slate-900"
                          : "bg-white text-slate-700 ring-slate-200"
                      }`}
                    >
                      {fuel.label}
                    </button>
                  ))}
                </div>

                <div className="mb-3 grid grid-cols-4 gap-2">
                  {affordable.map((value) => (
                    <button
                      key={value}
                      type="button"
                      onClick={() => setAmount(value)}
                      className={`rounded-lg py-2 text-sm font-medium ring-1 ${
                        amount === value
                          ? "bg-slate-900 text-white ring-slate-900"
                          : "bg-white text-slate-700 ring-slate-200"
                      }`}
                    >
                      {value / 1000}k
                    </button>
                  ))}
                </div>

                {error ? (
                  <p className="mb-2 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
                ) : null}

                <button
                  type="button"
                  disabled={issuing || amount > remaining}
                  onClick={() => void takeVoucher()}
                  className="flex w-full items-center justify-center gap-2 rounded-xl bg-[var(--gerege-blue)] px-4 py-3 font-semibold text-white disabled:opacity-50"
                >
                  {issuing ? (
                    <Loader2 className="h-5 w-5 animate-spin" />
                  ) : (
                    <Zap className="h-5 w-5" />
                  )}
                  {money(amount)} ₮ · {t("fuel.sheet.take_voucher")}
                </button>
                <p className="mt-2 flex items-center justify-center gap-1.5 text-xs text-slate-400">
                  <Fuel className="h-3.5 w-3.5" />
                  {t("fuel.sheet.any_pump")}
                </p>
              </>
            )}
          </>
        )}
      </div>
    </aside>
  );
}
