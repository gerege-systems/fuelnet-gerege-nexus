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
//
// lib/api/_departed/ is gone. It held clients for nine modules that had moved to
// other repositories, on the argument that their screens were still working. The
// core served no route for any of them: every one of those screens was calling
// an endpoint that answered 404. They were removed with the clients.

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
};
