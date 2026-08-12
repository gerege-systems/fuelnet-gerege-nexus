"use client";

import Applications from "@/components/landing/Applications";
import Architecture from "@/components/landing/Architecture";
import Capabilities from "@/components/landing/Capabilities";
import Hero from "@/components/landing/Hero";
import PlatformDepth from "@/components/landing/PlatformDepth";
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
 *
 * The order is an argument that narrows. The first half answers "can I get in
 * and is that safe", which is what a citizen arriving here wants. The second
 * half answers "what am I adopting", which is what the person deciding for an
 * organisation wants, and which used to be argued separately on the
 * documentation site — one page making the case is one page to keep true.
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
        <Architecture />
        <Applications />
        <PlatformDepth />
      </main>
      <SiteFooter />
    </div>
  );
}
