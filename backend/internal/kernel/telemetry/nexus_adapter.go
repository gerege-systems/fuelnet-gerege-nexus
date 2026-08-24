/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package telemetry

import "github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

// AsSignatureCounter lets a module count a signature without reaching the
// registry.
//
// The registry stays here on purpose. A module that could declare its own
// metric could declare one with a tenant label, and no metric on this platform
// carries one — a cardinality decision that has to hold for every app, not just
// the ones that read the guidance.
func AsSignatureCounter() nexus.SignatureCounter { return signatureCounter{} }

type signatureCounter struct{}

func (signatureCounter) Signed(rail string, ok bool) { RecordDocumentSigned(rail, ok) }
