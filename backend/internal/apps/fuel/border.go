/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The border and the depots — the top of the chain.
 *
 * A consignment is declared at a port, cleared, and unloaded into a tank at a
 * depot. Clearing it is what mints the batch that everything downstream carries:
 * from that moment the laboratory certificate has something to travel on, and
 * the question "where did this litre come from" has an answer that reaches the
 * frontier.
 *
 * # Clearing is the act that creates a batch
 *
 * Not the declaration. A shipment sitting at the border under inspection may
 * still be turned back, and a batch that exists before it is allowed in is a
 * batch that can be referenced by fuel nobody may sell. Clearing is the moment
 * the state says this fuel is in the country, so it is the moment the chain
 * starts.
 */

package fuel

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// Shipment is a consignment at, or past, the border.
type Shipment struct {
	ID             string     `json:"id"`
	DeclarationNo  string     `json:"declaration_no"`
	BorderPort     string     `json:"border_port"`
	OriginCountry  string     `json:"origin_country"`
	Exporter       string     `json:"exporter"`
	FuelType       string     `json:"fuel_type"`
	FuelLabel      string     `json:"fuel_label"`
	DeclaredLiters float64    `json:"declared_liters"`
	DeclaredTons   *float64   `json:"declared_tons"`
	Wagons         int        `json:"wagons"`
	ConvoyCode     string     `json:"convoy_code"`
	Status         string     `json:"status"`
	LabStatus      string     `json:"lab_status"`
	QualityCertNo  string     `json:"quality_cert_no"`
	OctaneTested   *float64   `json:"octane_tested"`
	SulfurPPM      *float64   `json:"sulfur_ppm"`
	EnteredAt      time.Time  `json:"entered_at"`
	ClearedAt      *time.Time `json:"cleared_at"`
	ExpectedAt     *time.Time `json:"expected_at"`
	BatchCode      string     `json:"batch_code,omitempty"`
	Note           string     `json:"note"`
}

// The states a consignment moves through, and what may follow what.
//
// Written down rather than left to whatever a caller sends, because the whole
// value of the record is that it cannot be back-dated: a shipment that reached
// a depot must not quietly become "still under inspection" because a screen
// posted a stale value.
var shipmentNext = map[string][]string{
	"border_arrived": {"inspecting", "cleared"},
	"inspecting":     {"cleared", "border_arrived"},
	"cleared":        {"in_transit"},
	"in_transit":     {"at_depot"},
	"at_depot":       {},
}

// ShipmentDraft is what an importer declares.
type ShipmentDraft struct {
	DeclarationNo  string   `json:"declaration_no"`
	BorderPort     string   `json:"border_port"`
	OriginCountry  string   `json:"origin_country"`
	Exporter       string   `json:"exporter"`
	FuelType       string   `json:"fuel_type"`
	DeclaredLiters float64  `json:"declared_liters"`
	DeclaredTons   *float64 `json:"declared_tons"`
	Wagons         int      `json:"wagons"`
	ConvoyCode     string   `json:"convoy_code"`
	ExpectedAt     *string  `json:"expected_at"`
	Note           string   `json:"note"`
}

// handleCreateShipment records a consignment arriving at a port.
func (m *Module) handleCreateShipment(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	var draft ShipmentDraft
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&draft); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if draft.DeclarationNo == "" {
		nexus.Error(w, http.StatusBadRequest, "гаалийн мэдүүлгийн дугаар заавал шаардлагатай")
		return
	}
	if _, known := fuelLabels[draft.FuelType]; !known {
		nexus.Error(w, http.StatusBadRequest, "энэ түлшний төрлийг мэдэхгүй байна")
		return
	}
	if draft.DeclaredLiters <= 0 {
		nexus.Error(w, http.StatusBadRequest, "мэдүүлсэн литр 0-ээс их байх ёстой")
		return
	}

	var expected *time.Time
	if draft.ExpectedAt != nil && *draft.ExpectedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *draft.ExpectedAt)
		if err != nil {
			nexus.Error(w, http.StatusBadRequest, "хүрэх огноо RFC3339 хэлбэртэй байх ёстой")
			return
		}
		expected = &parsed
	}

	var shipment Shipment
	err = m.db.QueryRow(r.Context(), `
		INSERT INTO fuel_customs_shipments
		       (tenant_id, declaration_no, border_port, origin_country, exporter,
		        fuel_type, fuel_label, declared_liters, declared_tons,
		        wagons, convoy_code, expected_at, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id::text, declaration_no, border_port, origin_country, exporter,
		          fuel_type, fuel_label, declared_liters::float8, declared_tons::float8,
		          wagons, convoy_code, status, lab_status, quality_cert_no,
		          octane_tested::float8, sulfur_ppm::float8,
		          entered_at, cleared_at, expected_at, note`,
		tenantID, draft.DeclarationNo, draft.BorderPort, draft.OriginCountry, draft.Exporter,
		draft.FuelType, fuelLabel(draft.FuelType), draft.DeclaredLiters, draft.DeclaredTons,
		draft.Wagons, draft.ConvoyCode, expected, draft.Note).
		Scan(&shipment.ID, &shipment.DeclarationNo, &shipment.BorderPort,
			&shipment.OriginCountry, &shipment.Exporter, &shipment.FuelType, &shipment.FuelLabel,
			&shipment.DeclaredLiters, &shipment.DeclaredTons, &shipment.Wagons,
			&shipment.ConvoyCode, &shipment.Status, &shipment.LabStatus, &shipment.QualityCertNo,
			&shipment.OctaneTested, &shipment.SulfurPPM,
			&shipment.EnteredAt, &shipment.ClearedAt, &shipment.ExpectedAt, &shipment.Note)
	if isUniqueViolation(err) {
		nexus.Error(w, http.StatusConflict, "энэ дугаартай мэдүүлэг аль хэдийн бүртгэгдсэн байна")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not record the shipment")
		return
	}

	nexus.Audit(r.Context(), tenantID, claims.UserID, "fuel.shipment.declared", shipment.ID,
		map[string]any{"declaration_no": shipment.DeclarationNo, "liters": shipment.DeclaredLiters})

	nexus.JSON(w, http.StatusCreated, shipment)
}

// ClearRequest is customs releasing a consignment, with the laboratory result.
type ClearRequest struct {
	Status        string   `json:"status"`
	LabStatus     string   `json:"lab_status"`
	QualityCertNo string   `json:"quality_cert_no"`
	OctaneTested  *float64 `json:"octane_tested"`
	SulfurPPM     *float64 `json:"sulfur_ppm"`
}

// handleAdvanceShipment moves a consignment along, and mints its batch when it
// clears.
//
// The batch is created here and only here. A consignment may be turned back
// while it is under inspection, and a batch that existed before the state
// allowed the fuel in would be a chain link attached to nothing.
func (m *Module) handleAdvanceShipment(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	shipmentID := chi.URLParam(r, "id")

	var request ClearRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	tx, err := m.db.Begin(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not start")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var (
		current, fuelType, fuelLabelText, declarationNo, origin, exporter string
		declaredLiters                                                    float64
		existingBatch                                                     *string
	)
	err = tx.QueryRow(r.Context(), `
		SELECT s.status, s.fuel_type, s.fuel_label, s.declaration_no,
		       s.origin_country, s.exporter, s.declared_liters::float8,
		       (SELECT b.id::text FROM fuel_batches b WHERE b.customs_shipment_id = s.id LIMIT 1)
		  FROM fuel_customs_shipments s
		 WHERE s.id = $1::uuid FOR UPDATE`, shipmentID).
		Scan(&current, &fuelType, &fuelLabelText, &declarationNo, &origin, &exporter,
			&declaredLiters, &existingBatch)
	if errors.Is(err, pgx.ErrNoRows) {
		nexus.Error(w, http.StatusNotFound, "ийм мэдүүлэг олдсонгүй")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the shipment")
		return
	}

	if !allows(current, request.Status) {
		nexus.Error(w, http.StatusConflict,
			fmt.Sprintf("«%s» төлвөөс «%s» рүү шилжих боломжгүй", current, request.Status))
		return
	}

	if _, err := tx.Exec(r.Context(), `
		UPDATE fuel_customs_shipments
		   SET status = $2,
		       lab_status = COALESCE(NULLIF($3, ''), lab_status),
		       quality_cert_no = COALESCE(NULLIF($4, ''), quality_cert_no),
		       octane_tested = COALESCE($5, octane_tested),
		       sulfur_ppm = COALESCE($6, sulfur_ppm),
		       cleared_at = CASE WHEN $2 = 'cleared' THEN NOW() ELSE cleared_at END,
		       updated_at = NOW()
		 WHERE id = $1::uuid`,
		shipmentID, request.Status, request.LabStatus, request.QualityCertNo,
		request.OctaneTested, request.SulfurPPM); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not update the shipment")
		return
	}

	batchCode := ""
	if request.Status == "cleared" && existingBatch == nil {
		err = tx.QueryRow(r.Context(), `
			INSERT INTO fuel_batches
			       (tenant_id, batch_code, fuel_type, fuel_label, origin_country,
			        refinery, customs_decl_no, customs_shipment_id, imported_liters,
			        quality_cert_no, octane_tested, sulfur_ppm, lab_status)
			SELECT $1, $2, s.fuel_type, s.fuel_label, s.origin_country,
			       s.exporter, s.declaration_no, s.id, s.declared_liters,
			       s.quality_cert_no, s.octane_tested, s.sulfur_ppm, s.lab_status
			  FROM fuel_customs_shipments s WHERE s.id = $3::uuid
			RETURNING batch_code`,
			tenantID, batchCodeFor(declarationNo), shipmentID).Scan(&batchCode)
		if err != nil {
			nexus.Error(w, http.StatusInternalServerError, "could not mint the batch")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not save")
		return
	}

	nexus.Audit(r.Context(), tenantID, claims.UserID, "fuel.shipment."+request.Status, shipmentID,
		map[string]any{"declaration_no": declarationNo, "batch_code": batchCode})

	nexus.JSON(w, http.StatusOK, map[string]any{
		"id": shipmentID, "status": request.Status, "batch_code": batchCode,
	})
}

// allows reports whether a shipment may move from one state to another.
func allows(from, to string) bool {
	for _, candidate := range shipmentNext[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

// batchCodeFor names a batch after the declaration it came from.
//
// Derived rather than random: somebody holding a customs document should be
// able to find the batch without a lookup, and the two are one consignment.
func batchCodeFor(declarationNo string) string {
	return "BATCH-" + declarationNo
}

// handleListShipments is the importer's own consignments.
func (m *Module) handleListShipments(w http.ResponseWriter, r *http.Request) {
	if _, ok := nexus.RequireTenant(w, r); !ok {
		return
	}

	rows, err := m.db.Query(r.Context(), `
		SELECT s.id::text, s.declaration_no, s.border_port, s.origin_country, s.exporter,
		       s.fuel_type, s.fuel_label, s.declared_liters::float8, s.declared_tons::float8,
		       s.wagons, s.convoy_code, s.status, s.lab_status, s.quality_cert_no,
		       s.octane_tested::float8, s.sulfur_ppm::float8,
		       s.entered_at, s.cleared_at, s.expected_at, s.note,
		       COALESCE((SELECT b.batch_code FROM fuel_batches b
		                  WHERE b.customs_shipment_id = s.id LIMIT 1), '')
		  FROM fuel_customs_shipments s
		 ORDER BY s.entered_at DESC
		 LIMIT 200`)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not load the shipments")
		return
	}
	defer rows.Close()

	shipments := []Shipment{}
	for rows.Next() {
		var s Shipment
		if err := rows.Scan(&s.ID, &s.DeclarationNo, &s.BorderPort, &s.OriginCountry,
			&s.Exporter, &s.FuelType, &s.FuelLabel, &s.DeclaredLiters, &s.DeclaredTons,
			&s.Wagons, &s.ConvoyCode, &s.Status, &s.LabStatus, &s.QualityCertNo,
			&s.OctaneTested, &s.SulfurPPM, &s.EnteredAt, &s.ClearedAt, &s.ExpectedAt,
			&s.Note, &s.BatchCode); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "could not read the shipments")
			return
		}
		shipments = append(shipments, s)
	}
	if rows.Err() != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the shipments")
		return
	}

	nexus.JSON(w, http.StatusOK, map[string]any{"shipments": shipments, "count": len(shipments)})
}
