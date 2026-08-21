/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { esign } from "../addons/esign";
import { ar } from "../locales/ar/esign";
import { zh } from "../locales/zh/esign";
import { fr } from "../locales/fr/esign";
import { ru } from "../locales/ru/esign";
import { es } from "../locales/es/esign";

registerDictionary("esign", { ...source(esign), ar, zh, fr, ru, es });
