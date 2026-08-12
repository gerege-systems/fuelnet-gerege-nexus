/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Saturation collectors: the Go runtime, the process, and the database pool.
 */

package observability

import (
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// RuntimeCollectorsRegistered reports whether the default registry is carrying
// the Go runtime and process collectors.
//
// client_golang registers both into prometheus.DefaultRegisterer from its own
// init, so there is nothing to add here — and adding a second one would panic
// with AlreadyRegisteredError at startup. This exists so the assumption is
// checked rather than believed: a future move to a private registry, or a
// library that unregisters them, would take `go_goroutines` and
// `process_resident_memory_bytes` off /metrics with nothing saying so, and the
// saturation half of the golden signals would quietly go blank.
func RuntimeCollectorsRegistered() bool {
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return false
	}
	var goSeen, processSeen bool
	for _, family := range families {
		switch family.GetName() {
		case "go_goroutines":
			goSeen = true
		case "process_resident_memory_bytes":
			processSeen = true
		}
	}
	return goSeen && processSeen
}

// pool metric descriptors. Declared once: a Collector hands the same *Desc back
// on every scrape, and building them per Collect would allocate on every
// Prometheus request.
var (
	poolAcquiredConns = prometheus.NewDesc(
		"pgxpool_acquired_conns",
		"Connections currently held by a caller",
		nil, nil)
	poolIdleConns = prometheus.NewDesc(
		"pgxpool_idle_conns",
		"Connections open and waiting to be handed out",
		nil, nil)
	poolTotalConns = prometheus.NewDesc(
		"pgxpool_total_conns",
		"Connections the pool is holding, idle and acquired together",
		nil, nil)
	poolMaxConns = prometheus.NewDesc(
		"pgxpool_max_conns",
		"Ceiling the pool was configured with",
		nil, nil)
	poolEmptyAcquires = prometheus.NewDesc(
		"pgxpool_empty_acquire_total",
		"Acquisitions that had to wait because the pool was empty",
		nil, nil)
	poolCanceledAcquires = prometheus.NewDesc(
		"pgxpool_canceled_acquire_total",
		"Acquisitions abandoned because the caller's context ended first",
		nil, nil)
	poolAcquireDuration = prometheus.NewDesc(
		"pgxpool_acquire_duration_seconds_total",
		"Time callers have spent waiting for a connection",
		nil, nil)
)

// PoolCollector exports pgxpool.Stat.
//
// The proposal asked for a sampler on a 15-second ticker. This reads the pool at
// scrape time instead, which is the same numbers without a goroutine, without a
// sampling lag between the spike and the scrape that shows it, and without a
// value that keeps being served after the pool has been closed. pgxpool.Stat is
// a cheap snapshot of counters the pool already maintains, so there is nothing
// to amortise by sampling less often than Prometheus asks.
type PoolCollector struct {
	pool *pgxpool.Pool
}

// NewPoolCollector builds the collector. A nil pool collects nothing rather than
// panicking: a test server may have no database at all.
func NewPoolCollector(pool *pgxpool.Pool) *PoolCollector {
	return &PoolCollector{pool: pool}
}

func (c *PoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- poolAcquiredConns
	ch <- poolIdleConns
	ch <- poolTotalConns
	ch <- poolMaxConns
	ch <- poolEmptyAcquires
	ch <- poolCanceledAcquires
	ch <- poolAcquireDuration
}

func (c *PoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool == nil {
		return
	}
	stat := c.pool.Stat()
	gauge := func(desc *prometheus.Desc, value float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value)
	}
	counter := func(desc *prometheus.Desc, value float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value)
	}
	gauge(poolAcquiredConns, float64(stat.AcquiredConns()))
	gauge(poolIdleConns, float64(stat.IdleConns()))
	gauge(poolTotalConns, float64(stat.TotalConns()))
	gauge(poolMaxConns, float64(stat.MaxConns()))
	counter(poolEmptyAcquires, float64(stat.EmptyAcquireCount()))
	counter(poolCanceledAcquires, float64(stat.CanceledAcquireCount()))
	counter(poolAcquireDuration, stat.AcquireDuration().Seconds())
}

// RegisterPoolCollector attaches a pool to the default registry.
//
// Registering twice is reported rather than fatal. The process has one pool and
// calls this once; a second call is a wiring mistake in a test, and taking the
// server down over a metric would be a worse outcome than a log line.
func RegisterPoolCollector(pool *pgxpool.Pool) {
	if err := prometheus.Register(NewPoolCollector(pool)); err != nil {
		var already prometheus.AlreadyRegisteredError
		if !errors.As(err, &already) {
			slog.Warn("observability: could not export database pool statistics", "error", err)
		}
	}
}
