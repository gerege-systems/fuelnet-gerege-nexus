/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The module lives in business-gerege-nexus now; these screens have not followed
// it yet. The registration goes with the screens — see
// lib/api/_departed/README.md for the same arrangement on the API side.

import { registerDictionary, source } from "../registry";
import { inventory } from "../addons/inventory";
import { ar } from "../locales/ar/inventory";
import { zh } from "../locales/zh/inventory";
import { fr } from "../locales/fr/inventory";
import { ru } from "../locales/ru/inventory";
import { es } from "../locales/es/inventory";

registerDictionary("inventory", { ...source(inventory), ar, zh, fr, ru, es });
