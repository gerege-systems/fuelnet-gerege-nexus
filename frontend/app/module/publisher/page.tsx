"use client";

/**
 * Publisher Studio.
 *
 * What used to be a separate product at developer.gerege.mn — its own Next.js
 * application, its own OAuth2 client, its own token verifier and its own idea
 * of who a publisher is. Here the caller is a signed-in member of a tenant and
 * the tenant is the publisher, so all four are gone and this is a screen.
 *
 * It submits and stops. Publishing is the review queue's decision, held by
 * somebody else on purpose.
 */

import { useCallback, useEffect, useState } from "react";
import { Boxes, Send, Store, Upload } from "lucide-react";
import { api, type Publisher, type StoreApp, type StoreVersion } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Chip, Empty, ErrorNote, Loading, Modal, Panel, Screen } from "@/components/module/kit";

const statusTone: Record<string, "blue" | "slate" | "emerald" | "amber" | "rose"> = {
  published: "emerald",
  in_review: "amber",
  rejected: "rose",
  yanked: "rose",
  draft: "slate",
};

export default function PublisherStudioPage() {
  const { t } = useI18n();
  const [publisher, setPublisher] = useState<Publisher | null>(null);
  const [apps, setApps] = useState<StoreApp[]>([]);
  const [versions, setVersions] = useState<Record<string, StoreVersion[]>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [editingProfile, setEditingProfile] = useState(false);
  const [profileDraft, setProfileDraft] = useState({ slug: "", name: "", contact_email: "" });
  const [submitting, setSubmitting] = useState<StoreApp | null>(null);
  const [manifestDraft, setManifestDraft] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      // A tenant that has never published has no profile, and the server says so
      // with a 404. That is a state to render, not a failure to report.
      let profile: Publisher | null = null;
      try {
        profile = await api.getPublisherProfile();
      } catch {
        profile = null;
      }
      setPublisher(profile);
      if (!profile) {
        setApps([]);
        return;
      }
      const own = await api.getPublisherApps();
      setApps(own || []);
      const byApp: Record<string, StoreVersion[]> = {};
      await Promise.all(
        (own || []).map(async (app) => {
          try {
            byApp[app.slug] = (await api.getPublisherVersions(app.slug)) || [];
          } catch {
            byApp[app.slug] = [];
          }
        }),
      );
      setVersions(byApp);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveProfile(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api.savePublisherProfile(profileDraft);
      setEditingProfile(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    }
  }

  async function submit() {
    if (!submitting) return;
    setError("");
    setNotice("");
    let manifest: unknown;
    try {
      manifest = JSON.parse(manifestDraft);
    } catch {
      setError(t("mod.publisher.manifest_invalid"));
      return;
    }
    try {
      const saved = await api.submitVersion(submitting.slug, manifest);
      setSubmitting(null);
      setManifestDraft("");
      setNotice(t("mod.publisher.submitted", { version: saved.version }));
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("base.message.error"));
    }
  }

  const openProfile = () => {
    setProfileDraft({
      slug: publisher?.slug || "",
      name: publisher?.name || "",
      contact_email: publisher?.contact_email || "",
    });
    setEditingProfile(true);
  };

  return (
    <Screen
      icon={<Upload className="w-5 h-5" />}
      title={t("mod.publisher.title")}
      subtitle={t("mod.publisher.subtitle")}
      action={
        <button
          onClick={openProfile}
          className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-semibold flex items-center gap-2 shadow-sm"
        >
          <Store className="w-4 h-4" />
          {publisher ? t("mod.publisher.edit_profile") : t("mod.publisher.create_profile")}
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
      ) : !publisher ? (
        <Empty icon={<Store className="w-9 h-9 mx-auto" />}>{t("mod.publisher.no_profile")}</Empty>
      ) : (
        <>
          <Panel className="p-4 mb-4">
            <div className="flex items-center justify-between gap-3 flex-wrap">
              <div className="min-w-0">
                <h3 className="font-bold text-slate-900">{publisher.name}</h3>
                <code className="text-xs text-slate-500">{publisher.slug}</code>
              </div>
              <Chip tone={publisher.verified ? "emerald" : "slate"}>
                {publisher.verified
                  ? t("mod.publisher.verified")
                  : t("mod.publisher.unverified")}
              </Chip>
            </div>
          </Panel>

          {apps.length === 0 ? (
            <Empty icon={<Boxes className="w-9 h-9 mx-auto" />}>{t("mod.publisher.no_apps")}</Empty>
          ) : (
            <div className="space-y-3">
              {apps.map((app) => (
                <Panel key={app.id} className="p-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <h3 className="font-bold text-slate-900">{app.name}</h3>
                      <code className="text-xs font-mono text-slate-500">{app.id}</code>
                      <p className="text-sm text-slate-600 mt-1">{app.description}</p>
                    </div>
                    <button
                      onClick={() => { setSubmitting(app); setManifestDraft(""); }}
                      className="px-3 py-1.5 text-sm font-semibold rounded-lg bg-white hover:bg-slate-50 text-slate-700 border border-slate-200 flex items-center gap-1.5 shrink-0"
                    >
                      <Send className="w-4 h-4" /> {t("mod.publisher.submit_version")}
                    </button>
                  </div>

                  <div className="flex flex-wrap gap-2 mt-3">
                    {(versions[app.slug] || []).length === 0 ? (
                      <span className="text-xs text-slate-400">{t("mod.publisher.no_versions")}</span>
                    ) : (
                      (versions[app.slug] || []).map((v) => (
                        <Chip key={v.id} tone={statusTone[v.status] || "slate"}>
                          v{v.version} · {t(`mod.publisher.status.${v.status}` as never)}
                        </Chip>
                      ))
                    )}
                  </div>
                </Panel>
              ))}
            </div>
          )}
        </>
      )}

      {editingProfile && (
        <Modal onClose={() => setEditingProfile(false)}>
          <form onSubmit={saveProfile} className="space-y-4">
            <h2 className="text-lg font-bold text-slate-900">{t("mod.publisher.profile")}</h2>
            {([
              ["slug", t("mod.publisher.handle")],
              ["name", t("base.field.name")],
              ["contact_email", t("mod.publisher.contact")],
            ] as const).map(([field, label]) => (
              <label key={field} className="block">
                <span className="text-xs font-semibold text-slate-700">
                  {label}
                  {field !== "contact_email" ? " *" : ""}
                </span>
                <input
                  value={profileDraft[field]}
                  onChange={(e) => setProfileDraft({ ...profileDraft, [field]: e.target.value })}
                  required={field !== "contact_email"}
                  className="mt-1 w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none"
                />
              </label>
            ))}
            {/* The handle appears in storefront URLs, so it is not something to
                change casually once anybody has linked to it. */}
            <p className="text-xs text-slate-500">{t("mod.publisher.handle_note")}</p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setEditingProfile(false)}
                className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
              >
                {t("base.action.cancel")}
              </button>
              <button
                type="submit"
                className="px-4 py-2 text-sm bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-semibold"
              >
                {t("base.action.save")}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {submitting && (
        <Modal onClose={() => setSubmitting(null)}>
          <div className="space-y-3">
            <h2 className="text-lg font-bold text-slate-900">
              {t("mod.publisher.submit_version")} — {submitting.name}
            </h2>
            <p className="text-xs text-slate-500">{t("mod.publisher.submit_note")}</p>
            <textarea
              value={manifestDraft}
              onChange={(e) => setManifestDraft(e.target.value)}
              rows={14}
              spellCheck={false}
              placeholder={`{\n  "id": "${submitting.id}",\n  "version": "1.1.0",\n  "platform": ">=1.0.0",\n  "release_notes": {\n    "kind": "feature",\n    "summary": { "mn": "…", "en": "…" }\n  }\n}`}
              className="w-full px-3 py-2 text-xs font-mono border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none"
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setSubmitting(null)}
                className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
              >
                {t("base.action.cancel")}
              </button>
              <button
                onClick={() => void submit()}
                disabled={manifestDraft.trim() === ""}
                className="px-4 py-2 text-sm bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-semibold disabled:opacity-40"
              >
                {t("mod.publisher.submit_version")}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </Screen>
  );
}
