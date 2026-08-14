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
import { fetchStorefrontOnServer } from "@/lib/storefront";

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
 * Which page that is gets decided here, on the server, before anything is sent.
 * It was briefly decided in the browser instead, and that was wrong twice over:
 * a visitor saw the platform page and then watched it turn into a shop, and
 * anything that does not run JavaScript — every crawler — only ever saw the
 * first of those. A catalogue nobody can find is not much of a shop.
 *
 * It stays a run-time question rather than a build-time one, though: one image
 * serves every deployment, so the image cannot know which one it is. The
 * deployment says where its API is (`API_INTERNAL_URL`) and this asks.
 */

// Rendered per request, not prerendered at build.
//
// Which product this deployment is depends on an environment variable, and a
// build has no environment: prerendering baked the platform page into the image
// and served it to the first visitor after every deploy — for a full minute,
// until the first revalidation replaced it with the shop. A page that is wrong
// exactly when somebody first looks at it is wrong.
//
// The render is cheap and the fetch behind it is not repeated: it carries its
// own 60-second cache, so the API is asked once a minute however many people
// arrive.
export const dynamic = "force-dynamic";

export default async function LandingPage() {
  const apps = await fetchStorefrontOnServer();

  return (
    <div className="gp-landing" id="top">
      <SiteHeader />
      <main>
        {apps ? (
          <Storefront apps={apps} />
        ) : (
          <>
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
    </div>
  );
}
