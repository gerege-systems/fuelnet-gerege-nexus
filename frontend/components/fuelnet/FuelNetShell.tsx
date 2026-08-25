import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowUpRight, Fuel } from "lucide-react";

import FuelNetHeader from "./FuelNetHeader";

export default function FuelNetShell({children}: {children: ReactNode}) {
  return (
    <div className="fn-site">
      <FuelNetHeader />
      {children}
      <footer className="fn-footer">
        <div className="fn-container fn-footer__top">
          <Link href="/" className="fn-brand fn-brand--footer" aria-label="FuelNet нүүр хуудас">
            <span className="fn-brand__mark"><Fuel /></span>
            <span><b>FUELNET</b><small>Шатахууны нэгдсэн сүлжээ</small></span>
          </Link>
          <p>Монгол Улсын шатахууны урсгал, эрэлт нийлүүлэлтийн нэгдсэн платформ.</p>
          <div className="fn-footer__links">
            <Link href="/map">Газрын зураг</Link>
            <Link href="/supply">Урсгал</Link>
            <Link href="/stations">ШТС</Link>
            <Link href="/vouchers">Ваучер</Link>
            <Link href="/oversight">Хяналт</Link>
            <Link href="/rollout">Нэвтрүүлэлт</Link>
          </div>
        </div>
        <div className="fn-container fn-footer__bottom">
          <span>© 2026 Gerege Systems · FuelNet</span>
          <span>Төр–хувийн түншлэлийн дэд бүтэц</span>
          <Link href="/login">Платформд нэвтрэх <ArrowUpRight /></Link>
        </div>
      </footer>
    </div>
  );
}
