"use client";

/**
 * Customs declarations — the top of the chain.
 *
 * The one thing this screen has to make unmistakable is that **clearing** is
 * the act that mints a batch, not declaring. A consignment under inspection may
 * still be turned back, and a batch that existed before the state let the fuel
 * in would be a chain link attached to nothing. So the batch column is empty
 * until the row is cleared, and the button that clears it is the one that asks
 * for the laboratory result.
 */

import { useCallback, useEffect, useState } from "react";
import { Loader2, Plus, ShieldCheck, TestTube2 } from "lucide-react";

import { api, type Shipment } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  ErrorNote,
  Field,
  GhostButton,
  PrimaryButton,
  StatusPill,
  inputClass,
  litres,
} from "@/components/fuel/operatorUI";

/** The grades this platform knows. Kept in step with fuel/entitlement.go. */
const GRADES = [
  { code: "ai92", label: "АИ-92" },
  { code: "ai95", label: "АИ-95" },
  { code: "ai98", label: "АИ-98" },
  { code: "ai80", label: "АИ-80" },
  { code: "diesel", label: "Дизель (ДТ)" },
];

const PORTS = ["Сүхбаатар", "Замын-Үүд", "Алтанбулаг", "Боршоо", "Гашуунсухайт", "Бичигт"];

export default function ShipmentsPage() {
  const { t } = useI18n();
  const [shipments, setShipments] = useState<Shipment[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [declaring, setDeclaring] = useState(false);
  const [clearing, setClearing] = useState<string | null>(null);

  const load = useCallback(() => {
    api
      .listFuelShipments()
      .then((result) => setShipments(result.shipments))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  useEffect(load, [load]);

  const statusLabel = useCallback(
    (status: string) => t(`fuel.ship.status.${status}`) || status,
    [t],
  );

  return (
    <div>
      <div className="mb-6 flex items-start justify-between gap-6">
        <p className="max-w-2xl text-slate-500">{t("fuel.ship.subtitle")}</p>
        <PrimaryButton onClick={() => setDeclaring(true)} className="shrink-0">
          <Plus className="h-4 w-4" />
          {t("fuel.ship.declare")}
        </PrimaryButton>
      </div>

      {error ? <ErrorNote>{error}</ErrorNote> : null}

      {declaring ? (
        <DeclareForm
          onCancel={() => setDeclaring(false)}
          onDone={() => {
            setDeclaring(false);
            load();
          }}
        />
      ) : null}

      {shipments === null ? (
        <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
      ) : shipments.length === 0 ? (
        <p className="rounded-2xl border border-slate-200 bg-white p-10 text-center text-sm text-slate-500">
          {t("fuel.ship.empty")}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white">
          <table className="w-full text-sm">
            <thead className="border-b border-slate-200 text-left text-xs text-slate-500">
              <tr>
                <th className="px-4 py-3 font-medium">{t("fuel.ship.declaration_no")}</th>
                <th className="px-4 py-3 font-medium">{t("fuel.ship.border_port")}</th>
                <th className="px-4 py-3 font-medium">{t("fuel.ship.fuel_type")}</th>
                <th className="px-4 py-3 text-right font-medium">{t("fuel.ship.liters")}</th>
                <th className="px-4 py-3 font-medium">{t("fuel.ship.batch")}</th>
                <th className="px-4 py-3 font-medium">&nbsp;</th>
                <th className="px-4 py-3 font-medium">&nbsp;</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {shipments.map((shipment) => (
                <tr key={shipment.id} className="align-middle">
                  <td className="px-4 py-3">
                    <span className="font-medium text-slate-900">{shipment.declaration_no}</span>
                    <span className="block text-xs text-slate-400">
                      {shipment.exporter || shipment.origin_country || "—"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-slate-600">{shipment.border_port || "—"}</td>
                  <td className="px-4 py-3 text-slate-600">
                    {shipment.fuel_label || shipment.fuel_type}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums text-slate-900">
                    {litres(shipment.declared_liters)}
                    {shipment.declared_tons ? (
                      <span className="block text-xs text-slate-400">
                        {shipment.declared_tons.toLocaleString("mn-MN")} т
                      </span>
                    ) : null}
                  </td>
                  <td className="px-4 py-3">
                    {shipment.batch_code ? (
                      <code className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700">
                        {shipment.batch_code}
                      </code>
                    ) : (
                      <span className="text-xs text-slate-400">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill status={shipment.status} label={statusLabel(shipment.status)} />
                  </td>
                  <td className="px-4 py-3 text-right">
                    {shipment.status === "border_arrived" || shipment.status === "inspecting" ? (
                      <button
                        onClick={() => setClearing(shipment.id)}
                        className="inline-flex items-center gap-1.5 text-xs font-medium text-[var(--gerege-blue)] hover:underline"
                      >
                        <ShieldCheck className="h-3.5 w-3.5" />
                        {t("fuel.ship.clear")}
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {clearing ? (
        <ClearDialog
          shipmentId={clearing}
          onCancel={() => setClearing(null)}
          onDone={() => {
            setClearing(null);
            load();
          }}
        />
      ) : null}
    </div>
  );
}

/** Declaring a consignment. No status field: everything starts at the border. */
function DeclareForm({ onCancel, onDone }: { onCancel: () => void; onDone: () => void }) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState({
    declaration_no: "",
    border_port: PORTS[0],
    origin_country: "ОХУ",
    exporter: "",
    fuel_type: GRADES[0].code,
    declared_liters: "",
    declared_tons: "",
    wagons: "",
    convoy_code: "",
  });

  const set = (key: keyof typeof form) => (event: { target: { value: string } }) =>
    setForm((current) => ({ ...current, [key]: event.target.value }));

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.createFuelShipment({
        declaration_no: form.declaration_no.trim(),
        border_port: form.border_port,
        origin_country: form.origin_country,
        exporter: form.exporter,
        fuel_type: form.fuel_type,
        declared_liters: Number(form.declared_liters),
        declared_tons: form.declared_tons ? Number(form.declared_tons) : null,
        wagons: form.wagons ? Number(form.wagons) : 0,
        convoy_code: form.convoy_code,
      });
      onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form
      onSubmit={submit}
      className="mb-6 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Field label={t("fuel.ship.declaration_no")}>
          <input
            required
            className={inputClass}
            value={form.declaration_no}
            onChange={set("declaration_no")}
            placeholder="ГМ-2026-000000"
          />
        </Field>
        <Field label={t("fuel.ship.border_port")}>
          <select className={inputClass} value={form.border_port} onChange={set("border_port")}>
            {PORTS.map((port) => (
              <option key={port}>{port}</option>
            ))}
          </select>
        </Field>
        <Field label={t("fuel.ship.origin")}>
          <input className={inputClass} value={form.origin_country} onChange={set("origin_country")} />
        </Field>
        <Field label={t("fuel.ship.exporter")}>
          <input className={inputClass} value={form.exporter} onChange={set("exporter")} />
        </Field>
        <Field label={t("fuel.ship.fuel_type")}>
          <select className={inputClass} value={form.fuel_type} onChange={set("fuel_type")}>
            {GRADES.map((grade) => (
              <option key={grade.code} value={grade.code}>
                {grade.label}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t("fuel.ship.liters")}>
          <input
            required
            type="number"
            min="1"
            step="1"
            className={inputClass}
            value={form.declared_liters}
            onChange={set("declared_liters")}
          />
        </Field>
        <Field label={t("fuel.ship.tons")} hint={t("fuel.ship.tons_note")}>
          <input
            type="number"
            min="0"
            step="0.001"
            className={inputClass}
            value={form.declared_tons}
            onChange={set("declared_tons")}
          />
        </Field>
        <Field label={t("fuel.ship.wagons")}>
          <input type="number" min="0" className={inputClass} value={form.wagons} onChange={set("wagons")} />
        </Field>
        <Field label={t("fuel.ship.convoy")}>
          <input className={inputClass} value={form.convoy_code} onChange={set("convoy_code")} />
        </Field>
      </div>

      {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}

      <div className="mt-5 flex gap-2">
        <PrimaryButton type="submit" busy={busy}>
          {t("fuel.common.save")}
        </PrimaryButton>
        <GhostButton type="button" onClick={onCancel}>
          {t("fuel.common.cancel")}
        </GhostButton>
      </div>
    </form>
  );
}

/**
 * Clearing a consignment, with the laboratory result attached.
 *
 * The certificate travels on the batch this request mints, and from there to
 * every tanker and forecourt that carries those litres. Asking for it here —
 * at the one moment the chain starts — is what makes tracing a bad octane
 * figure back to its origin possible at all.
 */
function ClearDialog({
  shipmentId,
  onCancel,
  onDone,
}: {
  shipmentId: string;
  onCancel: () => void;
  onDone: () => void;
}) {
  const { t } = useI18n();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState({ quality_cert_no: "", octane_tested: "", sulfur_ppm: "" });

  const set = (key: keyof typeof form) => (event: { target: { value: string } }) =>
    setForm((current) => ({ ...current, [key]: event.target.value }));

  const send = async (status: string) => {
    setBusy(true);
    setError(null);
    try {
      await api.advanceFuelShipment(shipmentId, {
        status,
        lab_status: status === "cleared" ? "passed" : "pending",
        quality_cert_no: form.quality_cert_no,
        octane_tested: form.octane_tested ? Number(form.octane_tested) : null,
        sulfur_ppm: form.sulfur_ppm ? Number(form.sulfur_ppm) : null,
      });
      onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 grid place-items-center bg-slate-900/40 p-4">
      <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
        <h2 className="mb-1 flex items-center gap-2 text-lg font-semibold text-slate-900">
          <TestTube2 className="h-5 w-5 text-[var(--gerege-blue)]" />
          {t("fuel.ship.clear")}
        </h2>
        <p className="mb-5 text-sm text-slate-500">{t("fuel.ship.subtitle")}</p>

        <div className="grid gap-4">
          <Field label={t("fuel.ship.cert_no")}>
            <input className={inputClass} value={form.quality_cert_no} onChange={set("quality_cert_no")} />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label={t("fuel.ship.octane")}>
              <input
                type="number"
                step="0.1"
                className={inputClass}
                value={form.octane_tested}
                onChange={set("octane_tested")}
              />
            </Field>
            <Field label={t("fuel.ship.sulfur")}>
              <input
                type="number"
                step="0.1"
                className={inputClass}
                value={form.sulfur_ppm}
                onChange={set("sulfur_ppm")}
              />
            </Field>
          </div>
        </div>

        {error ? <p className="mt-4 text-sm text-red-600">{error}</p> : null}

        <div className="mt-6 flex flex-wrap gap-2">
          <PrimaryButton busy={busy} onClick={() => send("cleared")}>
            <ShieldCheck className="h-4 w-4" />
            {t("fuel.ship.clear")}
          </PrimaryButton>
          <GhostButton disabled={busy} onClick={() => send("inspecting")}>
            {t("fuel.ship.inspect")}
          </GhostButton>
          <GhostButton disabled={busy} onClick={onCancel} className="ml-auto">
            {t("fuel.common.cancel")}
          </GhostButton>
        </div>
      </div>
    </div>
  );
}
