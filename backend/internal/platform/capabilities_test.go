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

// This binary lends the assistant nothing, and that is the point.
//
// /ai/copilot and /ai/stock-forecast used to answer from products, contacts,
// warehouses and stock_levels — commerce's tables, which db/migrations still
// creates. On a deployment without commerce the queries succeeded and returned
// zeros: "0 products, 0 customers" from the copilot, `[]` from the forecast.
// An empty reorder list reads as "nothing to reorder", not as "this deployment
// cannot tell you".
//
// Nothing in this repository provides an assistant tool now, so the copilot
// declares only the platform's knowledge search and the forecast endpoint
// answers 404. Both routes stay mounted — a route table that changes shape with
// the environment is one nobody can reason about — which is why routes.txt does
// not move.
func TestThisBinaryLendsTheAssistantNothing(t *testing.T) {
	// Built for its side effects: every module in this binary is constructed,
	// and any of them could register a tool.
	_ = routerUnderTest(t)

	if tools := nexus.AssistantToolset(); len(tools) != 0 {
		t.Errorf("this binary provides assistant tools %v; the core is not supposed to have any", tools)
	}
}
