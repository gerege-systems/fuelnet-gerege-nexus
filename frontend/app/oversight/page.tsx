import type {Metadata} from "next";
import {AlertTriangle, BarChart3, DatabaseZap, Eye, FileClock, Landmark, LockKeyhole, Scale, ShieldCheck} from "lucide-react";

import FuelNetShell from "../../components/fuelnet/FuelNetShell";
import {CheckList, FeatureGrid, PageCTA, PageHero, SectionHeading} from "../../components/fuelnet/ui";

export const metadata: Metadata = {title: "Төрийн хяналт · FuelNet"};

export default function OversightPage() {
  return (
    <FuelNetShell>
      <main>
        <PageHero
          eyebrow="Төрийн хяналт ба ил тод байдал"
          title="Таамгаар биш,"
          accent="нэг эх сурвалжаар удирдана."
          body="Нөөц, үнэ, чанар, татвар, түгээлтийн мэдээллийг зохицуулагчийн эрхийн түвшнээр бодит цагт нэгтгэж, өөрчлөх боломжгүй аудитын мөр үлдээнэ."
        >
          <div className="fn-oversight-board">
            <div className="fn-oversight-board__top"><span><i /> УЛСЫН НӨӨЦ</span><b>14.2 хоног</b></div>
            <div className="fn-stock-bars">
              <div><span>АИ-92</span><i><em style={{width: "58%"}} /></i><b>11.6 өдөр</b></div>
              <div><span>Дизель</span><i><em style={{width: "81%"}} /></i><b>20.1 өдөр</b></div>
              <div><span>АИ-95</span><i><em style={{width: "66%"}} /></i><b>15.4 өдөр</b></div>
            </div>
            <div className="fn-oversight-alert"><AlertTriangle /><span><b>3 бүс анхаарах түвшинд</b><small>Дорнод · Ховд · Баянзүрх</small></span></div>
          </div>
        </PageHero>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="Хяналтын дөрвөн зорилт" title="Хямрал өнгөрсөн ч үнэ цэн нь үлдэнэ" />
            <FeatureGrid items={[
              {icon: DatabaseZap, title: "Нөөцийн аюулгүй байдал", body: "Бүтээгдэхүүн, бүс, импортлогч, терминал, ШТС-аар үлдэгдэл ба дуусах прогноз.", tag: "STOCK"},
              {icon: Scale, title: "Үнэ ба татвар", body: "Импортын өртөг, жижиглэн үнэ, НӨАТ, e-Barimt, хэвийн бус үнийн өөрчлөлтийн дохио.", tag: "MARKET"},
              {icon: ShieldCheck, title: "Чанар ба хэмжил зүй", body: "Лабораторийн паспорт, стандарт литр, усны хэмжээ, хошууны калибровк.", tag: "QUALITY"},
              {icon: BarChart3, title: "Алдагдал ба зөрүү", body: "Импорт–нөөц–түгээлтийн reconciliation, зөрүүний кейс, хариуцагч, шийдвэрлэлт.", tag: "CONTROL"},
            ]} />
          </div>
        </section>

        <section className="fn-section fn-section--soft">
          <div className="fn-container fn-role-layout">
            <div>
              <span className="fn-section-label">Role-based oversight</span>
              <h2>Хүн бүр бүхнийг харахгүй. Хариуцсан зүйлээ бүрэн харна.</h2>
              <p>Үндэсний, салбарын, бүсийн, байгууллагын эрхийг тусгаарлаж, харах ба өөрчлөх үйлдэл бүрийг аудитлана.</p>
            </div>
            <div className="fn-role-table">
              <div className="fn-role-table__head"><span>Түвшин</span><span>Харагдац</span><span>Үйлдэл</span></div>
              <div><b>Шуурхай штаб</b><span>Улс даяар</span><small>Горим · квот</small></div>
              <div><b>АҮЭБЯ</b><span>Нөөц · импорт</span><small>Бодлого · прогноз</small></div>
              <div><b>ТЕГ / ГЕГ</b><span>Гүйлгээ · гааль</span><small>Тулгалт · шалгалт</small></div>
              <div><b>Аймаг / дүүрэг</b><span>Өөрийн бүс</span><small>Дохиолол · кейс</small></div>
              <div><b>ШТС сүлжээ</b><span>Өөрийн салбар</span><small>Засвар · тайлбар</small></div>
            </div>
          </div>
        </section>

        <section className="fn-section fn-section--dark">
          <div className="fn-container fn-integrity-layout">
            <div>
              <span className="fn-section-label">Tamper evidence</span>
              <h2>Өгөгдөл өөрчлөгдвөл мөр нь үлдэнэ.</h2>
              <p>Эх төхөөрөмжөөс ирсэн үйл явдал бүр гарын үсэг, дарааллын дугаар, өмнөх бичлэгийн hash-тай. Засвар нь хуучныг устгахгүй, шинэ залруулах бичлэг үүсгэнэ.</p>
              <CheckList items={["Append-only үйл явдлын бүртгэл", "Edge төхөөрөмжийн түлхүүр ба баталгаажуулалт", "Hash chain ба өдөр тутмын баталгаажуулалт", "Идемпотент ingest ба replay хамгаалалт"]} />
            </div>
            <div className="fn-hash-chain">
              {["EDGE #042", "INGEST", "LEDGER", "AUDIT"].map((item, index) => <div key={item}><span>0{index + 1}</span><b>{item}</b><small>{index === 0 ? "signed event" : index === 3 ? "sealed" : "prev_hash + payload"}</small></div>)}
            </div>
          </div>
        </section>

        <section className="fn-section">
          <div className="fn-container fn-public-layout">
            <div><Eye /><span className="fn-section-label">Нийтийн ил тод байдал</span><h2>Хэрэгтэйг нь нийтэд. Хувийнхыг нь хамгаална.</h2></div>
            <div>
              <FeatureGrid items={[
                {icon: Landmark, title: "Нийтийн самбар", body: "Улсын ба бүсийн нөөцийн хоног, нээлттэй ШТС, ерөнхий нийлүүлэлтийн төлөв.", tag: "PUBLIC"},
                {icon: LockKeyhole, title: "Хувийн мэдээллийн хязгаар", body: "РД, VIN, байршлыг токенжуулж, зорилго ба хадгалалтын хугацааг хязгаарлана.", tag: "PRIVACY"},
                {icon: FileClock, title: "Тайлан ба түүх", body: "Бодлогын өөрчлөлт, нөөцийн хөдөлгөөн, хямралын шийдвэрийн огноотой тайлан.", tag: "REPORTING"},
              ]} />
            </div>
          </div>
        </section>

        <PageCTA title="Нэг самбарын өмнө нэг өгөгдлийн стандарт хэрэгтэй." body="Эрхийн матриц, тайлангийн түвшин, аудитын шаардлагыг нэвтрүүлэлтийн Фаз 0-д батална." />
      </main>
    </FuelNetShell>
  );
}
