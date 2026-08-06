"use client";

import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  AlertTriangle,
  Ban,
  CheckCircle,
  Clock,
  FileText,
  Loader2,
  PenLine,
  ShieldCheck,
  Users,
  X,
  XCircle,
} from "lucide-react";

/** A document_records row as the documents API returns it. */
export interface DocumentRecord {
  id: string;
  title: string;
  doc_type: string;
  status: string;
  signed_by?: string;
  signature_hash?: string;
  signer_reg_number?: string;
  signer_method?: string;
  signed_at?: string;
  /** How many signatures the document carries, and how many its type needs. */
  signature_count: number;
  required_signatures: number;
  created_at: string;
}

/** The national identity channel the signature is applied through. */
export type SignMethod = "EID" | "DAN";

/** The only state a document can be signed or rejected in. */
export const PENDING = "PENDING_APPROVAL";

/** What the citizen's device needs to be found, and what the operator reads out. */
export interface EIDSignSession {
  session_id: string;
  verification_code: string;
  expires_at: string;
  device_link_url?: string;
  display_text: string;
}

/**
 * Only the four states the module defines get a translated badge. Anything
 * else — the DRAFT the table still defaults to, or a state a later workflow
 * introduces — is shown verbatim rather than dressed up as "Pending", so a
 * screen never claims a document is awaiting signature when it is not.
 */
export function StatusBadge({ status }: { status: string }) {
  const { t } = useI18n();
  const shell = "inline-flex items-center space-x-1 text-[11px] font-bold px-2.5 py-0.5 rounded-full";

  if (status === "APPROVED") {
    return (
      <span className={`${shell} bg-emerald-50 text-emerald-700 border border-emerald-200`}>
        <CheckCircle className="w-3 h-3 text-emerald-500" />
        <span>{t("documents.state.approved")}</span>
      </span>
    );
  }
  if (status === "REJECTED") {
    return (
      <span className={`${shell} bg-red-50 text-red-700 border border-red-200`}>
        <XCircle className="w-3 h-3 text-red-500" />
        <span>{t("documents.state.rejected")}</span>
      </span>
    );
  }
  if (status === PENDING) {
    return (
      <span className={`${shell} bg-amber-50 text-amber-700 border border-amber-200`}>
        <Clock className="w-3 h-3 text-amber-500" />
        <span>{t("documents.state.pending")}</span>
      </span>
    );
  }
  if (status === "DRAFT") {
    return (
      <span className={`${shell} bg-slate-100 text-slate-600 border border-slate-200`}>
        <FileText className="w-3 h-3 text-slate-400" />
        <span>{t("documents.state.draft")}</span>
      </span>
    );
  }
  return <span className={`${shell} bg-slate-100 text-slate-600 border border-slate-200`}>{status}</span>;
}

/**
 * How far a document is through its approval chain. A type that needs one
 * signature says nothing: the status badge already carries that.
 */
export function SignatureProgress({ doc }: { doc: DocumentRecord }) {
  const { t } = useI18n();
  if (doc.required_signatures <= 1) return null;

  const complete = doc.signature_count >= doc.required_signatures;
  return (
    <span
      className={`inline-flex items-center space-x-1 text-[11px] font-semibold px-2 py-0.5 rounded-full border ${
        complete
          ? "bg-emerald-50 text-emerald-700 border-emerald-200"
          : "bg-indigo-50 text-indigo-700 border-indigo-200"
      }`}
      title={t("documents.message.signature_progress", {
        applied: doc.signature_count,
        required: doc.required_signatures,
      })}
    >
      <Users className="w-3 h-3" />
      <span>
        {doc.signature_count}/{doc.required_signatures}
      </span>
    </span>
  );
}

/** Who signed, plus the reg number and hash the identity provider returned. */
export function SignatureCell({ doc }: { doc: DocumentRecord }) {
  const { t } = useI18n();

  if (doc.signed_by) {
    return (
      <div className="space-y-1">
        <span className="bg-blue-50 text-blue-700 text-[11px] font-semibold px-2.5 py-0.5 rounded-full border border-blue-200 inline-flex items-center w-max space-x-1">
          <ShieldCheck className="w-3 h-3 text-blue-500" />
          <span>{doc.signed_by}</span>
        </span>
        {doc.signature_hash && (
          <div className="font-mono text-[10px] text-slate-400 truncate max-w-[220px]" title={doc.signature_hash}>
            {doc.signer_reg_number ? `${doc.signer_reg_number} · ` : ""}
            {doc.signature_hash}
          </div>
        )}
      </div>
    );
  }
  if (doc.status === "REJECTED") {
    return <span className="text-red-400 italic">{t("documents.message.rejected_not_signed")}</span>;
  }
  return <span className="text-slate-400 italic">{t("documents.state.pending_signature")}</span>;
}

/** The heading every Documents screen opens with. */
export function SectionHeader({
  icon,
  title,
  subtitle,
  actions,
}: {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  actions?: React.ReactNode;
}) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          {icon}
          {title}
        </h1>
        <p className="text-sm text-slate-500 mt-1">{subtitle}</p>
      </div>
      {actions}
    </header>
  );
}

export interface ActionMessage {
  type: "success" | "error";
  text: string;
}

export function Banner({ message, onDismiss }: { message: ActionMessage; onDismiss: () => void }) {
  const { t } = useI18n();
  const error = message.type === "error";
  return (
    <div
      className={`p-4 rounded-lg flex items-start space-x-3 text-sm font-medium border ${
        error ? "bg-red-50 border-red-200 text-red-800" : "bg-emerald-50 border-emerald-200 text-emerald-800"
      }`}
    >
      {error ? (
        <AlertTriangle className="w-5 h-5 text-red-600 shrink-0" />
      ) : (
        <CheckCircle className="w-5 h-5 text-emerald-600 shrink-0" />
      )}
      <span className="flex-1">{message.text}</span>
      <button onClick={onDismiss} aria-label={t("base.action.close")}>
        <X className="w-4 h-4" />
      </button>
    </div>
  );
}

/**
 * Reject and outcome reporting, shared by the documents list and the approval
 * queue so both screens say the same things. Signing itself lives in
 * SignatureDialog, because E-ID signing is a conversation with the citizen's
 * device rather than one call. `onChanged` reloads the caller's rows.
 */
export function useDocumentActions(onChanged: () => void | Promise<void>) {
  const { t } = useI18n();
  const [busyId, setBusyId] = useState<string | null>(null);
  const [message, setMessage] = useState<ActionMessage | null>(null);

  const succeed = async (text: string) => {
    setMessage({ type: "success", text });
    await onChanged();
  };

  const fail = (text: string) => setMessage({ type: "error", text });

  const reject = async (doc: DocumentRecord) => {
    if (!confirm(t("documents.message.reject_confirm", { title: doc.title }))) return;
    setBusyId(doc.id);
    setMessage(null);
    try {
      await api.rejectDocument(doc.id);
      setMessage({ type: "success", text: t("documents.message.reject_success", { title: doc.title }) });
      await onChanged();
    } catch (err: any) {
      setMessage({ type: "error", text: err?.message || t("documents.message.reject_failed") });
    } finally {
      setBusyId(null);
    }
  };

  return { busyId, message, setMessage, succeed, fail, reject };
}

/** The Sign / Reject pair, shown only while a document is still pending. */
export function RowActions({
  doc,
  busy,
  canSign,
  onSign,
  onReject,
}: {
  doc: DocumentRecord;
  busy: boolean;
  canSign: boolean;
  onSign: (doc: DocumentRecord) => void;
  onReject: (doc: DocumentRecord) => void;
}) {
  const { t } = useI18n();

  if (doc.status !== PENDING) return <span className="text-slate-300 text-[11px]">—</span>;
  if (!canSign) return <span className="text-slate-300 text-[11px]">—</span>;

  return (
    <div className="flex items-center justify-end space-x-2">
      <button
        onClick={() => onSign(doc)}
        disabled={busy}
        className="inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold border border-indigo-200 text-indigo-600 hover:bg-indigo-50 transition disabled:opacity-50"
      >
        <PenLine className="w-3.5 h-3.5" />
        <span>{t("documents.action.sign")}</span>
      </button>
      <button
        onClick={() => onReject(doc)}
        disabled={busy}
        className="inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold border border-red-200 text-red-600 hover:bg-red-50 transition disabled:opacity-50"
      >
        <Ban className="w-3.5 h-3.5" />
        <span>{t("documents.action.reject")}</span>
      </button>
    </div>
  );
}


/**
 * The signing ceremony, which is not the same shape for the two channels.
 *
 * E-ID has no document-signing endpoint. What it has is an approval the citizen
 * gives on their own registered device, against a display text that names the
 * document — and that approval *is* the signature. So the dialog pushes the
 * request, shows the operator the verification code to read out, and waits.
 *
 * DAN exposes no approval push, so it stays a registration number and a code.
 *
 * The dialog owns these calls rather than the caller: a conversation with
 * somebody's phone has states of its own, and they belong next to the screen
 * showing them.
 */
export function SignatureDialog({
  doc,
  onClose,
  onDone,
  onError,
}: {
  doc: DocumentRecord;
  onClose: () => void;
  onDone: (message: string) => void | Promise<void>;
  onError: (message: string) => void;
}) {
  const { t } = useI18n();
  const [method, setMethod] = useState<SignMethod>("EID");
  const [regNumber, setRegNumber] = useState("");
  const [otpCode, setOtpCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [session, setSession] = useState<EIDSignSession | null>(null);

  // Wait on the citizen for as long as the dialog is open, and stop the moment it
  // is not: the poll is a request the API holds open, so leaving one running
  // behind a closed dialog would keep asking about a signature nobody awaits.
  useEffect(() => {
    if (!session) return;

    const controller = new AbortController();
    let cancelled = false;

    const wait = async () => {
      while (!cancelled) {
        try {
          const progress = await api.pollEIDSignature(doc.id, session.session_id, controller.signal);
          if (cancelled) return;

          if (progress.state === "COMPLETE") {
            await onDone(t("documents.message.sign_success", { title: doc.title, method: "E-ID" }));
            return;
          }
          if (progress.state === "REFUSED") {
            onError(t("documents.message.approval_refused"));
            setSession(null);
            return;
          }
          if (progress.state === "EXPIRED") {
            onError(t("documents.message.approval_expired"));
            setSession(null);
            return;
          }
        } catch (err: any) {
          if (cancelled || controller.signal.aborted) return;
          onError(err?.message || t("documents.message.sign_failed"));
          setSession(null);
          return;
        }
      }
    };
    void wait();

    return () => {
      cancelled = true;
      controller.abort();
    };
    // onDone/onError are recreated every render by the caller; re-running this on
    // their identity would restart the wait on every keystroke elsewhere.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session, doc.id, doc.title]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      if (method === "DAN") {
        await api.signDocumentWithDAN(doc.id, { reg_number: regNumber, otp_code: otpCode });
        await onDone(t("documents.message.sign_success", { title: doc.title, method: "DAN" }));
        return;
      }
      setSession(await api.startEIDSignature(doc.id, regNumber));
    } catch (err: any) {
      onError(err?.message || t("documents.message.sign_failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl border border-slate-200">
        <h2 className="text-xl font-bold text-slate-900 mb-1 flex items-center space-x-2">
          <PenLine className="w-5 h-5 text-indigo-600" />
          <span>{t("documents.view.sign_title")}</span>
        </h2>
        <p className="text-xs text-slate-500 mb-4 truncate">{doc.title}</p>

        {session ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-indigo-200 bg-indigo-50 p-4 text-center">
              <p className="text-[11px] font-semibold text-indigo-700 uppercase tracking-wide">
                {t("documents.field.verification_code")}
              </p>
              <p className="text-3xl font-bold tracking-[0.2em] text-indigo-900 mt-1">
                {session.verification_code}
              </p>
              <p className="text-[11px] text-indigo-700 mt-2">{t("documents.message.verification_code_hint")}</p>
            </div>

            <div className="flex items-center gap-2 text-sm text-slate-600">
              <Loader2 className="w-4 h-4 animate-spin shrink-0" />
              <span>{t("documents.message.awaiting_approval", { reg: regNumber.toUpperCase() })}</span>
            </div>

            <p className="text-[11px] text-slate-500">
              {t("documents.message.approval_display_text")}: <span className="italic">{session.display_text}</span>
            </p>

            <button
              type="button"
              onClick={onClose}
              className="w-full bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
            >
              {t("base.action.cancel")}
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                {t("documents.field.signature_method")}
              </label>
              <div className="grid grid-cols-2 gap-2">
                {(["EID", "DAN"] as SignMethod[]).map((m) => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => setMethod(m)}
                    className={`py-2 rounded-lg text-xs font-semibold border transition ${
                      method === m
                        ? "bg-indigo-600 text-white border-indigo-600"
                        : "bg-white text-slate-700 border-slate-300 hover:bg-slate-50"
                    }`}
                  >
                    {m === "EID" ? "E-ID (eidmongolia.mn)" : "DAN (dan.gerege.mn)"}
                  </button>
                ))}
              </div>
              <p className="text-[11px] text-slate-500 mt-1.5">
                {method === "EID"
                  ? t("documents.message.eid_method_hint")
                  : t("documents.message.dan_method_hint")}
              </p>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                {t("documents.field.reg_number")} *
              </label>
              <input
                type="text"
                placeholder="e.g. AA90010111"
                value={regNumber}
                onChange={(e) => setRegNumber(e.target.value)}
                className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 font-mono"
                required
              />
            </div>

            {method === "DAN" && (
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  {t("documents.field.otp_code")}
                </label>
                <input
                  type="text"
                  placeholder="e.g. 123456"
                  value={otpCode}
                  onChange={(e) => setOtpCode(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 font-mono"
                />
              </div>
            )}

            <div className="flex items-center space-x-2 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="w-1/2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium py-2 rounded-lg text-xs"
              >
                {t("base.action.cancel")}
              </button>
              <button
                type="submit"
                disabled={busy}
                className="w-1/2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 rounded-lg text-xs disabled:opacity-50"
              >
                {busy
                  ? t("documents.message.signing")
                  : method === "EID"
                    ? t("documents.action.request_approval")
                    : t("documents.action.sign")}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
