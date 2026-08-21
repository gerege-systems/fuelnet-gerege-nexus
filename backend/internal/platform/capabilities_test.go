/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package platform

import (
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// A booking contract with nobody providing it is what this test exists for.
//
// MeetingBooker was declared in the SDK, an adapter was written for it the same
// day, and for six days nothing called either — there was no accessor and
// nothing published one. The interface compiled, the adapter compiled, and a
// module asking for a meeting had nowhere to ask. Nothing failed, because
// nothing ran.
func TestABuiltServerProvidesTheBookingCapability(t *testing.T) {
	// Built for its side effects: NewServer is what publishes the capability.
	_ = routerUnderTest(t)

	booker, err := nexus.Meetings()
	if err != nil {
		t.Fatalf("a built server provides no meeting booker: %v", err)
	}
	if booker == nil {
		t.Fatal("nexus.Meetings returned a nil booker and no error")
	}
}
