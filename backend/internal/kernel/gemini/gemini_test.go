package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// What the client promises: a key it does not have is not a request, a
// temporary failure is retried and then named, and a permanent one is not.

func TestWithoutAKeyNothingIsSent(t *testing.T) {
	asked := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { asked = true }))
	t.Cleanup(server.Close)

	_, err := NewClient(server.URL, "", "").GenerateContent(context.Background(), Request{})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error is %v, want ErrNotConfigured", err)
	}
	if asked {
		t.Error("a request was sent without a key")
	}
}

func TestATemporaryFailureIsRetriedAndThenNamed(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "key", "")
	client.sleep = func(context.Context, time.Duration) error { return nil }

	_, err := client.GenerateContent(context.Background(), Request{})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error is %v, want ErrUnavailable", err)
	}
	if attempts != maxAttempts {
		t.Errorf("tried %d times, want %d", attempts, maxAttempts)
	}
}

// A rejected key is not going to be accepted on the second attempt, and
// retrying it wastes the caller's request deadline.
func TestAPermanentFailureIsNotRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "key", "")
	client.sleep = func(context.Context, time.Duration) error { return nil }

	if _, err := client.GenerateContent(context.Background(), Request{}); err == nil {
		t.Fatal("a 403 was reported as success")
	} else if errors.Is(err, ErrUnavailable) {
		t.Errorf("a 403 was reported as temporary: %v", err)
	}
	if attempts != 1 {
		t.Errorf("tried %d times, want 1", attempts)
	}
}

func TestTheAnswerIsReadIntoItsParts(t *testing.T) {
	var path, key string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, key = r.URL.Path, r.Header.Get("x-goog-api-key")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]any{
				{"text": "Сайн "},
				{"text": "байна уу"},
				{"functionCall": map[string]any{"name": "listApps", "args": map[string]any{"limit": 5}}},
				{"inlineData": map[string]any{"mimeType": "audio/L16;rate=24000", "data": "AAAA"}},
			}}}},
		})
	}))
	t.Cleanup(server.Close)

	resp, err := NewClient(server.URL, "the-key", "gemini-test").
		GenerateContent(context.Background(), Request{Contents: []Content{{Role: "user"}}})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if path != "/models/gemini-test:generateContent" {
		t.Errorf("called %s", path)
	}
	if key != "the-key" {
		t.Errorf("the key header was %q", key)
	}
	if resp.Text() != "Сайн байна уу" {
		t.Errorf("Text() = %q — the parts should join", resp.Text())
	}
	calls := resp.FunctionCalls()
	if len(calls) != 1 || calls[0].Name != "listApps" {
		t.Errorf("FunctionCalls() = %v", calls)
	}
	if audio := resp.InlineAudio(); audio == nil || audio.Data != "AAAA" {
		t.Errorf("InlineAudio() = %v", audio)
	}
	// The model's turn goes back into the conversation during a function-calling
	// loop, and it has to carry a role even when the answer omitted one.
	if resp.ModelContent().Role != "model" {
		t.Errorf("ModelContent().Role = %q", resp.ModelContent().Role)
	}
}
