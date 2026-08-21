/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// the publisher portal — the module lives in appstore-gerege-mn now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";
import type { StoreApp, StoreVersion } from "../store";

/** An organisation's publishing identity. One per tenant. */
export interface Publisher {
  id: string;
  slug: string;
  name: string;
  contact_email: string;
  verified: boolean;
  verified_at?: string;
  created_at: string;
}

export const publisherApi = {
  getPublisherProfile: () => request<Publisher>("/publisher"),
  savePublisherProfile: (data: { slug: string; name: string; contact_email: string }) =>
    request<Publisher>("/publisher", { method: "PUT", body: JSON.stringify(data) }),
  getPublisherApps: () => request<StoreApp[]>("/publisher/apps"),
  getPublisherVersions: (slug: string) =>
    request<StoreVersion[]>(`/publisher/apps/${slug}/versions`),
  submitVersion: (slug: string, manifest: unknown, channel = "stable") =>
    request<StoreVersion>(`/publisher/apps/${slug}/versions`, {
      method: "POST",
      body: JSON.stringify({ channel, manifest }),
    }),

};
