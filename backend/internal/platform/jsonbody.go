/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package platform

import (
	"encoding/json"
	"io"
	"net/http"
)

// decodeLimitedJSON reads a request body no larger than max.
//
// A handler that decodes an unbounded body is a handler that can be handed a
// gigabyte, and the cost is paid before any check the handler makes. Every
// caller names its own ceiling because the honest size differs by two orders of
// magnitude across them — a PIN, a pasted document, a base64 recording.
//
// It lived in ai_handlers.go until 2026-08-23, which is where it was first
// needed rather than where it belonged; when the assistant left for
// internal/apps/ai it turned out nine platform handlers had been using it. A
// helper the platform depends on does not belong in an app's file.
func decodeLimitedJSON(r *http.Request, dst any, max int64) error {
	return json.NewDecoder(io.LimitReader(r.Body, max)).Decode(dst)
}
