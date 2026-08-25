import type {ReactNode} from "react";
import Link from "next/link";
import {ArrowRight, Check, Fuel, Ship, TestTube2, Truck, Warehouse} from "lucide-react";

export function SectionHeading({label, title, body, inverse = false}: {label: string; title: string; body?: string; inverse?: boolean}) {
  return (
    <div className={`fn-section-heading${inverse ? " is-inverse" : ""}`}>
      <span>{label}</span>
      <h2>{title}</h2>
      {body ? <p>{body}</p> : null}
    </div>
  );
}

export function FlowRail({compact = false}: {compact?: boolean}) {
  const steps = [
    [Ship, "Импорт", "Гэрээ · Ачилт"],
    [TestTube2, "Хил", "Гааль · Чанар"],
    [Warehouse, "Терминал", "Сав · Нөөц"],
    [Truck, "Тээвэр", "GPS · Цахим лац"],
    [Fuel, "ШТС", "Сав · Хошуу"],
    [Check, "Түгээлт", "Баримт · Тулгалт"],
  ] as const;

  return (
    <div className={`fn-flow-rail${compact ? " is-compact" : ""}`}>
      {steps.map(([Icon, title, body], index) => (
        <div className="fn-flow-step" key={title}>
          <span className="fn-flow-step__icon"><Icon /></span>
          <span><b>{title}</b><small>{body}</small></span>
          {index < steps.length - 1 ? <i className="fn-flow-step__line"><em /></i> : null}
        </div>
      ))}
    </div>
  );
}

export function PageHero({eyebrow, title, accent, body, children}: {eyebrow: string; title: string; accent: string; body: string; children?: ReactNode}) {
  return (
    <section className="fn-page-hero">
      <div className="fn-page-hero__grid" />
      <div className="fn-container fn-page-hero__inner">
        <div>
          <div className="fn-kicker"><span /> {eyebrow}</div>
          <h1>{title} <em>{accent}</em></h1>
          <p>{body}</p>
        </div>
        {children ? <div className="fn-page-hero__visual">{children}</div> : null}
      </div>
    </section>
  );
}

export function FeatureGrid({items}: {items: {icon: typeof Fuel; title: string; body: string; tag?: string}[]}) {
  return (
    <div className="fn-feature-grid">
      {items.map(({icon: Icon, title, body, tag}) => (
        <article className="fn-feature-card" key={title}>
          <div className="fn-feature-card__icon"><Icon /></div>
          {tag ? <span>{tag}</span> : null}
          <h3>{title}</h3>
          <p>{body}</p>
        </article>
      ))}
    </div>
  );
}

export function PageCTA({title, body, href = "/rollout", action = "Нэвтрүүлэх төлөвлөгөө"}: {title: string; body: string; href?: string; action?: string}) {
  return (
    <section className="fn-page-cta">
      <div className="fn-container">
        <div><h2>{title}</h2><p>{body}</p></div>
        <Link href={href} className="fn-button fn-button--light">{action} <ArrowRight /></Link>
      </div>
    </section>
  );
}

export function CheckList({items}: {items: string[]}) {
  return <ul className="fn-check-list">{items.map((item) => <li key={item}><Check />{item}</li>)}</ul>;
}
