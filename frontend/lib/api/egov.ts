/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The e-Government app: the state's registers, as this platform reaches them.

import { request } from "./client";
import { storeApi } from "./store";

/** One connection to a state system, as this deployment is configured. */
export interface EgovRail {
  id: string;
  name: string;
  /** "live" | "mock" | "unconfigured" — a mock rail answers, and none of it is authoritative. */
  mode: string;
  endpoint?: string;
}

export interface EgovConnections {
  rails: EgovRail[];
  /** Where a person manages their own linked identities. Not this app's. */
  identities_path: string;
}

export interface EgovHistoryEntry {
  action: string;
  user_id: string;
  details: Record<string, unknown>;
  created_at: string;
}

export const egovApi = {
  queryXYPCitizen: (regNumber: string) =>
    request<{
      reg_number: string;
      civil_id: string;
      last_name: string;
      first_name: string;
      gender: string;
      address: string;
      passport_status: string;
      verified: boolean;
    }>("/egov/citizen", {
      method: "POST",
      body: JSON.stringify({ reg_number: regNumber }),
    }),

  queryXYPCompany: (companyReg: string) =>
    request<{
      company_reg: string;
      name: string;
      executive: string;
      address: string;
      vat_payer: boolean;
      status: string;
      founding_date: string;
    }>("/egov/company", {
      method: "POST",
      body: JSON.stringify({ company_reg: companyReg }),
    }),

  getEgovConnections: () => request<EgovConnections>("/egov/connections"),
  getEgovHistory: () => request<EgovHistoryEntry[]>("/egov/history"),

  // Whether this tenant has the e-Government app.
  //
  // Asked rather than assumed, because the answer changes what a screen should
  // offer and not merely what it shows: contacts pre-fills an address from the
  // citizen registry when it can, and a button that always 403s is worse than
  // no button. Any failure answers false — a screen that cannot find out
  // should degrade the same way as one that found out the answer was no.
  egovInstalled: async (): Promise<boolean> => {
    try {
      const installed = await storeApi.getInstalledApps();
      return (installed || []).some((app) => app.app_id === "io.gerege.nexus.egov" && app.enabled);
    } catch {
      return false;
    }
  },

  // External Integrations Manager.
  //
  // Connectors are per tenant and stored server-side; the secret and any OAuth
  // grant are write-only, so nothing here ever reads a credential back.
};
