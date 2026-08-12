# Тайлан — reporting модуль

`io.gerege.nexus.reports`: суулгасан апп бүрийн тайланг нэг дэлгэцээс
ажиллуулах, график харах, Excel/CSV болгон гаргах, товлосон хугацаанд илгээх.

[Баримт бичгийн төв рүү буцах](README.md) ·
[Дизайны санал](MONITORING_AND_REPORTING_PROPOSAL.md) ·
[Модуль бичих заавар](MODULE_AUTHORING_GUIDE.md)

---

## 1. Гол санаа: тайлан бол дэлгэц биш, тунхаглал

Модуль өөрийн тайлангаа Go-гийн `Report` интерфейсээр **тунхаглана**: юу гэж
нэрлэгдэхээ (7 хэлээр), ямар үзүүлэлт хүлээж авахаа, ямар багана гаргахаа,
хэрхэн тооцоолохоо. Бусад бүхэн — жагсаалтын дэлгэц, үзүүлэлтийн форм,
хүснэгт, график, Excel гаргалт, хуваарь, audit бүртгэл — **нэг удаа** энэ
давхаргад бичигдсэн бөгөөд аль ч модулийн аль ч тайланд үйлчилнэ.

Энэ бол Odoo-гийн загвар бөгөөд `reports` модуль billing, inventory, esign
гэсэн үг мэдэхгүй байгаагийн шалтгаан. Эсрэгээр нь хийвэл тайлангийн модуль
бусад бүх модулийг import хийх ба тэр нь энэ архитектурын зайлсхийхийг зорьсон
холбоо юм.

---

## 2. Шинэ тайлан нэмэх

Модулийнхаа хавтсанд `reports.go` үүсгэ:

```go
package billing

import (
    "context"
    "github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
)

type revenueByMonth struct{}

func (revenueByMonth) Key() string { return "billing.revenue_by_month" }
func (revenueByMonth) App() string { return "io.gerege.nexus.billing" }

func (revenueByMonth) Titles() map[string]string {
    return map[string]string{"mn": "Орлого сараар", "en": "Revenue by month"}
}

func (revenueByMonth) Params() []reporting.ParamSpec {
    return []reporting.ParamSpec{{
        Key:           "period",
        Kind:          reporting.ParamDateRange,
        Titles:        map[string]string{"mn": "Хугацаа", "en": "Period"},
        DefaultWindow: 365 * 24 * time.Hour,
    }}
}

func (revenueByMonth) Columns() []reporting.ColumnSpec {
    return []reporting.ColumnSpec{
        {Key: "month", Kind: reporting.ColumnMonth, Chart: reporting.ChartCategory,
         Titles: map[string]string{"mn": "Сар", "en": "Month"}},
        {Key: "gross", Kind: reporting.ColumnMoney, Chart: reporting.ChartValue, Total: true,
         Titles: map[string]string{"mn": "Нийт дүн", "en": "Gross"}},
    }
}

func (revenueByMonth) Run(ctx context.Context, q reporting.Querier,
    p reporting.Params) (reporting.Result, error) {

    rows, err := q.Query(ctx, `
        SELECT date_trunc('month', created_at)::date, sum(amount + vat_amount)
          FROM billing_invoices
         WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3
         GROUP BY 1 ORDER BY 1`,
        reporting.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"))
    if err != nil {
        return reporting.Result{}, err
    }
    collected, err := reporting.Collect(rows, func() (map[string]any, error) {
        var month time.Time
        var gross float64
        if err := rows.Scan(&month, &gross); err != nil {
            return nil, err
        }
        return map[string]any{"month": month, "gross": gross}, nil
    })
    if err != nil {
        return reporting.Result{}, err
    }
    return reporting.Result{Rows: collected}, nil
}
```

Модулийнхаа `New`-д бүртгэ:

```go
func New(db *pgxpool.Pool) *BillingModule {
    m := &BillingModule{db: db}
    appregistry.Register(m)
    registerReports()   // reporting.Register(revenueByMonth{}) энд
    return m
}
```

Дууслаа. Дэлгэц дээр гарч ирнэ, экспортлогдоно, товлогдоно, audit-д бичигдэнэ.
Frontend-д ямар ч өөрчлөлт хэрэггүй.

### Мөрдөх дүрмүүд

| Дүрэм | Яагаад |
| --- | --- |
| `WHERE tenant_id = $1` **заавал** бич | Хэрэглээний давхаргын шүүлт нь үндсэн хамгаалалт. RLS бол доод давхарга — мартсан заалтыг хоосон үр дүн болгож барих сүүлчийн тор, эхнийх нь биш |
| Тенантыг `reporting.TenantOf(ctx)`-оос ав | Нэгдсэн тайланд яг энэ л зүйл өөр тенант болж солигдоно (§5) |
| Нэгтгэлийг SQL дотор хий | Мянган мөрийг Go руу татаад давталтаар нэмэх нь демо тенант дээр адилхан ажиллаж, бодит дээр унана |
| Хүний нэр биш, регистрийн дугаар биш | Тайлан бол экспортлогдож, и-мэйлээр явж, татсан хавтсанд үлддэг зүйл |
| `mn` гарчиг заавал | Байхгүй бол Register нь асах үед panic хийнэ |

Түлхүүр (`Key`) нь **тогтвортой**: түүгээр хуваарийн мөр, grant-ын мөр
холбогдоно. Нэрлээд өөр зүйлд дахин ашиглаж болохгүй.

---

## 3. Хамгаалалт

**Апп gate.** Тухайн аппыг суулгаагүй байгууллага түүний тайланг жагсаалтад
харахгүй, метадатаг нь авахгүй, түлхүүрээр нь дуудсан ч 404 авна. Гурвуулаа
шалгагдана — жагсаалт шүүх нь хангалттай биш, API нь тусдаа зам.

**Тенант тусгаарлалт.** Тайлан бүр дуудагчийн тенант binding дотор ажиллана
(`dbguard`, миграц 00029). Тенантын заалтаа мартсан тайлан **юу ч буцаахгүй** —
энэ нь `engine_integration_test.go`-д бодит өгөгдлийн сан дээр шалгагдсан тест.

**Зөвхөн уншина.** Тайлангийн query нь read-only гүйлгээнд ажиллана.
Бичих оролдлого нь өгөгдлийн сангаас татгалзагдана, review-ээс биш.

**30 секундын тааз.** `SET LOCAL statement_timeout` — контекстийн deadline биш
(тэр нь зөвхөн энэ процессын хүлээлтийг зогсооно). Удаан тайлан pool-ын
холболтыг барих нь тайлан удаан байснаас илүү аюултай.

**Эрх.** `reports.view` — ажиллуулах, экспортлох. `reports.schedule` —
хуваарь үүсгэх. Хоёрыг тусгаарласан шалтгаан: хуваарь бол хэн ч байхгүй үед
байгууллагын тоог хаягийн жагсаалт руу илгээх шийдвэр.

**Audit.** Гүйлт бүр (`reports.run`), экспорт бүр (`reports.export`), хуваарийн
үйлдэл бүр `audit_events`-д бичигдэнэ. Экспорт нь гүйлтээс тусдаа бичлэг:
экспорт бол өгөгдөл платформоос гарч байгаа хэрэг.

---

## 4. Товлосон тайлан

`report_schedules` хүснэгт (миграц 00045), backend доторх минут тутмын
goroutine. **Шинэ процесс байхгүй** — энэ платформ нэг бинари.

Хуваарь нь cron-ийн 5 талбар: `минут цаг өдөр сар гараг`. `0 9 1 * *` нь
сарын 1-нд 09:00. Илэрхийллийг хадгалах үед шалгана — хэзээ ч ажиллахгүй
хуваарь чимээгүй суух ёсгүй.

**Давхар илгээлт.** Хэд хэдэн replica зэрэг ажиллаж болно. Sweep нь
PostgreSQL-ийн advisory lock барьж, болзсон мөрүүдийг эхлээд `last_run_at`-аар
тэмдэглэж, дараа нь ажиллуулна. Тэмдэглээд ажиллуулах дараалал санаатай:
амжилттай илгээснийхээ дараа тэмдэглэдэг байсан бол хоёрын хооронд дахин
эхэлсэн replica тайланг хоёр удаа илгээх байсан бөгөөд хоёр дахь хувь нь
жинхэнэ хувиас ялгагдахгүй, тоо нь өөр байж болно.

### И-мэйл

```bash
REPORT_SMTP_URL=smtp://user:password@relay.example.mn:587
REPORT_MAIL_FROM=nexus@gerege.mn
```

Хоосон бол хуваарь **ажиллах боловч илгээгдэхгүй** — үр дүн нь "delivery not
configured" гэж бүртгэгдэж, дэлгэц дээр анхааруулга гарна. "Ажиллаагүй"
гэдгээс "бэлтгэгдсэн, хүргэх газаргүй" гэдэг нь өөр бөгөөд илүү хэрэгтэй
байдал.

> **Дизайнаас зөрсөн зүйл.** Санал нь товлосон тайланг "одоогийн hosted email
> үйлчилгээгээр" илгээхээр бичсэн. Тэр үйлчилгээ ганц зүйл илгээдэг —
> баталгаажуулах холбоос — бөгөөд гарчиг, бие, хавсралт өгөх endpoint байхгүй.
> Тиймээс товлосон тайланд өөрийн gate хэрэгтэй болсон ба SMTP нь суулгац
> бүрийн аль хэдийн хариулттай зүйл.

---

## 5. Тенант дамнасан тайлан

Уурхай/тээврийн компанийн кейс — саналын §3.5 — нь `report_grants` механизмаар
шийдэгдэнэ. Дэлгэрэнгүйг [`REPORT_SHARING.md`](REPORT_SHARING.md)-ээс үзнэ үү.

Энд чухал нь: тэр механизм **энэ хөдөлгүүрийг өөрчлөхгүй**. Нэгдсэн тайлан нь
ижил `Run`-ыг grantor бүрийн тенант контекст дотор дуудна, ямар ч бодлого
сулрахгүй, тайлан өөрөө ялгааг мэдэхгүй.

---

## 6. API

| Аргачлал | Зам | Тайлбар |
| --- | --- | --- |
| `GET` | `/api/v1/reports` | Аппаар бүлэглэсэн жагсаалт |
| `GET` | `/api/v1/reports/{key}` | Метадата: үзүүлэлт, багана |
| `POST` | `/api/v1/reports/{key}/run` | JSON үр дүн |
| `POST` | `/api/v1/reports/{key}/export?format=xlsx\|csv` | Файл |
| `GET` | `/api/v1/reports/schedules` | Хуваариуд |
| `POST` | `/api/v1/reports/schedules` | Хуваарь үүсгэх |
| `PUT` | `/api/v1/reports/schedules/{id}` | Засах |
| `DELETE` | `/api/v1/reports/schedules/{id}` | Устгах |

Бүгд апп gate-ийн ард. Нээлттэй тайлангийн endpoint байхгүй бөгөөд байх ч
ёсгүй.

---

## 7. Экспорт

**xlsx** (`excelize`): гарчгийн мөр, тод толгой, багана бүрийн тоо/огнооны
формат, нийт дүнгийн мөр, толгойн мөр царцаасан. Тоонууд нь **тоо** байдлаар
орно — нийлбэр гаргаж болдоггүй хүснэгт бол дэлгэцийн зураг л гэсэн үг.

**csv**: UTF-8 BOM-той. BOM байхгүй бол Windows дээрх Excel монгол толгойг
mojibake болгож уншина — тэр нь тайлангийн бүх агуулга.

Файлын нэр нь тайлангийн түлхүүр + огноо. Локалчилсан нэр биш: кирилл үсэгтэй
файлын нэр браузер, и-мэйл клиентээр жигд бус дамждаг.
