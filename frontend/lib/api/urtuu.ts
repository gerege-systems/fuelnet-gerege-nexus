/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Өртөө: peers and the request codes (Settings → Өртөө).

import { request } from "./client";

/**
 * One Өртөө link, as Settings → Өртөө sees it.
 *
 * `role` is what *this* installation is on the link, not what the other side
 * is: the graph is directed and an installation in the middle of a chain is a
 * child on one link and a parent on the next.
 */
export interface UrtuuPeer {
  id: string;
  name: string;
  role: "parent" | "child";
  base_url?: string;
  status: "pending" | "active" | "revoked";
  peer_public_key?: string;
  invite_expires_at?: string;
  last_seen_at?: string;
  last_error?: string;
  /** Seconds the other side's clock differs by. Reported, never corrected for. */
  clock_skew_seconds: number;
  /** Envelopes queued for this link that have not been acknowledged. */
  undelivered: number;
  revoked_at?: string;
  created_at: string;
}

/** One request code: what a task may be raised under. */
export interface UrtuuCode {
  id: string;
  code: string;
  names: Record<string, string>;
  /** Which line work raised under this code belongs to. The code decides, not the raiser. */
  line: "service" | "assignment";
  schema?: unknown;
  /** Null where the code names no norm, which is not the same as a norm of zero. */
  default_sla_seconds: number | null;
  source: "ring" | "link" | "local";
  source_peer_id?: string;
  source_peer_name?: string;
  ring_process_ref?: string;
  version: number;
  active: boolean;
  /** Link ids this code has been announced on. */
  open_to: string[];
  updated_at: string;
}

export const urtuuApi = {
  getUrtuuPeers: () =>
    request<{
      peers: UrtuuPeer[];
      enabled: boolean;
      installation_id: string;
      public_key: string;
    }>("/urtuu/peers"),

  /** The invitation code comes back exactly once, in this response. */
  inviteUrtuuPeer: (name: string) =>
    request<{ id: string; invite_code: string; expires_in_hours: number }>("/urtuu/peers/invite", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),

  joinUrtuuParent: (input: { invite_code: string; base_url: string; name: string }) =>
    request<{ id: string; parent_installation_id: string }>("/urtuu/peers", {
      method: "POST",
      body: JSON.stringify(input),
    }),

  confirmUrtuuPeer: (id: string) =>
    request<{ id: string }>(`/urtuu/peers/${id}/confirm`, { method: "POST" }),

  revokeUrtuuPeer: (id: string) =>
    request<{ id: string }>(`/urtuu/peers/${id}/revoke`, { method: "POST" }),

  /** Replaces the whole set: what is sent is what the link may carry. */
  setUrtuuPeerCodes: (id: string, codes: string[]) =>
    request<{ peer_id: string; codes: number }>(`/urtuu/peers/${id}/codes`, {
      method: "PUT",
      body: JSON.stringify({ codes }),
    }),

  getUrtuuCodes: () =>
    request<{ codes: UrtuuCode[]; ring_configured: boolean }>("/urtuu/codes"),

  createUrtuuCode: (input: {
    code: string;
    names: Record<string, string>;
    line?: "service" | "assignment";
    schema?: unknown;
    default_sla_seconds?: number | null;
  }) => request<{ id: string; code: string }>("/urtuu/codes", {
    method: "POST",
    body: JSON.stringify(input),
  }),

  updateUrtuuCode: (
    id: string,
    input: {
      names?: Record<string, string>;
      schema?: unknown;
      default_sla_seconds?: number | null;
      active?: boolean;
    },
  ) => request<{ id: string }>(`/urtuu/codes/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  }),

  /** `unchanged` means the register published nothing new — an answer, not a failure. */
  syncUrtuuRing: () =>
    request<{ imported: number; unchanged?: boolean }>("/urtuu/codes/ring-sync", { method: "POST" }),
};
