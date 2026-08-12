"use client";

import Capabilities from "@/components/landing/Capabilities";
import Hero from "@/components/landing/Hero";
import SiteFooter from "@/components/landing/SiteFooter";
import SiteHeader from "@/components/landing/SiteHeader";
import Technology from "@/components/landing/Technology";
import Trust from "@/components/landing/Trust";

/**
 * The public landing page — the only screen a visitor sees before signing in.
 *
 * It is composed rather than written out: each section is a self-contained
 * piece of the argument the page makes, and the ones carrying anchors
 * (`#features`, `#trust`, `#technology`) own those ids themselves, so the
 * header's menu keeps working without this file knowing what is inside them.
 */
export default function LandingPage() {
  return (
    <div className="gp-landing" id="top">
      <SiteHeader />
      <main>
        <Hero />
        <Capabilities />
        <Trust />
        <Technology />
      </main>
      <SiteFooter />
    </div>
  );
}
