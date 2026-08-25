# FuelNet танилцуулга — ppt.fuelnet.gerege.mn

FuelNet-ийн төлөвлөгөөг төрийн шийдвэр гаргагч, зохицуулагч, импортлогч,
ШТС-ын сүлжээний удирдлагад тайлбарлах 20 слайдтай standalone HTML deck.
Build алхам, гаднын font эсвэл JavaScript dependency байхгүй.

## Локал ажиллуулах

```bash
cd ppt
python3 -m http.server 8090
```

`http://localhost:8090`-г нээнэ. Удирдлага:

- `←` / `→`, Space, Page Up / Page Down — слайд солих
- Home / End — эхний / сүүлийн слайд
- `f` — fullscreen
- `s` — тухайн слайдын эх сурвалж
- Touch дэлгэц дээр зүүн / баруун swipe
- `#s=12` — тодорхой слайд руу шууд орох

Print dialog-аас landscape PDF болгон хадгалж болно. Слайд бүр 1280×720 буюу
16:9 харьцаатай.

## Production deploy

`main` branch-д push хийсний дараа GitHub Actions:

1. CI-г бүрэн ажиллуулна.
2. `ppt/index.html` болон nginx bootstrap config-ийг production серверт хуулна.
3. Deck-ийг `/var/www/fuelnet-ppt` рүү atomically install хийнэ.
4. Анхны deploy дээр `ppt.fuelnet.gerege.mn` vhost болон Let's Encrypt TLS
   гэрчилгээг үүсгэнэ.
5. HTTPS хаягийг smoke-test хийнэ.

DNS нь `*.fuelnet.gerege.mn` wildcard-аар production host руу заасан.
Vhost-ыг certbot анх удаа TLS болгон өргөтгөсний дараа CI зөвхөн content-ийг
шинэчилж, certbot-ийн удирддаг TLS мөрүүдийг дахин бичихгүй.

## Агуулгын эх сурвалж

Deck-ийн гол эх сурвалж нь repository root дахь `FUELNET_PLAN.md`. Слайд бүр
далд `[Sources]` notes блоктой бөгөөд танилцуулга дээр `s` дарж харна.
