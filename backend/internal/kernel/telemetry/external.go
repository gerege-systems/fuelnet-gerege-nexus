/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * One histogram for every call this platform makes to somebody else's system.
 */

package telemetry

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// The systems this platform calls out to. A closed list, because `system` is a
// Prometheus label: an open one would let a typo in a call site mint a time
// series that never goes away, and a caller that passed a hostname would mint
// one per host.
const (
	SystemXYP         = "xyp"         // ХУР — xyp.gerege.mn
	SystemEID         = "eid"         // eID Mongolia / sso.gov.mn
	SystemDAN         = "dan"         // ДАН — dan.gerege.mn
	SystemESign       = "esign"       // Gerege eSign HSM
	SystemGemini      = "gemini"      // the AI copilot's model
	SystemEmailVerify = "emailverify" // the hosted address-verification service
)

// Outcomes. Two values, deliberately: an error string, an HTTP status or a
// provider's own code would each multiply the series count by however many
// distinct failures the far end can invent.
const (
	statusOK    = "ok"
	statusError = "error"
)

// knownSystems is what ExternalSystem accepts. Anything else is folded into
// "other" rather than being trusted, so a call site added later without a
// constant cannot widen the label set.
var knownSystems = map[string]bool{
	SystemXYP:         true,
	SystemEID:         true,
	SystemDAN:         true,
	SystemESign:       true,
	SystemGemini:      true,
	SystemEmailVerify: true,
}

// ExternalRequestDuration times outbound calls, by system, operation and
// outcome.
//
// `operation` is a constant chosen at the call site — "citizen_query",
// "poll", "sign_pdf" — never anything taken from a request. The buckets run
// further out than prometheus.DefBuckets because these are the calls that go
// slow: eID waits on a citizen reaching for their phone, and the HSM's own
// client allows ninety seconds.
var ExternalRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "external_request_duration_seconds",
		Help:    "Latency of calls to systems outside this platform",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30, 60, 120},
	},
	[]string{"system", "operation", "status"},
)

func init() {
	prometheus.MustRegister(ExternalRequestDuration)
}

// ExternalSystem folds an unrecognised name into "other".
func ExternalSystem(name string) string {
	if knownSystems[name] {
		return name
	}
	return "other"
}

// ObserveExternal times one outbound call and records how it ended.
//
// It wraps the call rather than the HTTP transport underneath it. Three of the
// six systems are reached through a client this repository does not own — the
// eID relying-party client and the Gemini client both come from
// open-gerege-core and keep their http.Client private — so a RoundTripper could
// only have covered half of them, and the half it covered would have been
// labelled by URL rather than by the operation the caller meant.
func ObserveExternal(ctx context.Context, system, operation string,
	call func(context.Context) error) error {

	_, err := ObserveExternalValue(ctx, system, operation,
		func(callCtx context.Context) (struct{}, error) {
			return struct{}{}, call(callCtx)
		})
	return err
}

// ObserveExternalValue is ObserveExternal for a call that returns something.
//
// It also opens a span, so a trace of a slow sign-in shows the eID call as a
// segment of it rather than as an unexplained gap. With tracing off the tracer
// is a no-op and the two extra lines cost an interface call.
func ObserveExternalValue[T any](ctx context.Context, system, operation string,
	call func(context.Context) (T, error)) (T, error) {

	// The call is handed the span's context rather than the caller's, so that
	// anything instrumented further down — an otelhttp transport, a nested
	// call — hangs off this span instead of off the request.
	ctx, span := Tracer().Start(ctx, ExternalSystem(system)+"."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(ExternalSpanAttributes(system, operation)...))
	defer span.End()

	start := time.Now()
	result, err := call(ctx)
	status := statusOK
	if err != nil {
		status = statusError
		// The message, not the error value: an error from an identity provider
		// can carry a subject or a token fragment, and a span is stored for as
		// long as Tempo keeps it. RecordError would attach the full string.
		span.SetStatus(codes.Error, "the call failed")
	}
	ExternalRequestDuration.
		WithLabelValues(ExternalSystem(system), operation, status).
		Observe(time.Since(start).Seconds())
	return result, err
}
