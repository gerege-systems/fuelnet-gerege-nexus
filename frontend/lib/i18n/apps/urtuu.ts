/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { urtuu } from "../addons/urtuu";
import { ar } from "../locales/ar/urtuu";
import { zh } from "../locales/zh/urtuu";
import { fr } from "../locales/fr/urtuu";
import { ru } from "../locales/ru/urtuu";
import { es } from "../locales/es/urtuu";

registerDictionary("urtuu", { ...source(urtuu), ar, zh, fr, ru, es });
