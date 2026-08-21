/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The module lives in appstore-gerege-mn now; these screens have not followed
// it yet. The registration goes with the screens — see
// lib/api/_departed/README.md for the same arrangement on the API side.

import { registerDictionary, source } from "../registry";
import { appstore_modules } from "../addons/appstore_modules";
import { ar } from "../locales/ar/appstore_modules";
import { zh } from "../locales/zh/appstore_modules";
import { fr } from "../locales/fr/appstore_modules";
import { ru } from "../locales/ru/appstore_modules";
import { es } from "../locales/es/appstore_modules";

registerDictionary("appstore_modules", { ...source(appstore_modules), ar, zh, fr, ru, es });
