/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The resilience machinery, seen from outside.
 */

package observability

import "github.com/prometheus/client_golang/prometheus"

// There is no breaker metric here.
//
// The design document asked for resilience_breaker_state alongside these, but
// the adaptive breaker it referred to was deliberately removed from
// internal/platform/resilience — see that package's header for why. A gauge
// that never leaves zero because nothing writes to it is worse than no gauge:
// it makes a dashboard panel that says "all breakers closed" on a platform that
// has no breakers at all. When one arrives attached to the call it guards, the
// gauge belongs next to it in this file.
var (
	// LoadShedTotal counts requests refused because the in-flight ceiling was
	// already reached.
	LoadShedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "resilience_load_shed_total",
			Help: "Requests refused with 503 because the in-flight ceiling was reached",
		},
	)

	// InFlightRequests is what the load shedder is comparing against its
	// ceiling. Without it, a shed count says something went wrong but not how
	// close to the edge the normal state is.
	InFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "resilience_in_flight_requests",
			Help: "Requests currently being served",
		},
	)

	// RetryTotal counts retried attempts, by the operation doing the retrying.
	// `name` is a constant chosen at the call site, never anything from a
	// request — a subscriber URL in this label would be unbounded cardinality
	// driven by tenant input.
	RetryTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resilience_retry_total",
			Help: "Attempts made after a first attempt failed, by operation",
		},
		[]string{"name"},
	)
)

func init() {
	prometheus.MustRegister(LoadShedTotal, InFlightRequests, RetryTotal)
}

// RecordLoadShed counts one refused request.
func RecordLoadShed() { LoadShedTotal.Inc() }

// RecordRetry counts one retried attempt of a named operation.
func RecordRetry(name string) { RetryTotal.WithLabelValues(name).Inc() }
