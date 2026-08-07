package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/integration"
)

func TestIntegrationManagerList(t *testing.T) {
	mgr := integration.NewManager()

	list := mgr.List()
	if len(list) < 2 {
		t.Fatalf("expected at least 2 default integrations, got %d", len(list))
	}
}

// A webhook signing secret is written, never read back. It used to ride along
// in the JSON of the admin endpoint that renders this list.
func TestIntegrationManagerListOmitsSecret(t *testing.T) {
	mgr := integration.NewManager()
	mgr.Register(&integration.IntegrationConfig{
		ID: "int_secret", Name: "Signed", Type: "webhook",
		TargetURL: "https://example.invalid/hook", SecretKey: "s3cret",
	})

	for _, cfg := range mgr.List() {
		if cfg.SecretKey != "" {
			t.Fatalf("List() exposed the signing secret for %s", cfg.ID)
		}
	}
}

// Register must not keep the caller's pointer: mutating it afterwards would
// change registered state with no lock held.
func TestIntegrationManagerRegisterCopies(t *testing.T) {
	mgr := integration.NewManager()
	cfg := &integration.IntegrationConfig{
		ID: "int_copy", Name: "Original", Type: "webhook", TargetURL: "https://example.invalid/hook",
	}
	mgr.Register(cfg)
	cfg.Name = "Mutated after registration"

	for _, stored := range mgr.List() {
		if stored.ID == "int_copy" && stored.Name != "Original" {
			t.Fatalf("Register aliased the caller's config: name is now %q", stored.Name)
		}
	}
}

// Dispatch is asynchronous, so it must not inherit the caller's cancellation.
// ctx is a request context in every real caller and is cancelled as soon as the
// handler returns — which is before the POST leaves the process, so every
// webhook used to race its own cancellation.
func TestDispatchEventSurvivesCallerCancellation(t *testing.T) {
	delivered := make(chan string, 1)
	subscriber := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivered <- r.Header.Get("X-ERP-Event")
		w.WriteHeader(http.StatusOK)
	}))
	defer subscriber.Close()

	mgr := integration.NewManager()
	mgr.Register(&integration.IntegrationConfig{
		ID: "int_test_webhook", Name: "Test Webhook", Type: "webhook", TargetURL: subscriber.URL,
	})

	ctx, cancel := context.WithCancel(context.Background())
	if err := mgr.DispatchEvent(ctx, integration.EventPayload{
		EventID:   "evt_1001",
		EventType: "contact.created",
		TenantID:  "00000000-0000-0000-0000-000000000001",
		Timestamp: time.Now(),
		Data:      map[string]any{"name": "Test User"},
	}); err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}
	// Exactly what a returning HTTP handler does to its request context.
	cancel()

	select {
	case eventType := <-delivered:
		if eventType != "contact.created" {
			t.Fatalf("X-ERP-Event = %q, want contact.created", eventType)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("webhook was never delivered after the caller's context was cancelled")
	}
}
