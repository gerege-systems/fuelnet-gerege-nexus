"use client";

import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Boxes, Warehouse as WarehouseIcon, ArrowUpRight, Plus, Sliders, History } from "lucide-react";

interface Warehouse {
  id: string;
  code: string;
  name: string;
  address: string;
}

interface Product {
  id: string;
  sku: string;
  name: string;
}

interface StockLevel {
  id: string;
  warehouse_id: string;
  product_id: string;
  quantity: number;
}

interface StockMovement {
  id: string;
  warehouse_id: string;
  product_id: string;
  quantity_change: number;
  reference: string;
  created_at: string;
}

export default function InventoryPage() {
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [stockLevels, setStockLevels] = useState<StockLevel[]>([]);
  const [movements, setMovements] = useState<StockMovement[]>([]);
  const [loading, setLoading] = useState(true);

  const [showWhModal, setShowWhModal] = useState(false);
  const [whForm, setWhForm] = useState({ code: "", name: "", address: "" });

  const [showAdjModal, setShowAdjModal] = useState(false);
  const [adjForm, setAdjForm] = useState({ warehouse_id: "", product_id: "", quantity_change: 0, reference: "" });

  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const loadData = async () => {
    try {
      const [wh, prod, stock, mov] = await Promise.all([
        api.getWarehouses(),
        api.getProducts(),
        api.getStockLevels(),
        api.getStockMovements(),
      ]);
      setWarehouses(wh || []);
      setProducts(prod || []);
      setStockLevels(stock || []);
      setMovements(mov || []);
    } catch (err: any) {
      setError(err.message || "Failed to load inventory data. Ensure Inventory app is installed & enabled.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreateWarehouse = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await api.createWarehouse(whForm);
      setShowWhModal(false);
      setWhForm({ code: "", name: "", address: "" });
      setSuccess("Warehouse created successfully!");
      await loadData();
    } catch (err: any) {
      setError(err.message || "Failed to create warehouse");
    }
  };

  const handleAdjustStock = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess("");
    try {
      await api.adjustStock(adjForm);
      setShowAdjModal(false);
      setAdjForm({ warehouse_id: "", product_id: "", quantity_change: 0, reference: "" });
      setSuccess("Stock adjustment recorded successfully!");
      await loadData();
    } catch (err: any) {
      setError(err.message || "Stock adjustment failed");
    }
  };

  const getWhName = (id: string) => warehouses.find((w) => w.id === id)?.name || id;
  const getProdName = (id: string) => products.find((p) => p.id === id)?.name || id;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
            <Boxes className="w-6 h-6 text-amber-600" />
            <span>Inventory & Warehouse Operations</span>
          </h1>
          <p className="text-sm text-slate-500">Track stock levels across warehouses and log adjustments</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowWhModal(true)}
            className="bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 font-medium text-sm py-2 px-3 rounded-lg flex items-center space-x-1.5 transition"
          >
            <WarehouseIcon className="w-4 h-4 text-slate-500" />
            <span>New Warehouse</span>
          </button>
          <button
            onClick={() => {
              if (warehouses.length > 0 && products.length > 0) {
                setAdjForm({
                  warehouse_id: warehouses[0].id,
                  product_id: products[0].id,
                  quantity_change: 1,
                  reference: "Initial stock intake",
                });
              }
              setShowAdjModal(true);
            }}
            className="bg-amber-600 hover:bg-amber-700 text-white font-medium text-sm py-2 px-4 rounded-lg flex items-center space-x-2 transition"
          >
            <Sliders className="w-4 h-4" />
            <span>Adjust Stock</span>
          </button>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg">
          {error}
        </div>
      )}

      {success && (
        <div className="p-4 bg-emerald-50 border border-emerald-200 text-emerald-700 text-sm rounded-lg">
          {success}
        </div>
      )}

      {loading ? (
        <div className="py-8 text-slate-500 text-sm">Loading inventory data...</div>
      ) : (
        <div className="space-y-8">
          {/* Section 1: Warehouses */}
          <div>
            <h2 className="text-lg font-bold text-slate-900 mb-3 flex items-center space-x-2">
              <WarehouseIcon className="w-5 h-5 text-slate-600" />
              <span>Warehouses</span>
            </h2>
            {warehouses.length === 0 ? (
              <div className="bg-white border border-slate-200 rounded-xl p-6 text-center text-slate-500 text-sm">
                No warehouses configured yet. Click "New Warehouse" to add a location.
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {warehouses.map((w) => (
                  <div key={w.id} className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm">
                    <span className="text-xs font-mono font-bold bg-amber-50 text-amber-700 px-2 py-0.5 rounded">
                      {w.code}
                    </span>
                    <h3 className="text-base font-bold text-slate-900 mt-2">{w.name}</h3>
                    <p className="text-xs text-slate-500 mt-1">{w.address || "No address specified"}</p>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Section 2: Stock Levels Overview */}
          <div>
            <h2 className="text-lg font-bold text-slate-900 mb-3">Live Stock Levels</h2>
            {stockLevels.length === 0 ? (
              <div className="bg-white border border-slate-200 rounded-xl p-6 text-center text-slate-500 text-sm">
                No stock recorded yet. Click "Adjust Stock" to perform stock intake.
              </div>
            ) : (
              <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
                <table className="w-full text-left border-collapse text-sm">
                  <thead>
                    <tr className="bg-slate-50 border-b border-slate-200 text-slate-600 font-semibold text-xs uppercase tracking-wider">
                      <th className="py-3 px-4">Warehouse</th>
                      <th className="py-3 px-4">Product</th>
                      <th className="py-3 px-4">Available Quantity</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {stockLevels.map((sl) => (
                      <tr key={sl.id} className="hover:bg-slate-50">
                        <td className="py-3.5 px-4 font-bold text-slate-800">{getWhName(sl.warehouse_id)}</td>
                        <td className="py-3.5 px-4 font-semibold text-slate-900">{getProdName(sl.product_id)}</td>
                        <td className="py-3.5 px-4">
                          <span
                            className={`font-bold font-mono text-sm px-3 py-1 rounded-lg ${
                              sl.quantity > 0
                                ? "bg-emerald-50 text-emerald-700"
                                : "bg-red-50 text-red-700"
                            }`}
                          >
                            {sl.quantity} units
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Section 3: Append-Only Stock Movements Log */}
          <div>
            <h2 className="text-lg font-bold text-slate-900 mb-3 flex items-center space-x-2">
              <History className="w-5 h-5 text-slate-600" />
              <span>Stock Movements History</span>
            </h2>
            {movements.length === 0 ? (
              <div className="bg-white border border-slate-200 rounded-xl p-6 text-center text-slate-500 text-sm">
                No stock movement audit events logged.
              </div>
            ) : (
              <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
                <table className="w-full text-left border-collapse text-sm">
                  <thead>
                    <tr className="bg-slate-50 border-b border-slate-200 text-slate-600 font-semibold text-xs uppercase tracking-wider">
                      <th className="py-3 px-4">Date & Time</th>
                      <th className="py-3 px-4">Warehouse</th>
                      <th className="py-3 px-4">Product</th>
                      <th className="py-3 px-4">Change</th>
                      <th className="py-3 px-4">Reference Note</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {movements.map((m) => (
                      <tr key={m.id} className="hover:bg-slate-50">
                        <td className="py-3 px-4 text-xs font-mono text-slate-500">
                          {new Date(m.created_at).toLocaleString()}
                        </td>
                        <td className="py-3 px-4 font-semibold text-slate-800">{getWhName(m.warehouse_id)}</td>
                        <td className="py-3 px-4 font-semibold text-slate-900">{getProdName(m.product_id)}</td>
                        <td className="py-3 px-4">
                          <span
                            className={`font-mono font-bold ${
                              m.quantity_change > 0 ? "text-emerald-600" : "text-red-600"
                            }`}
                          >
                            {m.quantity_change > 0 ? `+${m.quantity_change}` : m.quantity_change}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-xs text-slate-600">{m.reference || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Warehouse Modal */}
      {showWhModal && (
        <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-xl font-bold text-slate-900 mb-4">Create Warehouse</h2>
            <form onSubmit={handleCreateWarehouse} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Code *</label>
                <input
                  type="text"
                  placeholder="e.g. WH-MAIN"
                  value={whForm.code}
                  onChange={(e) => setWhForm({ ...whForm, code: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Warehouse Name *</label>
                <input
                  type="text"
                  value={whForm.name}
                  onChange={(e) => setWhForm({ ...whForm, name: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Address</label>
                <textarea
                  value={whForm.address}
                  onChange={(e) => setWhForm({ ...whForm, address: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500"
                  rows={3}
                />
              </div>

              <div className="flex items-center space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowWhModal(false)}
                  className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-sm"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="w-1/2 bg-amber-600 hover:bg-amber-700 text-white font-medium py-2 rounded-lg text-sm"
                >
                  Save Warehouse
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Stock Adjustment Modal */}
      {showAdjModal && (
        <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-xl font-bold text-slate-900 mb-4">Stock Adjustment</h2>
            <form onSubmit={handleAdjustStock} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Select Warehouse *</label>
                <select
                  value={adjForm.warehouse_id}
                  onChange={(e) => setAdjForm({ ...adjForm, warehouse_id: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500 bg-white"
                  required
                >
                  {warehouses.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name} ({w.code})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Select Product *</label>
                <select
                  value={adjForm.product_id}
                  onChange={(e) => setAdjForm({ ...adjForm, product_id: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500 bg-white"
                  required
                >
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} (SKU: {p.sku})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Quantity Change (+ to add, - to reduce) *
                </label>
                <input
                  type="number"
                  step="1"
                  value={adjForm.quantity_change}
                  onChange={(e) => setAdjForm({ ...adjForm, quantity_change: parseFloat(e.target.value) || 0 })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Reference / Reason</label>
                <input
                  type="text"
                  placeholder="e.g. PO-98421 or Physical count adjustment"
                  value={adjForm.reference}
                  onChange={(e) => setAdjForm({ ...adjForm, reference: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-amber-500"
                />
              </div>

              <div className="flex items-center space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAdjModal(false)}
                  className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-sm"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="w-1/2 bg-amber-600 hover:bg-amber-700 text-white font-medium py-2 rounded-lg text-sm"
                >
                  Confirm Adjustment
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
