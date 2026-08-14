/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Sharing a report with another organisation: request, accept, revoke, read.
 */

package reports

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxGrantBody = 8 << 10

// handleListGrants returns every grant this organisation is a party to.
func (m *Module) handleListGrants(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	grants, err := reporting.ListGrants(r.Context(), m.db, tenantID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the sharing agreements")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"grants": grants})
}

type grantRequest struct {
	// Whose data is being asked for, by registration number rather than by
	// tenant id. A tenant id is an internal identifier a requester has no
	// legitimate way to know, and letting one be typed in would turn this form
	// into a way to enumerate the organisations on the deployment.
	GrantorRegistrationNumber string `json:"grantor_registration_number"`
	ReportKey                 string `json:"report_key"`
	Scope                     string `json:"scope"`
	ValidUntil                string `json:"valid_until"`
	Note                      string `json:"note"`
}

// handleRequestGrant is the grantee asking to be shown a report.
//
// It creates a request, not a permission: accepted_at is null until the owning
// organisation's administrator answers, and ActiveGrants ignores anything
// unaccepted. §3.5's second principle — the data owner decides — is enforced
// here and again in the query that reads grants.
func (m *Module) handleRequestGrant(w http.ResponseWriter, r *http.Request) {
	granteeTenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxGrantBody)
	var request grantRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid request")
		return
	}

	report, found := reporting.Get(strings.TrimSpace(request.ReportKey))
	if !found {
		nexus.Error(w, http.StatusBadRequest, "no such report")
		return
	}

	scope := strings.TrimSpace(request.Scope)
	if scope == "" {
		scope = reporting.ScopeCounterparty
	}
	if scope != reporting.ScopeCounterparty && scope != reporting.ScopeFull {
		nexus.Error(w, http.StatusBadRequest, "scope must be counterparty or full")
		return
	}
	// Default deny in the type system: a report that has not been written to be
	// shared cannot be shared, and one that cannot filter by counterparty
	// cannot be granted that scope — rather than the scope quietly widening to
	// everything.
	if !reporting.SupportsScope(report, scope) {
		nexus.Error(w, http.StatusBadRequest,
			"this report cannot be shared with that scope")
		return
	}

	grantorTenantID, err := m.tenantByRegistration(r, request.GrantorRegistrationNumber)
	if err != nil {
		nexus.Error(w, http.StatusNotFound, "no organisation with that registration number")
		return
	}
	if grantorTenantID == granteeTenantID {
		nexus.Error(w, http.StatusBadRequest, "an organisation cannot ask itself for a report")
		return
	}

	// The counterparty reference is decided once, here, and stored. Matching it
	// again on every run would mean a grant silently pointing at different data
	// after the requesting organisation edited its own profile.
	counterpartyRef := ""
	if scope == reporting.ScopeCounterparty {
		counterpartyRef, err = m.registrationOf(r, granteeTenantID)
		if err != nil || counterpartyRef == "" {
			nexus.Error(w, http.StatusBadRequest,
				"your organisation has no registration number; set it in Organisation before asking for a counterparty report")
			return
		}
	}

	var validUntil *time.Time
	if raw := strings.TrimSpace(request.ValidUntil); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			nexus.Error(w, http.StatusBadRequest, "valid_until must be YYYY-MM-DD")
			return
		}
		validUntil = &parsed
	}

	userID := actorOf(r)
	var id string
	err = m.db.QueryRow(nexus.WithTenantID(r.Context(), granteeTenantID), `
		INSERT INTO report_grants
		    (grantor_tenant_id, grantee_tenant_id, report_key, scope,
		     counterparty_ref, valid_until, created_by, note)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid, $8)
		RETURNING id`,
		grantorTenantID, granteeTenantID, report.Key(), scope,
		counterpartyRef, validUntil, userID, strings.TrimSpace(request.Note)).Scan(&id)
	if err != nil {
		// The partial unique index: one live agreement per pair per report.
		nexus.Error(w, http.StatusConflict,
			"a request or agreement for this report already exists between these organisations")
		return
	}

	m.record(r, granteeTenantID, "reports.grant_requested", report.Key(), map[string]any{
		"grant_id": id, "grantor_tenant_id": grantorTenantID, "scope": scope,
	})
	nexus.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleAcceptGrant is the grantor agreeing. Only they can.
func (m *Module) handleAcceptGrant(w http.ResponseWriter, r *http.Request) {
	grantorTenantID, id, ok := m.grantParty(w, r)
	if !ok {
		return
	}

	// `grantor_tenant_id = $2` is the whole authorization for this endpoint:
	// the row-level policy lets both parties see the row, so without this
	// clause a grantee could accept their own request.
	var reportKey string
	err := m.db.QueryRow(nexus.WithTenantID(r.Context(), grantorTenantID), `
		UPDATE report_grants
		   SET accepted_by = NULLIF($3, '')::uuid, accepted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND grantor_tenant_id = $2 AND revoked_at IS NULL AND accepted_at IS NULL
		 RETURNING report_key`, id, grantorTenantID, actorOf(r)).Scan(&reportKey)
	if errors.Is(err, pgx.ErrNoRows) {
		nexus.Error(w, http.StatusNotFound, "no such pending request")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not accept the request")
		return
	}

	m.record(r, grantorTenantID, "reports.grant_accepted", reportKey, map[string]any{"grant_id": id})
	nexus.JSON(w, http.StatusOK, map[string]any{"id": id})
}

// handleRevokeGrant ends an agreement.
//
// Either side may. The owner revoking is the point — §3.5's second principle
// is that the permission is revocable and takes effect immediately — and the
// reader withdrawing is a request they no longer want, which nobody benefits
// from leaving open.
//
// The row is not deleted. "Who could see our data, and when" is a question the
// owner is entitled to an answer to after the fact.
func (m *Module) handleRevokeGrant(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid grant id")
		return
	}

	var reportKey, direction string
	err := m.db.QueryRow(nexus.WithTenantID(r.Context(), tenantID), `
		UPDATE report_grants
		   SET revoked_at = NOW(), updated_at = NOW()
		 WHERE id = $1
		   AND (grantor_tenant_id = $2 OR grantee_tenant_id = $2)
		   AND revoked_at IS NULL
		 RETURNING report_key,
		           CASE WHEN grantor_tenant_id = $2 THEN 'given' ELSE 'received' END`,
		id, tenantID).Scan(&reportKey, &direction)
	if errors.Is(err, pgx.ErrNoRows) {
		nexus.Error(w, http.StatusNotFound, "no such agreement")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not revoke the agreement")
		return
	}

	m.record(r, tenantID, "reports.grant_revoked", reportKey, map[string]any{
		"grant_id": id, "side": direction,
	})
	nexus.JSON(w, http.StatusOK, map[string]any{"id": id})
}

// handleRunConsolidated runs a report across every organisation that shares it.
func (m *Module) handleRunConsolidated(w http.ResponseWriter, r *http.Request) {
	tenantID, report, ok := m.resolve(w, r)
	if !ok {
		return
	}
	// The consolidated view is not gated on the grantee having the *grantor's*
	// app: it is gated on the grant, and on this organisation having the
	// reports app and the permission. resolve has already checked that the
	// report's own app is installed here, which is what makes the report
	// meaningful to them at all.

	raw, ok := decodeParams(w, r)
	if !ok {
		return
	}
	locale := localeOf(r)

	params, err := reporting.Bind(report, raw, locale)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := m.engine.RunConsolidated(r.Context(), tenantID, report, params, actorOf(r))
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	nexus.JSON(w, http.StatusOK, map[string]any{
		"key":    report.Key(),
		"title":  reporting.LocalizedTitle(report.Titles(), locale, report.Key()),
		"result": result,
	})
}

// handleAccessHistory is the owner's answer to "who has read our data".
//
// It reads this organisation's own audit rows for the one action the
// consolidated engine writes on the owner's side. Ordinary audit is not exposed
// through any API; this slice of it is, because §3.5 requires the data owner to
// be able to see it and a trail they cannot read is not a control.
func (m *Module) handleAccessHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	rows, err := m.db.Query(nexus.WithTenantID(r.Context(), tenantID), `
		SELECT a.created_at, a.resource, a.details, coalesce(t.name, '—')
		  FROM audit_events a
		  LEFT JOIN tenants t
		    ON t.id = (a.details->>'grantee_tenant_id')::uuid
		 WHERE a.tenant_id = $1 AND a.action = 'reports.data_shared'
		 ORDER BY a.created_at DESC
		 LIMIT 200`, tenantID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the access history")
		return
	}
	defer rows.Close()

	type entry struct {
		At        time.Time      `json:"at"`
		ReportKey string         `json:"report_key"`
		By        string         `json:"by"`
		Details   map[string]any `json:"details"`
	}
	history := make([]entry, 0, 32)
	for rows.Next() {
		var item entry
		if err := rows.Scan(&item.At, &item.ReportKey, &item.Details, &item.By); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "could not read the access history")
			return
		}
		history = append(history, item)
	}
	if err := rows.Err(); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the access history")
		return
	}

	nexus.JSON(w, http.StatusOK, map[string]any{"history": history})
}

// grantParty resolves the caller's tenant and the grant id together.
func (m *Module) grantParty(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return "", "", false
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid grant id")
		return "", "", false
	}
	return tenantID, id, true
}

// tenantByRegistration resolves an organisation from its registration number.
//
// On the platform path, because it deliberately looks outside the caller's own
// organisation — and narrowly: it answers with an id or a not-found, takes an
// exact registration number rather than a name, and is behind the same
// authenticated, app-gated, permission-checked route as everything else here.
func (m *Module) tenantByRegistration(r *http.Request, registration string) (string, error) {
	registration = strings.TrimSpace(registration)
	if registration == "" {
		return "", errors.New("a registration number is required")
	}

	var tenantID string
	err := m.db.QueryRow(nexus.WithoutTenant(r.Context()),
		`SELECT tenant_id FROM tenant_profiles WHERE registration_number = $1`,
		registration).Scan(&tenantID)
	return tenantID, err
}

// registrationOf reads one organisation's own registration number.
func (m *Module) registrationOf(r *http.Request, tenantID string) (string, error) {
	var registration string
	err := m.db.QueryRow(nexus.WithTenantID(r.Context(), tenantID),
		`SELECT registration_number FROM tenant_profiles WHERE tenant_id = $1`,
		tenantID).Scan(&registration)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return strings.TrimSpace(registration), err
}

func actorOf(r *http.Request) string {
	if claims, err := nexus.UserFromContext(r.Context()); err == nil {
		return claims.UserID
	}
	return ""
}
