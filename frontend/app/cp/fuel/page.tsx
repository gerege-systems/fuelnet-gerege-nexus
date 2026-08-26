"use client";

/**
 * Шатахууны харагдац — every operator's records, added up.
 *
 * Read-only, and read straight from the operators' own tables through the
 * console's role (internal/platform/fuel). Nothing here is a copy, so nothing
 * here can be out of step with what an operator sees on their own screen.
 *
 * The screen refuses to merge two different things: what has been reported, and
 * what has merely been left where it was. A row nobody has touched in a day is
 * counted as stale beside the total rather than inside it — a national stock
 * figure that cannot tell those apart is how a shortage is noticed late.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Fuel, RefreshCw, TriangleAlert } from "lucide-react";

import Console from "@/components/cp/Console";
import { Badge, Card, Table, formatMoment } from "@/components/cp/ui";
import { cp, type FuelOverview } from "@/lib/cp";
import { useI18n } from "@/lib/i18n";

const FUEL_LABELS: Record<string, string> = {
  ai80: "АИ-80", ai92: "АИ-92", ai95: "АИ-95", ai98: "АИ-98",
  diesel: "Дизель (ДТ)", euro5_diesel: "Euro-5 ДТ", euro92: "Euro-92", lpg: "Газ (LPG)",
};

function litres(value: number): string {
  return `${new Intl.NumberFormat("mn-MN", { maximumFractionDigits: 0 }).format(value || 0)} л`;
}

function percent(level: number, capacity: number): string {
  return capacity > 0 ? `${Math.round((level / capacity) * 100)}%` : "—";
}

export default function ControlPlaneFuelPage() {
  return (
    <Console>
      <FuelOversight />
    </Console>
  );
}

function FuelOversight() {
  const { locale } = useI18n();
  const [overview, setOverview] = useState<FuelOverview | null>(null);
  const [failure, setFailure] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    setRefreshing(true);
    try {
      setOverview(await cp.fuelOverview());
      setFailure("");
    } catch (error) {
      setFailure(error instanceof Error ? error.message : String(error));
    } finally {
      setRefreshing(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  if (failure) return <p className="rounded-lg bg-rose-50 p-4 text-sm text-rose-700">{failure}</p>;
  if (!overview) return <p className="text-sm text-slate-500">Ачааллаж байна…</p>;

  if (!overview.installed) {
    return (
      <Card title="Шатахууны харагдац">
        <p className="text-sm text-slate-600">
          Нэг ч байгууллага шатахууны аппыг суулгаагүй байна. Суулгасны дараа тэдний ШТС, бааз,
          нөөц, тээвэр энд нэгдэж харагдана.
        </p>
      </Card>
    );
  }

  const stationLitres = overview.stock.reduce((total, row) => total + row.station_liters, 0);
  const depotLitres = overview.stock.reduce((total, row) => total + row.depot_liters, 0);
  const staleRows = overview.stock.reduce((total, row) => total + row.stale, 0);

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span className="grid h-10 w-10 shrink-0 place-content-center rounded-xl bg-amber-50 text-amber-600">
            <Fuel className="h-5 w-5" />
          </span>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">Шатахууны харагдац</h1>
            <p className="mt-0.5 text-sm text-slate-500">
              Суулгац дээрх эрхлэгч бүрийн бүртгэлээс нэгтгэсэн дүр зураг. Зөвхөн унших.
            </p>
          </div>
        </div>
        <button onClick={() => { void load(); }}
                className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700">
          <RefreshCw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} /> Сэргээх
        </button>
      </header>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-6">
        <Stat label="Эрхлэгч" value={String(overview.totals.operators)} />
        <Stat label="ШТС" value={String(overview.totals.stations)} />
        <Stat label="Нөөцийн бааз" value={String(overview.totals.depots)} />
        <Stat label="ШТС дээрх нөөц" value={litres(stationLitres)}
              hint={`баазад ${litres(depotLitres)}`} />
        <Stat label="Замд яваа" value={String(overview.totals.in_transit)}
              hint={`${litres(overview.totals.in_transit_liters)} · 7 хоногт ${litres(overview.totals.received_7d_liters)} буусан`} />
        <Stat label="Хилд" value={String(overview.totals.at_border)}
              hint="баазад хүрээгүй мэдүүлэг" />
      </div>

      {staleRows > 0 && (
        <p className="flex items-center gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-800">
          <TriangleAlert className="h-4 w-4 shrink-0" />
          {staleRows} бүртгэлийн заалт 24 цагаас хуучин. Эдгээрийн үлдэгдэл нь мэдүүлсэн тоо
          бөгөөд хэмжсэн тоо биш.
        </p>
      )}

      <Card title="Түлшний төрлөөр">
        <Table
          head={["Түлш", "ШТС дээр", "Баазад", "Хилд", "Нийт нөөц", "Багтаамжийн", "Хуучирсан"]}
          empty="Нөөцийн мөр алга."
          rows={overview.stock.map((row) => [
            FUEL_LABELS[row.fuel_type] ?? row.fuel_type,
            litres(row.station_liters),
            litres(row.depot_liters),
            litres(row.border_liters),
            litres(row.station_liters + row.depot_liters),
            percent(row.station_liters + row.depot_liters, row.capacity_liters),
            row.stale > 0 ? <Badge tone="amber">{row.stale}</Badge> : "—",
          ])}
        />
      </Card>

      <Card title="Эрхлэгчээр">
        <Table
          head={["Байгууллага", "ШТС", "Бааз", "ШТС дээрх нөөц", "Баазын нөөц", "Замд", "Хилд", "Сүүлийн мэдээлэл"]}
          empty="Бүртгэл хөтөлж буй байгууллага алга."
          rows={overview.operators.map((row) => [
            <Link key={row.id} href={`/cp/tenants/${row.id}`} className="font-medium text-slate-900 hover:underline">
              {row.name}
            </Link>,
            row.stations,
            row.depots,
            litres(row.station_liters),
            litres(row.depot_liters),
            row.in_transit,
            row.at_border,
            row.stale_rows > 0
              ? <Badge tone="amber">{formatMoment(row.last_report_at, locale)}</Badge>
              : formatMoment(row.last_report_at, locale),
          ])}
        />
      </Card>

      <Card title="Түлшгүй, эсвэл дуусах дөхсөн">
        <Table
          head={["ШТС", "Аймаг / хот", "Түлш", "Үлдэгдэл", "Мэдээлсэн"]}
          empty="Дууссан гэж бүртгэгдсэн түлш алга."
          rows={overview.dry.map((row) => [
            `${row.name}${row.brand_label ? ` · ${row.brand_label}` : ""}`,
            `${row.aimag}${row.district ? `, ${row.district}` : ""}`,
            FUEL_LABELS[row.fuel_type] ?? row.fuel_type,
            litres(row.liters),
            formatMoment(row.last_reported_at, locale),
          ])}
        />
      </Card>

      <Card title="Аймаг, хотоор">
        <Table
          head={["Аймаг / хот", "ШТС", "Нөөц"]}
          empty="Байршлын мэдээлэл алга."
          rows={overview.aimags.map((row) => [row.aimag, row.stations, litres(row.liters)])}
        />
      </Card>

      {overview.totals.batches_open > 0 && (
        <p className="text-xs text-slate-400">
          {overview.totals.batches_open} парти бүрэн хүлээж авагдаагүй байна — импортолсон ба ШТС-д
          хүрсэн хэмжээ хоорондоо зөрүүтэй.
        </p>
      )}
    </div>
  );
}

function Stat({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-bold tabular-nums text-slate-900">{value}</p>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  );
}
