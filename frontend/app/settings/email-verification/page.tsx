"use client";

import React, { useCallback, useEffect, useState } from "react";
import {
  EmailVerification,
  EmailVerifyClient,
  EmailVerifyOverview,
  api,
} from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  AlertTriangle, CheckCircle2, Clock, Copy, KeyRound, MailCheck, Plus,
  Power, RefreshCw, Send, ShieldCheck, Trash2, XCircle,
} from "lucide-react";

/**
 * Email verification lives in Settings, not in an app.
 *
 * Proving an address is not one module's business: Contacts, Documents and Gov
 * Services all want it, and so does a platform running next to this one. The
 * screen therefore administers the shared service — who holds a key, what has
 * been sent — rather than being a feature of whatever app happened to ask first.
 *
 * A key appears exactly once, in the response that creates it. That is why the
 * new-key panel is a separate, deliberately loud block instead of a column in
 * the table: there is no second chance to read it.
 */

const emptyForm = { name: "", hourly_limit: 60, hosts: "" };

export default function EmailVerificationPage() {
  const { t } = useI18n();
  const [overview, setOverview] = useState<EmailVerifyOverview | null>(null);
  const [clients, setClients] = useState<EmailVerifyClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [banner, setBanner] = useState<{ kind: "ok" | "error"; text: string } | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [issued, setIssued] = useState<EmailVerifyClient | null>(null);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [testEmail, setTestEmail] = useState("");
  const [testRedirect, setTestRedirect] = useState("");

  const report = useCallback((err: unknown, fallback: string) => {
    const message = err instanceof Error && err.message ? err.message : fallback;
    setBanner({ kind: "error", text: message });
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [summary, list] = await Promise.all([
        api.getEmailVerifyOverview(),
        api.getEmailVerifyClients(),
      ]);
      setOverview(summary);
      setClients(list || []);
    } catch (err) {
      report(err, t("emailverify.message.load_failed"));
    } finally {
      setLoading(false);
    }
  }, [report, t]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setBanner(null);
    try {
      const client = await api.createEmailVerifyClient({
        name: form.name,
        hourly_limit: Number(form.hourly_limit) || undefined,
        allowed_redirect_hosts: form.hosts
          .split(",")
          .map((entry) => entry.trim())
          .filter(Boolean),
      });
      setShowModal(false);
      setForm(emptyForm);
      setCopied(false);
      // Held in state rather than shown in a toast: the administrator has to be
      // able to read it at their own pace, and a toast that closes takes the
      // only copy of the key with it.
      setIssued(client);
      await loadData();
    } catch (err) {
      report(err, t("emailverify.message.create_failed"));
    }
  }

  async function handleToggle(client: EmailVerifyClient) {
    setBusy(client.id);
    try {
      await api.updateEmailVerifyClient(client.id, {
        name: client.name,
        status: client.status === "ACTIVE" ? "DISABLED" : "ACTIVE",
        hourly_limit: client.hourly_limit,
        allowed_redirect_hosts: client.allowed_redirect_hosts,
      });
      await loadData();
    } catch (err) {
      report(err, t("emailverify.message.update_failed"));
    } finally {
      setBusy(null);
    }
  }

  async function handleDelete(client: EmailVerifyClient) {
    if (!window.confirm(t("emailverify.message.confirm_delete", { name: client.name }))) return;
    setBusy(client.id);
    try {
      await api.deleteEmailVerifyClient(client.id);
      if (issued?.id === client.id) setIssued(null);
      await loadData();
    } catch (err) {
      report(err, t("emailverify.message.delete_failed"));
    } finally {
      setBusy(null);
    }
  }

  async function handleTestSend(e: React.FormEvent) {
    e.preventDefault();
    setBanner(null);
    setBusy("test");
    try {
      await api.sendEmailVerification({
        email: testEmail,
        redirect_url: testRedirect || undefined,
        purpose: "portal_test",
      });
      setBanner({ kind: "ok", text: t("emailverify.message.test_sent", { email: testEmail }) });
      setTestEmail("");
      await loadData();
    } catch (err) {
      report(err, t("emailverify.message.send_failed"));
    } finally {
      setBusy(null);
    }
  }

  async function copyKey(secret: string) {
    try {
      await navigator.clipboard.writeText(secret);
      setCopied(true);
    } catch {
      // Clipboard access can be refused; the key is on screen either way.
    }
  }

  const stats = overview?.stats;
  const curl = [
    `curl -X POST ${overview?.send_url || ""} \\`,
    `  -H "Authorization: Bearer evk_..." \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -d '{"email":"user@example.com","redirect_url":"https://theirapp.com/verified"}'`,
  ].join("\n");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
            <MailCheck className="w-7 h-7 text-indigo-600" />
            <span>{t("emailverify.view.title")}</span>
          </h1>
          <p className="text-sm text-slate-500 mt-1">{t("emailverify.view.subtitle")}</p>
        </div>
        <div className="flex items-center space-x-2">
          <button
            onClick={() => void loadData()}
            aria-label={t("base.action.retry")}
            className="p-2 text-slate-600 hover:bg-slate-100 rounded-lg border border-slate-200 transition"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          <button
            onClick={() => setShowModal(true)}
            className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg flex items-center space-x-2 shadow-sm transition"
          >
            <Plus className="w-4 h-4" />
            <span>{t("emailverify.action.create_client")}</span>
          </button>
        </div>
      </div>

      {banner && (
        <div
          role="status"
          className={`p-4 text-sm rounded-lg border ${
            banner.kind === "ok"
              ? "bg-emerald-50 border-emerald-200 text-emerald-800"
              : "bg-red-50 border-red-200 text-red-700"
          }`}
        >
          {banner.text}
        </div>
      )}

      {/* Mail that is only logged looks exactly like mail that was sent, right
          up until somebody asks why nothing arrived. */}
      {overview && !overview.mail_configured && (
        <div className="p-4 text-sm rounded-lg border bg-amber-50 border-amber-200 text-amber-800 flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
          <span>{t("emailverify.message.mail_not_configured")}</span>
        </div>
      )}

      {/* The one moment the key exists outside the caller's own storage. */}
      {issued?.secret && (
        <div className="p-4 rounded-xl border-2 border-indigo-300 bg-indigo-50">
          <div className="flex items-start gap-2 text-indigo-900 text-sm font-semibold">
            <KeyRound className="w-4 h-4 mt-0.5 shrink-0" />
            <span>{issued.name}</span>
          </div>
          <p className="text-xs text-indigo-800 mt-1">{t("emailverify.message.secret_once")}</p>
          <div className="mt-3 flex items-center gap-2">
            <code className="flex-1 bg-white border border-indigo-200 rounded-lg px-3 py-2 text-xs font-mono text-slate-800 break-all">
              {issued.secret}
            </code>
            <button
              onClick={() => void copyKey(issued.secret!)}
              className="shrink-0 flex items-center gap-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-3 py-2 rounded-lg"
            >
              <Copy className="w-3.5 h-3.5" />
              {copied ? t("emailverify.message.copied") : t("emailverify.action.copy")}
            </button>
            <button
              onClick={() => setIssued(null)}
              aria-label={t("base.action.close")}
              className="shrink-0 p-2 text-indigo-700 hover:bg-indigo-100 rounded-lg"
            >
              <XCircle className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="py-12 text-center text-slate-400">{t("emailverify.message.loading")}</div>
      ) : (
        <>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
            <StatCard label={t("emailverify.stat.total")} value={stats?.total ?? 0} />
            <StatCard label={t("emailverify.stat.verified")} value={stats?.verified ?? 0} tone="ok" />
            <StatCard label={t("emailverify.stat.pending")} value={stats?.pending ?? 0} tone="wait" />
            <StatCard label={t("emailverify.stat.expired")} value={stats?.expired ?? 0} tone="off" />
            <StatCard
              label={t("emailverify.stat.verified_pct")}
              value={`${Math.round(stats?.verified_pct ?? 0)}%`}
            />
          </div>

          <section className="bg-white border border-slate-200 rounded-xl overflow-hidden">
            <header className="px-5 py-3 border-b border-slate-200 flex items-center justify-between">
              <h2 className="font-bold text-slate-900 text-sm flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-indigo-600" />
                {t("emailverify.view.clients_title")}
              </h2>
              <span className="text-[11px] text-slate-500">{t("emailverify.message.disabled_note")}</span>
            </header>
            {clients.length === 0 ? (
              <p className="p-8 text-center text-sm text-slate-500">{t("emailverify.message.no_clients")}</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-slate-50 text-[11px] uppercase tracking-wide text-slate-500">
                    <tr>
                      <th className="text-left font-semibold px-5 py-2">{t("emailverify.field.client_name")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("base.field.status")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("emailverify.field.hourly_limit")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("emailverify.field.allowed_hosts")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("emailverify.field.last_used")}</th>
                      <th className="text-right font-semibold px-5 py-2">{t("base.field.actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {clients.map((client) => (
                      <tr key={client.id} className="hover:bg-slate-50">
                        <td className="px-5 py-3">
                          <div className="font-semibold text-slate-900">{client.name}</div>
                          <code className="text-[11px] text-slate-500 font-mono">{client.key_prefix}…</code>
                        </td>
                        <td className="px-5 py-3">
                          <span
                            className={`inline-flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-full border ${
                              client.status === "ACTIVE"
                                ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                                : "bg-slate-100 text-slate-600 border-slate-200"
                            }`}
                          >
                            {client.status === "ACTIVE" ? (
                              <CheckCircle2 className="w-3 h-3" />
                            ) : (
                              <Power className="w-3 h-3" />
                            )}
                            {client.status === "ACTIVE" ? t("base.state.active") : t("base.state.inactive")}
                          </span>
                        </td>
                        <td className="px-5 py-3 text-slate-600">{client.hourly_limit}/h</td>
                        <td className="px-5 py-3 text-slate-600 text-xs">
                          {client.allowed_redirect_hosts.length > 0
                            ? client.allowed_redirect_hosts.join(", ")
                            : "—"}
                        </td>
                        <td className="px-5 py-3 text-slate-500 text-xs">
                          {client.last_used_at ? new Date(client.last_used_at).toLocaleString() : "—"}
                        </td>
                        <td className="px-5 py-3">
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => void handleToggle(client)}
                              disabled={busy === client.id}
                              className="border border-slate-300 text-slate-700 hover:bg-slate-100 text-xs font-semibold px-3 py-1.5 rounded-lg disabled:opacity-50"
                            >
                              {client.status === "ACTIVE"
                                ? t("emailverify.action.disable")
                                : t("emailverify.action.enable")}
                            </button>
                            <button
                              onClick={() => void handleDelete(client)}
                              disabled={busy === client.id}
                              aria-label={t("base.action.delete")}
                              className="p-1.5 border border-slate-300 text-red-600 hover:bg-red-50 rounded-lg disabled:opacity-50"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <section className="bg-white border border-slate-200 rounded-xl p-5 space-y-3">
              <h2 className="font-bold text-slate-900 text-sm">{t("emailverify.view.usage_title")}</h2>
              <p className="text-xs text-slate-500">{t("emailverify.message.usage")}</p>
              <pre className="bg-slate-900 text-slate-100 text-[11px] rounded-lg p-3 overflow-x-auto">
                <code>{curl}</code>
              </pre>
              <p className="text-xs text-slate-500">{t("emailverify.message.in_app_usage")}</p>
            </section>

            <section className="bg-white border border-slate-200 rounded-xl p-5 space-y-3">
              <h2 className="font-bold text-slate-900 text-sm">{t("emailverify.view.test_title")}</h2>
              <form onSubmit={handleTestSend} className="space-y-3">
                <input
                  type="email"
                  required
                  value={testEmail}
                  onChange={(e) => setTestEmail(e.target.value)}
                  placeholder="user@example.com"
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
                <input
                  type="url"
                  value={testRedirect}
                  onChange={(e) => setTestRedirect(e.target.value)}
                  placeholder="https://theirapp.com/verified"
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
                <button
                  type="submit"
                  disabled={busy === "test"}
                  className="w-full flex items-center justify-center gap-2 bg-slate-900 hover:bg-slate-800 text-white text-xs font-semibold py-2 rounded-lg disabled:opacity-50"
                >
                  <Send className="w-3.5 h-3.5" />
                  {t("emailverify.action.send_test")}
                </button>
              </form>
            </section>
          </div>

          <section className="bg-white border border-slate-200 rounded-xl overflow-hidden">
            <header className="px-5 py-3 border-b border-slate-200">
              <h2 className="font-bold text-slate-900 text-sm flex items-center gap-2">
                <Clock className="w-4 h-4 text-indigo-600" />
                {t("emailverify.view.recent_title")}
              </h2>
            </header>
            {(overview?.recent.length ?? 0) === 0 ? (
              <p className="p-8 text-center text-sm text-slate-500">
                {t("emailverify.message.no_verifications")}
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-slate-50 text-[11px] uppercase tracking-wide text-slate-500">
                    <tr>
                      <th className="text-left font-semibold px-5 py-2">{t("base.field.email")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("emailverify.field.source")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("emailverify.field.purpose")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("base.field.status")}</th>
                      <th className="text-left font-semibold px-5 py-2">{t("base.field.date")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {overview?.recent.map((row) => (
                      <tr key={row.id} className="hover:bg-slate-50">
                        <td className="px-5 py-3 text-slate-900">{row.email}</td>
                        <td className="px-5 py-3 text-slate-600 text-xs">{row.source}</td>
                        <td className="px-5 py-3 text-slate-500 text-xs">{row.purpose || "—"}</td>
                        <td className="px-5 py-3">
                          <VerificationBadge row={row} />
                        </td>
                        <td className="px-5 py-3 text-slate-500 text-xs">
                          {new Date(row.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      )}

      {showModal && (
        <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
            <h2 className="text-xl font-bold text-slate-900 mb-4">{t("emailverify.view.create_title")}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("emailverify.field.client_name")} *
                </label>
                <input
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder={t("emailverify.field.client_name_placeholder")}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("emailverify.field.hourly_limit")}
                </label>
                <input
                  type="number"
                  min={1}
                  value={form.hourly_limit}
                  onChange={(e) => setForm({ ...form, hourly_limit: Number(e.target.value) })}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
                <p className="mt-1 text-[11px] text-slate-500">{t("emailverify.message.limit_hint")}</p>
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("emailverify.field.allowed_hosts")}
                </label>
                <input
                  type="text"
                  value={form.hosts}
                  onChange={(e) => setForm({ ...form, hosts: e.target.value })}
                  placeholder={t("emailverify.field.allowed_hosts_placeholder")}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                />
                <p className="mt-1 text-[11px] text-slate-500">{t("emailverify.message.hosts_hint")}</p>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-lg"
                >
                  {t("base.action.cancel")}
                </button>
                <button
                  type="submit"
                  className="bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold px-4 py-2 rounded-lg"
                >
                  {t("emailverify.action.issue")}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: number | string;
  tone?: "ok" | "wait" | "off";
}) {
  const toneClass =
    tone === "ok"
      ? "text-emerald-700"
      : tone === "wait"
        ? "text-amber-700"
        : tone === "off"
          ? "text-slate-500"
          : "text-slate-900";
  return (
    <div className="bg-white border border-slate-200 rounded-xl p-4">
      <div className={`text-2xl font-bold ${toneClass}`}>{value}</div>
      <div className="text-[11px] uppercase tracking-wide text-slate-500 mt-1">{label}</div>
    </div>
  );
}

/**
 * A link whose deadline has passed is shown as expired even while the row still
 * says PENDING: the sweep that rewrites it runs on a timer, and until it does,
 * "pending" would claim somebody is still able to act on a dead link.
 */
function VerificationBadge({ row }: { row: EmailVerification }) {
  const { t } = useI18n();
  const expired = row.status === "EXPIRED" || (row.status === "PENDING" && new Date(row.expires_at) <= new Date());
  const state = row.status === "VERIFIED" ? "verified" : expired ? "expired" : "pending";
  const style =
    state === "verified"
      ? "bg-emerald-50 text-emerald-700 border-emerald-200"
      : state === "pending"
        ? "bg-amber-50 text-amber-700 border-amber-200"
        : "bg-slate-100 text-slate-600 border-slate-200";
  const label =
    state === "verified"
      ? t("emailverify.state.verified")
      : state === "pending"
        ? t("emailverify.state.pending")
        : t("emailverify.state.expired");
  return (
    <span className={`inline-flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded-full border ${style}`}>
      {state === "verified" ? <CheckCircle2 className="w-3 h-3" /> : <Clock className="w-3 h-3" />}
      {label}
    </span>
  );
}
