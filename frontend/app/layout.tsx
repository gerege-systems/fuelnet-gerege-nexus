import "./globals.css";
import "./fuelnet.css";
import type { Metadata, Viewport } from "next";
import React from "react";

import { cookies } from "next/headers";

import Providers from "./providers";
import { brandFromEnv } from "@/lib/brandEnv";
import { brandCopyFromEnv } from "@/lib/brandCopy";
import { localizedBrand } from "@/lib/brand";
import { DEFAULT_LOCALE, LOCALE_KEY } from "@/lib/locale";

/**
 * Rendered per request, because the deployment's name is not the build's to
 * know.
 *
 * Without this the shell's HTML — title, description, launcher name, chrome
 * colour — would be produced by `next build` and carried inside the image, and
 * an image carrying a name is an image one customer owns. It is the same
 * reasoning `lib/apiBase.ts` sets out for the address, and it costs the same
 * thing: the static HTML for these screens is no longer prebuilt. Cheap here,
 * because there was little to prebuild. Every screen under this layout is
 * either behind a session or already rendered per request (the landing page
 * fetches the storefront on the server), so what was being cached was the empty
 * frame around them.
 */
export const dynamic = "force-dynamic";

export async function generateMetadata(): Promise<Metadata> {
  // The name in the reader's language, when the deployment wrote one. The
  // language comes from the cookie the switcher sets (lib/i18n): metadata is
  // produced on the server, where localStorage does not exist, and a page read
  // in English inside a Mongolian-titled tab is the same mismatch the header
  // had.
  const locale = (await cookies()).get(LOCALE_KEY)?.value || DEFAULT_LOCALE;
  const brand = localizedBrand(brandFromEnv(), brandCopyFromEnv(), locale);
  return {
    title: brand.name,
    description: brand.description,
    // Apple reads none of the manifest: on iOS the name under the icon and the
    // status bar are decided by these instead. The short name is what there is
    // room for under an icon.
    appleWebApp: { capable: true, title: brand.shortName, statusBarStyle: "default" },
    // The tab icon and the one iOS puts on a home screen. A deployment that
    // named its own gets it in both places; one that did not keeps the files
    // this image was built with.
    //
    // The favicon moved from app/favicon.ico to public/ for this. As a file
    // convention Next emits a <link rel="icon"> for it unconditionally, so a
    // deployment that named its own ended up with two of them — the built-in
    // one and the brand's — and which the browser drew was a coin toss. In
    // public/ it is an ordinary file at the same address, named here when
    // nothing overrides it, so there is exactly one either way.
    icons: brand.iconUrl
      ? { icon: brand.iconUrl, apple: brand.iconUrl }
      : { icon: "/favicon.ico", apple: "/icons/apple-touch-icon.png" },
    // Both spellings, because the standard one is not yet the only one that
    // works. `appleWebApp.capable` above emits only `mobile-web-app-capable` —
    // the name every current browser reads, and the one Chrome warns about the
    // absence of. Safari on iOS still honours nothing but Apple's prefixed
    // name for a standalone launch, and it is not written by anything in the
    // metadata API, so it is written here. It can go when iOS reads the
    // unprefixed name.
    other: { "apple-mobile-web-app-capable": "yes" },
  };
}

export function generateViewport(): Viewport {
  return { themeColor: brandFromEnv().themeColor };
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // I18nProvider keeps <html lang> in step with the selected locale.
    <html lang="mn">
      <body>
        <Providers brand={brandFromEnv()} copy={brandCopyFromEnv()}>
          {children}
        </Providers>
      </body>
    </html>
  );
}
