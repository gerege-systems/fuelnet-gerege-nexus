"use client";

import {useEffect, useState} from "react";
import Link from "next/link";

import LanguageSwitcher from "@/components/LanguageSwitcher";
import { SECTION_LINKS, type LandingSection } from "@/lib/landing";
import {useI18n} from "@/lib/i18n";
import {useBrand} from "@/lib/brandContext";

/**
 * The public header.
 *
 * The section items scroll to a part of this page and are listed in the order
 * those sections are rendered, so the first item is never the furthest away —
 * which is why the list comes from the page rather than being written out
 * here: a deployment that drops a section must not be left with a menu item
 * that scrolls nowhere.
 *
 * The last item leaves for the published documentation, so it opens in a new
 * tab and carries `rel="noopener"` rather than silently replacing the page
 * someone is reading. Its address is the deployment's (`BRAND_DOCS_URL`): a
 * deployment with its own name has its own manual, and sending its readers to
 * the platform's is sending them somewhere that does not describe what they
 * are looking at.
 *
 * On a narrow screen the same items are behind a button rather than gone.
 * They used to be gone: the stylesheet hid the nav below 900px and the
 * language switcher below 640px, and nothing replaced either — a phone got a
 * logo and a Sign in button, and no way to reach any section of the page or to
 * read it in another language.
 */
export default function SiteHeader({sections}: {sections: LandingSection[]}) {
  const {t} = useI18n();
  const brand = useBrand();
  const [open, setOpen] = useState(false);

  // Escape closes it, because a panel that covers the page and can only be
  // dismissed by finding the same small button again is a trap on a phone.
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  const items = (
    <>
      {sections.map((section) => {
        const link = SECTION_LINKS[section];
        if (!link) return null;
        return (
          <a key={section} href={`#${link.anchor}`} onClick={() => setOpen(false)}>
            {t(link.label)}
          </a>
        );
      })}
      <a
        href={brand.docsUrl}
        target="_blank"
        rel="noopener noreferrer"
        onClick={() => setOpen(false)}
      >
        {t("website.menu.docs")}
      </a>
    </>
  );

  return (
    <header className="gp-nav">
      <a href="#top" className="gp-brand">
        <img src={brand.logoUrl} alt="" />
        <span>{brand.name}</span>
      </a>
      <nav>{items}</nav>
      <div className="gp-actions">
        <LanguageSwitcher variant="dark" />
        <Link href="/login" className="gp-gold">
          {t("website.action.sign_in")}
        </Link>
        {/* The button is the only part of this header that a wide screen never
            shows: above 900px every item is already on the bar. */}
        <button
          type="button"
          className="gp-nav__toggle"
          aria-expanded={open}
          aria-controls="gp-mobile-menu"
          aria-label={t("website.menu.toggle")}
          onClick={() => setOpen((was) => !was)}
        >
          <span aria-hidden="true">{open ? "✕" : "☰"}</span>
        </button>
      </div>
      {open && (
        <div className="gp-nav__menu" id="gp-mobile-menu">
          <nav>{items}</nav>
          {/* The language switcher lives here too: below 640px the stylesheet
              hides the one on the bar, and a reader who cannot find their own
              language leaves. */}
          <div className="gp-nav__menu-actions">
            <LanguageSwitcher variant="dark" />
          </div>
        </div>
      )}
    </header>
  );
}
