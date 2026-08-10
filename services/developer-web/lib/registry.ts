import { readSession } from "./session";

/**
 * The console's calls to the registry.
 *
 * All of them run on the server with the identity token from the httpOnly
 * cookie. The browser never holds the token and never talks to the registry, so
 * there is no CORS to configure and nothing to steal out of a page.
 */

export const REGISTRY_URL = (
  process.env.REGISTRY_URL || "http://127.0.0.1:8083"
).replace(/\/$/, "");

export interface Publisher {
  id: string;
  slug: string;
  name: string;
  contact_email: string;
  verified: boolean;
  owner_tenant_slug: string;
}

export interface Me {
  subject: string;
  email: string;
  name: string;
  tenant_slug: string;
  admin: boolean;
  publisher: Publisher | null;
}

export interface App {
  id: string;
  slug: string;
  type: "module" | "external";
  name: string;
  description: string;
  category: string;
  visibility: string;
  latest_version?: string;
}

export interface Version {
  id: string;
  app_id: string;
  version: string;
  channel: string;
  min_platform: string;
  status: "draft" | "in_review" | "published" | "rejected" | "yanked";
  submitted_by?: string;
  review_note?: string;
  published_at?: string;
  created_at: string;
  app?: App;
  manifest?: Record<string, unknown>;
}

export type Result<T> = { data: T } | { error: string; status: number };

async function call<T>(path: string, init?: RequestInit): Promise<Result<T>> {
  const token = await readSession();
  if (!token) return { error: "not signed in", status: 401 };

  try {
    const res = await fetch(`${REGISTRY_URL}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
        ...(init?.body ? { "Content-Type": "application/json" } : {}),
        ...(init?.headers || {}),
      },
      cache: "no-store",
    });

    if (!res.ok) {
      // The registry answers with {"error": "..."} and those messages are
      // written for the person reading them, so they are passed through rather
      // than replaced with something vaguer.
      const body = (await res.json().catch(() => ({}))) as { error?: string };
      return { error: body.error || `request failed (${res.status})`, status: res.status };
    }
    if (res.status === 204) return { data: undefined as T };
    return { data: (await res.json()) as T };
  } catch (cause) {
    return { error: `the registry is unreachable: ${(cause as Error).message}`, status: 503 };
  }
}

export const getMe = () => call<Me>("/api/v1/dev/me");
export const listMyApps = () => call<App[]>("/api/v1/dev/apps");
export const listVersions = (slug: string) =>
  call<Version[]>(`/api/v1/dev/apps/${encodeURIComponent(slug)}/versions`);
export const reviewQueue = () => call<Version[]>("/api/v1/admin/review");

export const createPublisher = (body: { slug: string; name: string; contact_email: string }) =>
  call<Publisher>("/api/v1/dev/publishers", { method: "POST", body: JSON.stringify(body) });

export const upsertApp = (body: Record<string, unknown>) =>
  call<App>("/api/v1/dev/apps", { method: "POST", body: JSON.stringify(body) });

export const submitVersion = (slug: string, body: Record<string, unknown>) =>
  call<Version>(`/api/v1/dev/apps/${encodeURIComponent(slug)}/versions`, {
    method: "POST",
    body: JSON.stringify(body),
  });

export const decideVersion = (id: string, action: string, note: string) =>
  call<{ status: string }>(`/api/v1/admin/review/${encodeURIComponent(id)}`, {
    method: "POST",
    body: JSON.stringify({ action, note }),
  });
