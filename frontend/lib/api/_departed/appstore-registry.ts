/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// the registry — the module lives in appstore-gerege-mn now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";

export const appstoreRegistryApi = {
  getRegistryState: () =>
    request<{ revision: number; key_id: string; public_key: string }>("/appstore/registry/state"),
  rebuildCatalogue: () =>
    request<{ discarded: number }>("/appstore/registry/rebuild", { method: "POST" }),

  // Whether an app follows the catalogue on its own. Turning it on also clears
  // a hold, which is why this refreshes the menus like the other store
  // mutations: an app held back can start contributing menus again.
};
