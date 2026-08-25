"use client";

import Link from "next/link";
import {useEffect, useState} from "react";
import {ArrowRight, FileSignature} from "lucide-react";

import EIDLogin from "@/components/EIDLogin";
import {useAccess} from "@/lib/access";
import {contracts, InboxItem} from "@/lib/contracts";
import {useI18n} from "@/lib/i18n";

/**
 * The first screen: what the platform is, and the eID panel to act on it.
 *
 * The headline is about the platform rather than the sign-in, because that is
 * the question a visitor arrives with. The sign-in panel still sits beside it
 * rather than behind the header button: the shortest path from landing to
 * signed-in is worth keeping even when it is no longer the argument.
 *
 * `seeMoreAnchor` is where the second button goes — the first section below
 * this one, decided by the page rather than written in here, because a
 * deployment may not render the section this used to point at. Absent, there
 * is nothing below the hero and the button is not drawn: a page whose only
 * call to action scrolls nowhere is worse than one with a single button.
 *
 * `localSignIn` is whether this deployment signs people in itself. False means
 * it is a client of somebody else's provider, and then the eID card here is a
 * form that cannot submit: the endpoints behind it sit past `requireLocalLogin`
 * and answer 403 — a visitor types a registration number, presses the button
 * and nothing happens. So the card goes, and the first button becomes what the
 * header's already is: a link to /login, which knows to hand the visitor on.
 */
export default function Hero({
  seeMoreAnchor,
  localSignIn = true,
}: {seeMoreAnchor?: string; localSignIn?: boolean}) {
  const {t} = useI18n();
  // Нэвтэрсэн хүнд энэ хуудас ӨӨР асуултад хариулна: «надад юу ирсэн бэ».
  // Нөхцөл нь зөвхөн `me` — эхний client render дээр null тул сервертэй яг
  // ижил markup гарна (hydration зөрөхгүй), нэвтрээгүй зочинд юу ч
  // өөрчлөгдөхгүй.
  const {me} = useAccess();

  return (
    <section className="gp-hero">
      <div className="gp-pattern" />
      <div className={`gp-hero__inner${localSignIn ? "" : " gp-hero__inner--solo"}`}>
        <div className="gp-copy">
          <span className="gp-eyebrow">
            <i /> OPEN SOURCE · APACHE 2.0 · GO
          </span>
          <h1>
            {t("website.view.hero_title_lead")} <em>{t("website.view.hero_title_highlight")}</em>{" "}
            {t("website.view.hero_title_tail")}
          </h1>
          <p>{t("website.view.hero_lede")}</p>
          <div className="gp-cta">
            {me ? (
              <Link href="/module/documents/inbox" className="gp-gold gp-gold--large">
                {t("website.action.my_contracts")} <ArrowRight />
              </Link>
            ) : localSignIn ? (
              <a href="#eid-login" className="gp-gold gp-gold--large">
                {t("website.action.eid_sign_in")} <ArrowRight />
              </a>
            ) : (
              <Link href="/login" className="gp-gold gp-gold--large">
                {t("website.action.sign_in")} <ArrowRight />
              </Link>
            )}
            {me ? (
              <Link href="/apps" className="gp-outline">
                {t("website.action.open_platform")}
              </Link>
            ) : seeMoreAnchor ? (
              <a href={`#${seeMoreAnchor}`} className="gp-outline">
                {t("website.action.see_features")}
              </a>
            ) : null}
          </div>
          <div className="gp-stats">
            <span>
              <b>{t("website.stat.apps_count")}</b>
              {t("website.stat.apps")}
            </span>
            <span>
              <b>{t("website.stat.languages_count")}</b>
              {t("website.stat.languages")}
            </span>
            <span>
              <b>{t("website.stat.binary_count")}</b>
              {t("website.stat.binary")}
            </span>
          </div>
        </div>
        {me ? (
          <div className="gp-login-slot">
            <HeroInbox />
          </div>
        ) : localSignIn ? (
          <div id="eid-login" className="gp-login-slot">
            <EIDLogin compact />
          </div>
        ) : null}
      </div>
    </section>
  );
}

/**
 * Нэвтэрсэн хүний ИРСЭН ГЭРЭЭ, eID картын суусан яг тэр нүдэнд.
 *
 * Захирал eID-ээрээ орж ирээд өөр юу ч хайхгүй: гэрээ нь нүүрэн дээр нь
 * байна. Жагсаалт нь хариу хүлээж буй гэрээ л — түүх биш: нүүр хуудас бол
 * ажлын ширээ, архив нь Ирсэн гэрээ дэлгэцэд.
 */
function HeroInbox() {
  const {t} = useI18n();
  const [items, setItems] = useState<InboxItem[] | null>(null);

  useEffect(() => {
    let alive = true;
    contracts
      .inbox(false)
      .then((res) => alive && setItems(res.items))
      // Documents апп суугаагүй, эсвэл эрхгүй — карт мэндчилгээ болно.
      .catch(() => alive && setItems([]));
    return () => {
      alive = false;
    };
  }, []);

  return (
    <div className="rounded-2xl bg-white/95 shadow-xl border border-slate-200 p-6 w-full max-w-md">
      <div className="flex items-center gap-2 mb-4">
        <FileSignature className="w-5 h-5 text-indigo-600" />
        <h3 className="font-bold text-slate-900">{t("website.view.hero_inbox_title")}</h3>
      </div>
      {items === null ? (
        <p className="text-sm text-slate-400">…</p>
      ) : items.length === 0 ? (
        <p className="text-sm text-slate-500">{t("website.view.hero_inbox_empty")}</p>
      ) : (
        <ul className="space-y-2">
          {items.slice(0, 4).map((item) => (
            <li key={item.party_id}>
              <Link
                href={`/module/documents/inbox/${item.party_id}`}
                className="block rounded-xl border border-slate-200 hover:border-indigo-300 hover:bg-indigo-50/40 px-4 py-3"
              >
                <div className="text-sm font-semibold text-slate-800">{item.title}</div>
                <div className="text-xs text-slate-500">
                  {item.issuer_name}
                  {" · "}
                  {item.state === "invited"
                    ? t("website.view.hero_inbox_new")
                    : t("website.view.hero_inbox_opened")}
                </div>
              </Link>
            </li>
          ))}
          {items.length > 4 ? (
            <li className="text-xs text-slate-400 pt-1">
              <Link href="/module/documents/inbox" className="hover:underline">
                +{items.length - 4} {t("website.view.hero_inbox_more")}
              </Link>
            </li>
          ) : null}
        </ul>
      )}
    </div>
  );
}
