"use client";

/**
 * Connections — the three rails, and what mode each is in.
 *
 * The mode is the point of the screen. A deployment running ХУР in mock mode
 * answers every lookup instantly with plausible fixtures, and from any other
 * screen that is indistinguishable from a working integration. Saying so here,
 * in the place somebody goes to ask "are we connected", is the only cheap way
 * to keep a fixture off a government form.
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import { Share2, TriangleAlert, User } from "lucide-react";
import { api, type EgovConnections, type EgovRail } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";

const toneFor = (mode: string) =>
  mode === "live" ? "emerald" : mode === "mock" ? "amber" : "rose";

// The three modes, spelled out rather than interpolated: the translation keys
// are a closed set and a template string would slip a typo past the compiler.
const MODE_LABELS = {
  live: "egov.mode.live",
  mock: "egov.mode.mock",
  unconfigured: "egov.mode.unconfigured",
} as const;

export default function EgovConnectionsPage() {
  const { t } = useI18n();
  const [data, setData] = useState<EgovConnections | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void (async () => {
      try {
        setData(await api.getEgovConnections());
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const rails: EgovRail[] = data?.rails || [];
  const anyMock = rails.some((rail) => rail.mode === "mock");

  return (
    <Screen
      icon={<Share2 className="w-6 h-6" />}
      title={t("egov.view.connections_title")}
      subtitle={t("egov.view.connections_subtitle")}
    >
      {error && <ErrorNote>{error}</ErrorNote>}
      {loading ? (
        <Loading label={t("egov.view.connections_title")} />
      ) : (
        <>
          {anyMock && (
            <p className="text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 flex items-start gap-2">
              <TriangleAlert className="w-4 h-4 shrink-0 mt-0.5" />
              {t("egov.message.mock_warning")}
            </p>
          )}

          <Panel>
            <table className="w-full text-sm">
              <thead className="text-left text-xs font-semibold text-slate-500 border-b border-slate-100">
                <tr>
                  <th className="px-4 py-3">{t("egov.field.rail")}</th>
                  <th className="px-4 py-3">{t("egov.field.mode")}</th>
                  <th className="px-4 py-3">{t("egov.field.endpoint")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {rails.map((rail) => (
                  <tr key={rail.id}>
                    <td className="px-4 py-3 font-semibold text-slate-900">{rail.name}</td>
                    <td className="px-4 py-3">
                      <Chip tone={toneFor(rail.mode)}>
                        {t(MODE_LABELS[rail.mode as keyof typeof MODE_LABELS] ?? "egov.mode.unconfigured")}
                      </Chip>
                    </td>
                    <td className="px-4 py-3 text-slate-500 break-all font-mono text-xs">
                      {rail.endpoint || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Panel>

          {/* The one thing this screen does not own. See the egov package
              comment: a person's linked identities are theirs, not their
              employer's, so they stay on the platform's profile screen and
              this points at them rather than duplicating them. */}
          <Panel className="p-4 flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-slate-600 flex items-start gap-2">
              <User className="w-4 h-4 shrink-0 mt-0.5 text-slate-400" />
              {t("egov.message.identities_hint")}
            </p>
            <Link
              href={data?.identities_path || "/profile"}
              className="px-3 py-1.5 text-xs font-semibold border border-slate-200 rounded-lg hover:bg-slate-50"
            >
              {t("egov.action.open_profile")}
            </Link>
          </Panel>
        </>
      )}
    </Screen>
  );
}
