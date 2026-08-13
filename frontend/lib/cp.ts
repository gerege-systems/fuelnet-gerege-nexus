/**
 * The operator console's client.
 *
 * Deliberately not part of lib/api.ts. That module speaks to the platform on
 * behalf of a tenant user: it knows about the session cookie, the tenant
 * header, the re-login dance the shell performs on a 401. None of that applies
 * here, and sharing the module would mean every screen in the product carrying
 * the console's calls in its bundle.
 *
 * Addresses are relative. The console is served on its own hostname and its API
 * is /cp/api on that same hostname, so a relative path is always right and an
 * absolute one — the pattern lib/apiBase.ts has to unpick for the device lines
 * — would be a way to get it wrong.
 */

const BASE = "/cp/api";

export type OperatorRole = "superadmin" | "operator" | "support" | "auditor";

export interface Operator {
  id: string;
  email: string;
  name: string;
  role: OperatorRole;
}

export interface Me {
  operator: Operator;
  expires_at: string;
  stepped_up: boolean;
}

export interface TenantSummary {
  id: string;
  slug: string;
  name: string;
  registration_number: string;
  created_at: string;
  user_count: number;
  app_count: number;
  last_activity_at: string | null;
}

export interface TenantApp {
  id: string;
  name: string;
  version: string;
  status: string;
  enabled: boolean;
  installed_at: string;
}

export interface TenantMember {
  user_id: string;
  email: string;
  name: string;
  roles: string[];
}

export interface TenantActivity {
  action: string;
  resource: string;
  user_id: string;
  created_at: string;
}

export interface AuditEntry {
  id: string;
  operator_id: string;
  operator_email: string;
  action: string;
  target_type: string;
  target_id: string;
  reason: string;
  before: unknown;
  after: unknown;
  ip: string;
  created_at: string;
}

export interface TenantDetail extends TenantSummary {
  legal_name: string;
  tax_number: string;
  apps: TenantApp[];
  members: TenantMember[];
  activity: TenantActivity[];
  operator_actions: AuditEntry[];
}

/**
 * Unauthorized is what every screen checks for to decide whether to show the
 * sign-in form. A distinct class rather than a status code compared at each
 * call site, so that a screen cannot forget which number meant what.
 */
export class Unauthorized extends Error {
  constructor() {
    super("unauthorized");
    this.name = "Unauthorized";
  }
}

/** StepUpRequired mirrors the API's `step_up_required` code. */
export class StepUpRequired extends Error {
  constructor() {
    super("step up required");
    this.name = "StepUpRequired";
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(BASE + path, {
    ...init,
    // The session is a cookie, and fetch does not send cookies unless told to
    // even on same-origin requests with a custom method.
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init.headers || {}) },
  });

  if (response.status === 401) throw new Unauthorized();

  let body: { error?: string; code?: string } | null = null;
  const text = await response.text();
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      // A response that is not JSON is a proxy's error page, not the API's.
      // The status is the only thing worth reporting from it.
    }
  }

  if (!response.ok) {
    if (body?.code === "step_up_required") throw new StepUpRequired();
    throw new Error(body?.error || `request failed with ${response.status}`);
  }
  return (body ?? {}) as T;
}

export const cp = {
  me: () => request<Me>("/me"),

  signIn: (email: string, password: string, code: string) =>
    request<{ operator: Operator; expires_at: string }>("/session", {
      method: "POST",
      body: JSON.stringify({ email, password, code }),
    }),

  signOut: () => request<{ status: string }>("/session", { method: "DELETE" }),

  stepUp: (code: string) =>
    request<{ stepped_up_until: string }>("/step-up", {
      method: "POST",
      body: JSON.stringify({ code }),
    }),

  tenants: (search: string) =>
    request<{ tenants: TenantSummary[] }>(`/tenants?q=${encodeURIComponent(search)}`),

  tenant: (id: string) => request<TenantDetail>(`/tenants/${encodeURIComponent(id)}`),

  audit: (params: { action?: string; target_type?: string; target_id?: string } = {}) => {
    const query = new URLSearchParams(
      Object.entries(params).filter(([, value]) => value) as [string, string][],
    );
    return request<{ entries: AuditEntry[] }>(`/audit?${query.toString()}`);
  },

  operators: () => request<{ operators: (Operator & { disabled_at: string | null; last_login_at: string | null; created_at: string })[] }>("/operators"),
};
