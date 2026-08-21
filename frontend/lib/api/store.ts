/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The app store screens. Core: this is the catalogue a deployment installs from, and every deployment has one.

import { request, APP_MENU_CHANGED_EVENT } from "./client";

/** What kind of change a release was. The store colours by it. */
export type ReleaseKind = "feature" | "fix" | "security" | "breaking" | "docs";

/**
 * One line of an app's timeline.
 *
 * `type` is "release" for the publisher's own record — the chronicle entry,
 * already reduced to one language by the server — and the installation event
 * type ("installed", "upgraded", "held", …) for everything this organisation
 * did. `system` marks the lines the auto-update sweep is responsible for
 * rather than a person.
 */
export interface AppHistoryEntry {
  at: string;
  type: string;
  version?: string;
  from?: string;
  kind?: ReleaseKind;
  summary?: string;
  details?: string;
  authors?: string[];
  refs?: string[];
  actor_id?: string;
  actor_name?: string;
  system?: boolean;
  reason?: string;
  added?: string;
}

/** One row of the administrator's store overview. */
export interface StoreOverviewApp {
  app_id: string;
  slug: string;
  name: string;
  binary_version?: string;
  catalog_version: string;
  installed_version?: string;
  installed: boolean;
  enabled: boolean;
  update_available: boolean;
  auto_update: boolean;
  held?: boolean;
  pinned_version?: string;
  /** The compiled module and the catalogue disagree. Always a fault. */
  drifted?: boolean;
  release_kind?: ReleaseKind;
  release_summary?: string;
}

/** An app as the registry holds it — what is *offered*, not what is installed. */
export interface StoreApp {
  id: string;
  slug: string;
  type: "module" | "external";
  name: string;
  description: string;
  category: string;
  visibility: string;
  publisher_name?: string;
  latest_version?: string;
  license?: string;
  repository?: string;
}

/** One submitted or published release. */
export interface StoreVersion {
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
  manifest?: { release_notes?: ManifestReleaseNotes };
}

/** A manifest's release notes, as they arrive inside a catalogue entry. */
export interface ManifestReleaseNotes {
  kind?: ReleaseKind;
  summary?: Record<string, string>;
  details?: Record<string, string>;
  authors?: string[];
  refs?: string[];
}

// A store action that changes what the tenant has installed.
//
// Layout lives above the App Store pages, so a route refresh does not recreate
// it. The mounted shell is told to refetch the tenant's menus immediately.
async function mutateApp(url: string) {
  const result = await request<{ status: string; app: string }>(url, { method: "POST" });
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(APP_MENU_CHANGED_EVENT, { detail: result }));
  }
  return result;
}

export const storeApi = {
  getStoreApps: () =>
    request<
      Array<{
        id: string;
        slug: string;
        name: string;
        description: string;
        icon_url: string;
        category: string;
        version: string;
        installed: boolean;
        enabled: boolean;
        installed_version?: string;
        latest_version: string;
        update_available: boolean;
        manifest: any;
      }>
    >("/store/apps"),

  getInstalledApps: () =>
    request<
      Array<{
        id: string;
        app_id: string;
        slug: string;
        name: string;
        installed_version: string;
        status: string;
        enabled: boolean;
        installed_at: string;
        auto_update: boolean;
        pinned_version?: string;
        latest_version?: string;
        update_available: boolean;
        // What a waiting version asks for that the installed one did not.
        // Non-empty means the update is being held for an administrator to
        // approve rather than offered as an ordinary one.
        held_for?: string[];
        held_reason?: string;
        // Part of the platform. Such an app has no Disable button, because
        // disabling it is refused server-side and a button that only ever
        // fails is worse than no button.
        core: boolean;
      }>
    >("/installed-apps"),

  // What changed in an app, and what this organisation did about it, merged
  // into one timeline newest first. A "release" line is the publisher's; every
  // other kind is this tenant's own installation history.
  getAppHistory: (slug: string) =>
    request<{
      app_id: string;
      slug: string;
      name: string;
      installed_version: string;
      latest_version: string;
      timeline: AppHistoryEntry[];
    }>(`/store/apps/${slug}/history`),

  // The administrator's single view of the store: which versions the binary,
  // the catalogue and this tenant each hold, and where they disagree.
  getStoreOverview: () =>
    request<{
      platform_version: string;
      sync: {
        source: "file" | "registry";
        sync_interval: string;
        last_sync_at?: string;
        last_sync_ok?: boolean;
        last_sync_error?: string;
      };
      apps: StoreOverviewApp[];
      summary: { catalog: number; installed: number; updates: number; held: number; drifted: number };
    }>("/admin/store/overview"),

  // --- App Store: the three module surfaces ---------------------------------
  //
  // Only reachable on the instance that *is* the store; every other deployment
  // has these apps uninstalled and the routes gated off.
  setAutoUpdate: (slug: string, enabled: boolean) =>
    request<{ app: string; auto_update: boolean }>(`/store/apps/${slug}/auto-update`, {
      method: "POST",
      body: JSON.stringify({ enabled }),
    }),

  installApp: (slug: string) => mutateApp(`/store/apps/${slug}/install`),

  // An upgrade changes which menus an app contributes just as an install does,
  // so it goes through the same notification rather than beside it.
  upgradeApp: (slug: string) => mutateApp(`/store/apps/${slug}/upgrade`),

  // Ask the registry for a catalog now rather than at the next scheduled sync.
  // Answers 501 on a deployment that reads its catalog from a file, which is
  // every self-hosted one — the button is hidden there rather than failing.
  // Where the catalogue comes from and how the last refresh went. The hourly
  // sync leaves only a log line, so this is the one place a registry that has
  // been failing for a week is distinguishable from one that has published
  // nothing.
  getCatalogStatus: () =>
    request<{
      source: "file" | "registry";
      apps: number;
      sync_interval: string;
      last_sync_at?: string;
      last_sync_ok?: boolean;
      last_sync_error?: string;
    }>("/admin/store/status"),

  syncStore: () =>
    request<{ status: "updated" | "unchanged"; apps: number }>("/admin/store/sync", { method: "POST" }),

  enableApp: (slug: string) => mutateApp(`/store/apps/${slug}/enable`),

  disableApp: (slug: string) => mutateApp(`/store/apps/${slug}/disable`),

  // Organisation & People — the platform's own core app. What the organisation
  // is, how it is arranged, and who works in it.
};
