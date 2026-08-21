/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { reports } from "../addons/reports";
import { ar } from "../locales/ar/reports";
import { zh } from "../locales/zh/reports";
import { fr } from "../locales/fr/reports";
import { ru } from "../locales/ru/reports";
import { es } from "../locales/es/reports";

registerDictionary("reports", { ...source(reports), ar, zh, fr, ru, es });
