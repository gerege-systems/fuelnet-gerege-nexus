/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The two response helpers moved to `backend/pkg/nexus`, where an app module in
 * another repository can reach them. These forward, so the platform's own
 * handlers did not all have to change on the same day — and so there is one
 * implementation rather than two that could answer differently.
 */

package httpx

import (
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// JSON writes value as the whole response body.
func JSON(w http.ResponseWriter, status int, value any) { nexus.JSON(w, status, value) }

// Error answers with {"error": message}.
func Error(w http.ResponseWriter, status int, message string) { nexus.Error(w, status, message) }
