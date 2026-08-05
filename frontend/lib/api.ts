const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

async function fetcher<T>(url: string, options: RequestInit = {}): Promise<T> {
  const token = typeof window !== "undefined" ? localStorage.getItem("session_token") : null;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers,
    credentials: "include",
  });

  if (!res.ok) {
    let errMessage = "Request failed";
    try {
      const errData = await res.json();
      errMessage = errData.error || errMessage;
    } catch {
      // ignore
    }
    throw new Error(errMessage);
  }

  return res.json();
}

export const api = {
  // Auth
  login: (email: string, password: string) =>
    fetcher<{ token: string; user: any }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  loginWithEID: (ssoToken?: string, regNumber?: string, otpCode?: string) =>
    fetcher<{ token: string; user: any; identity: any }>("/auth/eid/login", {
      method: "POST",
      body: JSON.stringify({ sso_token: ssoToken, reg_number: regNumber, otp_code: otpCode }),
    }),

  logout: () => fetcher<{ status: string }>("/auth/logout", { method: "POST" }),

  getMe: () => fetcher<{ id: string; tenant_id: string; tenant_name: string; name: string; email: string; is_admin: boolean }>("/auth/me"),

  getMenus: () => fetcher<Array<{ id: string; label: string; path: string; icon: string; order: number }>>("/menus"),

  // Store
  getStoreApps: () =>
    fetcher<
      Array<{
        id: string;
        slug: string;
        name: string;
        description: string;
        icon_url: string;
        category: string;
        version: string;
        installed: boolean;
        enabled: boolean;
        manifest: any;
      }>
    >("/store/apps"),

  getInstalledApps: () =>
    fetcher<
      Array<{
        id: string;
        app_id: string;
        slug: string;
        name: string;
        installed_version: string;
        status: string;
        enabled: boolean;
        installed_at: string;
      }>
    >("/installed-apps"),

  installApp: (slug: string) => fetcher<{ status: string; app: string }>(`/store/apps/${slug}/install`, { method: "POST" }),

  enableApp: (slug: string) => fetcher<{ status: string; app: string }>(`/store/apps/${slug}/enable`, { method: "POST" }),

  disableApp: (slug: string) => fetcher<{ status: string; app: string }>(`/store/apps/${slug}/disable`, { method: "POST" }),

  // Contacts App
  getContacts: () =>
    fetcher<
      Array<{
        id: string;
        name: string;
        email: string;
        phone: string;
        company: string;
        active: boolean;
        created_at: string;
      }>
    >("/contacts"),

  createContact: (data: { name: string; email: string; phone: string; company: string; active: boolean }) =>
    fetcher("/contacts", { method: "POST", body: JSON.stringify(data) }),

  updateContact: (id: string, data: { name: string; email: string; phone: string; company: string; active: boolean }) =>
    fetcher(`/contacts/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  // Products App
  getProducts: () =>
    fetcher<
      Array<{
        id: string;
        sku: string;
        name: string;
        price: number;
        active: boolean;
        created_at: string;
      }>
    >("/products"),

  createProduct: (data: { sku: string; name: string; price: number; active: boolean }) =>
    fetcher("/products", { method: "POST", body: JSON.stringify(data) }),

  updateProduct: (id: string, data: { sku: string; name: string; price: number; active: boolean }) =>
    fetcher(`/products/${id}`, { method: "PUT", body: JSON.stringify(data) }),

  // Inventory App
  getWarehouses: () =>
    fetcher<
      Array<{
        id: string;
        code: string;
        name: string;
        address: string;
        created_at: string;
      }>
    >("/inventory/warehouses"),

  createWarehouse: (data: { code: string; name: string; address: string }) =>
    fetcher("/inventory/warehouses", { method: "POST", body: JSON.stringify(data) }),

  getStockLevels: () =>
    fetcher<
      Array<{
        id: string;
        warehouse_id: string;
        product_id: string;
        quantity: number;
        updated_at: string;
      }>
    >("/inventory/stock-levels"),

  getStockMovements: () =>
    fetcher<
      Array<{
        id: string;
        warehouse_id: string;
        product_id: string;
        quantity_change: number;
        reference: string;
        created_at: string;
      }>
    >("/inventory/movements"),

  adjustStock: (data: { warehouse_id: string; product_id: string; quantity_change: number; reference: string }) =>
    fetcher("/inventory/adjustments", { method: "POST", body: JSON.stringify(data) }),

  // AI Assistant & Forecasting
  queryAICopilot: (prompt: string) =>
    fetcher<{ answer: string; intent: string; data?: any; actionable?: string[] }>("/ai/copilot", {
      method: "POST",
      body: JSON.stringify({ prompt }),
    }),

  getAIForecast: () =>
    fetcher<
      Array<{
        product_id: string;
        sku: string;
        product_name: string;
        current_stock: number;
        recommended_min: number;
        reorder_alert: boolean;
        suggested_reorder: number;
      }>
    >("/ai/stock-forecast"),
};
