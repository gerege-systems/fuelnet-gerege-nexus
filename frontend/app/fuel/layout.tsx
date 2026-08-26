"use client";

/**
 * The fuel network's three screens, under one heading.
 *
 * Tabs rather than three sidebar entries. The module declares one menu entry
 * and the platform's menu test asserts every entry has a page; three entries
 * would mean three sets of seven locale labels for what is one job — a company
 * looking at its own half of the chain. The order is the chain's own: what
 * crossed the border, where it was stored, where it is sold.
 */

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Fuel } from "lucide-react";

import { useI18n } from "@/lib/i18n";

export default function FuelLayout({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const pathname = usePathname();

  const tabs = [
    { href: "/fuel/shipments", label: t("fuel.tab.shipments") },
    { href: "/fuel/depots", label: t("fuel.tab.depots") },
    { href: "/fuel", label: t("fuel.tab.stations") },
  ];

  return (
    <div className="p-6 max-w-6xl">
      <header className="mb-6">
        <h1 className="flex items-center gap-3 text-2xl font-semibold text-slate-900">
          <Fuel className="h-6 w-6 text-[var(--gerege-blue)]" />
          {t("fuel.view.title")}
        </h1>
      </header>

      <nav className="mb-8 flex gap-1 border-b border-slate-200">
        {tabs.map((tab) => {
          const active = pathname === tab.href;
          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={`-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
                active
                  ? "border-[var(--gerege-blue)] text-[var(--gerege-blue)]"
                  : "border-transparent text-slate-500 hover:text-slate-900"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </nav>

      {children}
    </div>
  );
}
