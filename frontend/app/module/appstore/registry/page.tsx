"use client";

/**
 * The registry's own state.
 *
 * Small on purpose. Everything a person does with the store happens on the
 * other two screens; this is for the two questions that are about the registry
 * itself — which key it signs with, and whether the catalogue it is serving was
 * built under the current one.
 */

import { useCallback, useEffect, useState } from "react";
import { Boxes, KeyRound, RefreshCw } from "lucide-react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";

export default function AppStoreRegistryPage() {
  const { t } = useI18n();
  const [state, setState] = useState<{ revision: number; key_id: string; public_key: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setState(await api.getRegistryState());
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function rebuild() {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const { discarded } = await api.rebuildCatalogue();
      setNotice(t("mod.appstore_registry.rebuilt", { count: discarded }));
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Screen
      icon={<Boxes className="w-5 h-5" />}
      title={t("mod.appstore_registry.title")}
      subtitle={t("mod.appstore_registry.subtitle")}
      action={
        <button
          onClick={() => void rebuild()}
          disabled={busy || !state}
          className="bg-white hover:bg-slate-50 text-slate-700 border border-slate-200 px-4 py-2 rounded-lg text-sm font-semibold flex items-center gap-2 disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${busy ? "animate-spin" : ""}`} />
          {t("mod.appstore_registry.rebuild")}
        </button>
      }
    >
      {error && <ErrorNote>{error}</ErrorNote>}
      {notice && (
        <p className="text-sm text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2">
          {notice}
        </p>
      )}

      {loading ? (
        <Loading label={t("base.message.loading")} />
      ) : !state ? null : (
        <div className="grid gap-3 md:grid-cols-2">
          <Panel className="p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
              {t("mod.appstore_registry.revision")}
            </p>
            <p className="text-2xl font-bold text-slate-900 mt-1">{state.revision}</p>
            {/* The number that says the catalogue changed. A snapshot built
                under an older one is rebuilt rather than served. */}
            <p className="text-xs text-slate-500 mt-1">{t("mod.appstore_registry.revision_note")}</p>
          </Panel>

          <Panel className="p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide flex items-center gap-1.5">
              <KeyRound className="w-3.5 h-3.5" />
              {t("mod.appstore_registry.key")}
            </p>
            <p className="font-mono text-sm font-bold text-slate-900 mt-1">{state.key_id}</p>
            <code className="text-[11px] text-slate-500 break-all block mt-1">{state.public_key}</code>
            {/* Rotating a key moves no revision, so every cached document stays
                looking valid while signed by one the instances have stopped
                accepting. Rebuilding is the way out, which is why it is here. */}
            <p className="text-xs text-slate-500 mt-2">{t("mod.appstore_registry.key_note")}</p>
          </Panel>
        </div>
      )}
    </Screen>
  );
}
