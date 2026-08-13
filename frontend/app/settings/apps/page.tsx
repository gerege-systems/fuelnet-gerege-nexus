"use client";

import React, { useEffect, useState } from "react";
import { api, type StoreOverviewApp } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Settings, Clock, ArrowUpCircle, ShieldAlert, Pin, GitCompareArrows } from "lucide-react";

interface InstalledApp {
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
  held_for?: string[];
  held_reason?: string;
  core: boolean;
}

interface CatalogStatus {
  source: "file" | "registry";
  apps: number;
  sync_interval: string;
  last_sync_at?: string;
  last_sync_ok?: boolean;
  last_sync_error?: string;
}

export default function InstalledAppsSettingsPage() {
  const { t, locale } = useI18n();
  const [apps, setApps] = useState<InstalledApp[]>([]);
  const [status, setStatus] = useState<CatalogStatus | null>(null);
  // Rows where the compiled module and the catalogue disagree. Unlike an update
  // waiting or an app held back, this is nobody's decision — it means this
  // instance is serving a catalogue that does not describe the code it runs,
  // which from every other screen looks exactly like a healthy one.
  const [drifted, setDrifted] = useState<StoreOverviewApp[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const loadInstalled = async () => {
    try {
      const data = await api.getInstalledApps();
      setApps(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
    // Administrator-only, and this screen is reachable by anyone who types the
    // address, so a refusal here is expected rather than a fault.
    try {
      setStatus(await api.getCatalogStatus());
      // Administrator-only, and this page already is. A failure here must not
      // cost the page its list of installed apps, so it is asked for on its own.
      try {
        const overview = await api.getStoreOverview();
        setDrifted((overview.apps || []).filter((row) => row.drifted));
      } catch {
        setDrifted([]);
      }
    } catch {
      setStatus(null);
    }
  };

  useEffect(() => {
    setLoading(true);
    loadInstalled();
  }, [locale]);

  // An app whose new version asks for more is held rather than applied, so
  // approving it is the same action as updating it: the server moves the pin
  // forward with the version.
  const handleUpdate = async (app: InstalledApp) => {
    setActionLoading(app.slug);
    try {
      await api.upgradeApp(app.slug);
      await loadInstalled();
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleAutoUpdate = async (app: InstalledApp) => {
    setActionLoading(app.slug);
    try {
      await api.setAutoUpdate(app.slug, !app.auto_update);
      await loadInstalled();
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleToggle = async (app: InstalledApp) => {
    setActionLoading(app.slug);
    try {
      if (app.enabled) {
        await api.disableApp(app.slug);
      } else {
        await api.enableApp(app.slug);
      }
      await loadInstalled();
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border-b border-slate-200 pb-4">
        <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
          <Settings className="w-6 h-6 text-slate-600" />
          <span>{t("app_store.view.installed_title")}</span>
        </h1>
        <p className="text-sm text-slate-500">
          {t("app_store.view.installed_subtitle")}
        </p>
      </div>

      {status && (
        <div className="bg-white border border-slate-200 rounded-xl px-4 py-3 text-sm flex flex-wrap items-center gap-x-4 gap-y-1">
          <span className="font-semibold text-slate-700">
            {status.source === "registry"
              ? t("app_store.state.source_registry")
              : t("app_store.state.source_file")}
          </span>
          <span className="text-slate-500">{t("app_store.field.app_count", { count: status.apps })}</span>
          {status.last_sync_at && (
            <span className="text-slate-500">
              {t("app_store.field.last_sync")}: {new Date(status.last_sync_at).toLocaleString()}
            </span>
          )}
          {/* A failing registry is the thing nobody notices: the store keeps
              serving the catalogue it already has, so nothing looks wrong. */}
          {status.last_sync_error && (
            <span className="text-red-600">
              {t("app_store.message.sync_failed")}: {status.last_sync_error}
            </span>
          )}
        </div>
      )}

      {drifted.length > 0 && (
        <div className="bg-rose-50 border border-rose-200 rounded-xl px-4 py-3 text-sm">
          <p className="font-semibold text-rose-800 flex items-center gap-2">
            <GitCompareArrows className="w-4 h-4" />
            {t("app_store.overview.drifted")}
          </p>
          <p className="text-rose-700 mt-0.5">{t("app_store.overview.drift_note")}</p>
          <ul className="mt-2 space-y-0.5">
            {drifted.map((row) => (
              <li key={row.app_id} className="text-rose-800 font-mono text-xs">
                {row.name}: {t("app_store.overview.binary")} v{row.binary_version} ≠{" "}
                {t("app_store.overview.catalog")} v{row.catalog_version}
              </li>
            ))}
          </ul>
        </div>
      )}

      {loading ? (
        <div className="py-8 text-slate-500 text-sm">{t("app_store.message.loading_installed")}</div>
      ) : apps.length === 0 ? (
        <div className="bg-white border border-slate-200 rounded-xl p-8 text-center text-slate-500 text-sm">
          {t("app_store.message.none_installed")}{" "}
          <a href="/apps" className="text-indigo-600 font-semibold underline">
            {t("app_store.action.browse_store")}
          </a>
        </div>
      ) : (
        <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-sm">
          <table className="w-full text-left border-collapse text-sm">
            <thead>
              <tr className="bg-slate-50 border-b border-slate-200 text-slate-600 font-semibold text-xs uppercase tracking-wider">
                <th className="py-3 px-4">{t("app_store.field.application_name")}</th>
                <th className="py-3 px-4">{t("app_store.field.module_id")}</th>
                <th className="py-3 px-4">{t("app_store.field.installed_version")}</th>
                <th className="py-3 px-4">{t("app_store.field.updates")}</th>
                <th className="py-3 px-4">{t("base.field.status")}</th>
                <th className="py-3 px-4">{t("app_store.field.installed_date")}</th>
                <th className="py-3 px-4 text-right">{t("base.field.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {apps.map((app) => (
                <tr key={app.id} className="hover:bg-slate-50">
                  <td className="py-3.5 px-4 font-bold text-slate-900">{app.name}</td>
                  <td className="py-3.5 px-4 font-mono text-xs text-slate-500">{app.app_id}</td>
                  <td className="py-3.5 px-4 font-semibold text-slate-700">
                    v{app.installed_version}
                    {app.update_available && app.latest_version && (
                      <span className="ml-1.5 text-xs font-normal text-indigo-600">→ v{app.latest_version}</span>
                    )}
                  </td>
                  <td className="py-3.5 px-4">
                    <div className="flex flex-col gap-1.5 items-start">
                      {/* The switch, said in words rather than as a bare toggle:
                          this decides whether somebody else's release reaches
                          this organisation without anybody looking at it. */}
                      <button
                        onClick={() => handleAutoUpdate(app)}
                        disabled={actionLoading === app.slug}
                        className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold border transition ${
                          app.auto_update
                            ? "border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                            : "border-slate-200 bg-slate-50 text-slate-600 hover:bg-slate-100"
                        }`}
                      >
                        <span className={`w-1.5 h-1.5 rounded-full ${app.auto_update ? "bg-emerald-500" : "bg-slate-400"}`} />
                        <span>{app.auto_update ? t("app_store.state.auto_update_on") : t("app_store.state.auto_update_off")}</span>
                      </button>
                      {app.pinned_version && (
                        <span className="inline-flex items-center gap-1 text-xs text-amber-700">
                          <Pin className="w-3 h-3" />
                          {t("app_store.state.pinned", { version: app.pinned_version })}
                        </span>
                      )}
                      {(app.held_reason || (app.held_for && app.held_for.length > 0)) && (
                        <span className="inline-flex items-start gap-1 text-xs text-amber-700 max-w-56">
                          <ShieldAlert className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                          <span>
                            {app.held_for && app.held_for.length > 0 ? (
                              <>
                                {t("app_store.message.held_for_approval")}{" "}
                                <span className="font-mono">{app.held_for.join(", ")}</span>
                              </>
                            ) : (
                              /* Held with nothing to itemise — the installed
                                 version's manifest predates the history, so what
                                 the new one adds cannot be established. Saying so
                                 is better than an app that silently stops moving. */
                              app.held_reason
                            )}
                          </span>
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="py-3.5 px-4">
                    <span
                      className={`inline-flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-semibold ${
                        app.enabled
                          ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                          : "bg-slate-100 text-slate-600 border border-slate-200"
                      }`}
                    >
                      <span className={`w-1.5 h-1.5 rounded-full ${app.enabled ? "bg-emerald-500" : "bg-slate-400"}`}></span>
                      <span>{app.enabled ? t("base.state.active") : t("app_store.state.disabled")}</span>
                    </span>
                  </td>
                  <td className="py-3.5 px-4 text-xs text-slate-500 flex items-center space-x-1 pt-4">
                    <Clock className="w-3.5 h-3.5" />
                    <span>{new Date(app.installed_at).toLocaleDateString()}</span>
                  </td>
                  <td className="py-3.5 px-4 text-right">
                    {app.update_available && (
                      <button
                        onClick={() => handleUpdate(app)}
                        disabled={actionLoading === app.slug}
                        className="mr-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition border border-indigo-600 bg-indigo-600 text-white hover:bg-indigo-700 inline-flex items-center gap-1.5"
                      >
                        <ArrowUpCircle className="w-3.5 h-3.5" />
                        {/* Approving a held version and updating an ordinary
                            one are the same request; only the wording differs,
                            because only one of them is a decision. */}
                        {app.held_for && app.held_for.length > 0
                          ? t("app_store.action.approve_update")
                          : t("app_store.action.update")}
                      </button>
                    )}
                    {/* Every app can be turned off, including the ones a new
                        organisation starts with: the platform underneath —
                        sign-in, the organisation's own profile, settings —
                        is not an app and does not appear in this list. */}
                    <button
                        onClick={() => handleToggle(app)}
                        disabled={actionLoading === app.slug}
                        className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition border ${
                          app.enabled
                            ? "border-red-200 text-red-600 hover:bg-red-50"
                            : "border-emerald-600 bg-emerald-600 text-white hover:bg-emerald-700"
                        }`}
                      >
                        {app.enabled ? t("app_store.action.disable") : t("app_store.action.enable")}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
