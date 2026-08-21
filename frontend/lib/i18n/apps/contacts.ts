/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The module lives in business-gerege-nexus now; these screens have not followed
// it yet. The registration goes with the screens — see
// lib/api/_departed/README.md for the same arrangement on the API side.

import { registerDictionary, source } from "../registry";
import { contacts } from "../addons/contacts";
import { ar } from "../locales/ar/contacts";
import { zh } from "../locales/zh/contacts";
import { fr } from "../locales/fr/contacts";
import { ru } from "../locales/ru/contacts";
import { es } from "../locales/es/contacts";

registerDictionary("contacts", { ...source(contacts), ar, zh, fr, ru, es });
