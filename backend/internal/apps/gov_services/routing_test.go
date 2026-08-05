package gov_services

import (
	"errors"
	"testing"
)

// buildGraph wires a small hierarchy: ministry → agency → two districts.
func buildGraph() (*UnitGraph, map[string]string) {
	ids := map[string]string{
		"ministry": "11111111-1111-1111-1111-111111111111",
		"agency":   "22222222-2222-2222-2222-222222222222",
		"north":    "33333333-3333-3333-3333-333333333333",
		"south":    "44444444-4444-4444-4444-444444444444",
	}
	ministry := ids["ministry"]
	agency := ids["agency"]
	units := []*OrgUnit{
		{ID: ids["ministry"], Code: "MIN", Active: true},
		{ID: ids["agency"], Code: "AGY", ParentID: &ministry, Active: true},
		{ID: ids["north"], Code: "NORTH", ParentID: &agency, Active: true, RegionCode: "UB-NORTH"},
		{ID: ids["south"], Code: "SOUTH", ParentID: &agency, Active: true, RegionCode: "UB-SOUTH"},
	}
	return NewUnitGraph(units), ids
}

func TestResolveSelfAndParent(t *testing.T) {
	g, ids := buildGraph()

	got, err := g.Resolve(RouteSelf, RoutingRequest{CurrentUnitID: ids["agency"]}, nil, "")
	if err != nil || got != ids["agency"] {
		t.Fatalf("SELF must stay on the current unit, got %q err %v", got, err)
	}

	got, err = g.Resolve(RouteParent, RoutingRequest{CurrentUnitID: ids["agency"]}, nil, "")
	if err != nil || got != ids["ministry"] {
		t.Fatalf("PARENT must escalate, got %q err %v", got, err)
	}

	// The root has nowhere to escalate to.
	if _, err := g.Resolve(RouteParent, RoutingRequest{CurrentUnitID: ids["ministry"]}, nil, ""); err == nil {
		t.Fatal("escalating from the root must fail")
	}
}

func TestResolveChildRequiresAnUnambiguousTarget(t *testing.T) {
	g, ids := buildGraph()

	// Two children: the caller has to say which one.
	_, err := g.Resolve(RouteChild, RoutingRequest{CurrentUnitID: ids["agency"]}, nil, "")
	var domain *WorkflowError
	if !asWorkflowError(err, &domain) || domain.Code != "TARGET_UNIT_REQUIRED" {
		t.Fatalf("expected TARGET_UNIT_REQUIRED, got %v", err)
	}

	got, err := g.Resolve(RouteChild, RoutingRequest{
		CurrentUnitID: ids["agency"], TargetUnitID: ids["north"],
	}, nil, "")
	if err != nil || got != ids["north"] {
		t.Fatalf("a named child must be selected, got %q err %v", got, err)
	}

	// A unit that is not a child must be refused, not silently accepted.
	_, err = g.Resolve(RouteChild, RoutingRequest{
		CurrentUnitID: ids["agency"], TargetUnitID: ids["ministry"],
	}, nil, "")
	if !asWorkflowError(err, &domain) || domain.Code != "TARGET_NOT_CHILD" {
		t.Fatalf("expected TARGET_NOT_CHILD, got %v", err)
	}

	// One child needs no explicit target.
	got, err = g.Resolve(RouteChild, RoutingRequest{CurrentUnitID: ids["ministry"]}, nil, "")
	if err != nil || got != ids["agency"] {
		t.Fatalf("a single child must be chosen automatically, got %q err %v", got, err)
	}
}

func TestResolveRegionMatch(t *testing.T) {
	g, ids := buildGraph()

	got, err := g.Resolve(RouteRegionMatch, RoutingRequest{
		CurrentUnitID: ids["agency"],
		Fields:        map[string]string{"region": "UB-SOUTH"},
	}, nil, "")
	if err != nil || got != ids["south"] {
		t.Fatalf("the southern district must be picked, got %q err %v", got, err)
	}

	if _, err := g.Resolve(RouteRegionMatch, RoutingRequest{
		CurrentUnitID: ids["agency"],
		Fields:        map[string]string{"region": "GOBI"},
	}, nil, ""); err == nil {
		t.Fatal("an unmatched region must fail rather than route arbitrarily")
	}
}

func TestDescendantsCoverTheWholeBranch(t *testing.T) {
	g, ids := buildGraph()

	got := g.Descendants(ids["agency"])
	if len(got) != 3 {
		t.Fatalf("expected the agency and its two districts, got %v", got)
	}

	// A supervisor at the ministry sees everything below it.
	if len(g.Descendants(ids["ministry"])) != 4 {
		t.Fatalf("the ministry must cover the whole tree")
	}
}

func TestSelectRulePrefersServiceScopedThenPriority(t *testing.T) {
	serviceID := "service-1"
	other := "service-2"
	rules := []RoutingRule{
		{ID: "tenant-wide", Priority: 10, Strategy: RouteSelf, Active: true},
		{ID: "service", Priority: 10, Strategy: RouteChild, Active: true, ServiceID: &serviceID},
		{ID: "other-service", Priority: 1, Strategy: RouteParent, Active: true, ServiceID: &other},
		{ID: "inactive", Priority: 1, Strategy: RouteParent, Active: false, ServiceID: &serviceID},
	}

	rule, ok := SelectRule(rules, serviceID, nil)
	if !ok || rule.ID != "service" {
		t.Fatalf("the service-scoped active rule must win, got %+v (ok=%v)", rule, ok)
	}

	// A field match that does not hold must not select the rule.
	field := "region"
	rules = append(rules, RoutingRule{
		ID: "region", Priority: 1, Strategy: RouteSpecificUnit, Active: true,
		ServiceID: &serviceID, MatchField: field, MatchValue: "UB",
	})
	rule, _ = SelectRule(rules, serviceID, map[string]string{"region": "GOBI"})
	if rule.ID == "region" {
		t.Fatal("a non-matching field must not select the rule")
	}
	rule, _ = SelectRule(rules, serviceID, map[string]string{"region": "UB"})
	if rule.ID != "region" {
		t.Fatalf("a matching field must select the rule, got %s", rule.ID)
	}
}

func asWorkflowError(err error, target **WorkflowError) bool {
	return errors.As(err, target)
}
