/**
 * The storefront's view of the registry.
 *
 * Every call happens on the server. The registry is reachable on loopback from
 * the host this runs on, so the browser never talks to it directly: one origin
 * for the reader, no CORS, and no way for a page to leak an internal address.
 */

export const REGISTRY_URL =
  process.env.REGISTRY_URL?.replace(/\/$/, "") || "http://127.0.0.1:8083";

export interface StoreApp {
  id: string;
  slug: string;
  type: "module" | "external";
  name: string;
  description: string;
  icon_url: string;
  category: string;
  publisher: string;
  latest_version: string;
  updated_at: string;
}

export interface AppVersion {
  version: string;
  channel: string;
  min_platform: string;
  published_at?: string;
  manifest: {
    permissions?: Array<{ code: string; name: string; description: string }>;
    dependencies?: Array<{ id: string; version_constraint: string }>;
    menus?: Array<{ id: string; label: string }>;
    external?: { launch_url: string; embed?: string; scopes?: string[] };
  };
}

export interface AppDetail extends StoreApp {
  versions: AppVersion[];
  manifest?: AppVersion["manifest"];
  external?: { launch_url: string; sso_client_id: string; scopes: string[]; embed: string };
}

/**
 * Two different failures that must not look the same.
 *
 * "There is no such app" is a 404 the storefront should answer with — it is a
 * public, indexed site, and a soft 404 for every typo teaches a search engine
 * that missing pages are fine. "The registry is unavailable" is not the
 * reader's fault and must not be reported as if the app never existed.
 */
export type ReadResult<T> = { data: T } | { error: "missing" | "unavailable" };

/**
 * Revalidated rather than cached for ever: the catalogue changes when somebody
 * publishes, and a minute-old shop window is not a problem worth an
 * invalidation protocol.
 */
async function read<T>(path: string): Promise<ReadResult<T>> {
  try {
    const res = await fetch(`${REGISTRY_URL}${path}`, {
      headers: { Accept: "application/json" },
      next: { revalidate: 60 },
    });
    if (res.status === 404) return { error: "missing" };
    if (!res.ok) return { error: "unavailable" };
    return { data: (await res.json()) as T };
  } catch {
    return { error: "unavailable" };
  }
}

export const listApps = (locale: string) =>
  read<StoreApp[]>(`/api/v1/registry/apps?locale=${encodeURIComponent(locale)}`);

export const getApp = (slug: string, locale: string) =>
  read<AppDetail>(
    `/api/v1/registry/apps/${encodeURIComponent(slug)}?locale=${encodeURIComponent(locale)}`,
  );
