/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package nexus_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

type pricer interface{ Price() int }

type fixedPrice struct{ n int }

func (f fixedPrice) Price() int { return f.n }

// A capability goes in and comes back out.
func TestWhatIsProvidedIsWhatComesBack(t *testing.T) {
	t.Cleanup(func() { nexus.Provide[pricer](nil) })

	nexus.Provide[pricer](fixedPrice{7})

	got, err := nexus.Capability[pricer]()
	if err != nil {
		t.Fatalf("provided a pricer and could not get one: %v", err)
	}
	if got.Price() != 7 {
		t.Errorf("got %d, want 7", got.Price())
	}
}

// The error names the type, because "no capability" is not an answer anybody
// can act on.
func TestAMissingCapabilityNamesTheTypeItIsMissing(t *testing.T) {
	type absent interface{ Absent() }

	_, err := nexus.Capability[absent]()
	if err == nil {
		t.Fatal("nothing was provided and no error came back")
	}
	if !strings.Contains(err.Error(), "absent") {
		t.Errorf("the error does not name the missing type: %v", err)
	}
}

// Last writer wins, the same rule Register uses for modules.
func TestTheLastCapabilityProvidedWins(t *testing.T) {
	t.Cleanup(func() { nexus.Provide[pricer](nil) })

	nexus.Provide[pricer](fixedPrice{1})
	nexus.Provide[pricer](fixedPrice{2})

	got, err := nexus.Capability[pricer]()
	if err != nil {
		t.Fatal(err)
	}
	if got.Price() != 2 {
		t.Errorf("got %d, want the later value 2", got.Price())
	}
}

// Providing nil withdraws the capability rather than storing an empty one.
//
// This is how a test undoes its own Provide, and how UseDocumentFiler(nil) —
// which several tests still call — has always behaved.
func TestProvidingNilWithdrawsTheCapability(t *testing.T) {
	nexus.Provide[pricer](fixedPrice{3})
	nexus.Provide[pricer](nil)

	if _, err := nexus.Capability[pricer](); err == nil {
		t.Error("the capability was withdrawn and still came back")
	}
}

// Two types, two answers. A registry keyed by type has to keep them apart.
func TestCapabilitiesOfDifferentTypesDoNotCollide(t *testing.T) {
	type counter interface{ Count() int }
	t.Cleanup(func() { nexus.Provide[pricer](nil) })

	nexus.Provide[pricer](fixedPrice{5})

	if _, err := nexus.Capability[counter](); err == nil {
		t.Error("providing a pricer answered for a counter")
	}
}

// Modules are constructed concurrently in some distributions, and the registry
// is global. Run with -race.
func TestProvideAndCapabilityAreSafeInParallel(t *testing.T) {
	t.Cleanup(func() { nexus.Provide[pricer](nil) })
	nexus.Provide[pricer](fixedPrice{1})

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(2)
		go func() { defer wg.Done(); nexus.Provide[pricer](fixedPrice{i}) }()
		go func() { defer wg.Done(); _, _ = nexus.Capability[pricer]() }()
	}
	wg.Wait()
}
