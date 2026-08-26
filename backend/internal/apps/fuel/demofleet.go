/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Keeping invented lorries on the road.
 *
 * cmd/fuel-demo-trips puts a fleet out once. Every run then arrives, and half an
 * hour later the map is empty — which is a fair picture of a dispatch board
 * nobody is operating, and a poor demonstration of one that is.
 *
 * This closes the loop: arrived runs are marked done and replacements are
 * dispatched, so the board stays populated without anybody re-running a command.
 *
 * # It is off unless asked for
 *
 * FUEL_DEMO_DISPATCH=true. A deployment carrying real dispatch data must never
 * find invented lorries appearing beside it, and "the demo seeder was left on"
 * is exactly the kind of thing that survives into production when the default
 * is the convenient one. Everything it writes carries source='demo', so what it
 * made is always separable from what an operator did.
 */

package fuel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// How many invented runs to keep in flight, and how often to look.
//
// Forty-five seconds is far finer than the thing it watches — a run lasts an
// hour or more — and that is deliberate: the cost is one cheap query, and the
// alternative is a gap where a lorry has arrived and its replacement has not
// been thought of yet.
const (
	demoFleetSize     = 16
	demoSweepInterval = 45 * time.Second
	demoSource        = "demo"
)

// Depots, as in the seeder. Real rail terminals and storage.
var demoDepots = []struct {
	name     string
	lat, lon float64
}{
	{"Сүхбаатар боомт", 50.2350, 106.2070},
	{"Замын-Үүд боомт", 43.7200, 111.8980},
	{"Дархан нефть бааз", 49.4860, 105.9220},
	{"Улаанбаатар төв бааз", 47.9060, 106.8300},
	{"Багануур бааз", 47.8280, 108.3450},
	{"Эрдэнэт бааз", 49.0280, 104.0450},
}

var demoFuels = []struct{ code, label string }{
	{"ai92", "АИ-92"},
	{"ai95", "АИ-95"},
	{"diesel", "Дизель (ДТ)"},
}

var demoPlateSuffix = []string{"УБА", "УБЕ", "УНС", "УБН", "ХӨА", "СБА"}

// StartHousekeeping is nexus's background hook. The platform hands it a context
// that is cancelled on shutdown, so the sweep stops with the process rather than
// holding a database connection open through it.
func (m *Module) StartHousekeeping(ctx context.Context) {
	if os.Getenv("FUEL_DEMO_DISPATCH") != "true" {
		return
	}
	slog.Info("fuel: invented deliveries will be kept running", "fleet", demoFleetSize)

	go func() {
		// Once immediately: a deployment that has just started should not show
		// an empty board for the first forty-five seconds.
		m.rollDemoFleet(ctx)

		ticker := time.NewTicker(demoSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.rollDemoFleet(ctx)
			}
		}
	}()
}

// rollDemoFleet retires what has arrived and dispatches what is missing.
func (m *Module) rollDemoFleet(ctx context.Context) {
	// Arrived. `status` and `completed_at` together, so a run is either in
	// flight or finished and never half of each.
	if _, err := m.db.Exec(ctx, `
		UPDATE fuel_dispatch_trips
		   SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		 WHERE source = $1 AND completed_at IS NULL AND eta_at <= NOW()`, demoSource); err != nil {
		slog.Warn("fuel: could not retire arrived demo runs", "error", err)
		return
	}

	var active int
	if err := m.db.QueryRow(ctx, `
		SELECT count(*) FROM fuel_dispatch_trips
		 WHERE source = $1 AND completed_at IS NULL`, demoSource).Scan(&active); err != nil {
		slog.Warn("fuel: could not count demo runs", "error", err)
		return
	}

	for range demoFleetSize - active {
		if err := m.dispatchOneDemoRun(ctx); err != nil {
			// One failure is usually the router being slow. Stop this sweep
			// rather than hammering it; the next one is in forty-five seconds.
			slog.Warn("fuel: could not dispatch a demo run", "error", err)
			return
		}
	}
}

// dispatchOneDemoRun sends a lorry to a station somebody might be looking at.
func (m *Module) dispatchOneDemoRun(ctx context.Context) error {
	// A station around the capital, chosen at random. Nationwide was tried and
	// looked broken: the register spans Mongolia, so a city-sized viewport held
	// almost none of the fleet.
	var (
		stationID  string
		tenantID   string
		sLat, sLon float64
	)
	err := m.db.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, lat, lon
		  FROM fuel_stations
		 WHERE lat BETWEEN 47.65 AND 48.15 AND lon BETWEEN 106.5 AND 107.4
		 ORDER BY random()
		 LIMIT 1`).Scan(&stationID, &tenantID, &sLat, &sLon)
	if err != nil {
		return fmt.Errorf("pick a destination: %w", err)
	}

	// Usually the nearest depot — what a dispatcher would choose, and what keeps
	// a city delivery on city roads. One in four comes from further out, because
	// border runs are the shape of the supply chain this platform is for.
	depot := demoDepots[0]
	best := math.Inf(1)
	for _, candidate := range demoDepots {
		dLat := candidate.lat - sLat
		dLon := (candidate.lon - sLon) * math.Cos(sLat*math.Pi/180)
		if distance := dLat*dLat + dLon*dLon; distance < best {
			depot, best = candidate, distance
		}
	}
	if rand.IntN(4) == 0 {
		depot = demoDepots[rand.IntN(len(demoDepots))]
	}

	route, _, seconds, routeErr := roadRoute(ctx, depot.lat, depot.lon, sLat, sLon)
	if routeErr != nil {
		return routeErr
	}

	// A batch for this load.
	//
	// One per run, which is not how a real import works — a batch is thousands
	// of tonnes crossing a border and is split across many tankers. It is right
	// for a demonstration, because the point being shown is that the chain
	// holds: this litre came from that batch, which entered at that port, with
	// that laboratory certificate.
	fuel := demoFuels[rand.IntN(len(demoFuels))]

	origin, refinery := "ОХУ", "Ангарскийн НПЗ"
	if depot.name == "Замын-Үүд боомт" {
		origin, refinery = "БНХАУ", "Sinopec"
	}
	volume := float64(12000 + rand.IntN(4)*4000)
	var batchID string
	err = m.db.QueryRow(ctx, `
		INSERT INTO fuel_batches
		       (tenant_id, batch_code, fuel_type, fuel_label, origin_country,
		        refinery, imported_liters, quality_cert_no, octane_tested,
		        sulfur_ppm, lab_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'passed')
		RETURNING id::text`,
		tenantID,
		fmt.Sprintf("BATCH-%d-%06d", time.Now().Year(), rand.IntN(999999)),
		fuel.code, fuel.label, origin, refinery, volume,
		fmt.Sprintf("LAB-%05d", rand.IntN(99999)),
		octaneFor(fuel.code),
		// Euro-5 дизель 10 ppm хүртэл; бусад нь өндөр. Тоо нь дүр эсгэсэн ч
		// хэмжээсийн зэрэг нь бодитой — К4 энэ багана дээр ажиллана.
		float64(8+rand.IntN(42)),
	).Scan(&batchID)
	if err != nil {
		return fmt.Errorf("mint a batch: %w", err)
	}

	// Lorries are slower than the router's car profile and they stop.
	total := time.Duration(float64(seconds)*1.35) * time.Second
	if total < 20*time.Minute {
		total = 20 * time.Minute
	}
	routeJSON, _ := json.Marshal(route)

	// Departing now, every time. The first fleet was scattered along its runs so
	// the map did not look staged; a replacement joins a board that is already
	// staggered, so it starts where a real one would — at the depot gate.
	_, err = m.db.Exec(ctx, `
		INSERT INTO fuel_dispatch_trips
		       (tenant_id, trip_code, tanker_plate, driver_name, driver_phone,
		        from_depot, origin_lat, origin_lon, to_station_id,
		        fuel_type, fuel_label, volume_liters,
		        seal_no, seal_status, status,
		        departed_at, eta_at, source, source_ref,
		        route_geom, route_distance_m, route_duration_s, batch_id)
		VALUES ($1, $2, $3, '—', '—', $4, $5, $6, $7::uuid, $8, $9, $10,
		        $11, 'sealed_intact', 'in_transit',
		        NOW(), NOW() + $12::interval, $13, $14,
		        $15::jsonb, NULL, $16, $17::uuid)`,
		tenantID,
		fmt.Sprintf("TRIP-%d-%06d", time.Now().Year(), rand.IntN(999999)),
		fmt.Sprintf("%04d%s", 1000+rand.IntN(8999), demoPlateSuffix[rand.IntN(len(demoPlateSuffix))]),
		depot.name, depot.lat, depot.lon, stationID,
		fuel.code, fuel.label, volume,
		fmt.Sprintf("E-SEAL-%05d", rand.IntN(99999)),
		total.String(), demoSource,
		// Unique per run, so the seeder's rows and these never collide on the
		// (tenant, source, source_ref) index.
		fmt.Sprintf("auto-%d-%d", time.Now().UnixNano(), rand.IntN(1000)),
		routeJSON, seconds, batchID,
	)
	return err
}

// octaneFor is the grade's nominal octane, for a batch nobody has tested.
//
// Diesel has none — it is measured by cetane — so it answers nil rather than a
// number that would read as a very poor petrol.
func octaneFor(code string) *float64 {
	nominal := map[string]float64{"ai80": 80, "ai92": 92, "ai95": 95, "ai98": 98, "euro92": 92}
	if value, ok := nominal[code]; ok {
		// A real assay lands near the grade, not on it.
		measured := value + float64(rand.IntN(9))/10
		return &measured
	}
	return nil
}

// roadRoute asks the router how a lorry gets from one point to another.
//
// OSRM_URL points at whichever instance a deployment runs; the public demo is
// the default and is for development only, by its operators' own rules.
func roadRoute(ctx context.Context, fromLat, fromLon, toLat, toLon float64) (
	points [][2]float64, metres float64, seconds int, err error) {

	base := os.Getenv("OSRM_URL")
	if base == "" {
		base = "https://router.project-osrm.org"
	}
	endpoint := fmt.Sprintf("%s/route/v1/driving/%s,%s;%s,%s?%s", base,
		strconv.FormatFloat(fromLon, 'f', 6, 64), strconv.FormatFloat(fromLat, 'f', 6, 64),
		strconv.FormatFloat(toLon, 'f', 6, 64), strconv.FormatFloat(toLat, 'f', 6, 64),
		url.Values{"overview": {"full"}, "geometries": {"geojson"}}.Encode())

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	response, err := (&http.Client{Timeout: 25 * time.Second}).Do(request)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() { _ = response.Body.Close() }()

	var answer struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
			Geometry struct {
				Coordinates [][2]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil {
		return nil, 0, 0, err
	}
	if answer.Code != "Ok" || len(answer.Routes) == 0 || len(answer.Routes[0].Geometry.Coordinates) < 2 {
		return nil, 0, 0, fmt.Errorf("router answered %q", answer.Code)
	}
	r := answer.Routes[0]
	return r.Geometry.Coordinates, r.Distance, int(r.Duration), nil
}
