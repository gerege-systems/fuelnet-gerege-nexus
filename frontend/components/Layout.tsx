"use client";

import React, { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import brandLogo from "@/public/brand.webp";
import { usePathname, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { useTheme } from "@/lib/theme";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import AICopilot from "@/components/AICopilot";
import {
  LayoutGrid,
  Settings,
  Users,
  Package,
  Boxes,
  LogOut,
  Share2,
  CreditCard,
  FileText,
  Code2,
  Menu,
  X,
  ChevronDown,
  Palette,
  Moon,
  Sun,
  Building2,
  Database,
  Workflow,
  Layers3,
  ChevronRight,
} from "lucide-react";

interface Menu {
  id: string;
  parent_id?: string;
  label: string;
  path?: string;
  icon: string;
  order: number;
}

const iconMap: Record<string, React.ReactNode> = {
  users: <Users className="w-5 h-5" />,
  package: <Package className="w-5 h-5" />,
  boxes: <Boxes className="w-5 h-5" />,
  "credit-card": <CreditCard className="w-5 h-5" />,
  "file-text": <FileText className="w-5 h-5" />,
  code: <Code2 className="w-5 h-5" />,
  database: <Database className="w-5 h-5" />,
  workflow: <Workflow className="w-5 h-5" />,
  layers: <Layers3 className="w-5 h-5" />,
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
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [expandedMenus, setExpandedMenus] = useState<Set<string>>(new Set());
  const pathname = usePathname();
  const router = useRouter();
  const { t, locale } = useI18n();
  const theme = useTheme();

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
    // Menu labels are translated by the API, so switching language has to
    // refetch them — otherwise the sidebar keeps the labels of the locale that
    // was active when the page loaded.
  }, [pathname, router, isPublic, locale]);

  useEffect(() => setSidebarOpen(false), [pathname]);

  const menuTree = useMemo(() => {
    const children = new Map<string, Menu[]>();
    for (const item of menus) {
      if (!item.parent_id) continue;
      children.set(item.parent_id, [...(children.get(item.parent_id) || []), item]);
    }
    for (const siblings of children.values()) siblings.sort((a, b) => a.order - b.order);
    return menus
      .filter((item) => !item.parent_id)
      .sort((a, b) => a.order - b.order)
      .map((item) => ({ ...item, children: children.get(item.id) || [] }));
  }, [menus]);

  useEffect(() => {
    setExpandedMenus((current) => {
      const next = new Set(current);
      for (const parent of menuTree) {
        if (parent.children.some((child) => child.path && pathname.startsWith(child.path))) next.add(parent.id);
      }
      return next;
    });
  }, [menuTree, pathname]);

  const toggleMenu = (id: string) => setExpandedMenus((current) => {
    const next = new Set(current);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });

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
    <div className="gerege-shell min-h-screen flex flex-col">
      {/* Top Navbar */}
      <header className="gerege-topbar h-16 flex items-center justify-between px-4 lg:px-6 border-b sticky top-0 z-50">
        <div className="flex items-center gap-3 min-w-0">
          <button
            type="button"
            onClick={() => setSidebarOpen((open) => !open)}
            className="lg:hidden grid place-items-center w-9 h-9 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50"
            aria-label={sidebarOpen ? (locale === "en" ? "Close menu" : "Цэс хаах") : (locale === "en" ? "Open menu" : "Цэс нээх")}
            aria-expanded={sidebarOpen}
          >
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
          <Link href="/apps" className="flex items-center gap-2.5 min-w-0 group">
            {theme.design === "gerege" ? <img src={brandLogo.src} width={36} height={36} alt="Gerege" className="w-9 h-9 rounded-lg shadow-sm shrink-0" /> : <span className="original-brand-mark w-9 h-9 rounded-lg grid place-items-center shrink-0"><Building2 className="w-6 h-6" /></span>}
            <span className="hidden sm:flex flex-col leading-tight min-w-0">
              <span className="font-semibold text-[15px] text-slate-900 truncate">{theme.design === "gerege" ? "Gerege ERP" : "Gerege Template Platform"}</span>
              <span className="text-[11px] text-slate-500 tracking-wide">{theme.design === "gerege" ? "BUSINESS PLATFORM" : "ORIGINAL THEME"}</span>
            </span>
          </Link>
          <span className="hidden md:flex items-center gap-2 text-xs text-slate-600 border-l border-slate-200 pl-4 ml-1">
            <span className="gerege-session-dot w-1.5 h-1.5 rounded-full"></span>
            <span className="truncate max-w-48"><strong className="text-slate-800 font-medium">{user?.tenant_name || "Demo Tenant"}</strong> · {locale === "en" ? "active" : "идэвхтэй"}</span>
          </span>
        </div>

        <div className="flex items-center gap-2 sm:gap-3">
          <button onClick={theme.toggleMode} className="grid place-items-center w-9 h-9 rounded-lg border border-slate-200 text-slate-500 hover:bg-slate-50" aria-label={locale === "en" ? "Toggle theme" : "Theme солих"} title={locale === "en" ? "Toggle theme" : "Theme солих"}>
            {theme.resolvedMode === "dark" ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
          </button>
          <LanguageSwitcher />
          <div className="hidden md:flex items-center gap-2.5 pl-3 border-l border-slate-200">
            <span className="w-8 h-8 rounded-full bg-[var(--gerege-blue-soft)] text-[var(--gerege-blue)] grid place-items-center text-xs font-bold">
              {(user?.name || user?.email || "G").slice(0, 1).toUpperCase()}
            </span>
            <span className="flex flex-col leading-tight max-w-40">
              <span className="text-sm font-medium text-slate-800 truncate">{user?.name}</span>
              <span className="text-[11px] text-slate-500 truncate">{user?.email}</span>
            </span>
            <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
          </div>
          <button
            onClick={handleLogout}
            className="flex items-center gap-1.5 text-xs text-slate-500 hover:text-red-600 px-2.5 py-2 rounded-lg hover:bg-red-50 transition"
            title={t("shell.logout")}
          >
            <LogOut className="w-4 h-4" />
            <span className="hidden xl:inline">{t("shell.logout")}</span>
          </button>
        </div>
      </header>

      <div className="flex flex-1">
        {/* Sidebar */}
        {sidebarOpen && <button className="fixed inset-0 top-16 bg-slate-950/25 z-30 lg:hidden" onClick={() => setSidebarOpen(false)} aria-label={locale === "en" ? "Close menu" : "Цэс хаах"} />}
        <aside className={`gerege-sidebar fixed lg:static top-16 bottom-0 left-0 z-40 w-60 border-r flex flex-col py-5 justify-between transition-transform lg:translate-x-0 ${sidebarOpen ? "translate-x-0" : "-translate-x-full"}`}>
          <div className="space-y-6">
            <div>
              <div className="px-4 text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                {t("shell.modules")}
              </div>
              <nav className="space-y-1 px-2">
                <Link
                  href="/apps"
                  className={`gerege-nav-link flex items-center space-x-3 px-3 py-2.5 text-sm font-medium transition ${
                    pathname === "/apps"
                      ? "gerege-nav-link-active font-semibold"
                      : ""
                  }`}
                >
                  <LayoutGrid className="gerege-nav-icon w-5 h-5" />
                  <span>{t("shell.appStore")}</span>
                </Link>

                {menuTree.map((parent) => {
                  const expanded = expandedMenus.has(parent.id);
                  const active = parent.children.some((child) => child.path && pathname.startsWith(child.path));
                  return (
                    <div key={parent.id} className="gerege-menu-group">
                      <button
                        type="button"
                        onClick={() => toggleMenu(parent.id)}
                        className={`gerege-nav-link gerege-parent-menu w-full flex items-center gap-3 px-3 py-2.5 text-sm font-semibold transition ${active ? "text-[var(--gerege-blue)]" : ""}`}
                        aria-expanded={expanded}
                      >
                        <span className="gerege-nav-icon">{iconMap[parent.icon] || <Layers3 className="w-5 h-5" />}</span>
                        <span className="flex-1 text-left">{parent.label}</span>
                        <ChevronRight className={`w-4 h-4 text-slate-400 transition-transform ${expanded ? "rotate-90" : ""}`} />
                      </button>
                      {expanded && (
                        <div className="gerege-submenu ml-5 pl-3 py-1 space-y-0.5">
                          {parent.children.filter((child) => child.path).map((child) => (
                            <Link
                              key={child.id}
                              href={child.path!}
                              className={`gerege-nav-link flex items-center gap-3 px-3 py-2 text-[13px] font-medium transition ${pathname.startsWith(child.path!) ? "gerege-nav-link-active font-semibold" : ""}`}
                            >
                              <span className="gerege-nav-icon">{iconMap[child.icon] || <Package className="w-4 h-4" />}</span>
                              <span>{child.label}</span>
                            </Link>
                          ))}
                        </div>
                      )}
                    </div>
                  );
                })}
              </nav>
            </div>

            <div>
              <div className="px-4 text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">
                {t("shell.settings")}
              </div>
              <nav className="space-y-1 px-2">
                <Link
                  href="/settings/appearance"
                  className={`gerege-nav-link flex items-center space-x-3 px-3 py-2.5 text-sm font-medium transition ${
                    pathname === "/settings/appearance"
                      ? "gerege-nav-link-active font-semibold"
                      : ""
                  }`}
                >
                  <Palette className="gerege-nav-icon w-5 h-5" />
                  <span>{t("shell.appearance")}</span>
                </Link>

                <Link
                  href="/settings/apps"
                  className={`gerege-nav-link flex items-center space-x-3 px-3 py-2.5 text-sm font-medium transition ${
                    pathname === "/settings/apps"
                      ? "gerege-nav-link-active font-semibold"
                      : ""
                  }`}
                >
                  <Settings className="gerege-nav-icon w-5 h-5" />
                  <span>{t("shell.installedApps")}</span>
                </Link>

                <Link
                  href="/settings/integrations"
                  className={`gerege-nav-link flex items-center space-x-3 px-3 py-2.5 text-sm font-medium transition ${
                    pathname === "/settings/integrations"
                      ? "gerege-nav-link-active font-semibold"
                      : ""
                  }`}
                >
                  <Share2 className="gerege-nav-icon w-5 h-5" />
                  <span>{t("shell.integrations")}</span>
                </Link>
              </nav>
            </div>
          </div>

          <div className="px-4 text-[11px] text-slate-400 border-t border-slate-100 pt-3">
            <span className="text-slate-500 font-medium">Gerege Theme</span><br />
            <span>ERP Platform · 2026</span>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 p-4 sm:p-6 lg:p-8 overflow-y-auto min-w-0">{children}</main>
      </div>
      <AICopilot />
    </div>
  );
}
