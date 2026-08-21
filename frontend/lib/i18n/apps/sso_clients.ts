/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { registerDictionary, source } from "../registry";
import { sso_clients } from "../addons/sso_clients";
import { ar } from "../locales/ar/sso_clients";
import { zh } from "../locales/zh/sso_clients";
import { fr } from "../locales/fr/sso_clients";
import { ru } from "../locales/ru/sso_clients";
import { es } from "../locales/es/sso_clients";

registerDictionary("sso_clients", { ...source(sso_clients), ar, zh, fr, ru, es });
