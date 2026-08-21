/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package platform

import (
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eidmongolia"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/gerege"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/integration"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/staterail"
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

// Everything Bootstrap asks the registry for, the server has to have provided.
//
// The eight used to be parameters, and a missing one was a compile error. They
// are capabilities now, and a missing one is a slog.Error and a zero value in a
// module nobody looks at — which is a better failure for a distribution and a
// worse one for this repository, where forgetting a Provide is how a module
// would quietly be built without its dependency. This is the compile error,
// put back.
func TestTheServerProvidesEverythingBootstrapAsksFor(t *testing.T) {
	_ = routerUnderTest(t)

	// One line per capability rather than a table, because each is a different
	// type and the type is the whole of what is being asserted.
	provided[*integration.Manager](t)
	provided[*eidmongolia.Service](t)
	provided[*ssoprovider.SSOProvider](t)
	provided[*gerege.GeregeService](t)
	provided[staterail.Rails](t)
	provided[nexus.Link](t)
	provided[nexus.Signer](t)
	provided[apps.InstalledApps](t)
}

func provided[T any](t *testing.T) {
	t.Helper()
	if _, err := nexus.Capability[T](); err != nil {
		t.Errorf("%v — server.go has to Provide it before apps.Bootstrap", err)
	}
}
