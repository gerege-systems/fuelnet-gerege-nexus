"use client";

/**
 * The console's frame: who is signed in, and the sign-in form when nobody is.
 *
 * Every /cp page renders inside it, so there is one place that decides whether
 * the operator is signed in and one place that draws the form. A page that made
 * that decision for itself would eventually make it differently.
 *
 * It looks nothing like the tenant application on purpose. An operator who has
 * both open should never have to read the URL to know which window can suspend
 * an organisation — the dark chrome is the answer to that, and it is why the
 * console does not reuse the product's shell.
 */

import React, { createContext, useCallback, useContext, useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  Building2,
  CheckCheck,
  Fuel,
  LifeBuoy,
  LogOut,
  Megaphone,
  ScrollText,
  ShieldCheck,
  SlidersHorizontal,
} from "lucide-react";

import { cp, Unauthorized, type Operator } from "@/lib/cp";
import { useI18n } from "@/lib/i18n";
import { useBrand } from "@/lib/brandContext";

interface ConsoleState {
  operator: Operator;
  signOut: () => Promise<void>;
}

const ConsoleContext = createContext<ConsoleState | null>(null);

/** useConsole is how a page reaches the signed-in operator. */
export function useConsole(): ConsoleState {
  const state = useContext(ConsoleContext);
  if (!state) throw new Error("useConsole outside the console frame");
  return state;
}

export default function Console({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const brand = useBrand();
  const [operator, setOperator] = useState<Operator | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const me = await cp.me();
      setOperator(me.operator);
    } catch (error) {
      // Anything other than "not signed in" is still answered with the form:
      // there is nothing else the console can offer, and an error page that
      // cannot be signed in from is a dead end.
      if (!(error instanceof Unauthorized)) {
        console.error("control plane: could not read the session", error);
      }
      setOperator(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const signOut = useCallback(async () => {
    try {
      await cp.signOut();
    } finally {
      setOperator(null);
    }
  }, []);

  if (loading) {
    return (
      <div className="min-h-screen grid place-items-center text-slate-500">
        <div className="animate-pulse">…</div>
      </div>
    );
  }

  if (!operator) return <SignIn onSignedIn={setOperator} />;

  return (
    <ConsoleContext.Provider value={{ operator, signOut }}>
      {/*
        The product's own shell, class for class. The console had a dark bar of
        its own for a phase, and the argument for it — an operator with both
        windows open should know which is which — is answered better by the
        badge in the corner than by a different design system: two visual
        languages in one repository is two things to maintain and one of them
        always falls behind.
      */}
      <div className="gerege-shell min-h-screen flex flex-col">
        <header className="gerege-topbar h-16 flex items-center border-b sticky top-0 z-50 px-4 gap-3">
          <Link href="/cp" className="flex items-center gap-2.5 font-semibold text-slate-900">
            <span className="w-9 h-9 rounded-lg grid place-items-center bg-slate-900">
              <ShieldCheck className="w-5 h-5 text-amber-400" />
            </span>
            <span className="min-w-0">
              <small className="block text-[11px] leading-4 text-slate-500">{brand.name}</small>
              <strong className="block text-[15px] leading-5 truncate">{t("cp.view.title")}</strong>
            </span>
          </Link>

          <div className="flex-1" />

          <span className="hidden sm:block text-sm text-slate-600 truncate max-w-[16rem]">
            {operator.name} · {t(`cp.role.${operator.role}`)}
          </span>
          <button
            type="button"
            onClick={() => void signOut()}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-100 transition"
          >
            <LogOut className="w-4 h-4" />
            <span className="hidden sm:inline">{t("cp.action.sign_out")}</span>
          </button>
        </header>

        <div className="flex flex-1 min-h-0">
          <aside className="w-16 lg:w-60 shrink-0 border-r border-[var(--gerege-border)] bg-[var(--gerege-chrome)] py-4">
            <nav className="px-2 space-y-6">
              <MenuGroup title={t("cp.group.watch")}>
                <ConsoleLink href="/cp" exact icon={<Activity className="w-5 h-5" />} label={t("cp.section.health")} />
              </MenuGroup>

              <MenuGroup title={t("cp.group.organisations")}>
                <ConsoleLink href="/cp/tenants" icon={<Building2 className="w-5 h-5" />} label={t("cp.section.tenants")} />
                <ConsoleLink href="/cp/support" icon={<LifeBuoy className="w-5 h-5" />} label={t("cp.section.support")} />
                <ConsoleLink href="/cp/approvals" icon={<CheckCheck className="w-5 h-5" />} label={t("cp.section.approvals")} />
                <ConsoleLink href="/cp/fuel" icon={<Fuel className="w-5 h-5" />} label={t("cp.section.fuel")} />
              </MenuGroup>

              <MenuGroup title={t("cp.group.platform")}>
                <ConsoleLink href="/cp/config" icon={<SlidersHorizontal className="w-5 h-5" />} label={t("cp.section.config")} />
                <ConsoleLink href="/cp/announcements" icon={<Megaphone className="w-5 h-5" />} label={t("cp.section.announcements")} />
              </MenuGroup>

              <MenuGroup title={t("cp.group.investigation")}>
                <ConsoleLink href="/cp/audit" icon={<ScrollText className="w-5 h-5" />} label={t("cp.section.audit")} />
              </MenuGroup>
            </nav>
          </aside>

          <main className="gerege-main flex-1 min-w-0 p-4 sm:p-6 lg:p-8 overflow-y-auto">
            <div className="mx-auto max-w-6xl">{children}</div>
          </main>
        </div>
      </div>
    </ConsoleContext.Provider>
  );
}

/** A titled group of destinations, as the product's own sidebar has. */
function MenuGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="gerege-menu-group">
      <h3 className="hidden lg:block px-3 mb-2 text-[11px] font-bold uppercase tracking-wider text-slate-400">
        {title}
      </h3>
      <div className="space-y-0.5">{children}</div>
    </section>
  );
}

/**
 * One destination.
 *
 * `exact` exists for the front page: every other route begins with /cp, so a
 * prefix test would light the first entry on every screen in the console.
 */
function ConsoleLink({
  href,
  label,
  icon,
  exact,
}: {
  href: string;
  label: string;
  icon: React.ReactNode;
  exact?: boolean;
}) {
  const pathname = usePathname();
  const active = exact ? pathname === href : pathname === href || pathname.startsWith(href + "/");
  return (
    <Link
      href={href}
      title={label}
      aria-current={active ? "page" : undefined}
      className={`gerege-nav-link flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition ${
        active ? "gerege-nav-link-active font-semibold" : ""
      }`}
    >
      <span className="gerege-nav-icon shrink-0">{icon}</span>
      <span className="hidden lg:inline truncate">{label}</span>
    </Link>
  );
}

function SignIn({ onSignedIn }: { onSignedIn: (operator: Operator) => void }) {
  const { t } = useI18n();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setFailed(false);
    try {
      const result = await cp.signIn(email, password, code);
      onSignedIn(result.operator);
    } catch {
      // One message for every reason. Which of the three was wrong is
      // deliberately not said — the API does not distinguish them either.
      setFailed(true);
      setCode("");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-screen grid place-items-center px-4">
      <form
        onSubmit={submit}
        className="w-full max-w-sm bg-white rounded-2xl border border-slate-200 shadow-sm p-6 space-y-4"
      >
        <div className="flex items-center gap-2 text-slate-900">
          <ShieldCheck className="w-5 h-5 text-amber-500" />
          <h1 className="text-lg font-semibold">{t("cp.login.title")}</h1>
        </div>
        <p className="text-sm text-slate-500">{t("cp.login.hint")}</p>

        {failed && (
          <p className="text-sm rounded-lg bg-red-50 text-red-700 border border-red-200 px-3 py-2">
            {t("cp.login.failed")}
          </p>
        )}

        <label className="block text-sm">
          <span className="text-slate-600">{t("cp.field.email")}</span>
          <input
            type="email"
            autoComplete="username"
            required
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
          />
        </label>

        <label className="block text-sm">
          <span className="text-slate-600">{t("cp.field.password")}</span>
          <input
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
          />
        </label>

        <label className="block text-sm">
          <span className="text-slate-600">{t("cp.field.code")}</span>
          <input
            // A numeric keypad on a telephone, and no autofill: a one-time code
            // is not something a password manager should be filling from a
            // saved value.
            inputMode="numeric"
            autoComplete="one-time-code"
            pattern="[0-9]*"
            maxLength={6}
            required
            value={code}
            onChange={(event) => setCode(event.target.value.replace(/\D/g, ""))}
            className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 tracking-[0.4em] font-mono focus:outline-none focus:ring-2 focus:ring-slate-900/10"
          />
        </label>

        <button
          type="submit"
          disabled={busy}
          className="w-full rounded-lg bg-slate-900 text-white py-2.5 font-medium hover:bg-slate-800 disabled:opacity-60 transition"
        >
          {t("cp.action.sign_in")}
        </button>
      </form>
    </div>
  );
}
