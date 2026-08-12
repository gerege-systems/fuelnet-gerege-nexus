import { currentDeviceLine } from "@/lib/deviceLine";

/**
 * Build үед шингэдэг хаяг. Хөгжүүлэлтэд frontend :3000, API :8080 гэсэн хоёр
 * өөр порт дээр байдаг тул үнэмлэхүй хаяг зайлшгүй.
 */
const CONFIGURED = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

/**
 * Энэ хуудсанд ашиглах API-гийн үндэс.
 *
 * Төхөөрөмжийн domain шугам дээр **үргэлж same-origin `/api/v1`**. Энэ нь
 * гоо сайхны асуудал биш: `NEXT_PUBLIC_API_URL` нь build үед шингэдэг ба
 * production-д `https://nexus.gerege.mn/api/v1` гэж бичигддэг. Тэр утгыг
 * `mac.nexus.gerege.mn` дээр ашиглавал дуудлага cross-origin болж, session
 * cookie нь host-only учраас ОГТ илгээгдэхгүй — API 401 буцааж, ажлын муж
 * нэвтрэлт дууссан гэж үзээд web-ийн `/login` руу түлхдэг. Яг ийм зүйл
 * тохиолдсон: native талд амжилттай нэвтэрсэн хэрнээ ажлын мужид нэвтрэх
 * дэлгэц гарч ирж байв.
 *
 * Шугам бүр өөрийн host дээрээ `/api/v1`-ээ үйлчилдэг нь nginx-ээр
 * баталгаажсан (`deploy/nginx/device-lines.nexus.gerege.mn.conf`), тиймээс
 * харьцангуй зам нь тодорхойлолтоороо зөв.
 *
 * Хөтчийн шугам дээр юу ч өөрчлөгдөхгүй — build-ийн утга хэвээр.
 */
export function apiBase(): string {
  if (typeof window === "undefined") return CONFIGURED;
  return currentDeviceLine() ? "/api/v1" : CONFIGURED;
}
