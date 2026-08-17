"use client";

import React, { createContext, useContext, useMemo } from "react";

import { DEFAULT_BRAND, localizedBrand, type Brand } from "./brand";
import type { BrandCopy } from "./brandCopy";
import { useI18n } from "./i18n";

/**
 * The brand, as the browser half of the app sees it.
 *
 * It arrives once, from the server, and never changes: the deployment's name is
 * a property of the deployment, not of the session or of the tenant. That is
 * why there is no setter and no fetch here — the value is already in the HTML
 * by the time any of this runs, so no screen ever paints one name and then
 * swaps it for another.
 *
 * A tenant's own name is a different thing and lives elsewhere (the header
 * shows it beside this one): the brand says which product you are in, the
 * tenant says whose data you are looking at.
 */
const BrandContext = createContext<Brand>(DEFAULT_BRAND);

/**
 * The name follows the reader's language; nothing else does.
 *
 * That is why this sits *inside* the i18n provider rather than outside it,
 * which is where it used to be: a deployment writes its name per locale in
 * `BRAND_COPY_FILE`, so resolving it needs to know which language is being
 * read. Every `useBrand()` caller gets the right one without asking — there
 * are a dozen of them, and the one that forgot would be a header in the wrong
 * language beside a sentence in the right one.
 */
export function BrandProvider({
  brand,
  copy = {},
  children,
}: {
  brand: Brand;
  copy?: BrandCopy;
  children: React.ReactNode;
}) {
  const { locale } = useI18n();
  const value = useMemo(() => localizedBrand(brand, copy, locale), [brand, copy, locale]);
  return <BrandContext.Provider value={value}>{children}</BrandContext.Provider>;
}

/**
 * The default is the context's rather than a thrown error, unlike useI18n's.
 * A component rendered outside the provider — a test harness, a story — should
 * show the product's own name, not fail to render; nothing here is a
 * correctness boundary, it is a label.
 */
export function useBrand(): Brand {
  return useContext(BrandContext);
}
