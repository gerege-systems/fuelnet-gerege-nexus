/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Tankers on their way to a forecourt.
 *
 * # What a citizen is told, and what they are not
 *
 * A fuel tanker is hazardous cargo. Publishing the live position of every one
 * in the country, unauthenticated, is also publishing a target list — so what
 * leaves this file is the smallest set that answers "when does the fuel get
 * here": where the vehicle is, where it is going, what it is carrying and when
 * it should arrive.
 *
 * Withheld deliberately, and each for its own reason:
 *
 *	driver_name, driver_phone   a person's own details, and irrelevant to the
 *	                            question anybody is asking
 *	seal_no                     the seal is a control; knowing its number is
 *	                            the first step in defeating it
 *	volume_liters               the same competitive read the stock endpoint
 *	                            withholds, arriving by another route
 *
 * If a deployment decides the position itself should not be public either, the
 * change is to drop two fields from PublicTrip — the operator's view, which is
 * gated, already carries everything.
 */

package fuel

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

// PublicTrip is a tanker as somebody waiting for fuel sees it.
type PublicTrip struct {
	ID          string  `json:"id"`
	TripCode    string  `json:"trip_code"`
	Plate       string  `json:"tanker_plate"`
	Brand       string  `json:"brand"`
	FuelType    string  `json:"fuel_type"`
	FuelLabel   string  `json:"fuel_label"`
	FromDepot   string  `json:"from_depot"`
	ToStation   string  `json:"to_station"`
	ToStationID *string `json:"to_station_id"`
	Status      string  `json:"status"`

	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Heading float64 `json:"heading"`

	// PositionSource is "device" when a tracker last reported this, and
	// "schedule" when nobody has and the point is where the timetable says the
	// vehicle should be. Said out loud because the two are not the same claim,
	// and a map that showed them identically would be asserting a precision it
	// does not have.
	PositionSource string `json:"position_source"`

	// ProgressPercent is how far along the run it is, 0..100.
	ProgressPercent float64    `json:"progress_percent"`
	ETAMinutes      *int       `json:"eta_minutes"`
	ETAAt           *time.Time `json:"eta_at"`

	// DepartedAt is here so a client can work out where the vehicle is between
	// polls rather than waiting for the next one. With the route and these two
	// timestamps a browser has everything the server used, and can advance the
	// lorry every frame — which is the difference between a map that moves and
	// one that jumps every fifteen seconds.
	DepartedAt time.Time `json:"departed_at"`

	// Route is the road the load is taking, as [[lon,lat], …].
	//
	// Sent with every trip rather than fetched when somebody taps one: it is a
	// few dozen points, it is the same for everybody, and a second round trip
	// to draw a line somebody is already looking at is a second chance to fail.
	Route [][2]float64 `json:"route"`
}

// handlePublicTrips answers the tankers currently on the road.
//
// Not filtered by viewport, unlike the stations. A tanker's interest to
// somebody is where it is *going*, not where it is now — a driver in Bayanzürkh
// wants to know a delivery is coming to their forecourt whether the lorry is
// currently in Darkhan or on the ring road. There are tens of these in flight,
// not hundreds, so the whole set is the right answer and it is the same answer
// for everybody, which means it can be cached.
func (m *Module) handlePublicTrips(w http.ResponseWriter, r *http.Request) {
	rows, err := m.db.Query(r.Context(), `
		SELECT t.id::text, t.trip_code, t.tanker_plate,
		       COALESCE(s.brand, ''), t.fuel_type, t.fuel_label,
		       t.from_depot, COALESCE(s.name, ''), t.to_station_id::text,
		       t.status,
		       t.current_lat, t.current_lon, t.heading,
		       t.origin_lat, t.origin_lon, s.lat, s.lon,
		       t.departed_at, t.eta_at, t.route_geom
		  FROM fuel_dispatch_trips t
		  LEFT JOIN fuel_stations s ON s.id = t.to_station_id
		 WHERE t.completed_at IS NULL
		   AND t.status <> 'completed'
		 ORDER BY t.eta_at NULLS LAST
		 LIMIT 200`)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not load the deliveries")
		return
	}
	defer rows.Close()

	now := time.Now()
	trips := []PublicTrip{}
	for rows.Next() {
		var (
			t                      PublicTrip
			curLat, curLon, head   *float64
			oLat, oLon, dLat, dLon *float64
			departedAt             time.Time
			etaAt                  *time.Time
			routeJSON              []byte
		)
		if err := rows.Scan(&t.ID, &t.TripCode, &t.Plate,
			&t.Brand, &t.FuelType, &t.FuelLabel,
			&t.FromDepot, &t.ToStation, &t.ToStationID,
			&t.Status,
			&curLat, &curLon, &head,
			&oLat, &oLon, &dLat, &dLon,
			&departedAt, &etaAt, &routeJSON); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "could not read the deliveries")
			return
		}

		progress := runProgress(departedAt, etaAt, now)
		t.ProgressPercent = math.Round(progress*1000) / 10

		if len(routeJSON) > 0 {
			_ = json.Unmarshal(routeJSON, &t.Route)
		}

		switch {
		case curLat != nil && curLon != nil:
			t.Lat, t.Lon, t.PositionSource = *curLat, *curLon, "device"
			if head != nil {
				t.Heading = *head
			}
		case len(t.Route) >= 2:
			// Along the road, not across the map. A straight line between depot
			// and forecourt runs through ger districts and buildings, which is
			// not where a tanker is and reads as a bug to anybody who looks.
			t.Lat, t.Lon, t.Heading = alongRoute(t.Route, progress)
			t.PositionSource = "schedule"
		case oLat != nil && oLon != nil && dLat != nil && dLon != nil:
			// No route stored — the router was unreachable when this was
			// dispatched. A straight line is wrong, and saying so is better than
			// dropping the delivery off the map entirely.
			t.Lat = *oLat + (*dLat-*oLat)*progress
			t.Lon = *oLon + (*dLon-*oLon)*progress
			t.Heading = bearing(*oLat, *oLon, *dLat, *dLon)
			t.PositionSource = "estimate"
		default:
			// Neither a report nor a route. Nothing to draw, and a point at
			// 0,0 in the Gulf of Guinea is worse than no point at all.
			continue
		}

		t.DepartedAt = departedAt
		if etaAt != nil {
			t.ETAAt = etaAt
			minutes := int(math.Round(etaAt.Sub(now).Minutes()))
			if minutes < 0 {
				minutes = 0
			}
			t.ETAMinutes = &minutes
		}
		trips = append(trips, t)
	}
	if rows.Err() != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the deliveries")
		return
	}

	nexus.JSON(w, http.StatusOK, map[string]any{"trips": trips, "count": len(trips)})
}

// runProgress is how far through its run a trip is, 0..1.
//
// Clamped at both ends: a lorry that left early is not at a negative position,
// and one that is late sits at its destination rather than sailing past it.
// A trip with no ETA has no schedule to interpolate along and reads as not yet
// started.
func runProgress(departed time.Time, eta *time.Time, now time.Time) float64 {
	if eta == nil {
		return 0
	}
	total := eta.Sub(departed).Seconds()
	if total <= 0 {
		return 1
	}
	return math.Min(1, math.Max(0, now.Sub(departed).Seconds()/total))
}

// alongRoute is the point a given fraction of the way down a polyline, by
// distance, and the heading of the segment it lands on.
//
// By distance rather than by vertex: OSRM returns points where the road bends,
// so a junction-heavy stretch has many and a straight highway has two. Stepping
// vertex by vertex would crawl through town and then teleport across the steppe.
func alongRoute(route [][2]float64, progress float64) (lat, lon, head float64) {
	// Cumulative length. Degrees, scaled so a degree of longitude counts for
	// what it is worth at this latitude — the absolute unit does not matter
	// because only the ratio is used.
	lengths := make([]float64, len(route)-1)
	var total float64
	for i := 0; i < len(route)-1; i++ {
		dLon := (route[i+1][0] - route[i][0]) * math.Cos(route[i][1]*math.Pi/180)
		dLat := route[i+1][1] - route[i][1]
		lengths[i] = math.Hypot(dLon, dLat)
		total += lengths[i]
	}
	if total == 0 {
		return route[0][1], route[0][0], 0
	}

	target := progress * total
	var walked float64
	for i, segment := range lengths {
		if walked+segment >= target || i == len(lengths)-1 {
			within := 0.0
			if segment > 0 {
				within = math.Min(1, math.Max(0, (target-walked)/segment))
			}
			lon = route[i][0] + (route[i+1][0]-route[i][0])*within
			lat = route[i][1] + (route[i+1][1]-route[i][1])*within
			return lat, lon, bearing(route[i][1], route[i][0], route[i+1][1], route[i+1][0])
		}
		walked += segment
	}
	last := route[len(route)-1]
	return last[1], last[0], 0
}

// bearing is the initial compass heading from one point to another, in degrees.
func bearing(lat1, lon1, lat2, lon2 float64) float64 {
	φ1, φ2 := lat1*math.Pi/180, lat2*math.Pi/180
	Δλ := (lon2 - lon1) * math.Pi / 180
	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
}

// TelemetryReport is what a tracker on a tanker sends.
type TelemetryReport struct {
	Lat      float64  `json:"lat"`
	Lon      float64  `json:"lon"`
	SpeedKmh *float64 `json:"speed_kmh"`
	Heading  *float64 `json:"heading"`
}

// handleTripTelemetry records where a tanker says it is.
//
// Gated, and scoped by row-level policy to the organisation the caller is
// acting for: a position report is a write about somebody's vehicle, and an
// operator must not be able to move another operator's lorry. The policy is
// what enforces that — the UPDATE names no tenant and could not narrow itself
// if it wanted to.
func (m *Module) handleTripTelemetry(w http.ResponseWriter, r *http.Request) {
	if _, ok := nexus.RequireTenant(w, r); !ok {
		return
	}
	tripID := chi.URLParam(r, "id")

	var report TelemetryReport
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&report); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if report.Lat < -90 || report.Lat > 90 || report.Lon < -180 || report.Lon > 180 {
		nexus.Error(w, http.StatusBadRequest, "lat/lon out of range")
		return
	}

	tag, err := m.db.Exec(r.Context(), `
		UPDATE fuel_dispatch_trips
		   SET current_lat = $2, current_lon = $3,
		       speed_kmh = $4, heading = $5,
		       reported_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND completed_at IS NULL`,
		tripID, report.Lat, report.Lon, report.SpeedKmh, report.Heading)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not record the position")
		return
	}
	if tag.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "no such delivery in progress")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"recorded": true})
}
