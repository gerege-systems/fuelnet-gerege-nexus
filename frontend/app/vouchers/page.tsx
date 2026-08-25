import type {Metadata} from "next";
import {BellRing, CarFront, Clock3, Fingerprint, Gauge, MapPin, QrCode, ShieldCheck, SignalZero, Users} from "lucide-react";

import FuelNetShell from "../../components/fuelnet/FuelNetShell";
import {FeatureGrid, PageCTA, PageHero, SectionHeading} from "../../components/fuelnet/ui";

export const metadata: Metadata = {title: "Ваучер ба хуваарилалт · FuelNet"};

const allocationSteps = [
  ["01", "Хангалт баталгаажна", "ATG савны бодит өсөлтийг мэдээлж, ачсан хэмжээтэй тулгана."],
  ["02", "Боломжит литр тооцно", "Dead stock, өмнөх ваучер, тэргүүлэх ангиллын нөөцийг хасна."],
  ["03", "Хэрэглэгч эрэмбэлнэ", "Ойр байдал, хүлээсэн хугацаа, бүсийн хомсдол, хэрэглээг жинлэнэ."],
  ["04", "Цагийн цонх санал болгоно", "15 минутад баталгаажуулж, нэг ШТС-д уягдсан QR авна."],
  ["05", "Хошуугаар баталгаажна", "Redeem хийсэн литр ба түгээгүүрийн бодит гүйлгээг тулгана."],
];

export default function VouchersPage() {
  return (
    <FuelNetShell>
      <main>
        <PageHero
          eyebrow="Ваучер ба хуваарилалт"
          title="Дараалал биш,"
          accent="баталгаатай цагийн цонх."
          body="FuelNet бодитоор орж ирсэн түлшийг нөөцлөгдсөн литрийн эрх болгон, ойр байршил ба хэрэгцээгээр шударга хуваарилна."
        >
          <div className="fn-voucher-card">
            <div className="fn-voucher-card__top"><span>FUELNET VOUCHER</span><strong>RESERVED</strong></div>
            <div className="fn-voucher-card__main"><div className="fn-fake-qr"><QrCode /></div><div><small>АИ-92 · ШТС #042</small><b>20.9 литр</b><span>Өнөөдөр 18:00–18:30</span></div></div>
            <div className="fn-voucher-card__foot"><span>УБ • Баянзүрх</span><b>₮50,000 хүртэл</b></div>
          </div>
        </PageHero>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="Автомат үүсэлтийн гинж" title="Нөөц орж ирэхэд эрх үүснэ" body="Амлалт нөөцөөс давахгүй. Ашиглаагүй слот хугацаа дуусмагц дараагийн хэрэглэгчид автоматаар шилжинэ." />
            <div className="fn-allocation-flow">
              {allocationSteps.map(([index, title, body], i) => (
                <article key={index}><span>{index}</span><div><h3>{title}</h3><p>{body}</p></div>{i < allocationSteps.length - 1 ? <i /> : null}</article>
              ))}
            </div>
          </div>
        </section>

        <section className="fn-section fn-section--soft">
          <div className="fn-container fn-policy-layout">
            <div>
              <span className="fn-section-label">Policy engine</span>
              <h2>Лимитийг код бичихгүйгээр өөрчилнө.</h2>
              <p>Бүтээгдэхүүн, бүс, хугацаа, тээврийн хэрэгслийн ангилал, тэргүүлэх бүлгээр бодлогыг 5 минутаас бага хугацаанд идэвхжүүлнэ.</p>
            </div>
            <div className="fn-policy-card">
              <div><span>Нэг удаагийн лимит</span><b>₮50,000</b></div>
              <div><span>Давтамж</span><b>24 цагт 1</b></div>
              <div><span>7 хоногийн дээд</span><b>₮150,000</b></div>
              <div><span>Хүчинтэй хугацаа</span><b>6 цаг</b></div>
              <div><span>Гео бүс</span><b>Аймаг / дүүрэг</b></div>
              <div><span>Идэвхтэй бүтээгдэхүүн</span><b>АИ-92</b></div>
            </div>
          </div>
        </section>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="Шударга ба тэсвэртэй" title="Хэрэглэгч бүрт хүрэх олон суваг" />
            <FeatureGrid items={[
              {icon: MapPin, title: "Ойр байршил", body: "H3 бүсчлэлээр хамгийн ойрын хангалттай ШТС-ыг санал болгоно.", tag: "GEO"},
              {icon: Clock3, title: "Мухардалгүй дараалал", body: "48 цаг эрх аваагүй хэрэглэгч автоматаар хамгийн өндөр эрэмбэд орно.", tag: "FAIRNESS"},
              {icon: BellRing, title: "Шаталсан мэдэгдэл", body: "Сандралын оргил ачааллаас сэргийлж мэдэгдлийг 30–120 секундээр тараана.", tag: "LOAD"},
              {icon: SignalZero, title: "Офлайн QR", body: "Ed25519 гарын үсгийг локал шалгаж, нэг ШТС-д л хүчинтэй байлгана.", tag: "OFFLINE"},
              {icon: Fingerprint, title: "SMS ба USSD", body: "Аппгүй хэрэглэгч 8 оронтой код, USSD, операторын хайлтаар эрхээ ашиглана.", tag: "ACCESS"},
              {icon: Users, title: "Тэргүүлэх ангилал", body: "Түргэн, гал, цагдаа, нийтийн тээвэр, хүнс, эм, ХАА тусдаа квоттой.", tag: "PRIORITY"},
            ]} />
          </div>
        </section>

        <section className="fn-section fn-section--dark">
          <div className="fn-container fn-antifraud">
            <div><ShieldCheck /><span className="fn-section-label">Залилангийн эсрэг</span><h2>Нэг хүн, нэг машин, нэг бодит түгээлт.</h2></div>
            <div className="fn-antifraud__grid">
              <article><CarFront /><h3>Тээврийн хэрэгслийн танигч</h3><p>Утасны дугаар биш, VIN/арлын дугаар + эзэмшигчийн РД үндсэн танигч болно.</p></article>
              <article><Gauge /><h3>Хошууны нотолгоо</h3><p>Redeem хийсэн ваучерийг totalizer, урсгалын профиль, ээлжийн касстай тулгана.</p></article>
              <article><Fingerprint /><h3>Давхар ашиглалтаас хамгаалах</h3><p>QR нь нэг ШТС-д уягдаж, офлайн үед локал bloom filter ба аудитын жагсаалт ашиглана.</p></article>
            </div>
          </div>
        </section>

        <PageCTA title="Шударга хуваарилалт бодит нөөцөөс эхэлнэ." body="POS ба ATG өгөгдөл бэлэн болмогц ваучерийн туршилтыг нэг дүүрэг, нэг аймгаас эхлүүлнэ." />
      </main>
    </FuelNetShell>
  );
}
