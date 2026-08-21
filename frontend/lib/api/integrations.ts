/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Outbound integrations: conferencing, webhooks and the delivery log behind
// them. Core: the platform holds the credentials and the OAuth dance, and a
// module that books a meeting reaches them through nexus.Meetings() rather
// than through here.

import { request } from "./client";

export type IntegrationProvider =
  | "webhook"
  | "government"
  | "payment"
  | "custom_rest"
  | "google_drive"
  | "dropbox"
  | "google_meet";

export interface Integration {
  id: string;
  provider: IntegrationProvider;
  name: string;
  target_url: string;
  /** The administrator's intent. A failure is reported in last_error and does
   *  not switch the connector off. */
  status: "ACTIVE" | "INACTIVE";
  config: Record<string, string>;
  account_label: string;
  /** True once an OAuth grant is stored. The token itself never comes back. */
  connected: boolean;
  connected_at?: string;
  last_ping_at?: string;
  last_error?: string;
  capabilities: string[];
  created_at: string;
  updated_at: string;
}

export interface IntegrationInput {
  provider: IntegrationProvider;
  name: string;
  target_url?: string;
  /** Write-only. Left blank on an update it means "unchanged", not "clear it". */
  secret_key?: string;
  status?: string;
  config?: Record<string, string>;
}

export const integrationsApi = {
  getIntegrations: () => request<Integration[]>("/integrations"),

  // Which providers this deployment can actually offer. A provider whose OAuth
  // client was never configured comes back unavailable with the reason, so the
  // screen can say why instead of showing a form that cannot work.
  getIntegrationProviders: () =>
    request<{
      providers: Array<{
        provider: IntegrationProvider;
        oauth: boolean;
        capabilities: string[];
        available: boolean;
        reason?: string;
      }>;
      encryption_configured: boolean;
      redirect_uri: string;
    }>("/integrations/providers"),

  registerIntegration: (data: IntegrationInput) =>
    request<Integration>("/integrations", { method: "POST", body: JSON.stringify(data) }),

  updateIntegration: (id: string, data: IntegrationInput) =>
    request<Integration>(`/integrations/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  deleteIntegration: (id: string) =>
    request<{ status: string }>(`/integrations/${id}`, { method: "DELETE" }),

  // Starts the OAuth grant. The answer is the provider URL to send the
  // administrator to; the callback lands back on the settings screen.
  connectIntegration: (id: string) =>
    request<{ authorization_url: string }>(`/integrations/${id}/connect`, { method: "POST" }),

  disconnectIntegration: (id: string) =>
    request<{ status: string }>(`/integrations/${id}/disconnect`, { method: "POST" }),

  // What has recently left the platform. A signed document reaching an outside
  // account is a disclosure, and this is the record of it.
  getIntegrationDeliveries: (limit = 50) =>
    request<
      Array<{
        id: string;
        integration_id: string;
        kind: string;
        reference: string;
        outcome: "OK" | "FAILED";
        detail?: string;
        external_id?: string;
        external_url?: string;
        created_at: string;
      }>
    >(`/integrations/deliveries?limit=${limit}`),

  // Send an already-signed document to a storage connector. Automatic export
  // covers documents signed after a connector was set up; this covers the ones
  // signed before it, and the retry after a destination was unreachable.
};
