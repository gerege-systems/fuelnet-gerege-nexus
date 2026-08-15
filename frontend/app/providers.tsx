"use client";

import React, { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import Layout from "@/components/Layout";
import InstallApp from "@/components/InstallApp";
import { I18nProvider } from "@/lib/i18n";
import { ThemeProvider } from "@/lib/theme";
import { BrandProvider } from "@/lib/brandContext";
import type { Brand } from "@/lib/brand";

/**
 * Everything under the document that has to run in the browser.
 *
 * This used to be the root layout itself, which was a client component so the
 * providers could live in it. The brand is what split them: it is read from the
 * environment at request time, and only a server component can read it, so the
 * root became one and this file took the providers. The file comment there
 * once called that split larger than the metadata warranted — it was, for
 * metadata; a name the image is not allowed to know is a different bargain.
 *
 * The brand is passed down rather than fetched: it is in the first byte of HTML
 * the server sends, so no screen shows one product's name and then swaps it.
 */
export default function Providers({ brand, children }: { brand: Brand; children: React.ReactNode }) {
  // Created per mount rather than at module scope so the cache is not shared
  // across requests when the bundle is reused.
  const [queryClient] = useState(() => new QueryClient());

  return (
    <BrandProvider brand={brand}>
      <ThemeProvider>
        {/* The brand name reaches every translation as {brand}; see lib/i18n. */}
        <I18nProvider brand={brand.name}>
          <QueryClientProvider client={queryClient}>
            <Layout>{children}</Layout>
            <InstallApp />
          </QueryClientProvider>
        </I18nProvider>
      </ThemeProvider>
    </BrandProvider>
  );
}
