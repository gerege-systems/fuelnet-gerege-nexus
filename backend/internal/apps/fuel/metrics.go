/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The chain, as numbers a dashboard can watch.
 *
 * Everything else this module exposes answers a question somebody asked: an
 * operator opening a screen, a citizen tapping a station. These answer the
 * question nobody is awake to ask — is fuel still moving, and is it moving at
 * every point on the chain or only at some of them.
 *
 * # Why it is national, and has no tenant label
 *
 * Prometheus keeps what it is given for sixty days and Grafana's access
 * controls are coarser than the platform's. A `tenant` label here would put one
 * company's stock curve, delivery cadence and turnover in front of whoever
 * holds a dashboard login — a competitor's read of a business, published by
 * accident and then kept for two months. The national figure carries what an
 * operations dashboard is for, and carries nothing a rival could act on.
 *
 * The same rule as FUELNET_OVERSIGHT_PLAN.md sets for the regulator's queries:
 * COUNT and SUM, never a row. `fuel_vouchers.citizen_id` appears nowhere below,
 * not even inside a COUNT DISTINCT that could later be relaxed into a GROUP BY.
 *
 * # Why the queries run on scrape
 *
 * Prometheus asks every fifteen seconds and these are seven aggregate queries
 * over small tables. A cache refreshed on its own timer would be a second
 * schedule to reason about and a window in which the dashboard shows a number
 * that was true a minute ago — which is exactly the failure `stock_age` below
 * exists to make visible. If they ever become expensive the answer is a
 * materialised view, not a goroutine.
 */

package fuel

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/prometheus/client_golang/prometheus"
)

// How long the whole collection may take.
//
// Shorter than Prometheus's own scrape timeout, so a slow database shows up as
// missing fuel metrics rather than as a failed scrape that also loses the HTTP
// and pool metrics served from the same endpoint.
const metricsTimeout = 5 * time.Second

var (
	stationsDesc = prometheus.NewDesc(
		"fuel_stations_total",
		"Filling stations on the register, by operational status.",
		[]string{"status"}, nil)

	stationStockDesc = prometheus.NewDesc(
		"fuel_station_stock_liters",
		"Litres in forecourt tanks nationally, by grade.",
		[]string{"fuel_type"}, nil)

	stationCapacityDesc = prometheus.NewDesc(
		"fuel_station_capacity_liters",
		"Forecourt tank capacity nationally, by grade.",
		[]string{"fuel_type"}, nil)

	// Counted rather than averaged. A national mean hides the case that
	// matters: most forecourts comfortable and forty of them dry is the same
	// average as every forecourt at two thirds, and only one of those is an
	// incident.
	stationsLowDesc = prometheus.NewDesc(
		"fuel_stations_low_stock_total",
		"Forecourts whose fullest tank is below a fifth, by grade.",
		[]string{"fuel_type"}, nil)

	// Age, not freshness. A forecourt last heard from a day ago is not a
	// forecourt holding that much fuel — it is one nobody has heard from, and
	// its litres are still being added into the national total above.
	stockAgeDesc = prometheus.NewDesc(
		"fuel_station_stock_age_seconds",
		"Age of the oldest stock report on the register.",
		nil, nil)

	stockStaleDesc = prometheus.NewDesc(
		"fuel_station_stock_stale_total",
		"Forecourt tanks whose stock has not been reported in 24 hours.",
		nil, nil)

	shipmentsDesc = prometheus.NewDesc(
		"fuel_customs_shipments_total",
		"Consignments at or past the border, by state.",
		[]string{"status"}, nil)

	shipmentLitersDesc = prometheus.NewDesc(
		"fuel_customs_declared_liters",
		"Litres declared at customs and not yet at a depot, by state.",
		[]string{"status"}, nil)

	depotTankDesc = prometheus.NewDesc(
		"fuel_depot_tank_liters",
		"Litres in depot tanks nationally, by grade.",
		[]string{"fuel_type"}, nil)

	depotCapacityDesc = prometheus.NewDesc(
		"fuel_depot_tank_capacity_liters",
		"Depot tank capacity nationally, by grade.",
		[]string{"fuel_type"}, nil)

	depotsDesc = prometheus.NewDesc(
		"fuel_depots_total",
		"Storage bases on the register, by status.",
		[]string{"status"}, nil)

	tripsDesc = prometheus.NewDesc(
		"fuel_dispatch_trips_total",
		"Tanker runs, by state.",
		[]string{"status"}, nil)

	tripsInFlightDesc = prometheus.NewDesc(
		"fuel_dispatch_trips_in_flight",
		"Tanker runs that have departed and not arrived.",
		nil, nil)

	tripLitersDesc = prometheus.NewDesc(
		"fuel_dispatch_in_flight_liters",
		"Litres on the road right now.",
		nil, nil)

	// The gap the whole chain exists to expose. Imported is what crossed the
	// border; received is what reached a forecourt. The difference is fuel in
	// depots, fuel on lorries, and fuel nobody can account for — this cannot
	// say which, and a platform that stored only one of the two could not say
	// there was a difference at all.
	batchImportedDesc = prometheus.NewDesc(
		"fuel_batch_imported_liters_total",
		"Litres on batches minted at the border.",
		nil, nil)

	batchReceivedDesc = prometheus.NewDesc(
		"fuel_batch_received_liters_total",
		"Litres from those batches that reached a forecourt.",
		nil, nil)

	batchesDesc = prometheus.NewDesc(
		"fuel_batches_total",
		"Batches on the register, by laboratory result.",
		[]string{"lab_status"}, nil)

	// Vouchers and rations, in aggregate only. No citizen appears here, in a
	// label or in a distinct count that a later edit could widen into one.
	vouchersDesc = prometheus.NewDesc(
		"fuel_vouchers_today_total",
		"Vouchers drawn against today's ration, by state.",
		[]string{"status"}, nil)

	voucherAmountDesc = prometheus.NewDesc(
		"fuel_vouchers_today_mnt",
		"Tugrik on vouchers drawn today, by state.",
		[]string{"status"}, nil)

	entitlementGrantedDesc = prometheus.NewDesc(
		"fuel_entitlement_today_granted_mnt",
		"Tugrik of daily ration granted today.",
		nil, nil)

	entitlementUsedDesc = prometheus.NewDesc(
		"fuel_entitlement_today_used_mnt",
		"Tugrik of daily ration spent today.",
		nil, nil)

	entitlementPeopleDesc = prometheus.NewDesc(
		"fuel_entitlement_today_people",
		"How many people drew on the ration today.",
		nil, nil)

	// Whether the collection worked at all. Without it a database that has
	// stopped answering looks identical to a country that has stopped moving
	// fuel — every gauge simply absent, and no alert able to tell the two apart.
	upDesc = prometheus.NewDesc(
		"fuel_metrics_up",
		"1 when the last collection of fuel metrics succeeded.",
		nil, nil)

	scrapeDurationDesc = prometheus.NewDesc(
		"fuel_metrics_scrape_duration_seconds",
		"How long the last collection of fuel metrics took.",
		nil, nil)
)

// Collector reads the chain on every scrape.
type Collector struct{ db nexus.DB }

// RegisterMetrics puts the fuel chain on /metrics.
//
// Called by the module's constructor. A failure to register is logged rather
// than fatal: a deployment that cannot export these is a deployment with a
// blind spot, not one that should refuse to serve fuel.
//
// Registering twice is not one of those failures. The module is constructed
// once per process in production and more than once in tests, which build
// several servers against one registry — and a test that fell over on the
// second one would be failing for a reason that cannot happen in the thing it
// is testing. errors.As rather than a type assertion: the registry is free to
// wrap, and a wrapped AlreadyRegisteredError would otherwise read as a real
// fault and log a warning that means nothing.
func RegisterMetrics(db nexus.DB) {
	err := prometheus.Register(&Collector{db: db})
	if err == nil {
		return
	}
	var already prometheus.AlreadyRegisteredError
	if errors.As(err, &already) {
		return
	}
	slog.Warn("fuel: metrics are not being exported", "error", err)
}

// Describe is deliberately empty, which makes this an unchecked collector.
//
// The label values are whatever the database holds — a status somebody adds in
// a migration appears here without a code change — so the set of series cannot
// be declared in advance. Prometheus allows this and only gives up the
// duplicate-registration check, which the descriptors above already provide by
// being distinct.
func (c *Collector) Describe(chan<- *prometheus.Desc) {}

// Collect runs the aggregates and emits them.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), metricsTimeout)
	defer cancel()

	ok := 1.0
	fail := func(what string, err error) {
		ok = 0
		slog.Warn("fuel: could not collect a metric", "metric", what, "error", err)
	}

	gauge := func(desc *prometheus.Desc, value float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labels...)
	}

	// ---- the forecourts ------------------------------------------------

	if err := c.each(ctx, `
		SELECT status, count(*)::float8 FROM fuel_stations GROUP BY status`,
		func(label string, value float64) { gauge(stationsDesc, value, label) }); err != nil {
		fail("stations", err)
	}

	if err := c.eachTriple(ctx, `
		SELECT fuel_type,
		       COALESCE(sum(current_stock_liters), 0)::float8,
		       COALESCE(sum(tank_capacity_liters), 0)::float8,
		       count(*) FILTER (
		           WHERE tank_capacity_liters > 0
		             AND current_stock_liters / tank_capacity_liters < 0.2)::float8
		  FROM fuel_station_inventory
		 GROUP BY fuel_type`,
		func(label string, stock, capacity, low float64) {
			gauge(stationStockDesc, stock, label)
			gauge(stationCapacityDesc, capacity, label)
			gauge(stationsLowDesc, low, label)
		}); err != nil {
		fail("station_stock", err)
	}

	if err := c.pair(ctx, `
		SELECT COALESCE(EXTRACT(EPOCH FROM (NOW() - min(last_reported_at))), 0)::float8,
		       count(*) FILTER (WHERE last_reported_at < NOW() - INTERVAL '24 hours')::float8
		  FROM fuel_station_inventory`,
		func(age, stale float64) {
			gauge(stockAgeDesc, age)
			gauge(stockStaleDesc, stale)
		}); err != nil {
		fail("stock_age", err)
	}

	// ---- the border ----------------------------------------------------

	if err := c.eachPair(ctx, `
		SELECT status, count(*)::float8, COALESCE(sum(declared_liters), 0)::float8
		  FROM fuel_customs_shipments GROUP BY status`,
		func(label string, count, liters float64) {
			gauge(shipmentsDesc, count, label)
			gauge(shipmentLitersDesc, liters, label)
		}); err != nil {
		fail("shipments", err)
	}

	// ---- the depots ----------------------------------------------------

	if err := c.each(ctx, `
		SELECT status, count(*)::float8 FROM fuel_depots GROUP BY status`,
		func(label string, value float64) { gauge(depotsDesc, value, label) }); err != nil {
		fail("depots", err)
	}

	if err := c.eachPair(ctx, `
		SELECT fuel_type,
		       COALESCE(sum(current_liters), 0)::float8,
		       COALESCE(sum(capacity_liters), 0)::float8
		  FROM fuel_depot_tanks GROUP BY fuel_type`,
		func(label string, current, capacity float64) {
			gauge(depotTankDesc, current, label)
			gauge(depotCapacityDesc, capacity, label)
		}); err != nil {
		fail("depot_tanks", err)
	}

	// ---- the road ------------------------------------------------------

	if err := c.each(ctx, `
		SELECT status, count(*)::float8 FROM fuel_dispatch_trips GROUP BY status`,
		func(label string, value float64) { gauge(tripsDesc, value, label) }); err != nil {
		fail("trips", err)
	}

	if err := c.pair(ctx, `
		SELECT count(*)::float8, COALESCE(sum(volume_liters), 0)::float8
		  FROM fuel_dispatch_trips WHERE completed_at IS NULL`,
		func(count, liters float64) {
			gauge(tripsInFlightDesc, count)
			gauge(tripLitersDesc, liters)
		}); err != nil {
		fail("trips_in_flight", err)
	}

	// ---- the batches ---------------------------------------------------

	if err := c.pair(ctx, `
		SELECT COALESCE(sum(imported_liters), 0)::float8,
		       COALESCE(sum(received_liters), 0)::float8
		  FROM fuel_batches`,
		func(imported, received float64) {
			gauge(batchImportedDesc, imported)
			gauge(batchReceivedDesc, received)
		}); err != nil {
		fail("batches", err)
	}

	if err := c.each(ctx, `
		SELECT lab_status, count(*)::float8 FROM fuel_batches GROUP BY lab_status`,
		func(label string, value float64) { gauge(batchesDesc, value, label) }); err != nil {
		fail("batch_lab", err)
	}

	// ---- the ration ----------------------------------------------------

	if err := c.eachPair(ctx, `
		SELECT status, count(*)::float8, COALESCE(sum(amount_mnt), 0)::float8
		  FROM fuel_vouchers WHERE for_date = CURRENT_DATE GROUP BY status`,
		func(label string, count, amount float64) {
			gauge(vouchersDesc, count, label)
			gauge(voucherAmountDesc, amount, label)
		}); err != nil {
		fail("vouchers", err)
	}

	if err := c.triple(ctx, `
		SELECT COALESCE(sum(granted_mnt), 0)::float8,
		       COALESCE(sum(used_mnt), 0)::float8,
		       count(*)::float8
		  FROM fuel_entitlements WHERE for_date = CURRENT_DATE`,
		func(granted, used, people float64) {
			gauge(entitlementGrantedDesc, granted)
			gauge(entitlementUsedDesc, used)
			gauge(entitlementPeopleDesc, people)
		}); err != nil {
		fail("entitlements", err)
	}

	gauge(upDesc, ok)
	gauge(scrapeDurationDesc, time.Since(started).Seconds())
}

// The four shapes every query above takes. Written out rather than reached for
// with reflection: the SQL and the emit are next to each other in Collect, and
// these only carry the rows between them.

func (c *Collector) each(ctx context.Context, query string, emit func(string, float64)) error {
	rows, err := c.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var label string
		var value float64
		if err := rows.Scan(&label, &value); err != nil {
			return err
		}
		emit(label, value)
	}
	return rows.Err()
}

func (c *Collector) eachPair(ctx context.Context, query string, emit func(string, float64, float64)) error {
	rows, err := c.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var label string
		var first, second float64
		if err := rows.Scan(&label, &first, &second); err != nil {
			return err
		}
		emit(label, first, second)
	}
	return rows.Err()
}

func (c *Collector) eachTriple(ctx context.Context, query string, emit func(string, float64, float64, float64)) error {
	rows, err := c.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var label string
		var first, second, third float64
		if err := rows.Scan(&label, &first, &second, &third); err != nil {
			return err
		}
		emit(label, first, second, third)
	}
	return rows.Err()
}

func (c *Collector) pair(ctx context.Context, query string, emit func(float64, float64)) error {
	var first, second float64
	if err := c.db.QueryRow(ctx, query).Scan(&first, &second); err != nil {
		return err
	}
	emit(first, second)
	return nil
}

func (c *Collector) triple(ctx context.Context, query string, emit func(float64, float64, float64)) error {
	var first, second, third float64
	if err := c.db.QueryRow(ctx, query).Scan(&first, &second, &third); err != nil {
		return err
	}
	emit(first, second, third)
	return nil
}
