/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The reporting API: list, describe, run, export.
 */

package reports

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
)

// maxRunBody bounds a run request. The body is a parameter object of at most a
// dozen short values; anything larger is not a report request.
const maxRunBody = 16 << 10

// reportSummary is one entry of the list.
type reportSummary struct {
	Key    string            `json:"key"`
	App    string            `json:"app"`
	Title  string            `json:"title"`
	Titles map[string]string `json:"titles"`
}

// handleList returns the reports this tenant may run, grouped by app.
//
// Grouped rather than flat because that is how the screen reads and because the
// grouping is the app gate made visible: a tenant sees the sections for the
// apps it has, and a section it does not have simply is not there — rather than
// being there and refusing when clicked.
func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	installed, err := m.installedApps(r, tenantID)
	if err != nil {
		return
	}

	locale := config.LocaleFromRequest(r)
	groups := map[string][]reportSummary{}
	for _, report := range reporting.ForApps(installed) {
		groups[report.App()] = append(groups[report.App()], reportSummary{
			Key:    report.Key(),
			App:    report.App(),
			Title:  reporting.LocalizedTitle(report.Titles(), locale, report.Key()),
			Titles: report.Titles(),
		})
	}

	apps := make([]string, 0, len(groups))
	for app := range groups {
		apps = append(apps, app)
	}
	sort.Strings(apps)

	type group struct {
		App     string          `json:"app"`
		Reports []reportSummary `json:"reports"`
	}
	response := make([]group, 0, len(apps))
	for _, app := range apps {
		response = append(response, group{App: app, Reports: groups[app]})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"groups": response})
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
				httpx.Error(w, http.StatusInternalServerError, "could not load the parameter options")
				return
			}
			view.Options = options
		}
		// The query is not sent to the browser. It is SQL this platform runs,
		// and a client has no use for it.
		view.OptionsQuery = ""
		params = append(params, view)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
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
	locale := config.LocaleFromRequest(r)

	params, err := reporting.Bind(report, raw, locale)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := m.engine.Run(r.Context(), tenantID, report, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "the report could not be produced: "+err.Error())
		return
	}

	m.record(r, tenantID, "reports.run", report.Key(), map[string]any{
		"rows":   len(result.Rows),
		"params": raw,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
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
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	raw, ok := decodeParams(w, r)
	if !ok {
		return
	}
	locale := config.LocaleFromRequest(r)

	params, err := reporting.Bind(report, raw, locale)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := m.engine.Run(r.Context(), tenantID, report, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "the report could not be produced: "+err.Error())
		return
	}

	title := reporting.LocalizedTitle(report.Titles(), locale, report.Key())
	payload, err := reporting.Export(format, title, result, locale)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "the export could not be written")
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

// resolve is the guard every report endpoint shares: a tenant, a report that
// exists, and an app that tenant has installed.
//
// The third check is the one that matters and the easiest to leave out. Without
// it, a caller who knows a key can run a report belonging to an app their
// organisation never installed — the list would not show it and the API would
// serve it anyway.
func (m *Module) resolve(w http.ResponseWriter, r *http.Request) (string, reporting.Report, bool) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return "", nil, false
	}

	key := strings.TrimSpace(chi.URLParam(r, "key"))
	report, found := reporting.Get(key)
	if !found {
		httpx.Error(w, http.StatusNotFound, "no such report")
		return "", nil, false
	}

	installed, err := m.installedApps(r, tenantID)
	if err != nil {
		return "", nil, false
	}
	if !installed[report.App()] {
		// 404 rather than 403: whether another organisation's app exists is not
		// this caller's business, and the two answers together enumerate the
		// catalogue.
		httpx.Error(w, http.StatusNotFound, "no such report")
		return "", nil, false
	}
	return tenantID, report, true
}

// installedApps asks the platform, and refuses on error rather than admitting.
func (m *Module) installedApps(r *http.Request, tenantID string) (map[string]bool, error) {
	installed, err := m.installed(r.Context(), tenantID)
	if err != nil {
		return nil, err
	}
	return installed, nil
}

// loadOptions fills a UUID parameter's dropdown from the tenant's own rows.
func (m *Module) loadOptions(r *http.Request, tenantID string, spec reporting.ParamSpec) ([]reporting.ParamOption, error) {
	ctx := tenant.WithTenantID(r.Context(), tenantID)
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
		httpx.Error(w, http.StatusBadRequest, "invalid report parameters")
		return nil, false
	}
	if body.Params != nil {
		raw = body.Params
	}
	return raw, true
}

// record writes the audit entry. Every run and every export, by name.
//
// §3.5 requires this before one tenant may see anything of another's, and it is
// worth having on ordinary runs too: "who read the payroll figures" is a
// question an organisation is entitled to an answer to.
func (m *Module) record(r *http.Request, tenantID, action, key string, details map[string]any) {
	userID := ""
	if claims, err := auth.UserFromContext(r.Context()); err == nil {
		userID = claims.UserID
	}
	audit.Record(r.Context(), tenantID, userID, action, key, details)
}
