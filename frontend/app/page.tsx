"use client";

import { useEffect, useState } from "react";

import Applications from "@/components/landing/Applications";
import Architecture from "@/components/landing/Architecture";
import Capabilities from "@/components/landing/Capabilities";
import Hero from "@/components/landing/Hero";
import PlatformDepth from "@/components/landing/PlatformDepth";
import SiteFooter from "@/components/landing/SiteFooter";
import SiteHeader from "@/components/landing/SiteHeader";
import Storefront from "@/components/landing/Storefront";
import Technology from "@/components/landing/Technology";
import Trust from "@/components/landing/Trust";
import { fetchStorefront, type StoreApp } from "@/lib/storefront";

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
 *
 * # The store
 *
 * A deployment carrying the app-store modules answers a different question. Its
 * visitor is not asking what the platform is; they are asking what is in the
 * catalogue. So that deployment gets the catalogue, and the platform's argument
 * gives way to it.
 *
 * Which page to render is asked of the server at run time rather than decided
 * at build time, and that is the whole point: one image serves every
 * deployment, so the image cannot know which one it is. The same reasoning as
 * lib/apiBase.ts, for the same reason.
 */
export default function LandingPage() {
  const [apps, setApps] = useState<StoreApp[] | null>(null);
  const [asked, setAsked] = useState(false);

  useEffect(() => {
    let live = true;
    fetchStorefront().then((found) => {
      if (!live) return;
      setApps(found);
      setAsked(true);
    });
    return () => {
      live = false;
    };
  }, []);

  return (
    <div className="gp-landing" id="top">
      <SiteHeader />
      <main>
        {apps ? (
          <Storefront apps={apps} />
        ) : (
          <>
            {/* Until the question is answered, the platform page is what shows.
                A spinner over the whole landing would trade a correct page for
                a blank one on every visit, to spare a store one repaint. */}
            <Hero />
            <Architecture />
            <Applications />
            <PlatformDepth />
            <Trust />
            <Technology />
            <Capabilities />
          </>
        )}
      </main>
      <SiteFooter />
      {/* asked is read so the effect's completion is observable in the tree;
          without it a store's first paint and its second are indistinguishable
          to anything watching. */}
      <span hidden data-storefront-resolved={asked} />
    </div>
  );
}
