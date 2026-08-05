package gov_services

import (
	"errors"
	"testing"
)

// The state machine is pure, so its invariants are tested without a database.

func TestCompleteWaitsForVerificationWhenStepRequiresIt(t *testing.T) {
	step := &WorkflowStep{Code: "FULFILL", RequiresVerification: true}

	result, err := resolveTransition(TaskInProgress, ActionComplete, step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.to != TaskAwaitingVerification {
		t.Fatalf("a step requiring verification must land in %s, got %s", TaskAwaitingVerification, result.to)
	}

	// The same action on a step that needs no verification finishes outright.
	result, err = resolveTransition(TaskInProgress, ActionComplete, &WorkflowStep{Code: "FULFILL"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.to != TaskCompleted {
		t.Fatalf("expected %s, got %s", TaskCompleted, result.to)
	}
}

func TestInvalidTransitionsAreRejected(t *testing.T) {
	cases := []struct {
		name   string
		from   string
		action string
		code   string
	}{
		{"verify work that was never submitted", TaskReceived, ActionVerify, "INVALID_TRANSITION"},
		{"start a rejected task", TaskRejected, ActionStart, "TASK_TERMINAL"},
		{"complete a cancelled task", TaskCancelled, ActionComplete, "TASK_TERMINAL"},
		{"delegate after closing", TaskClosed, ActionDelegate, "TASK_TERMINAL"},
		{"return work that is not awaiting verification", TaskInProgress, ActionReturn, "INVALID_TRANSITION"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveTransition(tc.from, tc.action, nil, nil)
			var domain *WorkflowError
			if !errors.As(err, &domain) {
				t.Fatalf("expected a workflow error, got %v", err)
			}
			if domain.Code != tc.code {
				t.Fatalf("expected code %s, got %s", tc.code, domain.Code)
			}
		})
	}
}

func TestConfiguredWorkflowCanNarrowButNotWiden(t *testing.T) {
	// A version that offers only "reject" from IN_PROGRESS must refuse
	// "complete", even though the base machine allows it.
	allowed := map[transitionKey]string{
		{TaskInProgress, ActionReject}: TaskRejected,
	}
	if _, err := resolveTransition(TaskInProgress, ActionComplete, nil, allowed); err == nil {
		t.Fatal("a transition the version does not offer must be refused")
	}
	if _, err := resolveTransition(TaskInProgress, ActionReject, nil, allowed); err != nil {
		t.Fatalf("a configured transition must be allowed: %v", err)
	}

	// Configuration cannot invent a target the base machine does not allow.
	widened := map[transitionKey]string{
		{TaskReceived, ActionStart}: TaskClosed,
	}
	if _, err := resolveTransition(TaskReceived, ActionStart, nil, widened); err == nil {
		t.Fatal("configuration must not be able to widen the state machine")
	}
}

func TestActionsRequiringAReason(t *testing.T) {
	for _, action := range []string{ActionReject, ActionRequestInfo} {
		result, err := resolveTransition(TaskInProgress, action, nil, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", action, err)
		}
		if !result.requiresComment {
			t.Fatalf("%s must require a comment", action)
		}
	}
	result, err := resolveTransition(TaskAwaitingVerification, ActionReturn, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.requiresComment {
		t.Fatal("returning work must require a reason")
	}
}

func TestPermissionsPerAction(t *testing.T) {
	cases := map[string]struct {
		from   string
		action string
		want   string
	}{
		"delegation needs the delegate permission":  {TaskInProgress, ActionDelegate, PermDelegate},
		"verification needs the verify permission":  {TaskAwaitingVerification, ActionVerify, PermVerify},
		"processing needs the process permission":   {TaskReceived, ActionStart, PermProcess},
		"cancelling is an applicant-side operation": {TaskReceived, ActionCancel, PermApply},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := resolveTransition(tc.from, tc.action, nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.permission != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, result.permission)
			}
		})
	}
}

func TestApplicationStatusIsDerivedFromTheRootTask(t *testing.T) {
	cases := map[string]string{
		TaskReceived:             "SUBMITTED",
		TaskAssigned:             "SUBMITTED",
		TaskInProgress:           "IN_REVIEW",
		TaskForwarded:            "IN_REVIEW",
		TaskAwaitingVerification: "IN_REVIEW",
		TaskInfoRequested:        "INFO_REQUESTED",
		TaskCompleted:            "APPROVED",
		TaskClosed:               "COMPLETED",
		TaskRejected:             "REJECTED",
		TaskCancelled:            "CANCELLED",
	}
	for task, want := range cases {
		if got := applicationStatusFor(task); got != want {
			t.Errorf("%s: expected %s, got %s", task, want, got)
		}
	}
}

func TestTemplatesCoverTheRequiredShapes(t *testing.T) {
	for _, code := range []string{"LOCAL_FULFILMENT", "DELEGATE_ONE_LEVEL", "DELEGATE_MULTI_LEVEL"} {
		tpl, ok := TemplateByCode(code)
		if !ok {
			t.Fatalf("template %s is missing", code)
		}
		if len(tpl.Steps) == 0 {
			t.Fatalf("template %s has no steps", code)
		}
	}

	// Delegation templates must ask for verification somewhere, otherwise a
	// lower unit could close the request on its own.
	delegated, _ := TemplateByCode("DELEGATE_ONE_LEVEL")
	verifies := false
	for _, step := range delegated.Steps {
		if step.RequiresVerification {
			verifies = true
		}
	}
	if !verifies {
		t.Fatal("the delegation template must require verification")
	}
}
