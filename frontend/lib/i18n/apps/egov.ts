/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { egov } from "../addons/egov";
import { ar } from "../locales/ar/egov";
import { zh } from "../locales/zh/egov";
import { fr } from "../locales/fr/egov";
import { ru } from "../locales/ru/egov";
import { es } from "../locales/es/egov";

registerDictionary("egov", { ...source(egov), ar, zh, fr, ru, es });
