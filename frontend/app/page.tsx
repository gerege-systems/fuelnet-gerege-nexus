"use client";

import Link from "next/link";
import { useI18n } from "@/lib/i18n";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import {
  Building2,
  Shield,
  Zap,
  Cpu,
  Database,
  Code2,
  Layers,
  ArrowRight,
  CheckCircle2,
  Sparkles,
  Boxes,
  Lock,
  BarChart3,
  FileText,
  CreditCard,
  Users,
  Globe,
  Bot,
  Activity,
  Terminal,
} from "lucide-react";

export default function LandingPage() {
  const { t } = useI18n();
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 font-sans selection:bg-indigo-500 selection:text-white">
      {/* Dynamic Background Mesh */}
      <div className="fixed inset-0 z-0 opacity-30 pointer-events-none">
        <div className="absolute top-0 left-1/4 w-[600px] h-[600px] bg-indigo-600/30 rounded-full blur-[140px]" />
        <div className="absolute bottom-0 right-1/4 w-[600px] h-[600px] bg-cyan-500/20 rounded-full blur-[140px]" />
      </div>

      {/* Navigation Header */}
      <header className="relative z-10 border-b border-slate-800/80 backdrop-blur-xl bg-slate-950/60 sticky top-0">
        <div className="max-w-7xl mx-auto px-6 h-20 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="p-2.5 bg-gradient-to-tr from-indigo-600 to-cyan-500 rounded-xl shadow-lg shadow-indigo-500/20">
              <Building2 className="w-6 h-6 text-white" />
            </div>
            <div>
              <span className="font-extrabold text-xl tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-white via-slate-200 to-indigo-300">
                Gerege Template Platform
              </span>
              <span className="ml-2 px-2 py-0.5 text-[10px] font-bold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 rounded-full inline-flex items-center gap-1">
                {/* Flaticon asset — see docs/assets/icons/ATTRIBUTION.md */}
                <img src="/icons/flag-mn.png" alt="" width={12} height={12} className="inline-block" />
                Open Source
              </span>
            </div>
          </div>

          <nav className="hidden md:flex items-center space-x-8 text-sm font-medium text-slate-300">
            <a href="#features" className="hover:text-indigo-400 transition">{t("landing.nav.features")}</a>
            <a href="#architecture" className="hover:text-indigo-400 transition">{t("landing.nav.architecture")}</a>
            <a href="#modules" className="hover:text-indigo-400 transition">{t("landing.nav.modules")}</a>
            <a href="#sso" className="hover:text-indigo-400 transition">{t("landing.nav.sso")}</a>
            <a href="https://github.com/gerege-systems/open-gerege-mn-erp" target="_blank" rel="noreferrer" className="hover:text-white transition flex items-center space-x-1">
              <Globe className="w-4 h-4" />
              <span>GitHub</span>
            </a>
          </nav>

          <div className="flex items-center space-x-4">
            <LanguageSwitcher variant="dark" />
            <Link
              href="/login"
              className="px-5 py-2.5 text-sm font-semibold bg-gradient-to-r from-indigo-600 to-cyan-600 hover:from-indigo-500 hover:to-cyan-500 text-white rounded-xl shadow-lg shadow-indigo-500/25 transition transform hover:-translate-y-0.5"
            >
              {t("landing.cta.signIn")}
            </Link>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative z-10 pt-24 pb-20 px-6 max-w-7xl mx-auto text-center space-y-8">
        <div className="inline-flex items-center space-x-2 px-4 py-2 rounded-full bg-slate-900 border border-indigo-500/30 text-indigo-300 text-xs font-semibold backdrop-blur-md">
          <Sparkles className="w-4 h-4 text-cyan-400 animate-pulse" />
          <span>{t("landing.badge")}</span>
        </div>

        <h1 className="text-5xl md:text-7xl font-black tracking-tight text-white max-w-4xl mx-auto leading-tight">
          {t("landing.hero.lead")}{" "}
          <span className="bg-clip-text text-transparent bg-gradient-to-r from-indigo-400 via-cyan-300 to-emerald-400">
            {t("landing.hero.highlight")}
          </span>
        </h1>

        <p className="text-lg md:text-xl text-slate-400 max-w-3xl mx-auto font-light leading-relaxed">
          {t("landing.hero.body")}
        </p>

        {/* Demo Credentials Banner */}
        <div className="max-w-xl mx-auto p-4 bg-slate-900/90 border border-slate-800 rounded-2xl shadow-xl flex items-center justify-between text-left text-xs font-mono">
          <div>
            <span className="text-slate-400 block mb-0.5">{t("landing.demoBanner")}</span>
            <span className="text-indigo-300 font-bold">admin@example.com</span> / <span className="text-emerald-400">Password123!</span>
          </div>
          <Link
            href="/login"
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg font-sans font-semibold transition"
          >
            {t("landing.cta.enter")}
          </Link>
        </div>
      </section>

      {/* Verified quality band — every figure below is enforced by CI */}
      <section className="relative z-10 pb-16 px-6 max-w-5xl mx-auto">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { value: "6", label: t("landing.stats.modules"), accent: "text-indigo-300" },
            { value: "0", label: t("landing.stats.lint"), accent: "text-emerald-300" },
            { value: "0", label: t("landing.stats.vulns"), accent: "text-cyan-300" },
            { value: "100%", label: t("landing.stats.tests"), accent: "text-amber-300" },
          ].map((stat) => (
            <div
              key={stat.label}
              className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800 text-center space-y-1"
            >
              <div className={`text-3xl font-black ${stat.accent}`}>{stat.value}</div>
              <div className="text-[11px] text-slate-400 leading-snug">{stat.label}</div>
            </div>
          ))}
        </div>
        <p className="mt-4 text-center text-[11px] text-slate-500">
          {t("landing.stats.note")}
        </p>
      </section>

      {/* Feature Grid Section */}
      <section id="features" className="relative z-10 py-20 bg-slate-900/50 border-y border-slate-800/80">
        <div className="max-w-7xl mx-auto px-6 space-y-16">
          <div className="text-center space-y-4 max-w-2xl mx-auto">
            <h2 className="text-3xl md:text-4xl font-bold text-white">{t("landing.features.title")}</h2>
            <p className="text-slate-400 text-sm">
              {t("landing.features.subtitle")}
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Card 1 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-indigo-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-indigo-500/10 rounded-xl flex items-center justify-center text-indigo-400 group-hover:scale-110 transition transform">
                <Cpu className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">{t("landing.feature1.title")}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">{t("landing.feature1.body")}</p>
            </div>

            {/* Card 2 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-cyan-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-cyan-500/10 rounded-xl flex items-center justify-center text-cyan-400 group-hover:scale-110 transition transform">
                <Shield className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">{t("landing.feature2.title")}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">{t("landing.feature2.body")}</p>
            </div>

            {/* Card 3 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-emerald-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-emerald-500/10 rounded-xl flex items-center justify-center text-emerald-400 group-hover:scale-110 transition transform">
                <Lock className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">{t("landing.feature3.title")}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">{t("landing.feature3.body")}</p>
            </div>

            {/* Card 4 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-purple-500/10 rounded-xl flex items-center justify-center text-purple-400 group-hover:scale-110 transition transform">
                <Code2 className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">{t("landing.feature4.title")}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">{t("landing.feature4.body")}</p>
            </div>

            {/* Card 5 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-amber-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-amber-500/10 rounded-xl flex items-center justify-center text-amber-400 group-hover:scale-110 transition transform">
                <Bot className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">{t("landing.feature5.title")}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">{t("landing.feature5.body")}</p>
            </div>

            {/* Card 6 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-pink-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-pink-500/10 rounded-xl flex items-center justify-center text-pink-400 group-hover:scale-110 transition transform">
                <Activity className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">{t("landing.feature6.title")}</h3>
              <p className="text-slate-400 text-sm leading-relaxed">{t("landing.feature6.body")}</p>
            </div>
          </div>
        </div>
      </section>

      {/* Production Business Apps Suite */}
      <section id="modules" className="relative z-10 py-20 max-w-7xl mx-auto px-6 space-y-16">
        <div className="text-center space-y-4 max-w-2xl mx-auto">
          <h2 className="text-3xl md:text-4xl font-bold text-white">{t("landing.modules.title")}</h2>
          <p className="text-slate-400 text-sm">
            {t("landing.modules.subtitle")}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Users className="w-6 h-6 text-indigo-400" />
              <h3 className="font-bold text-white text-base">{t("landing.module1.title")}</h3>
            </div>
            <p className="text-xs text-slate-400">{t("landing.module1.body")}</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Boxes className="w-6 h-6 text-cyan-400" />
              <h3 className="font-bold text-white text-base">{t("landing.module2.title")}</h3>
            </div>
            <p className="text-xs text-slate-400">{t("landing.module2.body")}</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <CreditCard className="w-6 h-6 text-emerald-400" />
              <h3 className="font-bold text-white text-base">{t("landing.module3.title")}</h3>
            </div>
            <p className="text-xs text-slate-400">{t("landing.module3.body")}</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <FileText className="w-6 h-6 text-purple-400" />
              <h3 className="font-bold text-white text-base">{t("landing.module4.title")}</h3>
            </div>
            <p className="text-xs text-slate-400">{t("landing.module4.body")}</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Code2 className="w-6 h-6 text-pink-400" />
              <h3 className="font-bold text-white text-base">{t("landing.module5.title")}</h3>
            </div>
            <p className="text-xs text-slate-400">{t("landing.module5.body")}</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Globe className="w-6 h-6 text-amber-400" />
              <h3 className="font-bold text-white text-base">{t("landing.module6.title")}</h3>
            </div>
            <p className="text-xs text-slate-400">{t("landing.module6.body")}</p>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-slate-800 bg-slate-950 py-12 text-slate-500 text-xs">
        <div className="max-w-7xl mx-auto px-6 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center space-x-2">
            <Building2 className="w-4 h-4 text-indigo-500" />
            <span className="font-semibold text-slate-300">Gerege Template Platform</span>
            <span>{t("landing.footer.license")}</span>
          </div>

          <div className="flex items-center space-x-6">
            <span>Copyright © 2026 Gerege Systems Development Team, Gemini AI & Claude AI</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
