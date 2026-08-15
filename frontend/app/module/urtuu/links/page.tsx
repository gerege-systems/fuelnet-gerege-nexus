"use client";

/**
 * Холбоосууд — the app's read-only view of the channel underneath it.
 *
 * Establishing and closing a link is platform configuration and lives in
 * Settings → Өртөө; what belongs here is the answer to "is anything moving",
 * which is the question somebody looking at a stalled queue actually has.
 */

import React, { useEffect, useState } from "react";
import { Link2 } from "lucide-react";

import { api, type UrtuuPeer } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, Empty, ErrorNote, Loading, Panel, Screen } from "@/components/module/kit";

export default function UrtuuLinksPage() {
  const { t } = useI18n();
  const [peers, setPeers] = useState<UrtuuPeer[]>([]);
  const [loading, setLoading] = useState(true);
  const [failure, setFailure] = useState("");

  useEffect(() => {
    api
      .getUrtuuPeers()
      .then((answer) => setPeers(answer.peers || []))
      .catch((err) => setFailure(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false));
  }, []);

  return (
    <Screen
      icon={<Link2 className="w-5 h-5" />}
      title={t("urtuu.links.title")}
      subtitle={t("urtuu.links.subtitle")}
    >
      {failure && <ErrorNote>{failure}</ErrorNote>}
      {loading ? (
        <Loading label={t("base.message.loading")} />
      ) : peers.length === 0 ? (
        <Empty icon={<Link2 className="w-8 h-8" />}>{t("urtuu.message.no_links")}</Empty>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {peers.map((peer) => (
            <Panel key={peer.id} className="p-4 space-y-2">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-semibold text-slate-800">{peer.name || peer.id.slice(0, 8)}</p>
                  <p className="text-xs text-slate-500">
                    {peer.role === "parent" ? t("urtuu.role.parent") : t("urtuu.role.child")}
                  </p>
                </div>
                <Chip
                  tone={
                    peer.status === "active" ? "emerald" : peer.status === "revoked" ? "slate" : "amber"
                  }
                >
                  {peer.status === "active"
                    ? t("urtuu.status.active")
                    : peer.status === "revoked"
                      ? t("urtuu.status.revoked")
                      : t("urtuu.status.pending")}
                </Chip>
              </div>
              <dl className="text-xs text-slate-600 space-y-1">
                <div className="flex justify-between gap-2">
                  <dt>{t("urtuu.field.last_seen")}</dt>
                  <dd>
                    {peer.last_seen_at
                      ? new Date(peer.last_seen_at).toLocaleString()
                      : t("urtuu.message.never")}
                  </dd>
                </div>
                {peer.undelivered > 0 && (
                  <div className="flex justify-between gap-2 text-amber-700">
                    <dt>{t("urtuu.message.undelivered", { count: peer.undelivered })}</dt>
                    <dd />
                  </div>
                )}
                {peer.clock_skew_seconds !== 0 && (
                  <div className="flex justify-between gap-2">
                    <dt>{t("urtuu.message.clock_skew", { seconds: peer.clock_skew_seconds })}</dt>
                    <dd />
                  </div>
                )}
              </dl>
              {peer.last_error && <ErrorNote>{peer.last_error}</ErrorNote>}
            </Panel>
          ))}
        </div>
      )}
    </Screen>
  );
}
