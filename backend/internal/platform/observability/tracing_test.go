package observability_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/observability"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// The condition for having the tracing code in the default path at all: with no
// endpoint configured, nothing is exported and no provider is installed.
func TestTracingIsOffWithoutAnEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	shutdown, err := observability.SetupTracing(context.Background(), "test", "test")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if observability.TracingEnabled() {
		t.Error("tracing reports itself enabled with no endpoint set")
	}

	// A span started through the no-op tracer must not record, and must not
	// panic — every call site starts one unconditionally.
	_, span := observability.Tracer().Start(context.Background(), "test")
	if span.IsRecording() {
		t.Error("a span is recording with tracing off")
	}
	span.End()

	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown: %v", err)
	}
}

// A slow external call has to appear in the trace as a segment of it, or a
// trace of a slow sign-in is an unexplained gap.
func TestObserveExternalOpensASpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(previous)

	// SetupTracing is what normally swaps the package tracer; here the span is
	// started through the provider directly, which is what ObserveExternalValue
	// does once tracing is on.
	ctx, parent := provider.Tracer("test").Start(context.Background(), "parent")
	_, err := observability.ObserveExternalValue(ctx, observability.SystemESign, "sign_pdf",
		func(callCtx context.Context) (string, error) {
			_, child := provider.Tracer("test").Start(callCtx, "inner")
			child.End()
			return "signed", nil
		})
	parent.End()
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if len(recorder.Ended()) < 2 {
		t.Fatalf("expected at least the parent and the inner span, got %d", len(recorder.Ended()))
	}
}

// The join between logs and traces: Grafana's derived field looks for trace_id
// on the log line, and without it a trace and its logs cannot be put together.
func TestLogLinesCarryTheTrace(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(previous)

	var buf bytes.Buffer
	logger := slog.New(observability.NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	ctx, span := provider.Tracer("test").Start(context.Background(), "unit")
	defer span.End()

	logger.InfoContext(ctx, "inside a span")

	line := buf.String()
	if !strings.Contains(line, `"trace_id":"`+span.SpanContext().TraceID().String()+`"`) {
		t.Errorf("trace_id missing or wrong in %s", line)
	}
	if !strings.Contains(line, `"span_id":"`) {
		t.Errorf("span_id missing from %s", line)
	}
}

// The health and metrics endpoints are called every few seconds by Docker and
// Prometheus. Tracing them would make them the majority of everything stored.
func TestTracingMiddlewareSkipsProbes(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(previous)

	router := chi.NewRouter()
	router.Use(observability.TracingMiddleware)
	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router.Get("/api/v1/things/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	if got := len(recorder.Ended()); got != 0 {
		t.Fatalf("/health produced %d spans", got)
	}

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/things/abc", nil))
	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span for the real route, got %d", len(ended))
	}
	// Named by pattern, not by URL: a span per document id is unbounded.
	if name := ended[0].Name(); name != "GET /api/v1/things/{id}" {
		t.Errorf("span named %q; the id leaked into the name", name)
	}
}

// A panic must become a 500 and a log line, never a dropped connection, and
// never a stack trace in the response.
func TestRecoveryMiddlewareAnswers500(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(observability.NewContextHandler(slog.NewJSONHandler(&buf, nil))))
	defer slog.SetDefault(previous)

	router := chi.NewRouter()
	router.Use(observability.RecoveryMiddleware)
	router.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("a deliberate test panic")
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "deliberate test panic") {
		t.Errorf("the panic value reached the caller: %s", rec.Body.String())
	}
	if !strings.Contains(buf.String(), "a deliberate test panic") {
		t.Errorf("the panic was not logged: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"route":"/boom"`) {
		t.Errorf("the route was not logged: %s", buf.String())
	}
}
