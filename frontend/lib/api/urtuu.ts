/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Өртөө: peers, the code ring, and the task board that travels between installations.

import { request } from "./client";

/**
 * Fired when something on the Өртөө board has moved because of an action taken
 * in this tab.
 *
 * The same shape as APP_MENU_CHANGED_EVENT and for the same reason: the screens
 * sit under a layout that is not re-rendered by a route refresh, so a task
 * accepted on the detail page has to tell the queue behind it. It is a
 * complement to polling, not a replacement — what arrives from another
 * installation arrives without any tab doing anything, and only the poll sees
 * that. No WebSocket: a push from a subordinate lands in a database, and a
 * fifteen-second poll is a truthful description of how fresh that is.
 */
export const URTUU_CHANGED_EVENT = "gerege:urtuu-changed";

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

/**
 * One task at this installation.
 *
 * `direction` is derived by the server: "incoming" is work this organisation
 * owes somebody, "outgoing" is this side's mirror of work it gave to a
 * subordinate, and "local" is its own. `overdue` is computed on every read
 * rather than stored, so an edited deadline never leaves a stale flag.
 */
/**
 * Who asked, on the service line.
 *
 * It travels down with the request — the office that has to issue a certificate
 * cannot issue it to nobody — and nothing else the applying installation knows
 * about them travels with it.
 */
export interface UrtuuApplicant {
  kind: "citizen" | "organisation";
  name: string;
  registry_number?: string;
  contact?: string;
}

export interface UrtuuTask {
  id: string;
  /** This installation's own register number — "Д2026-00412". Quoted, never routed. */
  number?: string;
  /** The number the sending installation registered it under, cited beside that link's name. */
  origin_number?: string;
  code: string;
  /**
   * Which of Өртөө's two promises this is under.
   *
   * "service" — somebody outside the platform asked the state for something,
   * and an answer has to come back to them. "assignment" — a superior
   * organisation gave a subordinate work, and the organisation that raised it
   * is the one waiting.
   */
  line: "service" | "assignment";
  title: string;
  /** Set on the service line only. */
  applicant?: UrtuuApplicant;
  /** What is being told back to the applicant. A service task cannot complete without one. */
  answer?: string;
  payload?: unknown;
  direction: "incoming" | "outgoing" | "local";
  origin_peer_id?: string;
  origin_peer_name?: string;
  target_peer_id?: string;
  target_peer_name?: string;
  parent_task_id?: string;
  origin_chain: string[];
  status: string;
  deadline?: string;
  overdue: boolean;
  assigned_user_id?: string;
  assigned_name?: string;
  note?: string;
  evidence?: unknown;
  created_at: string;
  updated_at: string;
}

/**
 * A reference to an official document behind a task.
 *
 * A reference and never the document: it stays in the documents app of the
 * organisation that filed it, under that organisation's retention and access
 * policy. `installation` says whose `ref` this is — an id is local to whoever
 * filed it, and one quoted without an owner is an id the reader could look up
 * in their own database and find something else under.
 */
export interface UrtuuEvidence {
  kind: string;
  ref: string;
  installation: string;
  title: string;
  signatures: number;
  required_signatures: number;
  signed: boolean;
}

export interface UrtuuTaskEvent {
  from_status?: string;
  to_status: string;
  actor_name?: string;
  peer_name?: string;
  note?: string;
  created_at: string;
}

/** One channel on the board: is it speaking, and is anything stuck behind it. */
export interface UrtuuLinkHealth {
  id: string;
  name: string;
  role: "parent" | "child";
  status: string;
  last_seen_at?: string;
  undelivered: number;
  last_error?: string;
}

/** One delegated task and how far its branches have got. */
export interface UrtuuTreeProgress {
  id: string;
  title: string;
  code: string;
  done: number;
  total: number;
  late: number;
  deadline?: string;
  created_at: string;
}

export interface UrtuuTally {
  direction: "incoming" | "outgoing" | "local";
  line: "service" | "assignment";
  status: string;
  count: number;
  overdue: number;
}

function urtuuChanged<T>(result: T): T {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(URTUU_CHANGED_EVENT));
  }
  return result;
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

  // The Өртөө app (io.gerege.nexus.urtuu): the task board over the channel.
  getUrtuuTasks: (filter: {
    direction?: "incoming" | "outgoing" | "local";
    line?: "service" | "assignment";
    status?: string;
    code?: string;
    overdue?: boolean;
  } = {}) => {
    const query = new URLSearchParams();
    if (filter.direction) query.set("direction", filter.direction);
    if (filter.line) query.set("line", filter.line);
    if (filter.status) query.set("status", filter.status);
    if (filter.code) query.set("code", filter.code);
    if (filter.overdue) query.set("overdue", "true");
    const suffix = query.toString();
    return request<{ tasks: UrtuuTask[] }>(`/urtuu/tasks${suffix ? `?${suffix}` : ""}`);
  },

  /** The task, its whole history and its branches together — see handleGetTask. */
  getUrtuuTask: (id: string) =>
    request<{
      task: UrtuuTask;
      events: UrtuuTaskEvent[];
      branches: UrtuuTask[];
      evidence: UrtuuEvidence[];
      next: string[];
    }>(`/urtuu/tasks/${id}`),

  getUrtuuBoard: () =>
    request<{
      counts: UrtuuTally[];
      overdue: UrtuuTask[];
      links: UrtuuLinkHealth[];
      trees: UrtuuTreeProgress[];
      enabled: boolean;
    }>("/urtuu/tasks/board"),

  createUrtuuTask: (input: {
    code: string;
    title?: string;
    payload?: unknown;
    deadline?: string | null;
    peer_ids?: string[];
    note?: string;
    /** The official document behind the work: one already filed, or one to file now. */
    document?: { document_id?: string; title?: string; type?: string };
    /** Required on the service line, refused on the assignment line. */
    applicant?: UrtuuApplicant;
  }) => request<{ id: string; number: string; status: string }>("/urtuu/tasks", {
    method: "POST",
    body: JSON.stringify(input),
  }).then(urtuuChanged),

  /**
   * One move. The verb is a path segment rather than a status in the body, so a
   * client cannot ask for a transition the server has no handler for.
   */
  moveUrtuuTask: (
    id: string,
    action: "accept" | "return" | "complete" | "close" | "assign" | "delegate",
    body: Record<string, unknown> = {},
  ) => request<{ id: string; status: string }>(`/urtuu/tasks/${id}/${action}`, {
    method: "POST",
    body: JSON.stringify(body),
  }).then(urtuuChanged),

  // Billing App (io.gerege.nexus.billing)
};
