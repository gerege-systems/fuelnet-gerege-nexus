"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Boxes, FileText, Landmark, LayoutGrid, MonitorCog, Package, PenTool, Receipt, ScanLine, ShieldCheck, Users, Wallet } from "lucide-react";
import { invokeShell, useShell, SHELL_METHODS } from "@/lib/shell";
import { LINES, isLine, type LineContent } from "./lines";

const ICONS: Record<string, React.ReactNode> = {
  grid: <LayoutGrid />, file: <FileText />, pen: <PenTool />, landmark: <Landmark />,
  users: <Users />, monitor: <MonitorCog />, wallet: <Wallet />, boxes: <Boxes />,
  shield: <ShieldCheck />, package: <Package />, scan: <ScanLine />, receipt: <Receipt />,
};

/** `device.identity`-ийн хариу. Талбар бүр байхгүй байж болно. */
interface DeviceIdentity {
  id?: string;
  name?: string;
  site?: string;
  form_factor?: string;
}

const EMPTY = "—";

/**
 * Шугамын нүүр дэлгэц.
 *
 * Энэ бол маркетингийн хуудас БИШ. Энэ нь native хүрээний дотор, ribbon болон
 * rail-ын дунд рендерлэгддэг тул өөрийн толгой хэсэг, навигаци зурахгүй —
 * тэдгээрийг бүрхүүл эзэмшинэ.
 *
 * Хуудас нь нэвтрэлт шаардахгүй: ажлын мужид web-ийн нэвтрэх дэлгэц гарч
 * ирэхийг орлохын тулд байгаа юм. Session байхгүй бол гэрэгэ нь "олгоогүй"
 * гэж уншигдана, харин дэлгэц өөрөө хэвээр зогсоно.
 */
export default function LineHomePage() {
  const params = useParams<{ line: string }>();
  const { shell } = useShell();
  const [identity, setIdentity] = useState<DeviceIdentity | null>(null);
  const [host, setHost] = useState("");

  useEffect(() => setHost(window.location.host), []);
  useEffect(() => {
    if (!shell) return;
    let cancelled = false;
    void invokeShell<DeviceIdentity>(SHELL_METHODS.DEVICE_IDENTITY, {}).then((result) => {
      // Бүрхүүл `device.identity`-г зарлаагүй бол reject ирнэ — тэр үед гэрэгэ
      // нь бүрхүүлээс мэдэх зүйлээрээ (платформ, хэлбэр, гэрээ) уншигдана.
      if (!cancelled && result.ok) setIdentity(result.value);
    });
    return () => { cancelled = true; };
  }, [shell]);

  const line = isLine(params.line) ? params.line : null;
  if (!line) {
    return <div className="line-home line-home--unknown">
      <p className="line-eyebrow">ГЭРЭГЭ</p>
      <h1 className="line-title">Энэ шугам бүртгэлгүй</h1>
      <p className="line-lede">
        Хаяг нь <code>{params.line}</code> гэсэн шугамыг заасан байна. Бүртгэлтэй шугамууд
        <code> native-apps/shared/device_lines.json</code> дотор байна.
      </p>
    </div>;
  }

  const content: LineContent = LINES[line];
  const formFactor = identity?.form_factor || shell?.formFactor || EMPTY;

  return (
    <div
      className={`line-home line-home--${content.posture}`}
      style={{ ["--line-alloy" as string]: content.alloy, ["--line-alloy-rgb" as string]: content.alloyRGB }}
    >
      <header className="line-mast">
        <p className="line-eyebrow">{content.eyebrow}</p>
        <h1 className="line-title">{content.title}</h1>
        <p className="line-lede">{content.lede}</p>
      </header>

      <div className="line-body">
        {/*
          Гэрэгэ — энэ аппын нэрийг үүрсэн эд. Монголын эзэнт гүрэн элчдээ
          олгодог байсан төмөр пайз: эзэмшигчийн эрх, хаана хүчинтэйг нь сийлж
          бичсэн байдаг. Төхөөрөмжийн бүртгэл яг үүнтэй ижил зүйл хийдэг тул
          энд түүнийг чимэг биш, төхөөрөмжийн үнэмлэх болгож харууллаа.
        */}
        <figure className="paiza" aria-label="Энэ төхөөрөмжид олгосон гэрэгэ">
          <span className="paiza-cord" aria-hidden="true" />
          <div className="paiza-face">
            <p className="paiza-mark">ГЭРЭГЭ</p>
            <dl className="paiza-inscription">
              <div><dt>Шугам</dt><dd>{host || EMPTY}</dd></div>
              <div><dt>Хэлбэр</dt><dd>{formFactor}</dd></div>
              <div><dt>Байршил</dt><dd>{identity?.site || EMPTY}</dd></div>
              <div><dt>Дугаар</dt><dd>{identity?.id || EMPTY}</dd></div>
            </dl>
            <p className="paiza-seal">
              {shell ? `Гэрээ v${shell.version}` : "Олгоогүй"}
            </p>
          </div>
          <figcaption className="paiza-note">
            {shell
              ? "Энэ төхөөрөмж бүрхүүлээр нэвтэрсэн. Дугаар, байршлыг Тохиргоо → Төхөөрөмж дээрээс бүртгэнэ."
              : "Хөтчөөр нээсэн байна. Гэрэгэ нь зөвхөн native бүрхүүлд олгогдоно."}
          </figcaption>
        </figure>

        <nav className="line-actions" aria-label="Хаанаас эхлэх">
          <p className="line-actions-head">Хаанаас эхлэх</p>
          <ul>
            {content.actions.map((action) => (
              <li key={action.label + action.href}>
                <Link href={action.href} className="line-action">
                  <span className="line-action-icon" aria-hidden="true">{ICONS[action.icon]}</span>
                  <span className="line-action-text">
                    <strong>{action.label}</strong>
                    <small>{action.hint}</small>
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      </div>
    </div>
  );
}
