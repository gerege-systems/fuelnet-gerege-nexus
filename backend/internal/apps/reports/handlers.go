/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The reporting API: list, describe, run, export.
 *
 * What is left in this file is the engine's half. Running a report and writing
 * a spreadsheet are things the platform does; the app's own decisions — which
 * reports this organisation may see at all — moved to backend/domain/reports
 * and arrive here as a value or a refusal.
 */

package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	domain "github.com/gerege-systems/open-gerege-nexus/backend/domain/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
	"github.com/go-chi/chi/v5"
)

// maxRunBody bounds a run request. The body is a parameter object of at most a
// dozen short values; anything larger is not a report request.
const maxRunBody = 16 << 10

// handleList returns the reports this tenant may run, grouped by app.
func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	groups, err := m.svc.Available(r.Context(), tenantID, localeOf(r))
	if err != nil {
		fail(w, r, err)
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"groups": groups})
}

// paramView is a parameter as the form needs it: the spec, plus the options for
// a UUID parameter resolved against this tenant's own rows.
type paramView struct {
	reporting.ParamSpec
	Options []reporting.ParamOption `json:"options,omitempty"`
}

// handleMetadata describes one report: its parameters and its columns.
func (m *Module) handleMetadata(w http.ResponseWriter, r *http.Request) {
	tenantID, report, ok := m.resolve(w, r)
	if !ok {
		return
	}

	params := make([]paramView, 0, len(report.Params()))
	for _, spec := range report.Params() {
		view := paramView{ParamSpec: spec, Options: spec.Options}
		if spec.Kind == reporting.ParamUUID && spec.OptionsQuery != "" {
			options, err := m.loadOptions(r, tenantID, spec)
			if err != nil {
				nexus.Error(w, http.StatusInternalServerError, "could not load the parameter options")
				return
			}
			view.Options = options
		}
		// The query is not sent to the browser. It is SQL this platform runs,
		// and a client has no use for it.
		view.OptionsQuery = ""
		params = append(params, view)
	}

	nexus.JSON(w, http.StatusOK, map[string]any{
		"key":     report.Key(),
		"app":     report.App(),
		"titles":  report.Titles(),
		"params":  params,
		"columns": report.Columns(),
	})
}

// handleRun executes a report and answers with the rows.
func (m *Module) handleRun(w http.ResponseWriter, r *http.Request) {
	tenantID, report, ok := m.resolve(w, r)
	if !ok {
		return
	}

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

	result, err := m.engine.Run(r.Context(), tenantID, report, params)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "the report could not be produced: "+err.Error())
		return
	}

	m.record(r, tenantID, "reports.run", report.Key(), map[string]any{
		"rows":   len(result.Rows),
		"params": raw,
	})

	nexus.JSON(w, http.StatusOK, map[string]any{
		"key":    report.Key(),
		"title":  reporting.LocalizedTitle(report.Titles(), locale, report.Key()),
		"result": result,
	})
}

// handleExport runs a report and answers with a file.
func (m *Module) handleExport(w http.ResponseWriter, r *http.Request) {
	tenantID, report, ok := m.resolve(w, r)
	if !ok {
		return
	}

	format, err := reporting.ParseFormat(r.URL.Query().Get("format"))
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, err.Error())
		return
	}

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

	result, err := m.engine.Run(r.Context(), tenantID, report, params)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "the report could not be produced: "+err.Error())
		return
	}

	title := reporting.LocalizedTitle(report.Titles(), locale, report.Key())
	payload, err := reporting.Export(format, title, result, locale)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "the export could not be written")
		return
	}

	// An export is a copy of the organisation's data leaving the platform, so
	// it is a separate audit entry from the run rather than the same one.
	m.record(r, tenantID, "reports.export", report.Key(), map[string]any{
		"rows":   len(result.Rows),
		"format": string(format),
		"params": raw,
	})

	filename := reporting.Filename(report.Key(), format)
	w.Header().Set("Content-Type", format.ContentType())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	// The bytes are a spreadsheet built from a tenant's data; a cache anywhere
	// between here and the browser is a copy of it nobody accounted for.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

// resolve is the guard every report endpoint shares, asked of the domain: a
// report that exists, and an app this organisation has installed.
//
// It answers with the engine's report rather than the domain's description,
// because what happens next is running it. The gate has already been decided by
// then, which is the half that was easy to leave out.
func (m *Module) resolve(w http.ResponseWriter, r *http.Request) (string, reporting.Report, bool) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return "", nil, false
	}
	described, err := m.svc.Resolve(r.Context(), tenantID, chi.URLParam(r, "key"))
	if err != nil {
		fail(w, r, err)
		return "", nil, false
	}
	report, found := reporting.Get(described.Key)
	if !found {
		// Unreachable: the domain answered from this same registry a moment
		// ago. Written out rather than asserted because a nil Report here would
		// be a panic in a handler.
		nexus.Error(w, http.StatusNotFound, "no such report")
		return "", nil, false
	}
	return tenantID, report, true
}

// loadOptions fills a UUID parameter's dropdown from the tenant's own rows.
func (m *Module) loadOptions(r *http.Request, tenantID string, spec reporting.ParamSpec) ([]reporting.ParamOption, error) {
	ctx := nexus.WithTenantID(r.Context(), tenantID)
	rows, err := m.db.Query(ctx, spec.OptionsQuery, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := make([]reporting.ParamOption, 0, 16)
	for rows.Next() {
		var value, label string
		if err := rows.Scan(&value, &label); err != nil {
			return nil, err
		}
		// One label for every locale: it is a row's own name, not a phrase.
		options = append(options, reporting.ParamOption{
			Value: value, Titles: map[string]string{"mn": label},
		})
	}
	return options, rows.Err()
}

// decodeParams reads the parameter object from the body.
//
// A JSON object of strings rather than typed values: every parameter goes
// through reporting.Bind, which is the one place that knows what a report
// accepts, and giving the client a typed channel would be a second path into
// the same validation.
func decodeParams(w http.ResponseWriter, r *http.Request) (map[string]string, bool) {
	raw := map[string]string{}
	if r.Body == nil {
		return raw, true
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRunBody)

	var body struct {
		Params map[string]string `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// An empty body is a run with the defaults, which is what opening a
		// report does before anybody touches the form.
		if err.Error() == "EOF" {
			return raw, true
		}
		nexus.Error(w, http.StatusBadRequest, "invalid report parameters")
		return nil, false
	}
	if body.Params != nil {
		raw = body.Params
	}
	return raw, true
}

// localeOf is the caller's language, as every screen-facing handler reads it.
func localeOf(r *http.Request) string { return nexus.LocaleFromRequest(r) }

// record writes the audit entry. Every run and every export, by name.
//
// §3.5 requires this before one tenant may see anything of another's, and it is
// worth having on ordinary runs too: "who read the payroll figures" is a
// question an organisation is entitled to an answer to.
func (m *Module) record(r *http.Request, tenantID, action, key string, details map[string]any) {
	nexus.Audit(r.Context(), tenantID, actorOf(r), action, key, details)
}

// acting names the person making the request, for the one row that records it.
func acting(r *http.Request) context.Context {
	return domain.WithActor(r.Context(), actorOf(r))
}

func actorOf(r *http.Request) string {
	if claims, err := nexus.UserFromContext(r.Context()); err == nil {
		return claims.UserID
	}
	return ""
}

// fail answers with the domain's own words and the status that refusal has
// always had. Anything that is not a refusal is a platform that could not
// answer, which is a 500 and a log line.
func fail(w http.ResponseWriter, r *http.Request, err error) {
	code := status(err)
	if code >= http.StatusInternalServerError {
		slog.Error("reports: "+err.Error(), "error", errors.Unwrap(err), "path", r.URL.Path)
	}
	nexus.Error(w, code, err.Error())
}

// status is the whole of what this app says in HTTP that it does not say in Go.
//
// Not found is 404 rather than 403 throughout: whether a report, a schedule, an
// agreement or an organisation exists on the other side of a boundary is not
// this caller's business, and the two answers together would enumerate the
// deployment.
func status(err error) int {
	switch {
	case errors.Is(err, domain.ErrReportUnavailable),
		errors.Is(err, domain.ErrNoSuchSchedule),
		errors.Is(err, domain.ErrNoSuchGrant),
		errors.Is(err, domain.ErrNoPendingRequest),
		errors.Is(err, domain.ErrNoSuchTenant):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrGrantExists):
		return http.StatusConflict
	case domain.IsRefusal(err):
		// Every other refusal is a field the caller got wrong: an unknown
		// report key in a body, a scope that is not one, an address list
		// nobody can be reached at.
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
