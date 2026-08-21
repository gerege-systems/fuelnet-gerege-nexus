/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The API client, as eighty-one screens still import it.
//
// The methods and types now live in lib/api/, one file per app and one for the
// platform itself. This file keeps the single `api` object and re-exports every
// type, so nothing above it had to change in the commit that did the splitting:
// a refactor whose diff also touches every screen is a refactor nobody can
// review. Moving the screens to the direct imports is a later, separate pass.
//
// Add nothing here. A new endpoint belongs in the file for the app it serves —
// lib/api/<app>.ts — or, if it is genuinely the platform's, in lib/api/client.ts,
// where scripts/check-api-boundaries.mjs will hold it to the core's own paths.

export * from "./api/client";
export * from "./api/integrations";
export * from "./api/store";
export * from "./api/ai";
export * from "./api/documents";
export * from "./api/egov";
export * from "./api/esign";
export * from "./api/organisation";
export * from "./api/reports";
export * from "./api/sso-clients";
export * from "./api/urtuu";

export * from "./api/_departed/appstore-registry";
export * from "./api/_departed/billing";
export * from "./api/_departed/contacts";
export * from "./api/_departed/inventory";
export * from "./api/_departed/pos";
export * from "./api/_departed/products";
export * from "./api/_departed/publisher";
export * from "./api/_departed/shifts";
export * from "./api/_departed/store-review";

import { coreApi } from "./api/client";
import { integrationsApi } from "./api/integrations";
import { storeApi } from "./api/store";
import { aiApi } from "./api/ai";
import { documentsApi } from "./api/documents";
import { egovApi } from "./api/egov";
import { esignApi } from "./api/esign";
import { organisationApi } from "./api/organisation";
import { reportsApi } from "./api/reports";
import { ssoClientsApi } from "./api/sso-clients";
import { urtuuApi } from "./api/urtuu";

import { appstoreRegistryApi } from "./api/_departed/appstore-registry";
import { billingApi } from "./api/_departed/billing";
import { contactsApi } from "./api/_departed/contacts";
import { inventoryApi } from "./api/_departed/inventory";
import { posApi } from "./api/_departed/pos";
import { productsApi } from "./api/_departed/products";
import { publisherApi } from "./api/_departed/publisher";
import { shiftsApi } from "./api/_departed/shifts";
import { storeReviewApi } from "./api/_departed/store-review";

export const api = {
  ...coreApi,
  ...integrationsApi,
  ...storeApi,
  ...aiApi,
  ...documentsApi,
  ...egovApi,
  ...esignApi,
  ...organisationApi,
  ...reportsApi,
  ...ssoClientsApi,
  ...urtuuApi,

  ...appstoreRegistryApi,
  ...billingApi,
  ...contactsApi,
  ...inventoryApi,
  ...posApi,
  ...productsApi,
  ...publisherApi,
  ...shiftsApi,
  ...storeReviewApi,
};
