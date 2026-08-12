"use client";

/**
 * The review queue.
 *
 * Oldest first, because a queue is answered in the order people joined it.
 * Newest-first would mean a submission from a publisher nobody is chasing sinks
 * quietly to the bottom.
 *
 * Two decisions, held apart on purpose. Publishing needs no explanation — the
 * version speaks for itself. Turning one down does: a refusal a publisher
 * cannot act on wastes both sides, so the server refuses a rejection with no
 * note and this screen will not send one.
 */

import { useCallback, useEffect, useState } from "react";
import { CheckCircle2, ListChecks, ShieldCheck, XCircle } from "lucide-react";
import { api, type Publisher, type StoreVersion } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, Empty, ErrorNote, Loading, Modal, Panel, Screen } from "@/components/module/kit";

type Decision = "publish" | "reject" | "yank";

export default function StoreReviewPage() {
  const { t, locale } = useI18n();
  const [queue, setQueue] = useState<StoreVersion[]>([]);
  const [publishers, setPublishers] = useState<Publisher[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [deciding, setDeciding] = useState<{ version: StoreVersion; action: Decision } | null>(null);
  const [note, setNote] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [pending, pubs] = await Promise.all([api.getReviewQueue(), api.getReviewPublishers()]);
      setQueue(pending || []);
      setPublishers(pubs || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function decide() {
    if (!deciding) return;
    setBusy(deciding.version.id);
    setError("");
    try {
      await api.decideVersion(deciding.version.id, deciding.action, note.trim());
      setDeciding(null);
      setNote("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    } finally {
      setBusy(null);
    }
  }

  async function toggleVerified(publisher: Publisher) {
    setBusy(publisher.id);
    try {
      await api.verifyPublisher(publisher.id, !publisher.verified);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    } finally {
      setBusy(null);
    }
  }

  const summary = (version: StoreVersion) => {
    const notes = version.manifest?.release_notes?.summary;
    return notes ? notes[locale] || notes.mn || notes.en : "";
  };

  return (
    <Screen
      icon={<ListChecks className="w-5 h-5" />}
      title={t("mod.store_review.title")}
      subtitle={t("mod.store_review.subtitle")}
    >
      {error && <ErrorNote>{error}</ErrorNote>}

      {loading ? (
        <Loading label={t("base.message.loading")} />
      ) : (
        <>
          {queue.length === 0 ? (
            <Empty icon={<ListChecks className="w-9 h-9 mx-auto" />}>
              {t("mod.store_review.empty")}
            </Empty>
          ) : (
            <div className="space-y-3">
              {queue.map((version) => (
                <Panel key={version.id} className="p-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex items-baseline gap-2 flex-wrap">
                        <code className="font-mono text-sm font-bold text-slate-900">{version.app_id}</code>
                        <Chip tone="blue">v{version.version}</Chip>
                        <Chip tone="slate">{version.channel}</Chip>
                      </div>
                      {summary(version) && (
                        <p className="text-sm text-slate-700 mt-1">{summary(version)}</p>
                      )}
                      <p className="text-xs text-slate-500 mt-1">
                        {t("mod.store_review.submitted_by")}: {version.submitted_by || "—"} ·{" "}
                        {new Date(version.created_at).toLocaleDateString()}
                      </p>
                      <p className="text-xs text-slate-400 mt-0.5">
                        {t("mod.store_review.requires_platform")} {version.min_platform}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <button
                        disabled={busy === version.id}
                        onClick={() => { setDeciding({ version, action: "publish" }); setNote(""); }}
                        className="px-3 py-1.5 text-sm font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white flex items-center gap-1.5 disabled:opacity-50"
                      >
                        <CheckCircle2 className="w-4 h-4" /> {t("mod.store_review.publish")}
                      </button>
                      <button
                        disabled={busy === version.id}
                        onClick={() => { setDeciding({ version, action: "reject" }); setNote(""); }}
                        className="px-3 py-1.5 text-sm font-semibold rounded-lg bg-white hover:bg-rose-50 text-rose-600 border border-rose-200 flex items-center gap-1.5 disabled:opacity-50"
                      >
                        <XCircle className="w-4 h-4" /> {t("mod.store_review.reject")}
                      </button>
                    </div>
                  </div>
                </Panel>
              ))}
            </div>
          )}

          <h2 className="text-sm font-bold text-slate-900 mt-8 mb-3 flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-indigo-600" />
            {t("mod.store_review.publishers")}
          </h2>
          {publishers.length === 0 ? (
            <Empty icon={<ShieldCheck className="w-9 h-9 mx-auto" />}>{t("mod.store_review.no_publishers")}</Empty>
          ) : (
            <div className="grid gap-2 md:grid-cols-2">
              {publishers.map((publisher) => (
                <Panel key={publisher.id} className="p-3 flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="font-semibold text-slate-900 truncate">{publisher.name}</p>
                    <code className="text-xs text-slate-500">{publisher.slug}</code>
                  </div>
                  <button
                    disabled={busy === publisher.id}
                    onClick={() => void toggleVerified(publisher)}
                    className={`text-xs font-semibold px-2.5 py-1 rounded-lg border disabled:opacity-50 ${
                      publisher.verified
                        ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                        : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50"
                    }`}
                  >
                    {publisher.verified
                      ? t("mod.store_review.verified")
                      : t("mod.store_review.verify")}
                  </button>
                </Panel>
              ))}
            </div>
          )}
        </>
      )}

      {deciding && (
        <Modal onClose={() => setDeciding(null)}>
          <div className="space-y-4">
            <h2 className="text-lg font-bold text-slate-900">
              {deciding.action === "publish"
                ? t("mod.store_review.publish")
                : t("mod.store_review.reject")}{" "}
              <code className="font-mono text-sm">{deciding.version.app_id}</code> v
              {deciding.version.version}
            </h2>
            <label className="block">
              <span className="text-xs font-semibold text-slate-700">
                {t("mod.store_review.note")}
                {deciding.action !== "publish" ? " *" : ""}
              </span>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={3}
                className="mt-1 w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none"
              />
              {/* Said here rather than discovered from a 400: the server
                  refuses a rejection with no reason, and it is right to. */}
              {deciding.action !== "publish" && (
                <span className="text-xs text-slate-500 mt-1 block">
                  {t("mod.store_review.note_required")}
                </span>
              )}
            </label>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setDeciding(null)}
                className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
              >
                {t("base.action.cancel")}
              </button>
              <button
                onClick={() => void decide()}
                disabled={deciding.action !== "publish" && note.trim() === ""}
                className={`px-4 py-2 text-sm rounded-lg font-semibold text-white disabled:opacity-40 ${
                  deciding.action === "publish"
                    ? "bg-emerald-600 hover:bg-emerald-700"
                    : "bg-rose-600 hover:bg-rose-700"
                }`}
              >
                {t("base.action.save")}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Screen>
  );
}
