"use client";

/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The small parts the operator's fuel screens share.
 *
 * Here rather than duplicated across two pages because the two screens are one
 * job seen at two points on the chain, and a field that looks different on the
 * depot screen than on the customs screen reads as a different kind of field.
 */

import type { ReactNode } from "react";
import { Loader2 } from "lucide-react";

/** A labelled input. The label is a real <label>, so tapping it focuses. */
export function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-slate-600">{label}</span>
      {children}
      {hint ? <span className="mt-1 block text-xs text-slate-400">{hint}</span> : null}
    </label>
  );
}

export const inputClass =
  "w-full rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 " +
  "placeholder:text-slate-400 focus:border-[var(--gerege-blue)] focus:outline-none " +
  "focus:ring-2 focus:ring-[var(--gerege-blue-soft)]";

export function PrimaryButton({
  children,
  busy = false,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { busy?: boolean }) {
  return (
    <button
      {...rest}
      disabled={busy || rest.disabled}
      className={
        "inline-flex items-center gap-2 rounded-lg bg-[var(--gerege-blue)] px-4 py-2 " +
        "text-sm font-medium text-white transition-opacity hover:opacity-90 " +
        "disabled:cursor-not-allowed disabled:opacity-50 " +
        (rest.className ?? "")
      }
    >
      {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
      {children}
    </button>
  );
}

export function GhostButton({
  children,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...rest}
      className={
        "rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 " +
        "hover:bg-slate-50 disabled:opacity-50 " +
        (rest.className ?? "")
      }
    >
      {children}
    </button>
  );
}

/**
 * A consignment's state, in the colour the state deserves.
 *
 * Cleared is green because it is the moment the chain starts — the batch
 * exists from here and not before. Inspection is amber rather than red: a
 * consignment under inspection is an ordinary consignment, not a problem.
 */
export function StatusPill({ status, label }: { status: string; label: string }) {
  const tone: Record<string, string> = {
    border_arrived: "bg-slate-100 text-slate-600",
    inspecting: "bg-amber-100 text-amber-700",
    cleared: "bg-emerald-100 text-emerald-700",
    in_transit: "bg-sky-100 text-sky-700",
    at_depot: "bg-indigo-100 text-indigo-700",
  };
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${
        tone[status] ?? "bg-slate-100 text-slate-600"
      }`}
    >
      {label}
    </span>
  );
}

/**
 * A tank's level, drawn.
 *
 * Amber below a fifth and red below a twentieth: a base running out is the
 * thing this screen exists to make visible before somebody has to telephone
 * about it.
 */
export function FillBar({ percent }: { percent: number }) {
  const clamped = Math.max(0, Math.min(100, percent));
  const tone =
    clamped < 5 ? "bg-red-500" : clamped < 20 ? "bg-amber-500" : "bg-[var(--gerege-blue)]";
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-slate-100">
      <div className={`h-full rounded-full ${tone}`} style={{ width: `${clamped}%` }} />
    </div>
  );
}

/** Litres, grouped, with no decimals — nobody reads a depot to the millilitre. */
export function litres(value: number): string {
  return Math.round(value).toLocaleString("mn-MN");
}

export function ErrorNote({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
      {children}
    </p>
  );
}
