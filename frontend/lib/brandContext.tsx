"use client";

import React, { createContext, useContext } from "react";

import { DEFAULT_BRAND, type Brand } from "./brand";

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

export function BrandProvider({ brand, children }: { brand: Brand; children: React.ReactNode }) {
  return <BrandContext.Provider value={brand}>{children}</BrandContext.Provider>;
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
