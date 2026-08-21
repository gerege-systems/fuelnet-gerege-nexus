/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// commerce — the module lives in business-gerege-nexus now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";

export const billingApi = {
  getInvoices: () =>
    request<
      Array<{
        id: string;
        invoice_number: string;
        contact_name: string;
        amount: number;
        vat_amount: number;
        ebarimt_status: string;
        status: string;
        created_at: string;
      }>
    >("/billing/invoices"),

  createInvoice: (data: { contact_name: string; amount: number }) =>
    request("/billing/invoices", { method: "POST", body: JSON.stringify(data) }),

  // Documents App (io.gerege.nexus.documents)
  // One page of a tenant's documents, newest first, with how many there are in total —
  // each row counts its own signatures and outstanding steps, so the list cannot be
  // unbounded, and a screen showing part of it has to be able to say so.
};
