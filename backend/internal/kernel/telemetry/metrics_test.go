package telemetry_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/telemetry"
)

func TestMetricsMiddleware(t *testing.T) {
	handler := telemetry.MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test-metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestMetricsHandler(t *testing.T) {
	metricsHandler := telemetry.MetricsHandler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	metricsHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /metrics, got %d", rec.Code)
	}
}
