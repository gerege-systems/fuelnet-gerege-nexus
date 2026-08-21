/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { ai } from "../addons/ai";
import { ar } from "../locales/ar/ai";
import { zh } from "../locales/zh/ai";
import { fr } from "../locales/fr/ai";
import { ru } from "../locales/ru/ai";
import { es } from "../locales/es/ai";

registerDictionary("ai", { ...source(ai), ar, zh, fr, ru, es });
