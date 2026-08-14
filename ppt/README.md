# Танилцуулга — ppt.nexus.gerege.mn

Gerege Nexus-ийг тайлбарласан 20 хуудас танилцуулга. Нэг бие даасан
`index.html` — build алхамгүй, гадаад хамааралгүй, offline ч нээгдэнэ.

## Ашиглах

- Хөтчөөр `index.html`-ийг шууд нээнэ, эсвэл:

```bash
cd ppt && python3 -m http.server 8090   # http://localhost:8090
```

- Удирдлага: ← → сумнууд, Space, PgUp/PgDn, Home/End; `f` — бүтэн дэлгэц;
  утсан дээр зүүн/баруун шудрах. `#N` hash-аар тодорхой слайд руу очно
  (жишээ нь `index.html#17`).

## ppt.nexus.gerege.mn дээр байршуулах

Статик файл тул сервер дээр хавтсаа хуулаад nginx-д нэг server block
нэмэхэд хангалттай (жишээ нь `nginx-ppt.conf.example`):

```bash
# сервер дээр
sudo mkdir -p /var/www/nexus-ppt
sudo cp ppt/index.html /var/www/nexus-ppt/
```

DNS: `ppt.nexus.gerege.mn` A бичлэгийг үндсэн сервер рүү заана;
TLS-ийг бусад дэд домэйнтэй ижил аргаар (wildcard эсвэл certbot) авна.

Deploy автоматжуулах бол `.github/workflows/deploy.yml`-ийн серверт файл
хуулдаг алхамд `ppt/` хавтсыг нэмэхэд л болно — build шаардлагагүй.

## Засварлах

Слайд бүр `index.html` доторх нэг `<section class="slide">`. Дизайны
токенууд (өнгө, хэмжээ) файлын эхний `:root` блокт байгаа. Diagram-ууд
нь энгийн HTML/CSS (`.dg`, `.layer`, `.chips` классууд) тул текст
шиг засагдана.
