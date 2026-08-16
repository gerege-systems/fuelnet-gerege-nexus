/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Scheduled reports: the CRUD behind the screen.
 */

package reports

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// maxRecipients bounds an address list. Twenty is past every real distribution
// list and short of turning a schedule into a mailing service.
const maxRecipients = 20

const maxScheduleBody = 16 << 10

type scheduleRequest struct {
	ReportKey  string            `json:"report_key"`
	Name       string            `json:"name"`
	Params     map[string]string `json:"params"`
	Cron       string            `json:"cron"`
	Format     string            `json:"format"`
	Recipients []string          `json:"recipients"`
	Active     *bool             `json:"active"`
}

func (m *Module) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	schedules, err := reporting.ListSchedules(r.Context(), m.db, tenantID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not read the schedules")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{
		"schedules": schedules,
		// Whether anything can actually be sent. Without it the screen would
		// let somebody create a schedule, show it as active, and never say why
		// nothing arrives.
		"delivery_configured": reporting.NewSMTPDeliverer() != nil,
	})
}

func (m *Module) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	request, report, ok := m.decodeSchedule(w, r, tenantID)
	if !ok {
		return
	}

	userID := ""
	if claims, err := nexus.UserFromContext(r.Context()); err == nil {
		userID = claims.UserID
	}

	var id string
	err := m.db.QueryRow(nexus.WithTenantID(r.Context(), tenantID), `
		INSERT INTO report_schedules
		    (tenant_id, report_key, name, params, cron, format, recipients, active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')::uuid)
		RETURNING id`,
		tenantID, request.ReportKey, request.Name, request.Params, request.Cron,
		request.Format, request.Recipients, activeOrDefault(request.Active), userID).Scan(&id)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not save the schedule")
		return
	}

	m.record(r, tenantID, "reports.schedule_created", report.Key(), map[string]any{
		"schedule_id": id,
		"cron":        request.Cron,
		"recipients":  len(request.Recipients),
	})
	nexus.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (m *Module) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid schedule id")
		return
	}

	request, report, ok := m.decodeSchedule(w, r, tenantID)
	if !ok {
		return
	}

	// The tenant clause is here as well as in the policy. `WHERE id = $1` alone
	// would be a schedule id from one organisation editing another's row, and
	// the row-level policy is the layer that catches it — not the only one.
	tag, err := m.db.Exec(nexus.WithTenantID(r.Context(), tenantID), `
		UPDATE report_schedules
		   SET report_key = $3, name = $4, params = $5, cron = $6, format = $7,
		       recipients = $8, active = $9, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, request.ReportKey, request.Name, request.Params, request.Cron,
		request.Format, request.Recipients, activeOrDefault(request.Active))
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not update the schedule")
		return
	}
	if tag.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "no such schedule")
		return
	}

	m.record(r, tenantID, "reports.schedule_updated", report.Key(), map[string]any{
		"schedule_id": id, "cron": request.Cron, "active": activeOrDefault(request.Active),
	})
	nexus.JSON(w, http.StatusOK, map[string]any{"id": id})
}

func (m *Module) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid schedule id")
		return
	}

	var reportKey string
	err := m.db.QueryRow(nexus.WithTenantID(r.Context(), tenantID),
		`DELETE FROM report_schedules WHERE id = $1 AND tenant_id = $2 RETURNING report_key`,
		id, tenantID).Scan(&reportKey)
	if errors.Is(err, pgx.ErrNoRows) {
		nexus.Error(w, http.StatusNotFound, "no such schedule")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not remove the schedule")
		return
	}

	m.record(r, tenantID, "reports.schedule_removed", reportKey, map[string]any{"schedule_id": id})
	w.WriteHeader(http.StatusNoContent)
}

// decodeSchedule reads and validates the body.
//
// Everything is checked here rather than at the first run: a schedule with an
// unparseable expression or an address nobody can receive at is a schedule that
// fails silently at three in the morning, weeks after whoever created it has
// stopped watching.
func (m *Module) decodeSchedule(w http.ResponseWriter, r *http.Request, tenantID string) (scheduleRequest, reporting.Report, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxScheduleBody)

	var request scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid schedule")
		return request, nil, false
	}

	report, found := reporting.Get(strings.TrimSpace(request.ReportKey))
	if !found {
		nexus.Error(w, http.StatusBadRequest, "no such report")
		return request, nil, false
	}

	installed, err := m.installedApps(r, tenantID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not check the installed apps")
		return request, nil, false
	}
	if !installed[report.App()] {
		nexus.Error(w, http.StatusBadRequest, "no such report")
		return request, nil, false
	}

	if _, err := reporting.ParseCron(request.Cron); err != nil {
		nexus.Error(w, http.StatusBadRequest, "the schedule expression is not valid: "+err.Error())
		return request, nil, false
	}

	format, err := reporting.ParseFormat(request.Format)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, err.Error())
		return request, nil, false
	}
	request.Format = string(format)

	// The parameters have to be ones the report accepts, now. A schedule is
	// stored and run later, so this is the last moment anybody is present to
	// be told they are wrong.
	if request.Params == nil {
		request.Params = map[string]string{}
	}
	if _, err := reporting.Bind(report, request.Params, "mn"); err != nil {
		nexus.Error(w, http.StatusBadRequest, err.Error())
		return request, nil, false
	}

	recipients, err := normalizeRecipients(request.Recipients)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, err.Error())
		return request, nil, false
	}
	request.Recipients = recipients
	request.Name = strings.TrimSpace(request.Name)

	return request, report, true
}

func normalizeRecipients(given []string) ([]string, error) {
	if len(given) == 0 {
		return nil, errors.New("a schedule needs at least one recipient")
	}
	if len(given) > maxRecipients {
		return nil, errors.New("a schedule may not have more than twenty recipients")
	}

	seen := make(map[string]bool, len(given))
	cleaned := make([]string, 0, len(given))
	for _, raw := range given {
		address, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			return nil, errors.New(strings.TrimSpace(raw) + " is not an e-mail address")
		}
		lowered := strings.ToLower(address.Address)
		if seen[lowered] {
			continue
		}
		seen[lowered] = true
		cleaned = append(cleaned, lowered)
	}
	return cleaned, nil
}

func activeOrDefault(active *bool) bool {
	if active == nil {
		return true
	}
	return *active
}
