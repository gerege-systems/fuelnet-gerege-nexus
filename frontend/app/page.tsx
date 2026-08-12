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
 * The order answers questions in the order they are asked. What is this, and
 * how is it built. What do I get. What is underneath it. Only then how identity
 * works — which is why the page closes on the claim that signing in is not a
 * screen but the floor everything above it stands on. Put first, that claim is
 * a detail about a login box; put last, it is the point.
 */
export default function LandingPage() {
  return (
    <div className="gp-landing" id="top">
      <SiteHeader />
      <main>
        <Hero />
        <Architecture />
        <Applications />
        <PlatformDepth />
        <Trust />
        <Technology />
        <Capabilities />
      </main>
      <SiteFooter />
    </div>
  );
}
