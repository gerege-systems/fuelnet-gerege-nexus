/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// commerce — the module lives in business-gerege-nexus now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";

export const inventoryApi = {
  getWarehouses: () =>
    request<
      Array<{
        id: string;
        code: string;
        name: string;
        address: string;
        created_at: string;
      }>
    >("/inventory/warehouses"),

  createWarehouse: (data: { code: string; name: string; address: string }) =>
    request("/inventory/warehouses", { method: "POST", body: JSON.stringify(data) }),

  getStockLevels: () =>
    request<
      Array<{
        id: string;
        warehouse_id: string;
        product_id: string;
        quantity: number;
        updated_at: string;
      }>
    >("/inventory/stock-levels"),

  getStockMovements: () =>
    request<
      Array<{
        id: string;
        warehouse_id: string;
        product_id: string;
        quantity_change: number;
        reference: string;
        created_at: string;
      }>
    >("/inventory/movements"),

  adjustStock: (data: { warehouse_id: string; product_id: string; quantity_change: number; reference: string }) =>
    request("/inventory/adjustments", { method: "POST", body: JSON.stringify(data) }),

  // AI Assistant & Forecasting
};
