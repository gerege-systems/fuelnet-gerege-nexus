import type {Metadata} from "next";
import {Activity, Cable, CreditCard, Fuel, Gauge, MonitorSmartphone, ReceiptText, RefreshCcw, WifiOff} from "lucide-react";

import FuelNetShell from "../../components/fuelnet/FuelNetShell";
import {CheckList, FeatureGrid, PageCTA, PageHero, SectionHeading} from "../../components/fuelnet/ui";

export const metadata: Metadata = {title: "ШТС ба FuelNet POS"};

export default function StationsPage() {
  return (
    <FuelNetShell>
      <main>
        <PageHero
          eyebrow="ШТС ба FuelNet POS"
          title="Хошууны бодит заалт бол"
          accent="платформын суурь."
          body="FuelNet POS нь орчин үеийн controller-оос механик тоолуур хүртэлх ШТС-ыг нэг дотоод интерфейст холбож, интернэт тасарсан ч борлуулалтыг зогсоохгүй."
        >
          <div className="fn-pos-mock">
            <div className="fn-pos-mock__head"><span><i /> Ээлж нээлттэй</span><b>ШТС #042</b></div>
            <div className="fn-pos-nozzles">
              <div><span>01</span><b>АИ-92</b><strong>2,390₮</strong><small>Түгээж байна · 18.4 л</small></div>
              <div><span>02</span><b>Дизель</b><strong>3,140₮</strong><small>Бэлэн</small></div>
            </div>
            <div className="fn-pos-mock__foot"><WifiOff /> Offline queue · 0 хүлээгдэж байна</div>
          </div>
        </PageHero>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="ШТС-ын ажлын орон зай" title="Нэг дэлгэцээс бүх ээлжээ удирдана" body="Касс, хошуу, ваучер, төлбөр, баримт, ээлжийн тулгалтыг салангид програмд давхар оруулахгүй." />
            <FeatureGrid items={[
              {icon: Fuel, title: "Хошуу удирдах", body: "Төлөв харах, дүн эсвэл литрийн лимитээр authorize хийх, гүйлгээ сэргээх.", tag: "FORECOURT"},
              {icon: Gauge, title: "Савны түвшин", body: "Түлшний төрөл, ус, температур, дуусах хугацааны прогноз, огцом бууралтын дохиолол.", tag: "ATG"},
              {icon: CreditCard, title: "Бүх төлбөр", body: "Бэлэн, карт, QR, ваучер, байгууллагын дансны төлбөрийг нэг гүйлгээнд.", tag: "PAYMENT"},
              {icon: ReceiptText, title: "e-Barimt", body: "НӨАТ-ын баримт, буцаан олголт, кассын хаалт, борлуулалтын тайлан.", tag: "BILLING"},
              {icon: WifiOff, title: "Жинхэнэ офлайн", body: "Локал SQLite, гарын үсэгтэй QR шалгалт, холбогдмогц идемпотент нөхөн синк.", tag: "EDGE"},
              {icon: Activity, title: "Ээлжийн тулгалт", body: "Нээлт/хаалтын totalizer, касс, ваучер, бодит түгээлтийн зөрүүг ажилтнаар нь харна.", tag: "SHIFT"},
            ]} />
          </div>
        </section>

        <section className="fn-section fn-section--dark">
          <div className="fn-container fn-adapter-layout">
            <div>
              <span className="fn-section-label">Universal pump adapter</span>
              <h2>Брэнд бүрт шинэ систем биш. Драйвер бүрт нэг гэрээ.</h2>
              <p>FuelNet-ийн цөм нэг pump.Driver интерфейстэй. Шинэ үйлдвэрлэгч нэмэхэд POS, ваучер, тайлангийн цөм өөрчлөгдөхгүй.</p>
              <CheckList items={["IFSF LON ба TCP/IP", "RS-485 / RS-422 / RS-232", "Current loop ба Two-Wire", "Импульс тоолуур ба гар оруулалт"]} />
            </div>
            <div className="fn-adapter-stack">
              <div className="fn-adapter-core"><MonitorSmartphone /><span><b>FuelNet POS</b><small>Нэг дотоод интерфейс</small></span></div>
              <div className="fn-adapter-bus"><span /><Cable /><span /></div>
              <div className="fn-brand-chips">
                {["IFSF", "Gilbarco", "Wayne", "Tokheim", "Tatsuno", "Censtar", "Sanki", "Adast", "Petrotec", "Pulse"].map((brand) => <span key={brand}>{brand}</span>)}
              </div>
            </div>
          </div>
        </section>

        <section className="fn-section">
          <div className="fn-container fn-value-layout">
            <div><span className="fn-section-label">Нэвтрэлтийн хөшүүрэг</span><h2>ШТС-д зардал биш, ашигтай бүтээгдэхүүн.</h2></div>
            <div className="fn-value-list">
              {[
                [RefreshCcw, "Гар ажиллагаа багасна", "Ээлж хаалт, үнэ шинэчлэлт, тайлан, баримт автомат болно."],
                [Activity, "Алдагдал эрт харагдана", "Сав–хошуу–кассын зөрүүг өдөр дуусахаас өмнө илрүүлнэ."],
                [CreditCard, "Борлуулалт нэмэгдэнэ", "Бодит хангалтаа бүртгэсэн ШТС-д хэрэглэгчийн ваучер үүснэ."],
              ].map(([Icon, title, body]) => {
                const ValueIcon = Icon as typeof Fuel;
                return <article key={title as string}><ValueIcon /><div><h3>{title as string}</h3><p>{body as string}</p></div></article>;
              })}
            </div>
          </div>
        </section>

        <PageCTA title="POS → өгөгдөл → харагдац." body="ШТС-ын тоног төхөөрөмжийн аудит, драйверын эрэмбэ, туршилтын суурилуулалтаас эхэлнэ." />
      </main>
    </FuelNetShell>
  );
}
