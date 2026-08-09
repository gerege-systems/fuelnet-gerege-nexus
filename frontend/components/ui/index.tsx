"use client";

import React from "react";
import { AlertTriangle, CheckCircle2, Loader2, X } from "lucide-react";
import { useI18n } from "@/lib/i18n";

/**
 * The chrome every module screen is built from.
 *
 * Documents, E-Sign and Gov Services each grew their own copy of these four:
 * three PageHeaders that were identical character for character, three Banners
 * that differed only in padding and the shade of red, and two EmptyStates that
 * disagreed about whether the text is centred. Three copies of a header is not
 * three decisions — it is one decision recorded three times, and the screens had
 * already started to drift apart on it.
 *
 * Anything domain-specific stays in the module's own shared.tsx. What lands here
 * is what a screen needs regardless of which app it belongs to.
 */

export function PageHeader({
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

export type BannerTone = "error" | "success" | "info";

const BANNER_STYLE: Record<BannerTone, string> = {
  error: "bg-red-50 border-red-200 text-red-700",
  success: "bg-emerald-50 border-emerald-200 text-emerald-700",
  info: "bg-blue-50 border-blue-200 text-blue-700",
};

/**
 * What a screen says after an action, good or bad.
 *
 * onDismiss is optional because the two cases are genuinely different: a banner
 * reporting the outcome of something the operator pressed should be dismissable,
 * and one stating a standing condition of the screen — mock mode, a signature
 * placed off the page — should not be, because dismissing it would not make it
 * untrue.
 */
export function Banner({
  tone,
  message,
  onDismiss,
}: {
  tone: BannerTone;
  message: string;
  onDismiss?: () => void;
}) {
  const { t } = useI18n();
  return (
    <div className={`p-3 border text-sm rounded-lg flex items-start gap-2 ${BANNER_STYLE[tone]}`}>
      {tone === "error" ? (
        <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
      ) : (
        <CheckCircle2 className="w-4 h-4 mt-0.5 shrink-0" />
      )}
      <span className="flex-1">{message}</span>
      {onDismiss && (
        <button onClick={onDismiss} aria-label={t("base.action.close")}>
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  );
}

export function Loading({ label }: { label?: string }) {
  const { t } = useI18n();
  return (
    <div className="flex items-center gap-2 text-slate-500 text-sm">
      <Loader2 className="w-4 h-4 animate-spin" />
      {label || t("base.message.loading")}
    </div>
  );
}

export function EmptyState({ message }: { message: string }) {
  return <p className="p-6 text-sm text-slate-500 text-center italic">{message}</p>;
}
