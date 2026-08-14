import { apiBase } from "@/lib/apiBase";

/**
 * Энэ суулгац апп стор мөн үү, мөн бол юу санал болгож байна вэ.
 *
 * Стор бол платформын өөр загвар биш, ижил бинар дээр суусан гурван модуль
 * (`appstore-gerege-nexus`). Тэдгээр суугаагүй суулгацад доорх endpoint байхгүй
 * тул хариу нь `null` — тэгвэл нүүр хуудас платформынхаа танилцуулгыг үзүүлнэ.
 *
 * Хайлт нь build үед биш ажиллах үед хийгддэг нь чухал: нэг образ бүх
 * суулгацад үйлчилдэг байх ёстой (`lib/apiBase.ts`-ийг үз), тиймээс "би стор
 * мөн үү" гэдгийг образ мэдэж болохгүй — зөвхөн ажиллаж буй сервер нь мэднэ.
 */
export interface StoreApp {
  id: string;
  slug: string;
  name: string;
  description: string;
  icon_url: string;
  category: string;
  publisher: string;
  latest_version: string;
  homepage?: string;
  translations?: Record<string, { name?: string; description?: string; category?: string }>;
}

/** Каталогийн бичлэгийг хэрэглэгчийн хэл рүү буулгана. */
export function localizedApp(app: StoreApp, locale: string): StoreApp {
  const t = app.translations?.[locale];
  if (!t) return app;
  return {
    ...app,
    name: t.name || app.name,
    description: t.description || app.description,
    category: t.category || app.category,
  };
}

/**
 * Нийтлэгдсэн аппуудыг татна, эсвэл энэ суулгац стор биш бол `null`.
 *
 * Алдааг сурталчлахгүй: нүүр хуудас нь нэвтрээгүй зочны хардаг цорын ганц
 * дэлгэц бөгөөд стор биш суулгац дээр 404 нь алдаа биш, хариулт юм.
 */
export async function fetchStorefront(): Promise<StoreApp[] | null> {
  try {
    const res = await fetch(`${apiBase()}/registry/apps`, { credentials: "omit" });
    if (!res.ok) return null;
    const apps = (await res.json()) as StoreApp[] | null;
    return Array.isArray(apps) && apps.length > 0 ? apps : null;
  } catch {
    return null;
  }
}
