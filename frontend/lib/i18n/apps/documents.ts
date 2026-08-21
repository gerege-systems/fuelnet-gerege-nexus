/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { documents } from "../addons/documents";
import { ar } from "../locales/ar/documents";
import { zh } from "../locales/zh/documents";
import { fr } from "../locales/fr/documents";
import { ru } from "../locales/ru/documents";
import { es } from "../locales/es/documents";

registerDictionary("documents", { ...source(documents), ar, zh, fr, ru, es });
