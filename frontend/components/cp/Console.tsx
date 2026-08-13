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
import { ShieldCheck } from "lucide-react";

import { cp, Unauthorized, type Operator } from "@/lib/cp";
import { useI18n } from "@/lib/i18n";

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
      <header className="bg-slate-900 text-slate-100">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 h-14 flex items-center gap-3">
          <Link href="/cp" className="flex items-center gap-2 font-semibold">
            <ShieldCheck className="w-5 h-5 text-amber-400" />
            {t("cp.view.title")}
          </Link>
          <nav className="flex items-center gap-1 text-sm">
            <ConsoleLink href="/cp" label={t("cp.section.tenants")} />
            <ConsoleLink href="/cp/support" label={t("cp.section.support")} />
            <ConsoleLink href="/cp/approvals" label={t("cp.section.approvals")} />
            <ConsoleLink href="/cp/config" label={t("cp.section.config")} />
            <ConsoleLink href="/cp/announcements" label={t("cp.section.announcements")} />
          </nav>
          <div className="flex-1" />
          <span className="text-sm text-slate-300">
            {operator.name} · {t(`cp.role.${operator.role}`)}
          </span>
          <button
            type="button"
            onClick={() => void signOut()}
            className="text-sm rounded-lg px-3 py-1.5 bg-slate-800 hover:bg-slate-700 transition"
          >
            {t("cp.action.sign_out")}
          </button>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8">{children}</main>
    </ConsoleContext.Provider>
  );
}

function ConsoleLink({ href, label }: { href: string; label: string }) {
  return (
    <Link href={href} className="rounded-lg px-3 py-1.5 text-slate-300 hover:text-white hover:bg-slate-800 transition">
      {label}
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
