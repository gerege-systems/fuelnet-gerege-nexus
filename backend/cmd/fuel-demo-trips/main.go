/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Tankers on the road, invented.
 *
 * Scaffolding, like cmd/fuel-import's -demo-stock: real runs are dispatched by
 * an operator and tracked by a device in the cab. This exists so the map has
 * something moving on it before either of those is built, and so the shape of
 * the screen can be argued about against something rather than a mock.
 *
 * # Nothing here moves anything
 *
 * A trip is written once, with an origin, a destination and a window —
 * `departed_at` to `eta_at`. Where the lorry *is* comes out of that window when
 * somebody asks, in dispatch.go. So the vehicles slide along the map in real
 * time with no ticker running, no writes, and no background job to leak a
 * database connection; they simply arrive when their ETA passes and drop off
 * the map. Re-run this to put a fresh set on the road.
 *
 *	go run ./cmd/fuel-demo-trips -tenant demo -count 12
 */
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Where the road network is asked about.
//
// The public OSRM demo by default, which is fine for a seeder run a few times a
// day and explicitly not for production traffic — its operators say so. A
// deployment that dispatches for real points OSRM_URL at its own instance over
// a Mongolian extract; nothing else changes, because what is stored is the
// answer rather than the question.
func routerBase() string {
	if base := os.Getenv("OSRM_URL"); base != "" {
		return base
	}
	return "https://router.project-osrm.org"
}

// routeAlongRoads asks the router how a lorry actually gets from one point to
// another, and how long it takes.
//
// `overview=full` rather than simplified: simplified drops the points that make
// a city route follow streets, and the whole reason this call exists is that a
// straight line put tankers through people's yards.
func routeAlongRoads(ctx context.Context, fromLat, fromLon, toLat, toLon float64) (
	points [][2]float64, metres float64, seconds int, err error) {

	endpoint := fmt.Sprintf("%s/route/v1/driving/%f,%f;%f,%f?%s",
		routerBase(), fromLon, fromLat, toLon, toLat,
		url.Values{"overview": {"full"}, "geometries": {"geojson"}}.Encode())

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
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
	if answer.Code != "Ok" || len(answer.Routes) == 0 {
		return nil, 0, 0, fmt.Errorf("router answered %q", answer.Code)
	}
	r := answer.Routes[0]
	return r.Geometry.Coordinates, r.Distance, int(r.Duration), nil
}

// Where the fuel comes into the country and is stored. Real places: the rail
// terminals and depots the import actually moves through.
var depots = []struct {
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

var fuels = []struct{ code, label string }{
	{"ai92", "АИ-92"},
	{"ai95", "АИ-95"},
	{"diesel", "Дизель (ДТ)"},
}

// Plate letters used on Mongolian goods vehicles, enough for a plausible mix.
var plateSuffix = []string{"УБА", "УБЕ", "УНС", "УБН", "ХӨА", "СБА"}

func main() {
	tenantSlug := flag.String("tenant", "", "organisation the tankers belong to (required)")
	count := flag.Int("count", 12, "how many runs to put on the road")
	nearLat := flag.Float64("near-lat", 47.9185, "centre of the area to deliver into")
	nearLon := flag.Float64("near-lon", 106.9175, "centre of the area to deliver into")
	radiusKm := flag.Float64("radius-km", 45, "how far from that centre destinations may be; 0 for anywhere")
	clear := flag.Bool("clear", true, "remove previously invented runs first")
	flag.Parse()

	if *tenantSlug == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*tenantSlug, *count, *clear, *nearLat, *nearLon, *radiusKm); err != nil {
		fmt.Fprintln(os.Stderr, "fuel-demo-trips:", err)
		os.Exit(1)
	}
}

const sourceName = "demo"

// nearestDepot is the depot closest to a point, by straight-line distance.
//
// Equirectangular rather than haversine: over Mongolia the error is a fraction
// of a percent, and the answer only has to rank six candidates.
func nearestDepot(lat, lon float64) struct {
	name     string
	lat, lon float64
} {
	best, bestDistance := depots[0], math.Inf(1)
	for _, d := range depots {
		dLat := d.lat - lat
		dLon := (d.lon - lon) * math.Cos(lat*math.Pi/180)
		if distance := dLat*dLat + dLon*dLon; distance < bestDistance {
			best, bestDistance = d, distance
		}
	}
	return best
}

func run(tenantSlug string, count int, clear bool, nearLat, nearLon, radiusKm float64) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgrespassword@localhost:5432/platform_db?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	var tenantID string
	if err := pool.QueryRow(ctx,
		`SELECT id::text FROM platform.tenants WHERE slug = $1`, tenantSlug).Scan(&tenantID); err != nil {
		return fmt.Errorf("no organisation with slug %q: %w", tenantSlug, err)
	}

	// Destinations: real stations belonging to this organisation, drawn from
	// around a centre rather than from the whole register.
	//
	// Nationwide was the first attempt and it looked broken: the register spans
	// Mongolia, so fourteen runs scattered from Zamyn-Üüd to Khovd and a map
	// opened on Ulaanbaatar showed one lorry. Deliveries are worth watching
	// where somebody is watching, so the seeder defaults to the capital and
	// takes -radius-km 0 for the whole country.
	//
	// A degree of latitude is ~111 km and a degree of longitude ~75 km at this
	// latitude; the box is deliberately rough, because a seeder that needed
	// PostGIS to place a demo lorry would be the wrong trade.
	rows, err := pool.Query(ctx, `
		SELECT id::text, name, lat, lon FROM tenant.fuel_stations
		 WHERE tenant_id = $1
		   AND ($3 <= 0 OR (
		         lat BETWEEN $4 - $3 / 111.0 AND $4 + $3 / 111.0
		     AND lon BETWEEN $5 - $3 / 75.0  AND $5 + $3 / 75.0))
		 ORDER BY md5(id::text)
		 LIMIT $2`, tenantID, count, radiusKm, nearLat, nearLon)
	if err != nil {
		return fmt.Errorf("pick destinations: %w", err)
	}
	type dest struct {
		id, name string
		lat, lon float64
	}
	var dests []dest
	for rows.Next() {
		var d dest
		if err := rows.Scan(&d.id, &d.name, &d.lat, &d.lon); err != nil {
			rows.Close()
			return err
		}
		dests = append(dests, d)
	}
	rows.Close()
	if len(dests) == 0 {
		return fmt.Errorf("organisation %s has no stations within %.0f km of %.4f,%.4f; "+
			"widen -radius-km or run fuel-import first", tenantSlug, radiusKm, nearLat, nearLon)
	}

	if clear {
		tag, err := pool.Exec(ctx,
			`DELETE FROM tenant.fuel_dispatch_trips WHERE tenant_id = $1 AND source = $2`,
			tenantID, sourceName)
		if err != nil {
			return fmt.Errorf("clear previous runs: %w", err)
		}
		fmt.Printf("removed %d previously invented runs\n", tag.RowsAffected())
	}

	now := time.Now()
	written := 0
	for i, d := range dests {
		// Where the load comes from.
		//
		// Usually the nearest depot, which is what a dispatcher would choose and
		// also what keeps a delivery to Ulaanbaatar on Ulaanbaatar's roads. Round
		// robin was the first attempt: it sent every sixth load to the capital
		// from Zamyn-Üüd, 700 km away, so most of the fleet was somewhere in the
		// Gobi at any moment and the city map looked empty.
		//
		// One in four still comes from wherever, because some of them genuinely
		// do — a border run is the shape of the supply chain this platform exists
		// to show, and a demo where every lorry is twenty minutes out hides it.
		depot := nearestDepot(d.lat, d.lon)
		if i%4 == 3 {
			depot = depots[i%len(depots)]
		}
		fuel := fuels[i%len(fuels)]

		// A window that straddles now, so some lorries have just set off and
		// others are nearly there. Without the offset every vehicle would start
		// at its depot the moment this ran and the map would look staged.
		//
		// Between 5% and 80% of the way, never at either end. A run at 100% sits
		// exactly on its destination, underneath that station's own pin, and
		// reads as a lorry that is not there — which is what the first version
		// of this produced and what made the feature look broken.
		// The road, and how long the router thinks it takes. A journey time
		// invented here would put a lorry in the wrong place on a real road,
		// which is a more confusing kind of wrong than a straight line was.
		route, metres, seconds, routeErr := routeAlongRoads(ctx, depot.lat, depot.lon, d.lat, d.lon)
		if routeErr != nil {
			fmt.Fprintf(os.Stderr, "  route to %s: %v (falling back to a straight line)\n", d.name, routeErr)
		}

		total := time.Duration(60+rand.IntN(180)) * time.Minute
		if seconds > 0 {
			// Lorries are slower than the router's car profile and they stop.
			total = time.Duration(float64(seconds)*1.35) * time.Second
		}
		fraction := 0.05 + rand.Float64()*0.75
		departed := now.Add(-time.Duration(fraction * float64(total)))
		eta := departed.Add(total)

		var routeJSON []byte
		if len(route) >= 2 {
			routeJSON, _ = json.Marshal(route)
		}

		status := "in_transit"
		if eta.Sub(now) < 20*time.Minute {
			status = "arriving"
		}

		_, err := pool.Exec(ctx, `
			INSERT INTO tenant.fuel_dispatch_trips
			       (tenant_id, trip_code, tanker_plate, driver_name, driver_phone,
			        from_depot, origin_lat, origin_lon, to_station_id,
			        fuel_type, fuel_label, volume_liters,
			        seal_no, seal_status, status,
			        departed_at, eta_at, source, source_ref,
			        route_geom, route_distance_m, route_duration_s)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::uuid, $10, $11, $12,
			        $13, 'sealed_intact', $14, $15, $16, $17, $18,
			        $19::jsonb, NULLIF($20, 0), NULLIF($21, 0))`,
			tenantID,
			fmt.Sprintf("TRIP-%d-%04d", now.Year(), 1000+i),
			fmt.Sprintf("%04d%s", 1000+rand.IntN(8999), plateSuffix[i%len(plateSuffix)]),
			"—", "—", // a real dispatch carries a driver; an invented one must not pretend to
			depot.name, depot.lat, depot.lon, d.id,
			fuel.code, fuel.label, float64(12000+rand.IntN(4)*4000),
			fmt.Sprintf("E-SEAL-%05d", rand.IntN(99999)),
			status, departed, eta, sourceName, fmt.Sprintf("demo-%d", i),
			routeJSON, metres, seconds,
		)
		if err != nil {
			return fmt.Errorf("run to %s: %w", d.name, err)
		}
		written++
	}

	fmt.Printf("%d tankers on the road for %s\n", written, tenantSlug)
	return nil
}
