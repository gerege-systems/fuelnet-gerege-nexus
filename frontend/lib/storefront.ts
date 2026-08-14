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
 * Мөн адил асуулт, гэхдээ серверийн зүгээс — хуудсыг зурахаас өмнө.
 *
 * Client талд асуувал эхний зураглал платформын хуудас, дараа нь каталог болж
 * солигдоно: зочин буруу хуудсыг харж амжина, JS ажиллуулдаггүй хайлтын робот
 * зөвхөн түүнийг л харна. Апп сторын хувьд сүүлийнх нь ноцтой — каталог бол
 * хүмүүс хайж олох ёстой зүйл.
 *
 * Серверт origin гэж байхгүй тул хаягийг нь суулгац хэлж өгнө
 * (`API_INTERNAL_URL`, compose-ийн дотоод сүлжээгээр). Энэ нь build-ийн
 * тохиргоо биш ажиллуулах үеийн тохиргоо тул нэг образ бүх суулгацад
 * үйлчилсээр байна. Тавиагүй бол хариу нь `null` — платформын хуудас, өөрөөр
 * хэлбэл стор биш суулгацын зөв хариулт.
 */
export async function fetchStorefrontOnServer(): Promise<StoreApp[] | null> {
  const base = process.env.API_INTERNAL_URL;
  if (!base) return null;
  try {
    const res = await fetch(`${base.replace(/\/$/, "")}/registry/apps`, {
      // Каталог нь нийтлэгчийн үйлдлээр өөрчлөгддөг тул минут тутам хангалттай.
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    const apps = (await res.json()) as StoreApp[] | null;
    return Array.isArray(apps) && apps.length > 0 ? apps : null;
  } catch {
    return null;
  }
}
