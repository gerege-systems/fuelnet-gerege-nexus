/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Taking a delivery, and the only place stock goes up.
 *
 * Until this existed a tanker arrived, its run was marked completed, and the
 * forecourt's tank held exactly what it held before. Twelve thousand litres
 * went nowhere anybody could point at. The chain the whole platform is for —
 * border, depot, road, pump — had no last link, and the stock figures were
 * whatever the importer seeded.
 *
 * # One receipt per run
 *
 * Enforced by a partial unique index rather than by looking first. Two taps on
 * a phone with a poor signal are two requests, and a check followed by an
 * insert lets both through — which would add the load twice and quietly break
 * the one number the regulator is watching.
 *
 * # What the difference means
 *
 * `manifest_liters` is what the paperwork says; `liters` is what the attendant
 * measured. This records both and reconciles neither. A gap is loss, theft, or
 * a badly calibrated gauge, and nothing here can tell which — but a system that
 * stored only one figure could not even say a gap existed.
 */

package fuel

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// ReceiveRequest is what a station says when a tanker is unloaded.
type ReceiveRequest struct {
	// Liters measured into the tank. Required: it is the whole point.
	Liters float64 `json:"liters"`
	// ManifestLiters is what the delivery note claims, when it differs.
	ManifestLiters *float64 `json:"manifest_liters"`
	// SealStatus at arrival — sealed_intact, opened_authorized, seal_tampered.
	SealStatus string `json:"seal_status"`
	Note       string `json:"note"`
}

// Receipt is one delivery, recorded.
type Receipt struct {
	ID          string    `json:"id"`
	StationID   string    `json:"station_id"`
	StationName string    `json:"station_name"`
	TripID      *string   `json:"trip_id"`
	BatchID     *string   `json:"batch_id"`
	BatchCode   string    `json:"batch_code,omitempty"`
	FuelType    string    `json:"fuel_type"`
	FuelLabel   string    `json:"fuel_label"`
	Liters      float64   `json:"liters"`
	SealStatus  string    `json:"seal_status"`
	ReceivedAt  time.Time `json:"received_at"`
	// StockAfter is the tank level once this load is in it, so the screen that
	// confirmed the delivery can show the result rather than asking again.
	StockAfterLiters float64 `json:"stock_after_liters"`
}

// The seal states a station may report.
//
// A closed list, because this is the field a fraud case turns on: an attendant
// under pressure types what they are told to type, and free text would make
// "tampered" and "tamperd" two different facts.
var sealStates = map[string]bool{
	"sealed_intact":     true,
	"opened_authorized": true,
	"seal_tampered":     true,
}

// handleReceiveDelivery unloads a tanker into a forecourt's tank.
//
// Everything in one transaction: the receipt, the stock, the run's completion
// and the batch's running total. A half-applied delivery is a forecourt that
// has the fuel and no record of where it came from — which is the exact state
// this feature exists to make impossible.
func (m *Module) handleReceiveDelivery(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	tripID := chi.URLParam(r, "id")

	var request ReceiveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if request.Liters <= 0 {
		nexus.Error(w, http.StatusBadRequest, "хүлээж авсан литр 0-ээс их байх ёстой")
		return
	}
	if request.SealStatus == "" {
		request.SealStatus = "sealed_intact"
	}
	if !sealStates[request.SealStatus] {
		nexus.Error(w, http.StatusBadRequest, "лацны төлөв танигдахгүй байна")
		return
	}

	tx, err := m.db.Begin(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not start")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// The run being unloaded. Locked, so two attendants confirming the same
	// arrival at once queue behind each other rather than both proceeding.
	var (
		receipt   Receipt
		stationID string
		batchID   *string
	)
	err = tx.QueryRow(r.Context(), `
		SELECT t.to_station_id::text, t.fuel_type, t.fuel_label, t.batch_id::text,
		       COALESCE(s.name, '')
		  FROM fuel_dispatch_trips t
		  LEFT JOIN fuel_stations s ON s.id = t.to_station_id
		 WHERE t.id = $1::uuid
		   FOR UPDATE OF t`, tripID).
		Scan(&stationID, &receipt.FuelType, &receipt.FuelLabel, &batchID, &receipt.StationName)
	if errors.Is(err, pgx.ErrNoRows) {
		nexus.Error(w, http.StatusNotFound, "ийм рейс олдсонгүй")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the delivery")
		return
	}
	if stationID == "" {
		nexus.Error(w, http.StatusConflict, "энэ рейс аль ШТС рүү явахыг заагаагүй байна")
		return
	}

	// The receipt. The unique index on trip_id is what refuses a second one —
	// a conflict here means somebody already confirmed this arrival.
	err = tx.QueryRow(r.Context(), `
		INSERT INTO fuel_station_receipts
		       (tenant_id, station_id, trip_id, batch_id, fuel_type, liters,
		        seal_status, manifest_liters, received_by, note)
		VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9::uuid, $10)
		RETURNING id::text, received_at`,
		tenantID, stationID, tripID, batchID, receipt.FuelType, request.Liters,
		request.SealStatus, request.ManifestLiters, claims.UserID, request.Note).
		Scan(&receipt.ID, &receipt.ReceivedAt)
	if isUniqueViolation(err) {
		nexus.Error(w, http.StatusConflict, "энэ рейсийг аль хэдийн хүлээж авсан байна")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not record the delivery")
		return
	}

	// Into the tank. The row is created if this forecourt has never carried
	// this grade: a delivery of a fuel the station did not sell yesterday is an
	// ordinary event, not an error.
	err = tx.QueryRow(r.Context(), `
		INSERT INTO fuel_station_inventory
		       (station_id, tenant_id, fuel_type, fuel_label, price_mnt,
		        current_stock_liters, last_reported_at)
		VALUES ($1::uuid, $2, $3, $4, 0, $5, NOW())
		ON CONFLICT (station_id, fuel_type) DO UPDATE
		   SET current_stock_liters = fuel_station_inventory.current_stock_liters + $5,
		       last_reported_at = NOW()
		RETURNING current_stock_liters::float8`,
		stationID, tenantID, receipt.FuelType, receipt.FuelLabel, request.Liters).
		Scan(&receipt.StockAfterLiters)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not add the fuel to the tank")
		return
	}

	// The run is over.
	if _, err := tx.Exec(r.Context(), `
		UPDATE fuel_dispatch_trips
		   SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		 WHERE id = $1::uuid AND completed_at IS NULL`, tripID); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not close the delivery")
		return
	}

	// And the batch knows how much of it has arrived somewhere. The gap between
	// imported and received is the figure a regulator reads.
	if batchID != nil {
		if _, err := tx.Exec(r.Context(), `
			UPDATE fuel_batches
			   SET received_liters = received_liters + $2, updated_at = NOW()
			 WHERE id = $1::uuid`, *batchID, request.Liters); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "could not update the batch")
			return
		}
		if err := tx.QueryRow(r.Context(),
			`SELECT batch_code FROM fuel_batches WHERE id = $1::uuid`, *batchID).
			Scan(&receipt.BatchCode); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			nexus.Error(w, http.StatusInternalServerError, "could not read the batch")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not save the delivery")
		return
	}

	receipt.StationID = stationID
	receipt.TripID = &tripID
	receipt.BatchID = batchID
	receipt.Liters = request.Liters
	receipt.SealStatus = request.SealStatus

	nexus.Audit(r.Context(), tenantID, claims.UserID, "fuel.delivery.received", receipt.ID,
		map[string]any{
			"station_id":  stationID,
			"liters":      request.Liters,
			"seal_status": request.SealStatus,
			"batch_code":  receipt.BatchCode,
		})

	nexus.JSON(w, http.StatusCreated, receipt)
}

// handleListReceipts is a forecourt's delivery history.
func (m *Module) handleListReceipts(w http.ResponseWriter, r *http.Request) {
	if _, ok := nexus.RequireTenant(w, r); !ok {
		return
	}
	stationID := chi.URLParam(r, "id")

	rows, err := m.db.Query(r.Context(), `
		SELECT rc.id::text, rc.station_id::text, COALESCE(s.name, ''),
		       rc.trip_id::text, rc.batch_id::text, COALESCE(b.batch_code, ''),
		       rc.fuel_type, COALESCE(b.fuel_label, rc.fuel_type),
		       rc.liters::float8, rc.seal_status, rc.received_at
		  FROM fuel_station_receipts rc
		  LEFT JOIN fuel_stations s ON s.id = rc.station_id
		  LEFT JOIN fuel_batches b ON b.id = rc.batch_id
		 WHERE rc.station_id = $1::uuid
		 ORDER BY rc.received_at DESC
		 LIMIT 100`, stationID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not load the deliveries")
		return
	}
	defer rows.Close()

	receipts := []Receipt{}
	for rows.Next() {
		var rc Receipt
		if err := rows.Scan(&rc.ID, &rc.StationID, &rc.StationName,
			&rc.TripID, &rc.BatchID, &rc.BatchCode,
			&rc.FuelType, &rc.FuelLabel, &rc.Liters, &rc.SealStatus, &rc.ReceivedAt); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "could not read the deliveries")
			return
		}
		receipts = append(receipts, rc)
	}
	if rows.Err() != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the deliveries")
		return
	}

	nexus.JSON(w, http.StatusOK, map[string]any{"receipts": receipts, "count": len(receipts)})
}

// isUniqueViolation reports whether an error is Postgres refusing a duplicate.
//
// By SQLSTATE rather than by message: the message is localised and the code is
// not, and this one distinguishes "somebody already did this" from "the
// database is broken" — two answers a caller must not confuse.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
