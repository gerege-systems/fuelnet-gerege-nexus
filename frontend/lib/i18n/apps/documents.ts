/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import { documents } from "../addons/documents";
import { registerDictionary, source } from "../registry";

// The source dictionary travels with the optional Documents screens. Optional
// locale overlays can be added by the product distribution that ships them;
// until then the registry falls back to the reviewed English source strings.
registerDictionary("documents", source(documents));
