/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package canary

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// A distribution constructs its module the way a product's main() does, and
// everything it declared is there afterwards.
//
// This is the assertion the golden file cannot make. api.txt records that
// Provide, Capability, Register, Migrations, RegisterReport and ProvideAssistant
// exist with those signatures; it says nothing about whether providing a
// capability makes it gettable, or whether a registered module is in the list.
// A refactor can keep every signature and break every one of those.
func TestADistributionsModuleIsWiredUpAfterConstruction(t *testing.T) {
	module := New(nexus.NewPlatform(nil, nil))

	if got, ok := nexus.Get(module.ID()); !ok || got.ID() != module.ID() {
		t.Errorf("the module registered itself and nexus.Get does not have it")
	}

	// The capability this distribution published, of a type the core has never
	// heard of.
	pricing, err := nexus.Capability[Pricing]()
	if err != nil {
		t.Fatalf("the module provided a Pricing and cannot get one: %v", err)
	}
	price, err := pricing.Quote(context.Background(), "tenant-1", "SKU-1")
	if err != nil || price != 4200 {
		t.Errorf("Quote = %d, %v; want 4200", price, err)
	}

	if _, ok := nexus.MigrationsOf(module.ID()); !ok {
		t.Error("the module registered migrations and MigrationsOf does not have them")
	}

	tools := nexus.AssistantToolset()
	if len(tools) != 1 || tools[0].Name != "canary_quote" {
		t.Errorf("the assistant was lent %v; want one canary_quote", tools)
	}
	result, err := tools[0].Call(context.Background(), "tenant-1", map[string]any{"sku": "SKU-1"})
	if err != nil || result["price"] != int64(4200) {
		t.Errorf("the tool answered %v, %v", result, err)
	}

	if got := nexus.MenuPermissionOf(module); got != "canary.read" {
		t.Errorf("menu permission: got %q, want canary.read", got)
	}
	if got := nexus.RoutePermissionPrefixOf(module); got != "canary" {
		t.Errorf("route permission prefix: got %q, want canary", got)
	}
}

// A capability nothing provides comes back as an error that names the type.
//
// The behaviour a distribution depends on and no signature records: a module
// asks for something the deployment does not have and gets an answer it can act
// on, rather than a zero value it will dereference.
func TestAMissingCapabilityIsAnErrorAndNotAZeroValue(t *testing.T) {
	type absent interface{ Absent() }

	value, err := nexus.Capability[absent]()
	if err == nil {
		t.Fatal("a capability nothing provides came back without an error")
	}
	if value != nil {
		t.Errorf("a missing capability also returned a value: %v", value)
	}
	if !strings.Contains(err.Error(), "absent") {
		t.Errorf("the error does not name the missing type: %v", err)
	}

	// And the sentinel accessors keep answering the way v1 promised.
	if _, err := nexus.Ring(); !errors.Is(err, nexus.ErrNoLink) {
		t.Errorf("Ring on a deployment with no link returned %v, want ErrNoLink", err)
	}
	if _, err := nexus.Documents(); !errors.Is(err, nexus.ErrNoDocumentFiler) {
		t.Errorf("Documents with no filer returned %v, want ErrNoDocumentFiler", err)
	}
	if _, err := nexus.Meetings(); err == nil {
		t.Error("Meetings with no booker returned no error")
	}
}

// A permission that says who it reaches, and one that contradicts itself.
//
// Both are v1 promises a distribution writes against: the first is the whole
// point of DefaultRoles, and the second is the refusal that stops a permission
// quietly reaching more people than intended.
func TestAPermissionsDeclaredReachIsHonoured(t *testing.T) {
	module := New(nexus.NewPlatform(nil, nil))

	for _, perm := range module.Permissions() {
		if err := perm.Validate(); err != nil {
			t.Errorf("%s: %v", perm.Code, err)
		}
	}

	contradiction := nexus.PermissionDefinition{
		Code: "canary.read", AdminOnly: true, DefaultRoles: []string{nexus.DefaultRoleUser},
	}
	if err := contradiction.Validate(); err == nil {
		t.Error("a permission that is both AdminOnly and granted by default was accepted")
	}
}
