/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The developer portal: OAuth2 clients registered against this deployment's provider.

import { request, type OAuth2Scope } from "./client";

export type OAuth2ClientDraft = {
  client_name: string;
  client_uri?: string;
  client_type?: "confidential" | "public";
  redirect_uris: string[];
  /**
   * Where the platform may return somebody after this application signs them
   * out of it, matched exactly like redirect_uris. Optional: an application
   * that never ends a session here needs none.
   */
  post_logout_redirect_uris?: string[];
  grant_types: string[];
  scopes: string[];
  disabled?: boolean;
};

export type OAuth2Client = {
  id: string;
  client_id: string;
  client_name: string;
  client_uri?: string;
  client_type: "confidential" | "public";
  redirect_uris: string[];
  post_logout_redirect_uris: string[];
  grant_types: string[];
  scopes: string[];
  disabled: boolean;
  created_at: string;
  updated_at: string;
  secret_rotated_at?: string;
  last_used_at?: string;
  /** Present only in the response that created or rotated it. */
  client_secret?: string;
};

export type SigningKey = {
  kid: string;
  algorithm: string;
  active: boolean;
  created_at: string;
  retired_at?: string;
};

export type ClientActivity = {
  client_id: string;
  client_name: string;
  client_type: "confidential" | "public";
  disabled: boolean;
  active_access_tokens: number;
  active_refresh_tokens: number;
  consented_users: number;
  last_used_at?: string;
};

export type ConsentRecord = {
  client_id: string;
  client_name: string;
  user_id: string;
  user_email: string;
  user_name: string;
  scopes: string[];
  granted_at: string;
};

export const ssoClientsApi = {
  getSSOClients: () => request<OAuth2Client[]>("/sso-clients/apps"),
  getSSOClient: (clientID: string) =>
    request<OAuth2Client>(`/sso-clients/apps/${encodeURIComponent(clientID)}`),
  createSSOClient: (app: OAuth2ClientDraft) =>
    request<OAuth2Client>("/sso-clients/apps", { method: "POST", body: JSON.stringify(app) }),
  updateSSOClient: (clientID: string, app: OAuth2ClientDraft) =>
    request<OAuth2Client>(`/sso-clients/apps/${encodeURIComponent(clientID)}`, {
      method: "PUT",
      body: JSON.stringify(app),
    }),
  deleteSSOClient: (clientID: string) =>
    request<void>(`/sso-clients/apps/${encodeURIComponent(clientID)}`, { method: "DELETE" }),
  rotateSSOClientSecret: (clientID: string) =>
    request<OAuth2Client>(`/sso-clients/apps/${encodeURIComponent(clientID)}/rotate-secret`, {
      method: "POST",
    }),
  getSSOClientScopes: () =>
    request<{ scopes: OAuth2Scope[]; grant_types: string[] }>("/sso-clients/scopes"),
  getSSOClientEndpoints: () => request<Record<string, string>>("/sso-clients/endpoints"),
  getSSOSigningKeys: () =>
    request<{ keys: SigningKey[]; jwks_uri: string }>("/sso-clients/signing-keys"),
  getSSOClientAudit: () =>
    request<{ clients: ClientActivity[]; consents: ConsentRecord[] }>("/sso-clients/audit"),
  revokeSSOClientTokens: (clientID: string) =>
    request<{ revoked: number }>(`/sso-clients/apps/${encodeURIComponent(clientID)}/tokens`, {
      method: "DELETE",
    }),
  withdrawSSOClientConsent: (clientID: string, userID: string) =>
    request<void>(
      `/sso-clients/apps/${encodeURIComponent(clientID)}/consents/${encodeURIComponent(userID)}`,
      { method: "DELETE" },
    ),

  // OAuth2 consent screen. The query string is the authorization request the
  // browser arrived with; the server re-validates all of it rather than
  // trusting what the page echoes back.
};
