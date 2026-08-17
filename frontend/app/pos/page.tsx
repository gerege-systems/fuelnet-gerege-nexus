"use client";

/**
 * Кассын дэлгэц — хоёр өөр ээлжийн загварын аль нэгээр.
 *
 * `io.gerege.nexus.pos` модультай суулгац (Gerege POS) дээр ээлж нь **кассын
 * цэгт** хамаарна: хөтөч бол төхөөрөмж биш, төхөөрөмжийн токен барьдаггүй тул
 * цөмийн `/devices/shifts/*` түүнд 401 хариулна. Тэр модуль байхгүй үед — жишээ
 * нь native клиент цөмийн бүрхүүл рүү орж ирэхэд — доорх хуучин
 * төхөөрөмжийн ээлжийн удирдлага хэвээрээ ажиллана.
 *
 * Тиймээс эхлээд `/pos/registers`-ээс асууна. 403/404 нь "энэ суулгацад касс
 * суугаагүй" гэсэн жинхэнэ хариу бөгөөд алдаа биш: тэр үед төхөөрөмжийн
 * хувилбар руу шилжинэ.
 */

import { useState } from "react";
import { api } from "@/lib/api";
import { useLoadOnMount } from "@/lib/useResource";
import { Banknote, LockKeyhole, MonitorCheck, Plus, Receipt, Trash2 } from "lucide-react";

interface Register { id: string; code: string; name: string; warehouse_id?: string; active: boolean }
interface Report { sales: number; gross: number; vat: number; cash_taken: number; card_taken: number; expected_cash: number; variance: number }
interface Shift { id: string; register_id: string; register_code: string; opened_at: string; opening_float: number; closed_at?: string; counted_cash?: number; notes: string; report: Report }
interface Product { id: string; sku: string; name: string; price: number; active: boolean }
interface CartLine { product: Product; quantity: number }
interface Sale { id: string; receipt_no: string; total: number; vat_amount: number; cash: number; card: number; change_given: number }

const money = (value: number) => `${value.toLocaleString("mn-MN", { maximumFractionDigits: 2 })}₮`;

export default function PosPage() {
  const [hasModule, setHasModule] = useState<boolean | null>(null);
  const [registers, setRegisters] = useState<Register[]>([]);
  const [registerId, setRegisterId] = useState("");
  const [error, setError] = useState("");

  useLoadOnMount(async () => {
    try {
      const list = (await api.getPosRegisters()) || [];
      setRegisters(list);
      setRegisterId((current) => current || list.find((r) => r.active)?.id || "");
      setHasModule(true);
    } catch {
      // Касс суугаагүй суулгац. Алдаа биш — өөр бүтээгдэхүүн.
      setHasModule(false);
    }
  });

  if (hasModule === null) return <div className="py-8 text-sm text-slate-500">Ачаалж байна…</div>;
  if (!hasModule) return <DeviceShiftPanel />;

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <header className="border-b border-slate-300 pb-5">
        <p className="font-mono text-[10px] font-bold tracking-[.22em] text-cyan-700">POINT OF SALE</p>
        <h1 className="mt-2 flex items-center gap-3 text-3xl font-black">
          <MonitorCheck className="h-8 w-8 text-cyan-700" />
          Кассын ажлын хэсэг
        </h1>
      </header>

      {error && <div className="border-l-4 border-red-600 bg-red-50 p-3 text-sm text-red-800">{error}</div>}

      {registers.length === 0 ? (
        <div className="rounded-md border border-slate-300 bg-white p-8 text-center text-sm text-slate-600">
          Кассын цэг бүртгэгдээгүй байна. Эхлээд кассын цэг үүсгэнэ үү.
          <NewRegister onCreated={(reg) => { setRegisters((list) => [...list, reg]); setRegisterId(reg.id); }} onError={setError} />
        </div>
      ) : (
        <>
          <label className="block text-xs font-bold text-slate-600">
            Кассын цэг
            <select
              value={registerId}
              onChange={(e) => setRegisterId(e.target.value)}
              className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm font-bold sm:w-72"
            >
              {registers.map((r) => (
                <option key={r.id} value={r.id}>{r.code} · {r.name}</option>
              ))}
            </select>
          </label>
          {registerId && <RegisterWorkspace key={registerId} registerId={registerId} onError={setError} />}
        </>
      )}
    </div>
  );
}

/** Нэг кассын ээлж ба борлуулалт. */
function RegisterWorkspace({ registerId, onError }: { registerId: string; onError: (message: string) => void }) {
  const [shift, setShift] = useState<Shift | null>(null);
  const [products, setProducts] = useState<Product[]>([]);
  const [query, setQuery] = useState("");
  const [cart, setCart] = useState<CartLine[]>([]);
  const [amount, setAmount] = useState("0");
  const [cash, setCash] = useState("");
  const [card, setCard] = useState("");
  const [sale, setSale] = useState<Sale | null>(null);
  const [busy, setBusy] = useState(false);

  const load = async () => {
    try {
      setShift((await api.getPosShift(registerId)).shift);
    } catch (e) {
      onError(e instanceof Error ? e.message : "Ээлжийн төлөв уншсангүй");
    }
    try {
      setProducts(((await api.getProducts()) || []).filter((p: Product) => p.active));
    } catch {
      // Барааны эрхгүй хэрэглэгч ээлжээ нээж, хаана — сагс нь хоосон байна.
    }
  };
  useLoadOnMount(load);

  const total = cart.reduce((sum, line) => sum + line.product.price * line.quantity, 0);
  const tendered = (Number(cash) || 0) + (Number(card) || 0);

  async function toggleShift() {
    setBusy(true);
    try {
      if (shift) await api.closePosShift(shift.id, Number(amount) || 0);
      else await api.openPosShift(registerId, Number(amount) || 0);
      setAmount("0");
      setCart([]);
      await load();
    } catch (e) {
      onError(e instanceof Error ? e.message : "Ээлж шинэчилсэнгүй");
    } finally {
      setBusy(false);
    }
  }

  function addToCart(product: Product) {
    setCart((lines) => {
      const found = lines.find((l) => l.product.id === product.id);
      if (found) return lines.map((l) => (l.product.id === product.id ? { ...l, quantity: l.quantity + 1 } : l));
      return [...lines, { product, quantity: 1 }];
    });
  }

  async function takePayment() {
    setBusy(true);
    setSale(null);
    try {
      const created = await api.createPosSale(
        registerId,
        cart.map((l) => ({ product_id: l.product.id, quantity: l.quantity })),
        Number(cash) || 0,
        Number(card) || 0,
      );
      setSale(created);
      setCart([]);
      setCash("");
      setCard("");
      await load();
    } catch (e) {
      onError(e instanceof Error ? e.message : "Төлбөр бүртгэгдсэнгүй");
    } finally {
      setBusy(false);
    }
  }

  const matches = query
    ? products.filter((p) => (p.name + " " + p.sku).toLowerCase().includes(query.toLowerCase())).slice(0, 24)
    : products.slice(0, 24);

  return (
    <div className="space-y-6">
      <section className={`rounded-md border p-6 ${shift ? "border-emerald-300 bg-emerald-50" : "border-slate-300 bg-white"}`}>
        <div className="flex items-start gap-4">
          {shift ? <Banknote className="h-8 w-8 text-emerald-700" /> : <LockKeyhole className="h-8 w-8 text-slate-500" />}
          <div>
            <h2 className="text-xl font-black">{shift ? "Ээлж нээлттэй" : "Ээлж хаалттай"}</h2>
            <p className="mt-1 text-sm text-slate-600">
              {shift
                ? `${new Date(shift.opened_at).toLocaleString()} · Нээлтийн үлдэгдэл ${money(shift.opening_float)}`
                : "Борлуулалт эхлэхийн өмнө нээлтийн мөнгөн дүнг баталгаажуулна."}
            </p>
          </div>
        </div>

        {shift && (
          <dl className="mt-5 grid gap-3 text-sm sm:grid-cols-3 lg:grid-cols-5">
            <Figure label="Борлуулалт" value={String(shift.report.sales)} />
            <Figure label="Нийт дүн" value={money(shift.report.gross)} />
            <Figure label="Үүнээс НӨАТ" value={money(shift.report.vat)} />
            <Figure label="Бэлнээр" value={money(shift.report.cash_taken)} />
            <Figure label="Байх ёстой бэлэн" value={money(shift.report.expected_cash)} />
          </dl>
        )}

        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          <label className="text-xs font-bold text-slate-600">
            {shift ? "Тоолсон бэлэн мөнгө" : "Нээлтийн мөнгөн дүн"}
            <input
              type="number"
              min="0"
              step="0.01"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-3 text-lg font-bold"
            />
          </label>
        </div>
        <button
          disabled={busy || Number(amount) < 0}
          onClick={() => void toggleShift()}
          className={`mt-5 rounded px-5 py-3 text-sm font-black text-white ${shift ? "bg-slate-900" : "bg-cyan-700"}`}
        >
          {shift ? "Ээлж хаах" : "Ээлж нээх"}
        </button>
      </section>

      {shift && (
        <section className="grid gap-6 lg:grid-cols-[1fr_20rem]">
          <div className="rounded-md border border-slate-300 bg-white p-5">
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Бараа хайх — нэр эсвэл код"
              className="w-full rounded border border-slate-300 px-3 py-2 text-sm"
            />
            <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {matches.map((p) => (
                <button
                  key={p.id}
                  onClick={() => addToCart(p)}
                  className="rounded border border-slate-200 bg-slate-50 p-3 text-left hover:border-cyan-600"
                >
                  <span className="block font-mono text-[10px] font-bold text-indigo-600">{p.sku}</span>
                  <span className="block text-sm font-bold text-slate-900">{p.name}</span>
                  <span className="block text-sm font-semibold text-slate-700">{money(p.price)}</span>
                </button>
              ))}
              {matches.length === 0 && <p className="text-sm text-slate-500">Бараа олдсонгүй.</p>}
            </div>
          </div>

          <div className="space-y-3 rounded-md border border-slate-300 bg-white p-5">
            <h3 className="flex items-center gap-2 text-sm font-black text-slate-900"><Receipt className="h-4 w-4" /> Сагс</h3>
            {cart.length === 0 && <p className="text-sm text-slate-500">Хоосон.</p>}
            {cart.map((line) => (
              <div key={line.product.id} className="flex items-center justify-between gap-2 border-b border-slate-100 pb-2 text-sm">
                <span className="flex-1 font-semibold text-slate-800">{line.product.name}</span>
                <input
                  type="number"
                  min="0"
                  step="1"
                  value={line.quantity}
                  onChange={(e) => {
                    const quantity = Number(e.target.value);
                    setCart((lines) =>
                      quantity > 0
                        ? lines.map((l) => (l.product.id === line.product.id ? { ...l, quantity } : l))
                        : lines.filter((l) => l.product.id !== line.product.id),
                    );
                  }}
                  className="w-16 rounded border border-slate-300 px-2 py-1 text-right"
                />
                <span className="w-24 text-right font-bold">{money(line.product.price * line.quantity)}</span>
                <button onClick={() => setCart((lines) => lines.filter((l) => l.product.id !== line.product.id))} aria-label="Мөр хасах">
                  <Trash2 className="h-4 w-4 text-slate-400 hover:text-red-600" />
                </button>
              </div>
            ))}

            <p className="flex justify-between border-t border-slate-300 pt-3 text-lg font-black">
              <span>Нийт</span>
              <span>{money(total)}</span>
            </p>

            <label className="block text-xs font-bold text-slate-600">
              Бэлэн
              <input type="number" min="0" step="0.01" value={cash} onChange={(e) => setCash(e.target.value)}
                className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-right text-sm font-bold" />
            </label>
            <label className="block text-xs font-bold text-slate-600">
              Карт
              <input type="number" min="0" step="0.01" value={card} onChange={(e) => setCard(e.target.value)}
                className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-right text-sm font-bold" />
            </label>
            <p className="text-sm text-slate-600">Хариулт: <b>{money(Math.max(0, tendered - total))}</b></p>

            <button
              disabled={busy || cart.length === 0 || tendered + 0.005 < total}
              onClick={() => void takePayment()}
              className="w-full rounded bg-cyan-700 px-5 py-3 text-sm font-black text-white disabled:bg-slate-300"
            >
              Төлбөр авах
            </button>

            {sale && (
              <div className="rounded border border-emerald-300 bg-emerald-50 p-3 text-sm">
                <p className="font-black">Баримт {sale.receipt_no}</p>
                <p>Дүн {money(sale.total)} · НӨАТ {money(sale.vat_amount)}</p>
                <p>Хариулт {money(sale.change_given)}</p>
              </div>
            )}
          </div>
        </section>
      )}
    </div>
  );
}

function Figure({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border border-slate-200 bg-white p-3">
      <dt className="text-[10px] font-bold uppercase tracking-wider text-slate-500">{label}</dt>
      <dd className="mt-1 text-lg font-black text-slate-900">{value}</dd>
    </div>
  );
}

function NewRegister({ onCreated, onError }: { onCreated: (register: Register) => void; onError: (message: string) => void }) {
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);

  async function create() {
    setBusy(true);
    try {
      onCreated(await api.createPosRegister({ code: code.trim(), name: name.trim() }));
      setCode("");
      setName("");
    } catch (e) {
      onError(e instanceof Error ? e.message : "Кассын цэг үүсээгүй");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-5 flex flex-wrap items-end justify-center gap-3">
      <label className="text-left text-xs font-bold text-slate-600">
        Код
        <input value={code} onChange={(e) => setCode(e.target.value)} placeholder="A1"
          className="mt-1 block w-28 rounded border border-slate-300 px-3 py-2 text-sm font-mono" />
      </label>
      <label className="text-left text-xs font-bold text-slate-600">
        Нэр
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Гол лангуу"
          className="mt-1 block w-56 rounded border border-slate-300 px-3 py-2 text-sm" />
      </label>
      <button disabled={busy || !code.trim() || !name.trim()} onClick={() => void create()}
        className="flex items-center gap-2 rounded bg-cyan-700 px-4 py-2 text-sm font-black text-white disabled:bg-slate-300">
        <Plus className="h-4 w-4" /> Үүсгэх
      </button>
    </div>
  );
}

/**
 * Төхөөрөмжийн ээлж — кассын модульгүй суулгацын хувилбар.
 *
 * Энэ бол энэ хуудсанд анх байсан бүхэл дэлгэц. Native клиент төхөөрөмжийн
 * токентойгоо ирэхэд цөмийн `/devices/shifts/*` ажилладаг тул хэвээр үлдэв.
 */
function DeviceShiftPanel() {
  const [shift, setShift] = useState<null | { id: string; opened_at: string; opening_float: number }>(null);
  const [amount, setAmount] = useState("0");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    try {
      setShift((await api.getCurrentShift()).shift);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ээлжийн төлөв уншсангүй");
    }
  }
  useLoadOnMount(load);

  async function act() {
    setBusy(true);
    setError("");
    try {
      if (shift) await api.closeShift(Number(amount), notes);
      else await api.openShift(Number(amount), notes);
      setAmount("0");
      setNotes("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ээлж шинэчилсэнгүй");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <header className="border-b border-slate-300 pb-5">
        <p className="font-mono text-[10px] font-bold tracking-[.22em] text-cyan-700">POINT OF SALE / SHIFT CONTROL</p>
        <h1 className="mt-2 flex items-center gap-3 text-3xl font-black">
          <MonitorCheck className="h-8 w-8 text-cyan-700" />
          POS ажлын хэсэг
        </h1>
      </header>
      {error && <div className="border-l-4 border-red-600 bg-red-50 p-3 text-sm text-red-800">{error}</div>}
      <section className={`rounded-md border p-6 ${shift ? "border-emerald-300 bg-emerald-50" : "border-slate-300 bg-white"}`}>
        <div className="flex items-start gap-4">
          {shift ? <Banknote className="h-8 w-8 text-emerald-700" /> : <LockKeyhole className="h-8 w-8 text-slate-500" />}
          <div>
            <h2 className="text-xl font-black">{shift ? "Ээлж нээлттэй" : "Ээлж хаалттай"}</h2>
            <p className="mt-1 text-sm text-slate-600">
              {shift
                ? `${new Date(shift.opened_at).toLocaleString()} · Нээлтийн үлдэгдэл ${shift.opening_float}`
                : "Борлуулалт эхлэхийн өмнө нээлтийн мөнгөн дүнг баталгаажуулна."}
            </p>
          </div>
        </div>
        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          <label className="text-xs font-bold text-slate-600">
            {shift ? "Хаалтын нийт дүн" : "Нээлтийн мөнгөн дүн"}
            <input type="number" min="0" step="0.01" value={amount} onChange={(e) => setAmount(e.target.value)}
              className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-3 text-lg font-bold" />
          </label>
          <label className="text-xs font-bold text-slate-600">
            Тэмдэглэл
            <input value={notes} onChange={(e) => setNotes(e.target.value)}
              className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-3 text-sm" />
          </label>
        </div>
        <button disabled={busy || Number(amount) < 0} onClick={() => void act()}
          className={`mt-5 rounded px-5 py-3 text-sm font-black text-white ${shift ? "bg-slate-900" : "bg-cyan-700"}`}>
          {shift ? "Ээлж хаах" : "Ээлж нээх"}
        </button>
      </section>
    </div>
  );
}
