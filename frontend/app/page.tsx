"use client";

import Link from "next/link";
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
            <a href="#features" className="hover:text-indigo-400 transition">Боломжууд</a>
            <a href="#architecture" className="hover:text-indigo-400 transition">Архитектур</a>
            <a href="#modules" className="hover:text-indigo-400 transition">Модулиуд</a>
            <a href="#sso" className="hover:text-indigo-400 transition">OIDC SSO & ДАН</a>
            <a href="https://github.com/gerege-systems/open-gerege-mn-erp" target="_blank" rel="noreferrer" className="hover:text-white transition flex items-center space-x-1">
              <Globe className="w-4 h-4" />
              <span>GitHub</span>
            </a>
          </nav>

          <div className="flex items-center space-x-4">
            <Link
              href="/login"
              className="px-5 py-2.5 text-sm font-semibold bg-gradient-to-r from-indigo-600 to-cyan-600 hover:from-indigo-500 hover:to-cyan-500 text-white rounded-xl shadow-lg shadow-indigo-500/25 transition transform hover:-translate-y-0.5"
            >
              Платформ руу нэвтрэх
            </Link>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative z-10 pt-24 pb-20 px-6 max-w-7xl mx-auto text-center space-y-8">
        <div className="inline-flex items-center space-x-2 px-4 py-2 rounded-full bg-slate-900 border border-indigo-500/30 text-indigo-300 text-xs font-semibold backdrop-blur-md">
          <Sparkles className="w-4 h-4 text-cyan-400 animate-pulse" />
          <span>AI Native & National Digital Identity Ready</span>
        </div>

        <h1 className="text-5xl md:text-7xl font-black tracking-tight text-white max-w-4xl mx-auto leading-tight">
          Монгол Улсын Цахим Дэд Бүтэцтэй Нягт Холбогдох{" "}
          <span className="bg-clip-text text-transparent bg-gradient-to-r from-indigo-400 via-cyan-300 to-emerald-400">
            Modular Monolith ERP Platform
          </span>
        </h1>

        <p className="text-lg md:text-xl text-slate-400 max-w-3xl mx-auto font-light leading-relaxed">
          Odoo болон cloud-native экосистемээс санаа авсан, Go 1.25, Next.js 15, ДАН / E-ID, ХУР Төрийн мэдээлэл солилцоо болон ORY Hydra grade SSO Provider агуулсан нээлттэй эх бүхий бизнес платформ.
        </p>

        {/* Demo Credentials Banner */}
        <div className="max-w-xl mx-auto p-4 bg-slate-900/90 border border-slate-800 rounded-2xl shadow-xl flex items-center justify-between text-left text-xs font-mono">
          <div>
            <span className="text-slate-400 block mb-0.5">Демо Нэвтрэх Эрх (Demo Account):</span>
            <span className="text-indigo-300 font-bold">admin@example.com</span> / <span className="text-emerald-400">Password123!</span>
          </div>
          <Link
            href="/login"
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg font-sans font-semibold transition"
          >
            Нэвтрэх →
          </Link>
        </div>
      </section>

      {/* Verified quality band — every figure below is enforced by CI */}
      <section className="relative z-10 pb-16 px-6 max-w-5xl mx-auto">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { value: "6", label: "Бэлэн бизнес модуль", accent: "text-indigo-300" },
            { value: "0", label: "Lint & vet анхааруулга", accent: "text-emerald-300" },
            { value: "0", label: "Мэдэгдэж буй эмзэг байдал", accent: "text-cyan-300" },
            { value: "100%", label: "Race detector-тэй тест", accent: "text-amber-300" },
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
          Эдгээр үзүүлэлтийг push бүр дээр CI (golangci-lint · go vet · go test -race · govulncheck · gosec) шалгана.
        </p>
      </section>

      {/* Feature Grid Section */}
      <section id="features" className="relative z-10 py-20 bg-slate-900/50 border-y border-slate-800/80">
        <div className="max-w-7xl mx-auto px-6 space-y-16">
          <div className="text-center space-y-4 max-w-2xl mx-auto">
            <h2 className="text-3xl md:text-4xl font-bold text-white">Платформын Үндсэн Давуу Талууд</h2>
            <p className="text-slate-400 text-sm">
              Байгууллагын өдөр тутмын үйл ажиллагаа, аюулгүй байдал, өндөр бүтээмжийг нэг дороос хангах цогц систем.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {/* Card 1 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-indigo-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-indigo-500/10 rounded-xl flex items-center justify-center text-indigo-400 group-hover:scale-110 transition transform">
                <Cpu className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">Modular Monolith Engine</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Go хэл дээр компиллогдох Модулиар Монолит архитектур. Сүлжээний хоцрогдолгүй (zero-latency execution), тенант бүрийн Апп Стор тохиргоо ба DAG хамаарал шийдвэрлэгч.
              </p>
            </div>

            {/* Card 2 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-cyan-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-cyan-500/10 rounded-xl flex items-center justify-center text-cyan-400 group-hover:scale-110 transition transform">
                <Shield className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">Cloud-Native Resilience Engine</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                go-zero сангаас санаа авсан Adaptive Circuit Breaker, Load Shedder, Singleflight coalescing ба Exponential Backoff Retry системүүд.
              </p>
            </div>

            {/* Card 3 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-emerald-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-emerald-500/10 rounded-xl flex items-center justify-center text-emerald-400 group-hover:scale-110 transition transform">
                <Lock className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">E-ID & ДАН SSO Танилт Нэвтрэлт</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Төрийн ДАН ба E-ID системтэй холбогдон Тоон гарын үсэг, Mobile OTP, Банкны SSO болон Царай танилтаар баталгаажуулах интеграци.
              </p>
            </div>

            {/* Card 4 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-purple-500/10 rounded-xl flex items-center justify-center text-purple-400 group-hover:scale-110 transition transform">
                <Code2 className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">ORY Hydra SSO Identity Provider</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Өөрийн бие даасан OAuth2 ба OpenID Connect (OIDC) SSO Provider engine (`/.well-known/openid-configuration`) ба Developer Portal.
              </p>
            </div>

            {/* Card 5 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-amber-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-amber-500/10 rounded-xl flex items-center justify-center text-amber-400 group-hover:scale-110 transition transform">
                <Bot className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">Gemini AI Copilot & Forecaster</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Байгууллагын өгөгдлийн сантай холбогдсон Gemini AI туслах болон агуулахын аюулгүйн үлдэгдэл, захиалга таамаглах систем.
              </p>
            </div>

            {/* Card 6 */}
            <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 hover:border-pink-500/50 transition group space-y-4">
              <div className="w-12 h-12 bg-pink-500/10 rounded-xl flex items-center justify-center text-pink-400 group-hover:scale-110 transition transform">
                <Activity className="w-6 h-6" />
              </div>
              <h3 className="text-xl font-bold text-white">ХУР Мэдээлэл Солилцооны Систем</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Төрийн ХУР системээр иргэний бүртгэл (`WS100101`) ба Хуулийн этгээд/ААН (`WS100201`) автоматаар баталгаажуулан бөглөх модуль.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Production Business Apps Suite */}
      <section id="modules" className="relative z-10 py-20 max-w-7xl mx-auto px-6 space-y-16">
        <div className="text-center space-y-4 max-w-2xl mx-auto">
          <h2 className="text-3xl md:text-4xl font-bold text-white">Бэлэн Бизнес Аппликейшнүүд</h2>
          <p className="text-slate-400 text-sm">
            Апп Стороос тенант бүрээр идэвхжүүлэн ашиглах боломжтой Go бизнес модулиуд.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Users className="w-6 h-6 text-indigo-400" />
              <h3 className="font-bold text-white text-base">Contacts Module</h3>
            </div>
            <p className="text-xs text-slate-400">Харилцагч, бэлтгэн нийлүүлэгчдийн бүртгэл + ХУР төрийн системээс авто-бөглөлт.</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Boxes className="w-6 h-6 text-cyan-400" />
              <h3 className="font-bold text-white text-base">Products & Inventory</h3>
            </div>
            <p className="text-xs text-slate-400">Барааны бүртгэл, SKU, агуулахын хөдөлгөөний лог болон AI үлдэгдлийн таамаглал.</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <CreditCard className="w-6 h-6 text-emerald-400" />
              <h3 className="font-bold text-white text-base">Public Billing & e-Barimt</h3>
            </div>
            <p className="text-xs text-slate-400">Нийтийн нэхэмжлэх, 10% НӨАТ ба e-Barimt татварын баримт хэвлэх модуль.</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <FileText className="w-6 h-6 text-purple-400" />
              <h3 className="font-bold text-white text-base">Digital Documents & E-Sign</h3>
            </div>
            <p className="text-xs text-slate-400">Цахим баримт бичиг, батлах workflow болон E-ID/ДАН тоон гарын үсэг.</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Code2 className="w-6 h-6 text-pink-400" />
              <h3 className="font-bold text-white text-base">Developer Portal & OAuth2</h3>
            </div>
            <p className="text-xs text-slate-400">Гуравдагч системд зориулсан OAuth2 Client App бүртгэл, Secret ба Redirect URI тохиргоо.</p>
          </div>

          <div className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-3">
            <div className="flex items-center space-x-3">
              <Globe className="w-6 h-6 text-amber-400" />
              <h3 className="font-bold text-white text-base">Integrations & Webhooks</h3>
            </div>
            <p className="text-xs text-slate-400">HMAC-SHA256 гарын үсэгтэй асинхрон Webhook ба гадаад систем холбох Connector Manager.</p>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-slate-800 bg-slate-950 py-12 text-slate-500 text-xs">
        <div className="max-w-7xl mx-auto px-6 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center space-x-2">
            <Building2 className="w-4 h-4 text-indigo-500" />
            <span className="font-semibold text-slate-300">Gerege Template Platform</span>
            <span>— Distributed under Apache 2.0 License</span>
          </div>

          <div className="flex items-center space-x-6">
            <span>Copyright © 2026 Gerege Systems Development Team, Gemini AI & Claude AI</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
