# Клиентүүд эзнээ дагаж яваагүй

Эдгээр файлын endpoint-ууд нь **энэ репод байхаа больсон** модулиудынх.
Backend модуль нь өөрийн repo руу явсан, харин дэлгэцүүд нь цөмд үлдсэн —
`docs/ECOSYSTEM_GIT_STRATEGY.md` §2.3-ын зориудын шийдвэр: бүрхүүл нэг байх
ёстой, дэлгэц зөөх нь `@gerege/ui` гарсны дараа хамаагүй хямд болно.

Тиймээс эдгээр нь **устгах жагсаалт биш, харьяаллын жагсаалт**. Дэлгэцүүд
ажилласаар байна, эдгээр клиент тэднийг тэжээсээр байна.

| Файл | Аль модуль | Аль repo |
| --- | --- | --- |
| `contacts.ts` | Харилцагч | `business-gerege-nexus` |
| `products.ts` | Бараа | `business-gerege-nexus` |
| `inventory.ts` | Агуулах, үлдэгдэл, хөдөлгөөн | `business-gerege-nexus` |
| `billing.ts` | Нэхэмжлэх | `business-gerege-nexus` |
| `pos.ts` | Касс: бүртгэл, ээлж, худалдаа | `pos-gerege-nexus` |
| `shifts.ts` | Төхөөрөмжийн ээлж (`/devices/shifts/*`) | `pos-gerege-nexus` |
| `publisher.ts` | Нийтлэгчийн портал | `appstore-gerege-mn` |
| `store-review.ts` | Хувилбарын хяналт | `appstore-gerege-mn` |
| `appstore-registry.ts` | Каталогийн бүртгэл | `appstore-gerege-mn` |

## Юу нь энд байх ёсгүй вэ

Гурван зам явсан юм шиг харагддаг ч **цөмийнх**:

* `/store/apps`, `/admin/store/*` — суулгацын дэлгүүрийн дэлгэц. Суулгац бүр
  ямар нэг зүйл суулгадаг тул бүгдэд нь байна (`server.go:1069-1099`).
  `lib/api/store.ts`-д.
* `/esign/*` — esign нь миграц 00058-аар `documents` руу нэгдсэн, рельс нь
  `internal/platform/esign`-д амьд. `lib/api/esign.ts`-д.
* `/admin/devices/*` — төхөөрөмжийн бүртгэл нь платформынх. Зөвхөн
  `/devices/shifts/*` явсан.

## Хэзээ явах вэ

Үе 2d — дэлгэцүүдтэйгээ хамт. `docs/CORE_BOUNDARY_PLAN.md` §5 Үе 2d, §8.
Одоо устгавал ажиллаж байгаа дэлгэц эвдэрнэ.
