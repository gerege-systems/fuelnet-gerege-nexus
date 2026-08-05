"use client";

import "./globals.css";
import React, { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Layout from "@/components/Layout";
import { I18nProvider } from "@/lib/i18n";

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // Created per mount rather than at module scope so the cache is not shared
  // across requests when the bundle is reused.
  const [queryClient] = useState(() => new QueryClient());

  return (
    // I18nProvider keeps <html lang> in step with the selected locale.
    <html lang="mn">
      <body>
        <I18nProvider>
          <QueryClientProvider client={queryClient}>
            <Layout>{children}</Layout>
          </QueryClientProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
