"use client";

import {useEffect, useRef, useState} from "react";
import Link from "next/link";
import {Fuel, Menu, X} from "lucide-react";
import {usePathname} from "next/navigation";

const navigation = [
  {href: "/supply", label: "Урсгал"},
  {href: "/stations", label: "ШТС ба POS"},
  {href: "/vouchers", label: "Ваучер"},
  {href: "/oversight", label: "Хяналт"},
  {href: "/rollout", label: "Нэвтрүүлэлт"},
];

export default function FuelNetHeader() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const header = useRef<HTMLElement>(null);

  useEffect(() => setOpen(false), [pathname]);
  useEffect(() => {
    if (!open) return;
    const close = (event: KeyboardEvent) => event.key === "Escape" && setOpen(false);
    document.addEventListener("keydown", close);
    return () => document.removeEventListener("keydown", close);
  }, [open]);

  return (
    <header className="fn-header" ref={header}>
      <div className="fn-container fn-header__inner">
        <Link href="/" className="fn-brand" aria-label="FuelNet нүүр хуудас">
          <span className="fn-brand__mark"><Fuel /></span>
          <span><b>FUELNET</b><small>Шатахууны нэгдсэн сүлжээ</small></span>
        </Link>
        <nav className={`fn-nav${open ? " is-open" : ""}`} aria-label="Үндсэн цэс">
          {navigation.map((item) => (
            <Link key={item.href} href={item.href} className={pathname === item.href ? "is-active" : ""}>
              {item.label}
            </Link>
          ))}
          <Link href="/login" className="fn-nav__login">Нэвтрэх</Link>
        </nav>
        <Link href="/login" className="fn-header__login">Платформд нэвтрэх</Link>
        <button
          className="fn-menu-button"
          type="button"
          aria-label={open ? "Цэс хаах" : "Цэс нээх"}
          aria-expanded={open}
          onClick={() => setOpen((value) => !value)}
        >
          {open ? <X /> : <Menu />}
        </button>
      </div>
    </header>
  );
}
