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
нэмэхэд хангалттай (`nginx-ppt.conf.example`). Анхны суулгалт 2026-08-14-нд
хийгдсэн; шинэ орчинд давтахад:

```bash
# 1. файлаа хуулах (ssh alias: nexus / nexus-root)
scp ppt/index.html nexus-root:/tmp/
ssh nexus-root 'mkdir -p /var/www/nexus-ppt &&
                install -m 0644 /tmp/index.html /var/www/nexus-ppt/index.html'

# 2. vhost суулгаад асаах
scp ppt/nginx-ppt.conf.example nexus-root:/tmp/
ssh nexus-root 'install -m 0644 /tmp/nginx-ppt.conf.example \
                  /etc/nginx/sites-available/ppt.nexus.gerege.mn &&
                ln -sfn /etc/nginx/sites-available/ppt.nexus.gerege.mn \
                        /etc/nginx/sites-enabled/ppt.nexus.gerege.mn &&
                nginx -t && systemctl reload nginx'

# 3. гэрчилгээ (443 блокийг certbot өөрөө бичнэ)
ssh nexus-root 'certbot --nginx -d ppt.nexus.gerege.mn --redirect'
```

Зөвхөн агуулга шинэчлэх бол 1-р алхмыг давтаад л болно — nginx reload хэрэггүй.

DNS: `ppt.nexus.gerege.mn` нь `*.nexus.gerege.mn` wildcard-аар аль хэдийн
38.180.145.75 рүү заадаг. TLS нь энэ хостын конвенцоор дэд домэйн тус бүрдээ
certbot гэрчилгээтэй (wildcard биш), авто-шинэчлэлт certbot дээр тохирсон.

Deploy автоматжуулах бол `.github/workflows/deploy.yml`-ийн серверт файл
хуулдаг алхамд `ppt/` хавтсыг нэмэхэд л болно — build шаардлагагүй.

## Засварлах

Слайд бүр `index.html` доторх нэг `<section class="slide">`. Дизайны
токенууд (өнгө, хэмжээ) файлын эхний `:root` блокт байгаа. Diagram-ууд
нь энгийн HTML/CSS (`.dg`, `.layer`, `.chips` классууд) тул текст
шиг засагдана.
