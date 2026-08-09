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

export type BannerTone = "error" | "success" | "info" | "warning";

const BANNER_STYLE: Record<BannerTone, string> = {
  error: "bg-red-50 border-red-200 text-red-700",
  success: "bg-emerald-50 border-emerald-200 text-emerald-700",
  info: "bg-blue-50 border-blue-200 text-blue-700",
  // Not an outcome but a standing condition the screen cannot fix: no service
  // key, no encryption key. The settings screens had this in amber already.
  warning: "bg-amber-50 border-amber-200 text-amber-800",
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
    // role="status" so a screen reader announces the outcome of an action the
    // user cannot see the result of. The settings screens already did this; the
    // rest did not, and the answer to that disagreement is the accessible one.
    <div role="status" className={`p-3 border text-sm rounded-lg flex items-start gap-2 ${BANNER_STYLE[tone]}`}>
      {tone === "error" || tone === "warning" ? (
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

/**
 * The shapes below are style, not structure, so they are exported as class
 * strings rather than wrapped in components. A form field is an <input> on one
 * screen, a <select> on the next and a <textarea> on a third, all of them
 * carrying their own props; a component that took all of those would be a
 * worse <input>. What was actually duplicated is the look, and a name for the
 * look is enough to stop it drifting.
 */

/** Text, number, select and textarea inputs. Repeated 33 times before this. */
export const fieldClass =
  "w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500";

/** The white panel a section sits on. */
export const cardClass = "bg-white rounded-xl border border-slate-200 shadow-sm";

/** The header row of a listing table. */
export const tableHeadClass =
  "bg-slate-50 text-slate-700 font-semibold border-b border-slate-200 uppercase";

/** The small bordered button in a table row: Save, Use template, and friends. */
export const rowActionClass =
  "inline-flex items-center space-x-1 px-2.5 py-1.5 rounded-lg text-[11px] font-semibold " +
  "border border-indigo-200 text-indigo-600 hover:bg-indigo-50 disabled:opacity-50";

/**
 * A centred dialog over a dimmed page.
 *
 * Structure only: it deliberately does not close on Escape or on a backdrop
 * click, because none of the twelve dialogs it replaced did. Several of them
 * hold typed input, and one is a signing conversation with a citizen's device
 * — losing either to a stray click outside is worse than having to reach for
 * Cancel. Adding dismissal is a change to how these screens behave and belongs
 * in a change that says so.
 */
export function Modal({
  size = "md",
  scrollable = false,
  className,
  children,
}: {
  size?: "md" | "lg";
  /**
   * Let the backdrop scroll instead of the panel. The signing dialog needs it:
   * it grows with the number of steps, and a panel taller than the window
   * otherwise puts its own buttons past the bottom edge with no way to reach
   * them. Pair it with a vertical margin on the panel, or the top of a tall
   * dialog is clipped by the centring.
   */
  scrollable?: boolean;
  /** Extra panel classes. Height and scrolling genuinely differ per dialog. */
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={`fixed inset-0 bg-slate-900/50 flex items-center justify-center z-50 p-4${
        scrollable ? " overflow-y-auto" : ""
      }`}
    >
      <div
        className={`bg-white rounded-xl ${size === "lg" ? "max-w-lg" : "max-w-md"} w-full p-6 shadow-xl border border-slate-200${
          className ? ` ${className}` : ""
        }`}
      >
        {children}
      </div>
    </div>
  );
}

/** What a listing shows while its first load is outstanding. */
export function LoadingBlock({ label }: { label?: string }) {
  const { t } = useI18n();
  return <div className="py-12 text-center text-slate-400">{label || t("base.message.loading")}</div>;
}

/** A listing table on its panel, with the header row the caller supplies. */
export function TableCard({
  head,
  footer,
  children,
}: {
  head: React.ReactNode;
  /**
   * Rendered inside the panel, below the table. This is where the lists put
   * "these rows are stale" and their Load more button — both of them statements
   * about the table, so they belong on the same piece of paper as it.
   */
  footer?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className={`${cardClass} overflow-hidden`}>
      <table className="w-full text-left text-xs text-slate-600">
        <thead className={tableHeadClass}>{head}</thead>
        <tbody className="divide-y divide-slate-100">{children}</tbody>
      </table>
      {footer}
    </div>
  );
}
