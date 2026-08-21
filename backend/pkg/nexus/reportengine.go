/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package nexus

import "context"

// Running a report, as the app that shows them sees it.
//
// The engine stays the platform's: it binds parameters, opens a read-only
// transaction bound to the caller's organisation, runs the SQL a report
// declared, sums the columns marked total and renders a spreadsheet. None of
// that is a decision an app makes, and CORE_BOUNDARY_PLAN §4.2 says so —
// the engine is core, the screens are an app.
//
// What the app was doing instead was calling fifteen functions of the engine's
// package directly: Get, Bind, Run, Export, Filename, ParseFormat, ParseCron,
// ForApps, LocalizedTitle and the rest. Every one of them is under internal/,
// so the screens could not be built anywhere else — and publishing them as they
// stand would commit the engine's whole shape to semver so that one screen
// could move.
//
// Six methods instead, named for what the app does rather than for how the
// engine does it. Get-then-Bind-then-Run is one call here, because a caller
// that could do the three separately could do them in the wrong order.
type ReportEngine interface {
	// Available is what this tenant may run, given the apps it has installed.
	// A report belonging to an app nobody installed is not listed and cannot
	// be run: the gate is the installation, not the permission.
	Available(installed map[string]bool) []ReportDescription

	// Describe is one report, or false. Titles and scopes, no rows.
	Describe(key string) (ReportDescription, bool)

	// Run binds the parameters and produces the rows, in one call.
	//
	// The three steps are not offered separately on purpose: binding is what
	// rejects a parameter a report did not declare, and a caller that could
	// run without binding would be running a report with whatever the browser
	// sent.
	Run(ctx context.Context, tenantID, key string, params map[string]string, locale string) (*ReportRun, error)

	// Export runs a report and renders it to a file.
	//
	// It returns the bytes with the two things a caller has to send beside
	// them — a filename and a content type — because a caller that had to
	// derive those would be deriving them from a format string the engine
	// already parsed.
	Export(ctx context.Context, tenantID, key string, params map[string]string, locale, format string) (*ReportExport, error)

	// ValidateSchedule refuses a schedule the engine could not later run:
	// unknown report, parameters it did not declare, a cron it cannot parse,
	// a format it cannot render. It runs nothing.
	//
	// One call rather than four validators, because a schedule accepted with
	// three of the four checked is a schedule that fails at three in the
	// morning to nobody.
	ValidateSchedule(key string, params map[string]string, locale, cron, format string) error

	// Deliverable reports whether a scheduled report can actually be sent.
	// False on a deployment with no mail configured, which a screen should say
	// before somebody schedules something that will never arrive.
	Deliverable() bool
}

// ReportDescription is a report as a catalogue entry: what it is called and
// whether it can be shared. Not how it is run.
type ReportDescription struct {
	Key    string            `json:"key"`
	App    string            `json:"app"`
	Titles map[string]string `json:"titles"`
	// Scopes are the sharing scopes this report can honour. Empty means it was
	// not written to be shared, and no grant may name it — see
	// ReportScopeFull and ReportScopeCounterparty.
	Scopes []string `json:"scopes,omitempty"`
}

// ReportRun is a report's answer, with the title already resolved for the
// language it was asked in.
type ReportRun struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Result Result `json:"result"`
}

// ReportExport is a rendered report, ready to send.
type ReportExport struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Bytes       []byte `json:"-"`
	// Rows is what was exported, for the audit line the caller writes. An
	// export is a copy of an organisation's data leaving the platform, and the
	// count is the part of it worth recording.
	Rows int `json:"rows"`
}

// Reports returns the engine this deployment provides.
func Reports() (ReportEngine, error) { return Capability[ReportEngine]() }

// What this contract does not carry yet, and why it is written down here
// rather than discovered later.
//
// internal/apps/reports still reaches into the engine's package for three
// things these six methods do not cover. Each is a decision rather than an
// oversight:
//
//   - RunConsolidated. One organisation running a report over another's rows,
//     under a grant. It is a different act from Run — it names a counterparty,
//     it is audited on both sides, and a report that was not written to be
//     shared must refuse it. Folding it into Run would make the safe operation
//     and the dangerous one the same call.
//
//   - Listing schedules and grants. Both read platform tables and return
//     records of thirteen and fifteen fields, which is the shape MeetingConnector
//     is a warning about. They belong with the directory contract the
//     organisation app needs, not bolted onto the engine.
//
//   - The scheduler. The sweep that mails a report at three in the morning is
//     the engine's housekeeping, not a screen's, and it runs today because the
//     app happens to start it. Moving it to the platform is the right change
//     and is not this one.
//
// Until those land, the reports app keeps its import of
// internal/platform/reporting and stays in this repository.
