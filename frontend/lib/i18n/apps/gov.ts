/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The module lives in gerege-gov now; these screens have not followed
// it yet. The registration goes with the screens — see
// lib/api/_departed/README.md for the same arrangement on the API side.

import { registerDictionary, source } from "../registry";
import { gov } from "../addons/gov";
import { ar } from "../locales/ar/gov";
import { zh } from "../locales/zh/gov";
import { fr } from "../locales/fr/gov";
import { ru } from "../locales/ru/gov";
import { es } from "../locales/es/gov";

registerDictionary("gov", { ...source(gov), ar, zh, fr, ru, es });
