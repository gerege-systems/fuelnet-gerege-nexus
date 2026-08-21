/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The module lives in business-gerege-nexus now; these screens have not followed
// it yet. The registration goes with the screens — see
// lib/api/_departed/README.md for the same arrangement on the API side.

import { registerDictionary, source } from "../registry";
import { billing } from "../addons/billing";
import { ar } from "../locales/ar/billing";
import { zh } from "../locales/zh/billing";
import { fr } from "../locales/fr/billing";
import { ru } from "../locales/ru/billing";
import { es } from "../locales/es/billing";

registerDictionary("billing", { ...source(billing), ar, zh, fr, ru, es });
