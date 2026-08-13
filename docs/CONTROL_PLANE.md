# Control Plane — операторын консол

Платформыг удирддаг консол: `cp.nexus.gerege.mn`. Энэ баримт нь **хэрхэн
босгох, хэрхэн ажиллуулах** тухай. Яагаад ийм байдлаар зохиогдсоныг
[CONTROL_PLANE_PLAN.md](CONTROL_PLANE_PLAN.md)-аас үзнэ үү.

[Баримт бичгийн төв рүү буцах](README.md) ·
Холбоотой: [Мониторинг](MONITORING.md) · [Runbook](RUNBOOKS.md)

---

## 1. Одоо юу байгаа вэ (CP-1)

Энэ үе шат нь **суурь**: нэвтрэлт, эрх, audit, тусгаарлалт. Консол
**зөвхөн уншина** — тенант түдгэлзүүлэх, устгах, impersonation зэрэг
үйлдлүүд CP-2-т нэмэгдэнэ.

| Боломж | Төлөв |
| --- | --- |
| Операторын бүртгэл, нууц үг + TOTP, lockout | ✅ |
| Богино session (8 цаг, 30 мин idle), step-up механизм | ✅ |
| Append-only `operator_audit`, бичих үйлдэл бүрд заавал | ✅ |
| Тенантын жагсаалт/дэлгэрэнгүй (зөвхөн унших) | ✅ |
| Операторын жагсаалт, audit хайлт | ✅ |
| Тенантын амьдралын мөчлөг, quota, support, impersonation | CP-2 |
| Динамик тохиргоо, feature flag, `platform.access_mode` | CP-3 |
| Ажиглалтын тойм, deploy товч, backup төлөв | CP-4 |
| Metering | CP-5 |

## 2. Гурван давхарга

Консол руу хүрэхийн тулд гурвуулангаас өнгөрөх ёстой. Нэг нь нөгөөдөө
итгэдэггүй:

1. **nginx-ийн хаягийн allowlist** — `deploy/nginx/snippets/cp-allowlist.conf`.
   Жагсаалтад байхгүй хаягнаас ирсэн хүсэлт 403 авч, аппликейшн рүү огт
   хүрэхгүй. Репод ирдэг хувилбар нь `deny all` — тохируулаагүй консол
   нээлттэй байснаас хаалттай байх нь зөв.
2. **`CONTROL_PLANE_HOST`** — API ба frontend хоёулаа энэ нэрээр ирээгүй
   хүсэлтэд **404** хариулна (403 биш: 403 нь тэнд ямар нэг зүйл байгааг
   баталгаажуулна). Production дээр хоосон орхивол консол огт байхгүй.
3. **Нэвтрэлт** — нууц үг + TOTP. Хоёр дахь хүчин зүйлгүй бүртгэл нэвтэрч
   чадахгүй.

## 3. Босгох

### 3.1 nginx

```bash
sudo cp deploy/nginx/cp.nexus.gerege.mn.conf /etc/nginx/sites-available/
sudo cp deploy/nginx/snippets/cp-allowlist.conf /etc/nginx/snippets/
sudo ln -s /etc/nginx/sites-available/cp.nexus.gerege.mn.conf /etc/nginx/sites-enabled/

# Операторуудын бодит хаягийг бичнэ — эс бөгөөс консол хэнд ч нээгдэхгүй.
sudo nano /etc/nginx/snippets/cp-allowlist.conf

sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d cp.nexus.gerege.mn
```

DNS дээр `cp.nexus.gerege.mn` нь тухайн серверийг заасан байх ёстой.

### 3.2 Env

`deploy/.env.prod` дотор:

```
CONTROL_PLANE_HOST=cp.nexus.gerege.mn
```

Дараа нь `docker compose -f docker-compose.prod.yml up -d backend frontend`.
Энэ утга backend ба frontend хоёуланд очно — тэдгээр нь нэг дүрмийн хоёр тал.

### 3.3 Миграц

`00049_control_plane.sql` нь гурван хүснэгт ба `gerege_nexus_operator` role
үүсгэнэ. `docker compose up` нь миграцыг өөрөө ажиллуулна. Лог дээр:

```
dbguard: the control plane may bind its own database role role=gerege_nexus_operator
```

гэж гарвал бэлэн. Гараагүй бол консол ажиллахгүй (санаатай — role-гүйгээр
query нь login role-оор явахгүй, огт явахгүй).

### 3.4 Анхны оператор

Вэб бүртгэл **байхгүй**. Эхний бүртгэлийг DB-ийн эрхтэй хүн тушаалаар үүсгэнэ:

```bash
docker exec -it gerege_nexus_api /app/operator-bootstrap \
    -email you@gerege.mn -name "Таны нэр" -role superadmin
```

Тушаал нь нууц үгийг хоёр удаа асууж (харагдахгүй), TOTP-ийн нууц ба
`otpauth://` URI-г нэг удаа хэвлэнэ. Түүнийг authenticator-т нэмээд
(1Password, Aegis, Google Authenticator) кодыг нь буцааж бичиж
баталгаажуулна. **Баталгаажуулаагүй бүртгэл нэвтэрч чадахгүй** — тасалдсан
bootstrap нь нууц үгээр нээгддэг хаалга биш, түгжигдсэн хаалга үлдээнэ.

`docker exec` дээр `-it` заавал: нууц үг терминалаас уншигдана. Flag эсвэл
env-ээр нууц үг дамжуулах зам байхгүй — тэдгээр нь shell-ийн түүх,
процессын жагсаалт, контейнерийн `inspect`-д үлддэг.

## 4. Үүргүүд

| Үүрэг | CP-1 дээр юу хийж чадах вэ |
| --- | --- |
| `superadmin` | Бүгд |
| `operator` | Тенант унших, audit унших, операторуудыг харах |
| `support` | Тенант унших, audit унших |
| `auditor` | Тенант унших, audit унших, операторуудыг харах |

Дараагийн үе шатууд эрхийг `capabilities` (`internal/platform/controlplane/
operator.go`) хүснэгтэд мөр нэмэх байдлаар өргөтгөнө — handler дотор
`if role == "superadmin"` гэж бичихгүй.

## 5. Аюулгүй байдлын дүрмүүд

- **Бичих үйлдэл бүр audit-тай.** `Service.Do` нь үйлдэл ба `operator_audit`-ын
  мөрийг **нэг transaction**-д бичнэ. Түүнээс гадуур бичсэн handler-ийн
  хариуг `RequireAudit` буцаалгүй 500 өгнө — "бүртгэгдээгүй бол болоогүй".
- **`operator_audit` нь append-only.** UPDATE/DELETE-ийг DB-ийн trigger
  чангаар татгалзана (эзэн role ч гэсэн). Тестийн өгөгдөл ч устахгүй.
- **Консол бичих эрхгүй.** `gerege_nexus_operator` role нь тенантын
  хүснэгтүүдэд зөвхөн SELECT-тэй, нэрлэсэн жагсаалтаар. Шинэ хүснэгт
  автоматаар нэмэгдэхгүй.
- **Нэг код нэг удаа.** TOTP-ийн цагийн алхам хадгалагдаж, буурах/давтагдах
  кодыг татгалзана.
- **Step-up** — аюултай үйлдлийн өмнө 5 минутын дотор код дахин асуугдана.
  CP-1 дээр ашиглагдах үйлдэл байхгүй ч механизм бэлэн.

## 6. Хэмжүүр ба сэрэмжлүүлэг

`cp_login_attempts_total{result}` — `success`, `unknown`, `bad_password`,
`bad_code`, `locked`, `disabled`, `no_second_factor`, `step_up`,
`bad_step_up`.

Консолын нэвтрэлт долоо хоногт хэдхэн удаа болдог тул нэг цагт олон
амжилтгүй оролдлого нь шуугиан биш — хэн нэгэн оролдож байна гэсэн үг.
Alert дүрмийг [RUNBOOKS.md](RUNBOOKS.md)-оос үз.

## 7. Хөгжүүлэлт дээр

`CONTROL_PLANE_HOST` тохируулаагүй, `ENVIRONMENT` нь production биш үед
консол `http://localhost:3000/cp` дээр нээлттэй байна. Оператор бүртгэлээ
дээрхтэй ижил тушаалаар (`go run ./cmd/operator-bootstrap ...`) үүсгэнэ.

Тест:

```bash
cd backend && DATABASE_URL=<dsn> go test ./internal/platform/controlplane/...
```

Өгөгдлийн сангүй бол DB-ийн тестүүд алгасагдана — role, trigger, replay
хамгаалалтыг бодитоор шалгах тул CI дээр заавал ажиллана.
