import type {Metadata} from "next";
import {BadgeCheck, FileText, TestTube2, Thermometer, Truck, Warehouse} from "lucide-react";

import FuelNetShell from "../../components/fuelnet/FuelNetShell";
import {FeatureGrid, FlowRail, PageCTA, PageHero, SectionHeading} from "../../components/fuelnet/ui";

export const metadata: Metadata = {title: "Нийлүүлэлтийн урсгал · FuelNet"};

const tracePoints = [
  ["01", "Гэрээ ба захиалга", "Импортлогч, нийлүүлэгч, төрөл, тонн, үнэ, валют, хүлээгдэж буй огноо"],
  ["02", "Ачилт ба хил", "Вагон, партийн дугаар, гарал үүсэл, гаалийн мэдүүлэг, HS код, татвар"],
  ["03", "Чанар ба хэмжил зүй", "Октан, нягт, хүхэр, ус, температурын 15°C залруулга"],
  ["04", "Терминал ба стратегийн нөөц", "Савны түвшин, чөлөөт багтаамж, улсын нөөцөд шилжсэн хэмжээ"],
  ["05", "Тээвэр ба цахим лац", "Автоцистерн, жолооч, GPS замнал, зөрсөн зогсолт, ETA"],
  ["06", "ШТС-ын хүлээн авалт", "Ачсан ба хүлээн авсан литрийн зөрүү, ATG баталгаажуулалт"],
];

export default function SupplyPage() {
  return (
    <FuelNetShell>
      <main>
        <PageHero
          eyebrow="Нийлүүлэлтийн урсгал"
          title="Эх үүсвэрээс хошуу хүртэл"
          accent="тасралтгүй мөрдөнө."
          body="Импортын гэрээ, гааль, чанар, нөөц, тээвэр, ШТС-ын хүлээн авалтыг нэг парти ба нэг хэмжилтийн түүхээр холбоно."
        >
          <div className="fn-hero-equation">
            <span>ИМПОРТ</span><b>−</b><span>НӨӨЦ</span><b>−</b><span>ТҮГЭЭЛТ</span><strong>= 0 ± 0.5%</strong>
            <small>Автомат тулгалтын зорилтот зөрүү</small>
          </div>
        </PageHero>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="Chain of custody" title="Литрийн 17 цэгийн түүх" body="Зангилаа бүр өмнөх ба дараагийн хэмжилттэйгээ холбогдоно. Зөрүү гарвал хэзээ, хаана, хэний хариуцлагад үүссэнийг шууд тогтооно." />
            <FlowRail />
            <div className="fn-trace-grid">
              {tracePoints.map(([index, title, body]) => (
                <article key={index}><span>{index}</span><div><h3>{title}</h3><p>{body}</p></div></article>
              ))}
            </div>
          </div>
        </section>

        <section className="fn-section fn-section--soft">
          <div className="fn-container">
            <SectionHeading label="Хэмжилт ба баталгаа" title="Таамгийг хэмжилтээр солино" />
            <FeatureGrid items={[
              {icon: FileText, title: "Импортын нэг бүртгэл", body: "Гэрээ, ачилт, гаалийн мэдүүлэг, төлбөрийг нэг партийн дугаараар холбоно.", tag: "SUPPLY"},
              {icon: TestTube2, title: "Чанарын паспорт", body: "Итгэмжлэгдсэн лабораторийн октан, нягт, хүхэр, усны сорилтыг партитай уяна.", tag: "QUALITY"},
              {icon: Thermometer, title: "15°C залруулга", body: "Температур, нягтаас стандарт литр рүү шилжүүлж хэмжилтийн зөрүүг бодит болгоно.", tag: "METROLOGY"},
              {icon: Warehouse, title: "Савны бодит үлдэгдэл", body: "ATG түвшин, ус, температур, чөлөөт багтаамж, дуусах прогнозыг 5 минут тутам авна.", tag: "DEPOT"},
              {icon: Truck, title: "GPS ба цахим лац", body: "Тээврийн даалгавар, замнал, зөрсөн зогсолт, лацны төлөв, очих хугацааг хянана.", tag: "LOGISTICS"},
              {icon: BadgeCheck, title: "Хүлээн авалтын нотолгоо", body: "Ачсан ба хүлээн авсан литрийг ATG өсөлт, ажилтны баталгаажуулалттай тулгана.", tag: "RECEIPT"},
            ]} />
          </div>
        </section>

        <section className="fn-section">
          <div className="fn-container fn-split-panel">
            <div>
              <span className="fn-section-label">Reconciliation engine</span>
              <h2>Зөрүү бол тайлангийн төгсгөл биш, ажлын эхлэл.</h2>
              <p>FuelNet импорт, терминал, тээвэр, ШТС, хошууны бүх хэмжилтийг автоматаар тулгаж, хүлцлээс давсан зөрүүнд шалгах кейс нээнэ.</p>
            </div>
            <div className="fn-reconcile-table">
              <div><span>Ачилтын баримт</span><b>32,000 л</b></div>
              <div><span>15°C залруулсан хэмжээ</span><b>31,918 л</b></div>
              <div><span>ШТС хүлээн авсан</span><b>31,840 л</b></div>
              <div className="is-ok"><span>Зөрүү</span><b>−0.24% · ХЭВИЙН</b></div>
            </div>
          </div>
        </section>

        <PageCTA title="Урсгал харагдаж эхлэх цэг нь хэмжилт." body="Терминал, тээвэр, ШТС-ын аудит ба холболтыг үе шаттай эхлүүлнэ." />
      </main>
    </FuelNetShell>
  );
}
