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
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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
//
// These are now rows rather than names in a string column: a run has to be
// drawn out of a tank at one of them, and a tank needs somewhere to belong.
// `port` marks the two that sit on the frontier — the only places a consignment
// can enter, which is why they are where the invented imports arrive.
var demoDepots = []struct {
	name        string
	lat, lon    float64
	aimag       string
	railStation string
	port        string
}{
	{"Сүхбаатар боомт", 50.2350, 106.2070, "Сэлэнгэ", "СХБТ", "Сүхбаатар"},
	{"Замын-Үүд боомт", 43.7200, 111.8980, "Дорноговь", "ЗМҮД", "Замын-Үүд"},
	{"Дархан нефть бааз", 49.4860, 105.9220, "Дархан-Уул", "ДРХН", ""},
	{"Улаанбаатар төв бааз", 47.9060, 106.8300, "Улаанбаатар", "УБТЗ", ""},
	{"Багануур бааз", 47.8280, 108.3450, "Улаанбаатар", "БГНР", ""},
	{"Эрдэнэт бааз", 49.0280, 104.0450, "Орхон", "ЭРДН", ""},
}

// How big an invented tank is, and how full a restock leaves it.
//
// Five hundred cubic metres is an ordinary vertical steel tank at a Mongolian
// base. It matters that the number is realistic rather than enormous: a tank
// that never runs low never exercises the restock path, and the restock path is
// the one that walks a consignment through customs.
// The level a tank is topped up from, so a base that nobody draws from still
// shows fuel rather than an empty vessel that reads as a broken screen.
const (
	demoTankCapacity = 500_000.0
	demoRestockTo    = 0.85
	demoLowWater     = 0.20
)

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
	// The bases first, and before anything is asked about the fleet.
	//
	// They used to be created inside dispatchOneDemoRun, which meant they only
	// appeared when a replacement lorry was needed. On the deployment that first
	// carried this code the fleet was already full of runs dispatched by the
	// previous build, so nothing was dispatched, so no base was ever created —
	// and the depot screen was empty for reasons that had nothing to do with
	// depots. A base is a property of the deployment, not a side effect of
	// sending a lorry somewhere.
	m.ensureDemoBases(ctx)

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

	fuel := demoFuels[rand.IntN(len(demoFuels))]
	volume := float64(12000 + rand.IntN(4)*4000)

	// Usually the nearest base holding this grade — what a dispatcher would
	// choose, and what keeps a city delivery on city roads. One in four comes
	// from further out, because border runs are the shape of the supply chain
	// this platform is for.
	//
	// The choice is made in SQL, against tanks that actually exist, rather than
	// against a list of names in this file. That is the whole change: a run now
	// has somewhere it was loaded, and the litres it carries left a tank.
	order := "distance"
	if rand.IntN(4) == 0 {
		order = "random"
	}
	var (
		depotID, depotName, tankID string
		dLat, dLon                 float64
		tankLiters                 float64
	)
	err = m.db.QueryRow(ctx, `
		SELECT d.id::text, d.name, t.id::text, d.lat, d.lon, t.current_liters::float8
		  FROM fuel_depots d
		  JOIN fuel_depot_tanks t ON t.depot_id = d.id AND t.fuel_type = $4
		 WHERE d.tenant_id = $1 AND d.source = $5 AND d.lat IS NOT NULL
		 ORDER BY CASE WHEN $6 = 'random' THEN random()
		               ELSE (d.lat - $2) ^ 2 + ((d.lon - $3) * 0.67) ^ 2 END
		 LIMIT 1`,
		tenantID, sLat, sLon, fuel.code, demoSource, order).
		Scan(&depotID, &depotName, &tankID, &dLat, &dLon, &tankLiters)
	if err != nil {
		return fmt.Errorf("pick a depot: %w", err)
	}

	// A tank that cannot fill this lorry gets a consignment first — declared at
	// a port, cleared by customs, unloaded into the tank. That is the real
	// chain, walked by the demonstration rather than described by it.
	if tankLiters < volume {
		if err := m.restockDemoTank(ctx, tenantID, depotID, tankID, depotName, fuel); err != nil {
			return err
		}
	}

	route, _, seconds, routeErr := roadRoute(ctx, dLat, dLon, sLat, sLon)
	if routeErr != nil {
		return routeErr
	}

	// Out of the tank, and the batch that comes with those litres.
	//
	// A tank is commingled — yesterday's consignment and today's are the same
	// fuel once they are in it — so a load carries the most recent batch to
	// enter. That is an approximation, and it is the one the industry makes:
	// the alternative is claiming a precision about which molecule came from
	// where that no depot in the country can support.
	var batchID *string
	err = m.db.QueryRow(ctx, `
		WITH drawn AS (
			UPDATE fuel_depot_tanks
			   SET current_liters = current_liters - $2, updated_at = NOW()
			 WHERE id = $1::uuid AND current_liters >= $2
			RETURNING id
		)
		SELECT (SELECT rc.batch_id::text
		          FROM fuel_depot_receipts rc
		         WHERE rc.tank_id = $1::uuid AND rc.batch_id IS NOT NULL
		         ORDER BY rc.received_at DESC LIMIT 1)
		  FROM drawn`, tankID, volume).Scan(&batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		// The tank emptied between the read and the write. The next sweep will
		// restock it; nothing is broken.
		return nil
	}
	if err != nil {
		return fmt.Errorf("draw from a tank: %w", err)
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
		        from_depot, from_depot_id, from_tank_id, origin_lat, origin_lon,
		        to_station_id, fuel_type, fuel_label, volume_liters,
		        seal_no, seal_status, status,
		        departed_at, eta_at, source, source_ref,
		        route_geom, route_distance_m, route_duration_s, batch_id)
		VALUES ($1, $2, $3, '—', '—', $4, $5::uuid, $6::uuid, $7, $8,
		        $9::uuid, $10, $11, $12,
		        $13, 'sealed_intact', 'in_transit',
		        NOW(), NOW() + $14::interval, $15, $16,
		        $17::jsonb, NULL, $18, $19::uuid)`,
		tenantID,
		fmt.Sprintf("TRIP-%d-%06d", time.Now().Year(), rand.IntN(999999)),
		fmt.Sprintf("%04d%s", 1000+rand.IntN(8999), demoPlateSuffix[rand.IntN(len(demoPlateSuffix))]),
		depotName, depotID, tankID, dLat, dLon, stationID,
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

// ensureDemoBases puts the invented bases in place for every organisation that
// has stations the demo dispatcher might deliver to.
//
// By organisation rather than for one, because which tenant a run belongs to is
// decided by the station it is going to — so the bases have to exist for all of
// them before any station is picked. Failures are logged and the sweep goes on:
// a base that could not be created is a base the next sweep will try again in
// forty-five seconds, and it must not stop lorries that are already running.
func (m *Module) ensureDemoBases(ctx context.Context) {
	rows, err := m.db.Query(ctx, `
		SELECT DISTINCT tenant_id::text FROM fuel_stations
		 WHERE lat BETWEEN 47.65 AND 48.15 AND lon BETWEEN 106.5 AND 107.4`)
	if err != nil {
		slog.Warn("fuel: could not find who to build demo bases for", "error", err)
		return
	}
	defer rows.Close()

	var tenants []string
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			slog.Warn("fuel: could not read a tenant for demo bases", "error", err)
			return
		}
		tenants = append(tenants, tenantID)
	}
	if rows.Err() != nil {
		slog.Warn("fuel: could not read the tenants for demo bases", "error", rows.Err())
		return
	}

	// Read to the end before writing: ensureDemoDepots runs its own statements
	// on the same pool, and holding this cursor open across them would keep a
	// connection busy for the whole loop.
	for _, tenantID := range tenants {
		if err := m.ensureDemoDepots(ctx, tenantID); err != nil {
			slog.Warn("fuel: could not build the demo bases", "tenant", tenantID, "error", err)
		}
	}
}

// ensureDemoDepots puts one organisation's invented bases and tanks in place,
// and fills any tank that is running dry.
//
// Idempotent by the (tenant, source, source_ref) index the migration installs,
// so the sweep can call it every time without checking first — which is what
// makes it correct after a database is restored, a tenant is added, or somebody
// deletes a row by hand.
func (m *Module) ensureDemoDepots(ctx context.Context, tenantID string) error {
	for _, depot := range demoDepots {
		var depotID string
		err := m.db.QueryRow(ctx, `
			INSERT INTO fuel_depots
			       (tenant_id, name, brand, aimag, lat, lon,
			        has_rail_siding, rail_station_code, source, source_ref)
			VALUES ($1, $2, 'Демо', $3, $4, $5, TRUE, $6, $7, $2)
			ON CONFLICT (tenant_id, source, source_ref)
			    WHERE source IS NOT NULL AND source_ref IS NOT NULL
			DO UPDATE SET updated_at = NOW()
			RETURNING id::text`,
			tenantID, depot.name, depot.aimag, depot.lat, depot.lon,
			depot.railStation, demoSource).Scan(&depotID)
		if err != nil {
			return fmt.Errorf("ensure a depot: %w", err)
		}

		// One tank per grade. A real base has several of each and they are not
		// interchangeable; one apiece is enough to show a level moving, and
		// every extra invented vessel is a number somebody has to read.
		for _, fuel := range demoFuels {
			var tankID string
			var current, capacity float64
			err := m.db.QueryRow(ctx, `
				INSERT INTO fuel_depot_tanks
				       (depot_id, tenant_id, tank_no, fuel_type, fuel_label, capacity_liters)
				VALUES ($1::uuid, $2, $3, $4, $5, $6)
				ON CONFLICT (depot_id, tank_no) DO UPDATE SET updated_at = NOW()
				RETURNING id::text, current_liters::float8, capacity_liters::float8`,
				depotID, tenantID, "Т-"+fuel.code, fuel.code, fuel.label,
				demoTankCapacity).Scan(&tankID, &current, &capacity)
			if err != nil {
				return fmt.Errorf("ensure a tank: %w", err)
			}

			// A tank running dry gets a consignment — declared at a port,
			// cleared by customs, unloaded. Here rather than only at dispatch
			// so a base that nobody has drawn from still shows a level, and so
			// the customs register has something in it on a deployment where no
			// lorry has needed replacing yet.
			if current < capacity*demoLowWater {
				if err := m.restockDemoTank(ctx, tenantID, depotID, tankID, depot.name, fuel); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// restockDemoTank walks one consignment from the frontier into a tank.
//
// Declared, cleared, unloaded — the three states a real import passes through,
// written the same way the handlers write them, in one transaction. The point
// is not the fuel: it is that every litre a demo lorry carries can be traced to
// a customs declaration, so the chain the regulator's screens will read is a
// real chain rather than a picture of one.
func (m *Module) restockDemoTank(ctx context.Context, tenantID, depotID, tankID,
	depotName string, fuel struct{ code, label string }) error {

	var capacity, current float64
	if err := m.db.QueryRow(ctx,
		`SELECT capacity_liters::float8, current_liters::float8
		   FROM fuel_depot_tanks WHERE id = $1::uuid`, tankID).
		Scan(&capacity, &current); err != nil {
		return fmt.Errorf("read a tank: %w", err)
	}
	liters := math.Round(capacity*demoRestockTo - current)
	if liters <= 0 {
		return nil
	}

	// Which frontier it came through. Fuel reaching a base in the north came up
	// from Russia through Сүхбаатар; the Gobi bases are supplied from China.
	port, origin, exporter := "Сүхбаатар", "ОХУ", "Роснефть"
	if depotName == "Замын-Үүд боомт" {
		port, origin, exporter = "Замын-Үүд", "БНХАУ", "Sinopec"
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	declaration := fmt.Sprintf("ГМ-%d-%06d", time.Now().Year(), rand.IntN(999999))

	// Density is what turns a declaration written in tonnes into litres. The
	// paperwork figure is derived from the litres here, which is backwards from
	// reality and honest about it: what this stores is a plausible pair, not a
	// measurement.
	density := 725.0 + float64(rand.IntN(120))
	if fuel.code == "diesel" {
		density = 830.0 + float64(rand.IntN(20))
	}
	// litres → m³ → kg → tonnes. Both divisions are needed and the first draft
	// had only one, which declared a rail consignment at 359,550 tonnes: about
	// six thousand wagons, on a train reaching from the border to the capital.
	// A figure that wrong is easy to catch here and impossible to catch on a
	// screen that only ever shows one shipment at a time.
	tons := math.Round(liters/1000*density/1000*1000) / 1000

	var shipmentID string
	err = tx.QueryRow(ctx, `
		INSERT INTO fuel_customs_shipments
		       (tenant_id, declaration_no, border_port, origin_country, exporter,
		        fuel_type, fuel_label, declared_liters, declared_tons,
		        wagons, convoy_code, status, lab_status, quality_cert_no,
		        octane_tested, sulfur_ppm, cleared_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        'cleared', 'passed', $12, $13, $14, NOW())
		RETURNING id::text`,
		tenantID, declaration, port, origin, exporter,
		fuel.code, fuel.label, liters, tons,
		// A rail tanker wagon holds about 60 m³.
		int(math.Ceil(liters/60000)),
		fmt.Sprintf("ЦВ-%05d", rand.IntN(99999)),
		fmt.Sprintf("LAB-%05d", rand.IntN(99999)),
		octaneFor(fuel.code),
		// Euro-5 дизель 10 ppm хүртэл; бусад нь өндөр. Тоо нь дүр эсгэсэн ч
		// хэмжээсийн зэрэг нь бодитой — К4 энэ багана дээр ажиллана.
		float64(8+rand.IntN(42)),
	).Scan(&shipmentID)
	if err != nil {
		return fmt.Errorf("declare a shipment: %w", err)
	}

	// Clearing mints the batch, exactly as handleAdvanceShipment does.
	var batchID string
	err = tx.QueryRow(ctx, `
		INSERT INTO fuel_batches
		       (tenant_id, batch_code, fuel_type, fuel_label, origin_country,
		        refinery, customs_decl_no, customs_shipment_id, imported_liters,
		        quality_cert_no, octane_tested, sulfur_ppm, lab_status)
		SELECT $1, $2, s.fuel_type, s.fuel_label, s.origin_country,
		       s.exporter, s.declaration_no, s.id, s.declared_liters,
		       s.quality_cert_no, s.octane_tested, s.sulfur_ppm, s.lab_status
		  FROM fuel_customs_shipments s WHERE s.id = $3::uuid
		RETURNING id::text`,
		tenantID, batchCodeFor(declaration), shipmentID).Scan(&batchID)
	if err != nil {
		return fmt.Errorf("mint a batch: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO fuel_depot_receipts
		       (tenant_id, depot_id, tank_id, shipment_id, batch_id, liters, note)
		VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5::uuid, $6, 'демо нийлүүлэлт')`,
		tenantID, depotID, tankID, shipmentID, batchID, liters); err != nil {
		return fmt.Errorf("record a depot receipt: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE fuel_depot_tanks
		   SET current_liters = current_liters + $2, updated_at = NOW()
		 WHERE id = $1::uuid`, tankID, liters); err != nil {
		return fmt.Errorf("fill a tank: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE fuel_customs_shipments
		   SET status = 'at_depot', updated_at = NOW()
		 WHERE id = $1::uuid`, shipmentID); err != nil {
		return fmt.Errorf("close a shipment: %w", err)
	}

	return tx.Commit(ctx)
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
