"use client";
import { useCallback, useEffect, useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import EIDLogin from "@/components/EIDLogin";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import { api, API_BASE } from "@/lib/api";
import { resetAccess } from "@/lib/access";
import { useI18n } from "@/lib/i18n";
import { useTheme, type ColorMode } from "@/lib/theme";
import { invokeShell, SHELL_METHODS, useShell } from "@/lib/shell";
import {
  Lock,
  Mail,
  ShieldCheck,
  LayoutGrid,
  PenTool,
  Star,
  Settings,
  Sun,
  Moon,
  Laptop,
  Server,
  X,
  AlertTriangle,
} from "lucide-react";

/// Health хаяг нь API-гийн үндэс дээр сууна (`/api/v1` биш) — native
/// бүрхүүлүүд мөн үүнийг татдаг тул бүх клиент ижил үнэнийг харна.
function healthUrl(): string {
  try {
    return new URL(API_BASE, window.location.href).origin + "/health";
  } catch {
    return "/health";
  }
}

type LinkState = "checking" | "online" | "offline";

const LINK_BADGE: Record<LinkState, { label: string; chip: string; dot: string }> = {
  checking: {
    label: "Шалгаж байна...",
    chip: "bg-slate-500/10 text-slate-400 border-slate-500/20",
    dot: "bg-slate-400 animate-pulse",
  },
  online: {
    label: "Холбогдсон",
    chip: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
    dot: "bg-emerald-500",
  },
  offline: {
    label: "Холбогдоогүй",
    chip: "bg-rose-500/10 text-rose-400 border-rose-500/20",
    dot: "bg-rose-500",
  },
};

/** Тохиргооны цонх нь гэрлэн хэлбэртэй тул өөрийн өнгөний багцтай. */
const MODAL_LINK: Record<LinkState, { box: string; text: string }> = {
  checking: {
    box: "bg-slate-50 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700",
    text: "text-slate-600 dark:text-slate-300",
  },
  online: {
    box: "bg-emerald-50 dark:bg-emerald-950/30 border-emerald-200 dark:border-emerald-900/50",
    text: "text-emerald-700 dark:text-emerald-400",
  },
  offline: {
    box: "bg-rose-50 dark:bg-rose-950/30 border-rose-200 dark:border-rose-900/50",
    text: "text-rose-700 dark:text-rose-400",
  },
};

const THEME_MODES: Array<{ mode: ColorMode; label: string; Icon: typeof Sun }> = [
  { mode: "light", label: "Гэгээн", Icon: Sun },
  { mode: "dark", label: "Бараан", Icon: Moon },
  { mode: "system", label: "Систем", Icon: Laptop },
];

function LoginContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useI18n();
  const theme = useTheme();
  const { inShell } = useShell();
  const [next, setNext] = useState("/apps");
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [linkState, setLinkState] = useState<LinkState>("checking");

  useEffect(() => {
    const requested = searchParams.get("next");
    if (requested?.startsWith("/") && !requested.startsWith("//")) setNext(requested);
    if (searchParams.has("settings")) setShowSettingsModal(true);
  }, [searchParams]);

  // Native клиент дээр web login огт харуулахгүй. Session тасарсан эсвэл
  // `/login` руу шууд орсон аль ч тохиолдолд бүрхүүлийн native login-ыг нээнэ.
  useEffect(() => {
    if (inShell) void invokeShell(SHELL_METHODS.AUTH_RE_LOGIN, { reason: "web-login-route" });
  }, [inShell]);

  // Статусын индикатор нь бодит хариунаас л асна: "Холбогдсон" гэж бичээд
  // сервер унасан байх нь тасалдал оношилж буй хүнийг төөрөгдүүлнэ.
  const probe = useCallback(async () => {
    try {
      const res = await fetch(healthUrl(), { cache: "no-store", credentials: "omit" });
      setLinkState(res.ok ? "online" : "offline");
    } catch {
      setLinkState("offline");
    }
  }, []);

  useEffect(() => {
    void probe();
    const timer = window.setInterval(() => void probe(), 15_000);
    return () => window.clearInterval(timer);
  }, [probe]);

  const [authMode, setAuthMode] = useState<"eid" | "token" | "admin">("eid");
  const [tokenPin, setTokenPin] = useState("");

  async function passwordLogin(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api.login(email, password);
      resetAccess();
      router.push(next);
    } catch (err: any) {
      setError(err.message || t("auth.message.error_password"));
    }
  }

  if (inShell) {
    return <main className="min-h-screen bg-slate-950" aria-label="Native login нээгдэж байна" />;
  }

  return (
    <div className="w-screen h-screen flex flex-col bg-slate-950 text-slate-100 select-none overflow-hidden font-sans relative">
      {/* Ambient Radial Glowing Lights */}
      <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-blue-600/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-indigo-600/10 rounded-full blur-3xl pointer-events-none" />

      {/* HEADER: Logo, Version, Language, Network Status, Settings Button */}
      <header className="h-14 shrink-0 px-6 border-b border-slate-800/80 bg-slate-900/90 backdrop-blur-md flex items-center justify-between z-30">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-blue-600 to-blue-500 flex items-center justify-center shadow-md shadow-blue-500/20 text-white">
            <LayoutGrid className="w-5 h-5 text-white" />
          </div>
          <div className="flex items-center gap-2">
            <span className="font-bold text-base tracking-tight text-white">Gerege Nexus</span>
            <span className="text-slate-500 font-light">/</span>
            <span className="text-xs font-semibold px-2.5 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
              v1.0 Enterprise Native Workstation
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Live Network / HSM Status */}
          <div className={`hidden sm:flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium border ${LINK_BADGE[linkState].chip}`}>
            <span className={`w-1.5 h-1.5 rounded-full ${LINK_BADGE[linkState].dot}`} />
            <span>API: {LINK_BADGE[linkState].label}</span>
          </div>

          {/* Language Switcher */}
          <LanguageSwitcher variant="dark" />

          {/* Pre-login Quick Settings Icon Button */}
          <button
            type="button"
            onClick={() => setShowSettingsModal(true)}
            title="Системийн ба Серверийн Тохиргоо"
            className="p-2 rounded-xl border border-slate-700/80 bg-slate-800/80 hover:bg-slate-700 text-slate-200 hover:text-white transition shadow-sm flex items-center gap-1.5 text-xs font-semibold cursor-pointer"
          >
            <Settings className="w-4 h-4 text-blue-400" />
            <span className="hidden sm:inline">Тохиргоо</span>
          </button>
        </div>
      </header>

      {/* MAIN: Centered Native Workstation App Window Frame */}
      <main className="flex-1 flex items-center justify-center p-4 sm:p-6 overflow-y-auto z-20">
        <div className="w-full max-w-xl bg-slate-900/80 border border-slate-800/90 rounded-2xl shadow-2xl backdrop-blur-2xl overflow-hidden flex flex-col">

          {/* Native Window Titlebar */}
          <div className="h-11 px-4 bg-slate-900 border-b border-slate-800/80 flex items-center justify-between">
            {/* macOS Traffic Light Buttons */}
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-rose-500/80 inline-block border border-rose-600/50" />
              <span className="w-3 h-3 rounded-full bg-amber-500/80 inline-block border border-amber-600/50" />
              <span className="w-3 h-3 rounded-full bg-emerald-500/80 inline-block border border-emerald-600/50" />
            </div>

            <div className="text-xs font-bold text-slate-300 tracking-wide flex items-center gap-2">
              <ShieldCheck className="w-3.5 h-3.5 text-blue-400" />
              <span>Gerege Nexus Native Workstation Auth</span>
            </div>

            <div className="text-[11px] font-mono text-slate-500">
              PKI: eID Ready
            </div>
          </div>

          {/* Native Segmented Auth Navigation Bar */}
          <div className="p-3 bg-slate-950/60 border-b border-slate-800/60">
            <div className="grid grid-cols-3 gap-1.5 p-1 rounded-xl bg-slate-900 border border-slate-800/80">
              <button
                type="button"
                onClick={() => setAuthMode("eid")}
                className={`py-2 px-3 rounded-lg font-bold text-xs transition flex items-center justify-center gap-1.5 cursor-pointer ${
                  authMode === "eid"
                    ? "bg-blue-600 text-white shadow-md shadow-blue-500/25"
                    : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
                }`}
              >
                <Star className="w-3.5 h-3.5" />
                <span>eID Апп</span>
              </button>

              <button
                type="button"
                onClick={() => setAuthMode("token")}
                className={`py-2 px-3 rounded-lg font-bold text-xs transition flex items-center justify-center gap-1.5 cursor-pointer ${
                  authMode === "token"
                    ? "bg-blue-600 text-white shadow-md shadow-blue-500/25"
                    : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
                }`}
              >
                <PenTool className="w-3.5 h-3.5" />
                <span>USB Токен</span>
              </button>

              <button
                type="button"
                onClick={() => setAuthMode("admin")}
                className={`py-2 px-3 rounded-lg font-bold text-xs transition flex items-center justify-center gap-1.5 cursor-pointer ${
                  authMode === "admin"
                    ? "bg-blue-600 text-white shadow-md shadow-blue-500/25"
                    : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
                }`}
              >
                <Lock className="w-3.5 h-3.5" />
                <span>Админ Нэвтрэх</span>
              </button>
            </div>
          </div>

          {/* Native Workstation Content Body */}
          <div className="p-6">
            {authMode === "eid" && (
              <div className="space-y-4 eid-on-dark">
                <EIDLogin next={next} />
              </div>
            )}

            {/* USB токены PKCS#11 урсгал сервер талдаа хараахан байхгүй. Уншигч
                байгаа эсэх, сертификатын эзэн, хүчинтэй хугацаа — эдгээрийг
                зөвхөн бодит драйвер хэлж чадна. Тиймээс энд зохиомол утга
                харуулахгүй, форм нь идэвхгүй байна. */}
            {authMode === "token" && (
              <div className="space-y-5">
                <div className="p-4 rounded-xl bg-slate-950/80 border border-slate-800 space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                      <div className="w-9 h-9 rounded-lg bg-slate-500/10 text-slate-400 flex items-center justify-center">
                        <PenTool className="w-5 h-5" />
                      </div>
                      <div>
                        <div className="text-xs font-bold text-white">SmartCard USB Token</div>
                        <div className="text-[11px] text-slate-400">PKCS#11 уншигч</div>
                      </div>
                    </div>
                    <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-500/10 text-slate-400 border border-slate-500/20">
                      Илрээгүй
                    </span>
                  </div>

                  <div className="text-xs text-slate-400 bg-slate-900/90 p-3 rounded-lg border border-slate-800 font-mono">
                    <div className="text-slate-500 text-[10px] uppercase font-bold mb-1">Сертификат:</div>
                    <div>—</div>
                  </div>
                </div>

                <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-xs text-amber-300 flex items-start gap-2">
                  <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5 text-amber-400" />
                  <span>
                    Энэ орчинд USB токены баталгаажуулалт тохируулагдаагүй байна. eID
                    Апп эсвэл админ нэвтрэлтийг ашиглана уу.
                  </span>
                </div>

                <div className="space-y-1.5 opacity-50">
                  <label className="block text-xs font-bold text-slate-300">
                    Токены PIN Нууц үг
                  </label>
                  <div className="flex items-center gap-2 px-3 py-2.5 rounded-xl border border-slate-800 bg-slate-950/90">
                    <Lock className="w-4 h-4 text-slate-400" />
                    <input
                      type="password"
                      value={tokenPin}
                      onChange={(e) => setTokenPin(e.target.value)}
                      placeholder="Токены PIN оруулна уу"
                      className="w-full text-xs outline-none bg-transparent text-white placeholder-slate-500"
                      disabled
                    />
                  </div>
                </div>

                <button
                  type="button"
                  disabled
                  className="w-full py-3 rounded-xl bg-slate-800 text-slate-500 font-bold text-xs flex items-center justify-center gap-2 cursor-not-allowed"
                >
                  <PenTool className="w-4 h-4" />
                  <span>Токеноор нэвтрэх</span>
                </button>
              </div>
            )}

            {authMode === "admin" && (
              <form onSubmit={passwordLogin} className="space-y-4">
                <div className="p-3 rounded-xl bg-blue-500/10 border border-blue-500/20 text-xs text-blue-300 flex items-center gap-2">
                  <Lock className="w-4 h-4 shrink-0 text-blue-400" />
                  <span>Админ системд нэвтрэх хэрэглэгчийн нэр, нууц үгээ оруулна уу.</span>
                </div>

                {error && <p className="text-xs text-rose-400 font-medium">{error}</p>}

                <div className="space-y-1.5">
                  <label className="block text-xs font-bold text-slate-300">
                    И-мэйл хаяг
                  </label>
                  <div className="flex items-center gap-2 px-3 py-2.5 rounded-xl border border-slate-800 bg-slate-950/90 focus-within:border-blue-500 transition">
                    <Mail className="w-4 h-4 text-slate-400" />
                    <input
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      autoComplete="username"
                      placeholder="name@example.com"
                      className="w-full text-xs outline-none bg-transparent text-white placeholder-slate-600"
                      required
                    />
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="block text-xs font-bold text-slate-300">
                    Нууц үг
                  </label>
                  <div className="flex items-center gap-2 px-3 py-2.5 rounded-xl border border-slate-800 bg-slate-950/90 focus-within:border-blue-500 transition">
                    <Lock className="w-4 h-4 text-slate-400" />
                    <input
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      autoComplete="current-password"
                      placeholder="••••••••"
                      className="w-full text-xs outline-none bg-transparent text-white placeholder-slate-600"
                      required
                    />
                  </div>
                </div>

                <button
                  type="submit"
                  className="w-full py-3 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs transition shadow-lg shadow-blue-500/25 flex items-center justify-center gap-2 cursor-pointer"
                >
                  <Lock className="w-4 h-4" />
                  <span>{t("auth.action.admin_sign_in")}</span>
                </button>
              </form>
            )}
          </div>
        </div>
      </main>

      {/* FOOTER: System Info, Security Status, Copyright & Endpoint Info */}
      <footer className="h-9 shrink-0 px-6 border-t border-slate-800/80 bg-slate-900/90 backdrop-blur-md flex items-center justify-between text-[11px] text-slate-400 z-30">
        <div className="flex items-center gap-3">
          <span className="font-semibold text-slate-300">© 2026 Gerege Systems LLC. All rights reserved.</span>
          <span className="text-slate-700">•</span>
          <span>Gerege Nexus Shell v1.0 Enterprise</span>
        </div>

        <div className="flex items-center gap-4">
          <span className="hidden md:flex items-center gap-1.5 text-slate-400">
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
            <span>SSL 256-bit HSM Encrypted</span>
          </span>
          <span className="hidden lg:flex items-center gap-1 text-slate-400">
            <Server className="w-3 h-3 text-blue-400" />
            <span>API Endpoint: {API_BASE}</span>
          </span>
          <span className="font-mono text-[10px] text-slate-400 bg-slate-800/80 px-2 py-0.5 rounded border border-slate-700/60">
            MN · UTF-8 · eID Active
          </span>
        </div>
      </footer>

      {/* Pre-login Settings Popup Modal Overlay */}
      {showSettingsModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-md animate-in fade-in duration-200">
          <div className="w-full max-w-md rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl p-5 space-y-4 text-slate-800 dark:text-slate-100">
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-xl bg-blue-50 dark:bg-blue-950/50 text-blue-600 dark:text-blue-400">
                  <Settings className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="font-bold text-sm">Системийн ба Серверийн Тохиргоо</h3>
                  <p className="text-[11px] text-slate-500">Нэвтрэхийн өмнөх орчин ба сүлжээний тохиргоо</p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setShowSettingsModal(false)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Modal Body Content */}
            <div className="space-y-4 text-xs">
              {/* Server Endpoint — уншихад зориулагдсан. Хаягийг build-ийн
                  NEXT_PUBLIC_API_URL эсвэл бүрхүүлийн Тохиргоо цонх эзэмшдэг
                  тул энд солих товч тавьбал хуурамч мэдрэмж төрүүлнэ. */}
              <div>
                <label className="text-[11px] font-bold text-slate-500 uppercase tracking-wider mb-1.5 flex items-center gap-1.5">
                  <Server className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" />
                  <span>Сервер Холболт (API Endpoint)</span>
                </label>
                <p className="px-3 py-2.5 rounded-xl bg-slate-100 dark:bg-slate-800/80 font-mono text-[11px] break-all text-slate-700 dark:text-slate-300">
                  {API_BASE}
                </p>
                <p className="mt-1.5 text-[11px] text-slate-500">
                  {inShell
                    ? "Хаягийг солихдоо бүрхүүлийн «Тохиргоо…» цэсийг ашиглана уу."
                    : "Хаяг нь NEXT_PUBLIC_API_URL орчны хувьсагчаар тодорхойлогддог."}
                </p>
              </div>

              {/* Theme Mode — ThemeProvider-ийн хадгалдаг өгөгдөл рүү шууд
                  бичнэ; өөр түлхүүр рүү бичвэл нэвтэрмэгц дарагдана. */}
              <div>
                <label className="block text-[11px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">
                  Харагдац / Theme
                </label>
                <div className="grid grid-cols-3 gap-1.5 p-1 rounded-xl bg-slate-100 dark:bg-slate-800/80">
                  {THEME_MODES.map(({ mode, label, Icon }) => (
                    <button
                      key={mode}
                      type="button"
                      onClick={() => theme.updateTheme({ mode })}
                      className={`py-2 px-2 rounded-lg font-semibold text-xs flex items-center justify-center gap-1.5 transition cursor-pointer ${theme.mode === mode ? "bg-white dark:bg-slate-900 text-blue-600 dark:text-blue-400 shadow-sm border border-slate-200 dark:border-slate-700" : "text-slate-600 dark:text-slate-400"}`}
                    >
                      <Icon className="w-3.5 h-3.5" />
                      <span>{label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Network Status */}
              <div className={`p-3 rounded-xl border flex items-center justify-between text-xs ${MODAL_LINK[linkState].box}`}>
                <div className={`flex items-center gap-2 font-medium ${MODAL_LINK[linkState].text}`}>
                  <span className={`w-2.5 h-2.5 rounded-full ${LINK_BADGE[linkState].dot}`} />
                  <span>API Сервер</span>
                </div>
                <span className={`font-bold ${MODAL_LINK[linkState].text}`}>{LINK_BADGE[linkState].label}</span>
              </div>
            </div>

            {/* Modal Footer */}
            <div className="pt-2 flex gap-2">
              <button
                type="button"
                onClick={() => {
                  setLinkState("checking");
                  void probe();
                }}
                className="flex-1 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-200 font-bold text-xs transition hover:bg-slate-50 dark:hover:bg-slate-800 cursor-pointer"
              >
                Холболт шалгах
              </button>
              <button
                type="button"
                onClick={() => setShowSettingsModal(false)}
                className="flex-1 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white font-bold text-xs transition shadow-lg shadow-blue-500/25 cursor-pointer"
              >
                Хаах
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-slate-950 flex items-center justify-center text-white text-xs">Ачаалж байна...</div>}>
      <LoginContent />
    </Suspense>
  );
}
