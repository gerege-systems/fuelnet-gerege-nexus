"use client";

/**
 * The console's small shared pieces: a card, a table, a badge, and the one
 * date formatter every screen uses.
 *
 * They lived in the tenant detail page for a phase, and four other screens
 * imported them from it — a page importing a page, which works and reads as an
 * accident. This is where they belong.
 */

import React from "react";

export function Card({ title, action, children }: { title: string; action?: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <h2 className="px-4 py-3 border-b border-slate-100 font-medium text-slate-900 flex items-center gap-3">
        <span className="flex-1">{title}</span>
        {action}
      </h2>
      {children}
    </section>
  );
}

export function Table({ head, rows, empty }: { head: string[]; rows: React.ReactNode[][]; empty: string }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-slate-600">
          <tr>
            {head.map((cell, index) => (
              <th key={index} className="text-left font-medium px-4 py-2.5">
                {cell}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {rows.map((row, index) => (
            <tr key={index} className="hover:bg-slate-50">
              {row.map((cell, cellIndex) => (
                <td key={cellIndex} className="px-4 py-2.5 text-slate-700">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
          {rows.length === 0 && (
            <tr>
              <td colSpan={head.length} className="px-4 py-8 text-center text-slate-500">
                {empty}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

export type Tone = "red" | "amber" | "emerald" | "slate" | "green";

export function Badge({ tone, children }: { tone: Tone; children: React.ReactNode }) {
  const tones: Record<Tone, string> = {
    red: "bg-red-100 text-red-800",
    amber: "bg-amber-100 text-amber-900",
    emerald: "bg-emerald-100 text-emerald-800",
    green: "bg-emerald-100 text-emerald-800",
    slate: "bg-slate-100 text-slate-700",
  };
  return <span className={`text-xs font-medium rounded-full px-2 py-0.5 ${tones[tone]}`}>{children}</span>;
}

/**
 * Timestamps are rendered in the reader's own locale and time zone.
 *
 * The API sends RFC 3339 with an offset, and the browser is the only party
 * that knows where the person reading it is sitting — the same reasoning the
 * monitoring alerts were changed to follow.
 */
export function formatMoment(value: string | null | undefined, locale: string): string {
  if (!value) return "";
  const moment = new Date(value);
  if (Number.isNaN(moment.getTime())) return "";
  return moment.toLocaleString(locale === "mn" ? "mn-MN" : locale, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}
