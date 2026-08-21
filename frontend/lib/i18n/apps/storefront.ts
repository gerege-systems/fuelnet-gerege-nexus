/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { storefront } from "../addons/storefront";
import { ar } from "../locales/ar/storefront";
import { zh } from "../locales/zh/storefront";
import { fr } from "../locales/fr/storefront";
import { ru } from "../locales/ru/storefront";
import { es } from "../locales/es/storefront";

registerDictionary("storefront", { ...source(storefront), ar, zh, fr, ru, es });
