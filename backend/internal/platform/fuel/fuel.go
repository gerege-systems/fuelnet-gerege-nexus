/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Package fuel is the console's view of every operator's fuel records.
 *
 * It holds no tables. The rows belong to the operators — internal/apps/fuel
 * writes them, one organisation at a time — and this reads them through
 * `gerege_nexus_operator`, which migration 00008 of that module grants SELECT
 * and a `USING (TRUE)` policy for. The national figure is therefore a query
 * over the operators' own records rather than a copy kept in step by a nightly
 * job: there is no version of it that can drift from what a company sees on its
 * own screen.
 *
 * # What this is not
 *
 * It is not the regulator's surface. FUELNET_OVERSIGHT_PLAN.md §3 rejects
 * putting the ministry's oversight on this console, and the reason holds: an
 * operator here can suspend an organisation, impersonate a user and trigger a
 * deploy, and a sector regulator must not acquire those by being handed a
 * login. The regulator's own view belongs in the app behind `fuel.oversight`,
 * as that document sets out.
 *
 * What this is: the screen for whoever runs the deployment. They hold every one
 * of those powers already, and until now they could see how many organisations
 * exist but nothing about whether the platform's subject matter — fuel — is
 * being reported at all. One read, no writes, nothing to audit beyond the
 * request itself.
 */

package fuel

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/operator"
)

type Deps struct {
	DB *pgxpool.Pool
}

type Service struct {
	op *operator.Console
	db *pgxpool.Pool
}

func New(op *operator.Console, deps Deps) *Service {
	return &Service{op: op, db: deps.DB}
}

// Routes: one read, behind the capability every other read-only screen asks for.
func (s *Service) Routes(r chi.Router) {
	r.With(s.op.RequireCapability(operator.CapTenantRead)).Get("/fuel/overview", s.handleOverview)
}

// The whole screen in one statement.
//
// Per-operator figures are subqueries rather than joins: a tenant joined to
// both its stations and its depot tanks multiplies one by the other, and the
// stock figure that comes out is wrong in a way that looks plausible.
//
// The chain is read at all four of its points — border, depot, road, forecourt
// — because a shortage looks different at each: fuel stuck at the border is a
// customs problem, fuel sitting in depot tanks with empty forecourts is a
// haulage problem, and neither is visible in a single national total.
//
// `stale` is carried beside the totals and never folded into them. A forecourt
// whose stock was last reported a day ago is not a forecourt holding that much
// fuel — it is one nobody has heard from.
const overviewQuery = `
SELECT json_build_object(
  'installed', TRUE,
  'operators', (
    SELECT COALESCE(json_agg(row_to_json(o)), '[]') FROM (
      SELECT t.id, t.name, t.slug,
             (SELECT COUNT(*)::int FROM fuel_stations s WHERE s.tenant_id = t.id) AS stations,
             (SELECT COUNT(*)::int FROM fuel_depots d WHERE d.tenant_id = t.id)   AS depots,
             (SELECT COALESCE(SUM(i.current_stock_liters), 0)::float8
                FROM fuel_station_inventory i WHERE i.tenant_id = t.id)           AS station_liters,
             (SELECT COALESCE(SUM(k.current_liters), 0)::float8
                FROM fuel_depot_tanks k WHERE k.tenant_id = t.id)                 AS depot_liters,
             (SELECT COUNT(*)::int FROM fuel_station_inventory i
               WHERE i.tenant_id = t.id
                 AND (i.last_reported_at IS NULL
                      OR i.last_reported_at < NOW() - INTERVAL '24 hours'))       AS stale_rows,
             (SELECT COUNT(*)::int FROM fuel_dispatch_trips d
               WHERE d.tenant_id = t.id AND d.status = 'in_transit')              AS in_transit,
             (SELECT COUNT(*)::int FROM fuel_customs_shipments c
               WHERE c.tenant_id = t.id AND c.status <> 'at_depot')               AS at_border,
             (SELECT MAX(i.last_reported_at) FROM fuel_station_inventory i
               WHERE i.tenant_id = t.id)                                          AS last_report_at
        FROM platform.tenants t
       WHERE EXISTS (SELECT 1 FROM fuel_stations s WHERE s.tenant_id = t.id)
          OR EXISTS (SELECT 1 FROM fuel_depots d WHERE d.tenant_id = t.id)
       ORDER BY stations DESC, t.name) o),
  'stock', (
    SELECT COALESCE(json_agg(row_to_json(p)), '[]') FROM (
      SELECT fuel_type,
             SUM(station_liters)::float8  AS station_liters,
             SUM(depot_liters)::float8    AS depot_liters,
             SUM(border_liters)::float8   AS border_liters,
             SUM(capacity_liters)::float8 AS capacity_liters,
             SUM(stale)::int              AS stale
        FROM (
          SELECT fuel_type, current_stock_liters AS station_liters, 0 AS depot_liters,
                 0 AS border_liters, tank_capacity_liters AS capacity_liters,
                 CASE WHEN last_reported_at IS NULL
                        OR last_reported_at < NOW() - INTERVAL '24 hours'
                      THEN 1 ELSE 0 END AS stale
            FROM fuel_station_inventory
           UNION ALL
          SELECT fuel_type, 0, current_liters, 0, capacity_liters,
                 CASE WHEN updated_at < NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END
            FROM fuel_depot_tanks
           UNION ALL
          -- Хилд байгаа нь нөөц биш, ирж яваа зүйл. Тусад нь баганаар.
          SELECT fuel_type, 0, 0, declared_liters, 0, 0
            FROM fuel_customs_shipments
           WHERE status <> 'at_depot') AS everything
       GROUP BY fuel_type ORDER BY fuel_type) p),
  'aimags', (
    SELECT COALESCE(json_agg(row_to_json(a)), '[]') FROM (
      SELECT COALESCE(NULLIF(s.aimag, ''), '—') AS aimag,
             COUNT(DISTINCT s.id)::int          AS stations,
             COALESCE(SUM(i.current_stock_liters), 0)::float8 AS liters
        FROM fuel_stations s
        LEFT JOIN fuel_station_inventory i ON i.station_id = s.id
       GROUP BY COALESCE(NULLIF(s.aimag, ''), '—')
       ORDER BY stations DESC
       LIMIT 30) a),
  'dry', (
    SELECT COALESCE(json_agg(row_to_json(d)), '[]') FROM (
      SELECT s.id, s.name, s.aimag, s.district, s.brand_label,
             i.fuel_type, i.current_stock_liters::float8 AS liters, i.last_reported_at
        FROM fuel_station_inventory i
        JOIN fuel_stations s ON s.id = i.station_id
       WHERE i.status <> 'available' OR i.current_stock_liters <= 0
       ORDER BY s.aimag, s.name
       LIMIT 40) d),
  'totals', json_build_object(
    'operators',  (SELECT COUNT(DISTINCT tenant_id)::int FROM fuel_stations),
    'stations',   (SELECT COUNT(*)::int FROM fuel_stations),
    'depots',     (SELECT COUNT(*)::int FROM fuel_depots),
    'in_transit', (SELECT COUNT(*)::int FROM fuel_dispatch_trips WHERE status = 'in_transit'),
    'in_transit_liters', (SELECT COALESCE(SUM(volume_liters), 0)::float8
                            FROM fuel_dispatch_trips WHERE status = 'in_transit'),
    'at_border',  (SELECT COUNT(*)::int FROM fuel_customs_shipments WHERE status <> 'at_depot'),
    'received_7d_liters', (SELECT COALESCE(SUM(liters), 0)::float8
                             FROM fuel_station_receipts
                            WHERE received_at > NOW() - INTERVAL '7 days'),
    'batches_open', (SELECT COUNT(*)::int FROM fuel_batches
                      WHERE received_liters < imported_liters))
)::text`

func (s *Service) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := operator.Scoped(r.Context())

	var body string
	if err := s.db.QueryRow(ctx, overviewQuery).Scan(&body); err != nil {
		// A deployment where no organisation has installed the app has no fuel
		// tables at all. That is an empty screen, not a broken one: the module's
		// migrations run when it is first installed.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			httpx.JSON(w, http.StatusOK, map[string]any{"installed": false})
			return
		}
		slog.Error("control plane: could not build the fuel overview", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not load the overview")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}
