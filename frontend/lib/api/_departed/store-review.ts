/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// store review — the module lives in appstore-gerege-mn now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";
import type { StoreVersion } from "../store";
import type { Publisher } from "./publisher";

export const storeReviewApi = {
  getReviewQueue: () => request<StoreVersion[]>("/store-review/queue"),
  decideVersion: (id: string, action: "publish" | "reject" | "yank", note = "") =>
    request<{ status: string }>(`/store-review/versions/${id}`, {
      method: "POST",
      body: JSON.stringify({ action, note }),
    }),
  getReviewPublishers: () => request<Publisher[]>("/store-review/publishers"),
  verifyPublisher: (id: string, verified: boolean) =>
    request<{ verified: boolean }>(
      `/store-review/publishers/${id}/verify?verified=${verified}`, { method: "POST" }),

};
