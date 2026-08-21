/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The module lives in business-gerege-nexus now; these screens have not followed
// it yet. The registration goes with the screens — see
// lib/api/_departed/README.md for the same arrangement on the API side.

import { registerDictionary, source } from "../registry";
import { products } from "../addons/products";
import { ar } from "../locales/ar/products";
import { zh } from "../locales/zh/products";
import { fr } from "../locales/fr/products";
import { ru } from "../locales/ru/products";
import { es } from "../locales/es/products";

registerDictionary("products", { ...source(products), ar, zh, fr, ru, es });
