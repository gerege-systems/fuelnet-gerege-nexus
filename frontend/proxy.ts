import { NextResponse, type NextRequest } from "next/server";
import { DEVICE_LINE_HEADER, deviceLineFromHost, lineHomePath } from "@/lib/deviceLine";

/**
 * Төхөөрөмжийн domain шугамыг Host-оор нь салгана.
 *
 * (Next 16-д `middleware.ts` нь `proxy.ts` болж нэрлэгдсэн — функцийн нэр мөн
 * `proxy`. Үйлдэл нь өмнөхтэй ижил.)
 *
 * Хөтчийн шугам (`nexus.gerege.mn`) дээр энэ файл юу ч хийхгүй — толгой
 * нэмэхгүй, шилжүүлэхгүй. Хөтчийн горим ямар ч нөхцөлд өөрчлөгдөхгүй гэсэн
 * гэрээний нөхцөл эндээс эхэлнэ.
 *
 * Төхөөрөмжийн шугам дээр хийх зүйл гурав:
 *
 * 1. `/` дээр тухайн шугамын өөрийн нүүр дэлгэц рүү шилжүүлнэ. Шугам бүр
 *    өөрийн нүүрээ (`/line/<platform>`) өөрөө хөгжүүлнэ.
 *
 *    Rewrite биш redirect: rewrite үед хөтчийн хаяг `/` хэвээр үлддэг тул
 *    client талын router динамик сегментийг олж харахгүй, `useParams()` хоосон
 *    ирнэ. Redirect нь тэр эргэлзээг арилгаад зогсохгүй, аль шугам дээр байгааг
 *    хаягаас нь шууд уншуулна — оношлоход хамаагүй хялбар.
 *
 * 2. `/login` руу орохыг хаана. Тэдгээр шугам дээр нэвтрэлт бол native UI —
 *    web-ийн нэвтрэх дэлгэц бүрхүүлийн дотор гарч ирвэл хэрэглэгч нэг апп
 *    дотроос хоёр өөр нэвтрэлт хардаг болно. Session байхгүй үед бүрхүүл
 *    өөрөө `auth.reLogin`-оор нэвтрэлтээ эхлүүлнэ.
 *
 * 3. Тухайн шугамын нэрийг доош дамжуулж, `Vary: Host` тавина. Шугамууд нэг
 *    ижил зам дээр өөр өөр агуулга үйлчилдэг тул үүнгүйгээр CDN нэг шугамын
 *    хариуг нөгөөд өгөх боломжтой.
 */
export function proxy(request: NextRequest) {
  const line = deviceLineFromHost(request.headers.get("host"));
  if (!line) return NextResponse.next();

  const { pathname } = request.nextUrl;

  if (pathname === "/" || pathname === "/login") {
    const target = request.nextUrl.clone();
    target.pathname = lineHomePath(line);
    target.search = "";
    const redirect = NextResponse.redirect(target);
    redirect.headers.set("Vary", "Host");
    return redirect;
  }

  const headers = new Headers(request.headers);
  headers.set(DEVICE_LINE_HEADER, line.platform);
  const response = NextResponse.next({ request: { headers } });
  response.headers.set(DEVICE_LINE_HEADER, line.platform);
  response.headers.set("Vary", "Host");
  return response;
}

export const config = {
  // Статик хөрөнгө, зураг, manifest дээр ажиллуулах шаардлагагүй — тэдгээр нь
  // шугамаас үл хамааран ижил.
  matcher: ["/((?!_next/static|_next/image|favicon.ico|icons|brand.webp|manifest.webmanifest).*)"],
};
