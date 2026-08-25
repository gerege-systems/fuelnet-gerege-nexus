import type {Metadata} from "next";
import {Activity, Boxes, Cable, CheckCircle2, CloudCog, Database, Gauge, Map, RadioTower, ServerCog} from "lucide-react";

import FuelNetShell from "../../components/fuelnet/FuelNetShell";
import {FeatureGrid, PageHero, SectionHeading} from "../../components/fuelnet/ui";

export const metadata: Metadata = {title: "Нэвтрүүлэх төлөвлөгөө · FuelNet"};

const phases = [
  {phase: "ФАЗ 0", time: "0–3 долоо хоног", title: "Суурь ба бодит аудит", body: "ШТС, controller, ATG, протокол, сүлжээ, эрх зүйн шаардлагаа батална.", items: ["20–30 ШТС-ын тоног төхөөрөмжийн аудит", "Өгөгдлийн стандарт ба RBAC", "Туршилтын бүс, оператор, KPI сонгох"]},
  {phase: "ФАЗ 1", time: "3–10 долоо хоног", title: "POS ба харагдац", body: "FuelNet-ийн эхний бодит үнэ цэнэ — хошуу, сав, ээлжийн бодит өгөгдөл.", items: ["FuelNet Edge + POS MVP", "IFSF ба pulse/manual драйвер", "Нөөц, гүйлгээ, ээлжийн самбар"]},
  {phase: "ФАЗ 2", time: "10–18 долоо хоног", title: "Ваучер ба туршилтын бүс", body: "Бодит нөөцөөс эрх үүсгэх, хэрэглэгчид санал болгох, QR-аар олгох урсгал.", items: ["ХУР / e-Mongolia шалгалт", "Allocation ба policy engine", "Офлайн QR, SMS, USSD"]},
  {phase: "ФАЗ 3", time: "18–32 долоо хоног", title: "Улс даяар тэлэх", body: "Аймаг, дүүрэг, сүлжээгээр үе шаттай өргөжүүлж 90%-ийн хамралтад хүрнэ.", items: ["Брэндийн драйверуудын өргөтгөл", "24/7 ажиллагаа ба field support", "Төрийн бүх түвшний хяналтын самбар"]},
  {phase: "ФАЗ 4", time: "32+ долоо хоног", title: "Прогноз ба боловсронгуй хяналт", body: "Тайван үеийн байнгын дэд бүтэц болгон хөгжүүлж, эрсдэлийг урьдчилан харна.", items: ["Эрэлт ба дуусах хугацааны прогноз", "Залилангийн risk scoring", "Нийтийн ил тод байдал ба open data"]},
];

export default function RolloutPage() {
  return (
    <FuelNetShell>
      <main>
        <PageHero
          eyebrow="Нэвтрүүлэлт ба интеграц"
          title="Албадлагаар биш,"
          accent="үнэ цэнээр нэвтэрнэ."
          body="ШТС-д хэрэгтэй POS-оос эхэлж бодит өгөгдөл, харагдац, ваучер, төрийн хяналт руу хэмжигдэхүйц үе шат бүрээр тэлнэ."
        >
          <div className="fn-rollout-sequence">
            {[["POS", "01"], ["ӨГӨГДӨЛ", "02"], ["ХАРАГДАЦ", "03"], ["ВАУЧЕР", "04"], ["ХЯНАЛТ", "05"]].map(([name, index]) => <div key={name}><span>{index}</span><b>{name}</b></div>)}
          </div>
        </PageHero>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="32 долоо хоногийн замын зураг" title="Үе бүр өөрийн бодит үр дүнтэй" body="Улс даяар нэг дор асаахгүй. Тоног төхөөрөмжийн бодит нөхцөл, дата чанар, ачааллыг туршилт бүрээр баталгаажуулна." />
            <div className="fn-timeline">
              {phases.map((phase, index) => (
                <article key={phase.phase} className={index === 1 ? "is-highlight" : ""}>
                  <div className="fn-timeline__rail"><span>{index + 1}</span><i /></div>
                  <div className="fn-timeline__content">
                    <div className="fn-timeline__meta"><b>{phase.phase}</b><time>{phase.time}</time>{index === 1 ? <em>ЭХНИЙ БОДИТ ҮНЭ ЦЭН</em> : null}</div>
                    <h3>{phase.title}</h3><p>{phase.body}</p>
                    <ul>{phase.items.map((item) => <li key={item}><CheckCircle2 />{item}</li>)}</ul>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="fn-section fn-section--soft">
          <div className="fn-container">
            <SectionHeading label="Амжилтын хэмжүүр" title="Нэвтрүүлэлт бүр тоогоор баталгаажна" />
            <div className="fn-kpi-grid">
              {[
                ["< 15 мин", "Мэдээллийн саатал", "ШТС-ын 90%-д"],
                ["< 7 мин", "Дундаж хүлээлт", "Хямралын горимд"],
                ["< 0.5%", "Тулгалтын зөрүү", "Импорт vs түгээлт"],
                ["< 5 мин", "Бодлогын өөрчлөлт", "Код бичихгүйгээр"],
                ["> 99.5%", "Амжилттай хүсэлт", "Оргил ачаалалд"],
                ["Gini < 0.15", "Шударга хуваарилалт", "Бүсээр хэмжинэ"],
              ].map(([value, label, detail]) => <div key={label}><strong>{value}</strong><b>{label}</b><span>{detail}</span></div>)}
            </div>
          </div>
        </section>

        <section className="fn-section fn-section--dark">
          <div className="fn-container">
            <SectionHeading inverse label="Тэсвэртэй архитектур" title="Төв унасан ч ШТС ажилласаар байна" body="Гадаад системийг синхроноор хүлээхгүй, edge дээр ажлаа үргэлжлүүлж, холбогдмогц аюулгүй нөхөн синк хийнэ." />
            <div className="fn-architecture-stack">
              <div><RadioTower /><span><b>ШТС Edge</b><small>POS · SQLite · Pump / ATG drivers</small></span></div>
              <i />
              <div className="fn-architecture-stack__row">
                <div><CloudCog /><span><b>Ingest API</b><small>Идемпотент хүлээн авалт</small></span></div>
                <div><ServerCog /><span><b>Allocation</b><small>Event-driven worker</small></span></div>
                <div><Activity /><span><b>Oversight</b><small>Read model ба дохиолол</small></span></div>
              </div>
              <i />
              <div className="fn-architecture-stack__row">
                <div><Database /><span><b>PostgreSQL</b><small>Үндсэн бүртгэл</small></span></div>
                <div><Boxes /><span><b>NATS / Redis</b><small>Queue · lock · cache</small></span></div>
                <div><Gauge /><span><b>Timeseries</b><small>Сав · хошуу · GPS</small></span></div>
              </div>
            </div>
          </div>
        </section>

        <section className="fn-section">
          <div className="fn-container">
            <SectionHeading label="Фаз 0-д гаргах шийдвэр" title="Эхлэхийн өмнө батлах зургаан зүйл" />
            <FeatureGrid items={[
              {icon: Map, title: "Туршилтын бүс", body: "Нэг дүүрэг + нэг аймаг, өөр төрлийн тоног төхөөрөмжтэй 20–30 ШТС.", tag: "SCOPE"},
              {icon: Cable, title: "Драйверын эрэмбэ", body: "Аудитаар үйлдвэрлэгч, controller, физик протоколын бодит хувийг тогтооно.", tag: "HARDWARE"},
              {icon: Gauge, title: "Хүлцлийн босго", body: "Тээвэр, температур, сав, хошуу тус бүрийн зөвшөөрөгдөх зөрүүг батална.", tag: "POLICY"},
              {icon: CheckCircle2, title: "Ваучерын дүрэм", body: "Лимит, хугацаа, тэргүүлэх ангилал, бүс, мухардлаас сэргийлэх дүрэм.", tag: "ALLOCATION"},
              {icon: RadioTower, title: "Edge төхөөрөмж", body: "SBC эсвэл Android, порт, цахилгааны нөөц, SIM, алсын удирдлагын стандарт.", tag: "EDGE"},
              {icon: Activity, title: "Операторын загвар", body: "24/7 NOC, field support, SLA, сургалт, сэлбэг, ослын үеийн журам.", tag: "OPERATIONS"},
            ]} />
          </div>
        </section>

        <section className="fn-final-cta">
          <div className="fn-container">
            <span>ЭХНИЙ АЛХАМ</span>
            <h2>20–30 ШТС. 3 долоо хоног.<br />Нэг бодит аудит.</h2>
            <p>Төлөвлөгөөг тоног төхөөрөмж, өгөгдөл, эрх зүйн бодит нөхцөлтэй тулгаж FuelNet-ийн туршилтын хүрээг батална.</p>
            <div className="fn-final-cta__meta"><span><CheckCircle2 /> Төхөөрөмжийн матриц</span><span><CheckCircle2 /> Өгөгдлийн гэрээ</span><span><CheckCircle2 /> Туршилтын KPI</span></div>
          </div>
        </section>
      </main>
    </FuelNetShell>
  );
}
