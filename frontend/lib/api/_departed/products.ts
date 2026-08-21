/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// commerce — the module lives in business-gerege-nexus now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";

export const productsApi = {
  getProducts: () =>
    request<
      Array<{
        id: string;
        sku: string;
        name: string;
        price: number;
        active: boolean;
        created_at: string;
      }>
    >("/products"),

  createProduct: (data: { sku: string; name: string; price: number; active: boolean }) =>
    request("/products", { method: "POST", body: JSON.stringify(data) }),

  updateProduct: (id: string, data: { sku: string; name: string; price: number; active: boolean }) =>
    request(`/products/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  // Inventory App
};
