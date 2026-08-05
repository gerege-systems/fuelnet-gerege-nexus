"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import AICopilot from "@/components/AICopilot";
import {
  LayoutGrid,
  Settings,
  Users,
  Package,
  Boxes,
  LogOut,
  Building2,
  UserCheck,
  Share2,
  CreditCard,
  FileText,
  Code2,
} from "lucide-react";

interface Menu {
  id: string;
  label: string;
  path: string;
  icon: string;
  order: number;
}

const iconMap: Record<string, React.ReactNode> = {
  users: <Users className="w-5 h-5" />,
  package: <Package className="w-5 h-5" />,
  boxes: <Boxes className="w-5 h-5" />,
  "credit-card": <CreditCard className="w-5 h-5" />,
  "file-text": <FileText className="w-5 h-5" />,
};

// Routes that render without the authenticated app shell. The landing page
// used to be gated like every other route: Layout called getMe(), the call
// failed for an anonymous visitor and pushed them straight to /login, so the
// landing page was unreachable.
const PUBLIC_ROUTES = ["/", "/login"];

export default function Layout({ children }: { children: React.ReactNode }) {
  const [menus, setMenus] = useState<Menu[]>([]);
  const [user, setUser] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const pathname = usePathname();
  const router = useRouter();
  const { t } = useI18n();

  const isPublic = PUBLIC_ROUTES.includes(pathname);

  useEffect(() => {
    if (isPublic) {
      setLoading(false);
      return;
    }

    async function loadData() {
      try {
        const u = await api.getMe();
        setUser(u);
        const m = await api.getMenus();
        setMenus(m || []);
      } catch (err) {
        router.push("/login");
      } finally {
        setLoading(false);
      }
    }
    loadData();
  }, [pathname, router, isPublic]);

  const handleLogout = async () => {
    try {
      await api.logout();
    } catch {}
    localStorage.removeItem("session_token");
    router.push("/login");
  };

  if (pathname === "/login") {
    return <main className="min-h-screen bg-slate-100 flex items-center justify-center">{children}</main>;
  }

  // The landing page brings its own full-page chrome.
  if (pathname === "/") {
    return <>{children}</>;
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 text-slate-500 font-medium">
        {t("shell.loadingPlatform")}
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-slate-50">
      {/* Top Navbar */}
      <header className="h-14 bg-slate-900 text-white flex items-center justify-between px-4 border-b border-slate-800 sticky top-0 z-50">
        <div className="flex items-center space-x-3">
          <Link href="/apps" className="flex items-center space-x-2 font-bold text-base text-indigo-400 hover:text-indigo-300">
            <Building2 className="w-6 h-6 text-indigo-500" />
            <span>Gerege Template Platform</span>
          </Link>
          <span className="bg-slate-800 text-slate-300 text-xs font-semibold px-2.5 py-1 rounded-md flex items-center space-x-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
            <span>{user?.tenant_name || "Demo Tenant"}</span>
          </span>
        </div>

        <div className="flex items-center space-x-4">
          <LanguageSwitcher variant="dark" />
          <div className="flex items-center space-x-2 text-sm text-slate-300">
            <UserCheck className="w-4 h-4 text-emerald-400" />
            <span className="font-medium">{user?.name}</span>
            <span className="text-slate-500">({user?.email})</span>
          </div>
          <button
            onClick={handleLogout}
            className="flex items-center space-x-1 text-xs bg-slate-800 hover:bg-red-900/50 text-slate-300 hover:text-red-300 px-3 py-1.5 rounded-md border border-slate-700 transition"
          >
            <LogOut className="w-3.5 h-3.5" />
            <span>{t("shell.logout")}</span>
          </button>
        </div>
      </header>

      <div className="flex flex-1">
        {/* Sidebar */}
        <aside className="w-60 bg-white border-r border-slate-200 flex flex-col py-4 justify-between">
          <div className="space-y-6">
            <div>
              <div className="px-4 text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                {t("shell.modules")}
              </div>
              <nav className="space-y-1 px-2">
                <Link
                  href="/apps"
                  className={`flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-md transition ${
                    pathname === "/apps"
                      ? "bg-indigo-50 text-indigo-600 font-semibold"
                      : "text-slate-700 hover:bg-slate-100"
                  }`}
                >
                  <LayoutGrid className="w-5 h-5 text-indigo-500" />
                  <span>{t("shell.appStore")}</span>
                </Link>

                {menus.map((m) => (
                  <Link
                    key={m.id}
                    href={m.path}
                    className={`flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-md transition ${
                      pathname.startsWith(m.path)
                        ? "bg-indigo-50 text-indigo-600 font-semibold"
                        : "text-slate-700 hover:bg-slate-100"
                    }`}
                  >
                    {iconMap[m.icon] || <Package className="w-5 h-5" />}
                    <span>{m.label}</span>
                  </Link>
                ))}
              </nav>
            </div>

            <div>
              <div className="px-4 text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                {t("shell.settings")}
              </div>
              <nav className="space-y-1 px-2">
                <Link
                  href="/settings/apps"
                  className={`flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-md transition ${
                    pathname === "/settings/apps"
                      ? "bg-indigo-50 text-indigo-600 font-semibold"
                      : "text-slate-700 hover:bg-slate-100"
                  }`}
                >
                  <Settings className="w-5 h-5 text-slate-500" />
                  <span>{t("shell.installedApps")}</span>
                </Link>

                <Link
                  href="/settings/integrations"
                  className={`flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-md transition ${
                    pathname === "/settings/integrations"
                      ? "bg-indigo-50 text-indigo-600 font-semibold"
                      : "text-slate-700 hover:bg-slate-100"
                  }`}
                >
                  <Share2 className="w-5 h-5 text-slate-500" />
                  <span>{t("shell.integrations")}</span>
                </Link>

                <Link
                  href="/developer/apps"
                  className={`flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-md transition ${
                    pathname === "/developer/apps"
                      ? "bg-indigo-50 text-indigo-600 font-semibold"
                      : "text-slate-700 hover:bg-slate-100"
                  }`}
                >
                  <Code2 className="w-5 h-5 text-slate-500" />
                  <span>{t("shell.developerApps")}</span>
                </Link>
              </nav>
            </div>
          </div>

          <div className="px-4 text-[11px] text-slate-400 border-t border-slate-100 pt-3">
            Gerege Template Platform
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 p-6 overflow-y-auto">{children}</main>
      </div>
      <AICopilot />
    </div>
  );
}
