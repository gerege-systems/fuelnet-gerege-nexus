/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The contract a report implements, and the shapes it declares itself in.
 */

// Package reporting is the platform's reporting engine.
//
// A report is not a screen and not a query — it is a declaration. It says what
// it is called in every language the platform speaks, what parameters it will
// accept, what columns it produces, and how to produce them. Everything else —
// the list screen, the parameter form, the table, the chart, the Excel export,
// the schedule that mails it out, the audit entry recording that somebody ran
// it — is written once here and applies to every report any module ever adds.
//
// That is the Odoo shape, and it is the reason this is a platform package and
// not an app. A module that wants a report writes an implementation of Report
// and registers it in its init; it writes no handler, no CSV writer, no form.
//
// The isolation rules are unchanged. A report's Run receives a Querier bound to
// the caller's tenant, which is the same pool every handler uses, under the same
// row-level policies from migration 00029. A report that forgets its
// `WHERE tenant_id = $1` returns nothing rather than another organisation's
// numbers. Crossing that line at all is §3.5's business — see grants.go — and
// it happens by running the same query, unchanged, inside the *other* tenant's
// context.
package reporting

import (
	"context"
	"time"
)

// Report is what a module implements to add a report to the platform.
type Report interface {
	// Key identifies the report everywhere: in the API, in a schedule row, in
	// a grant. Dotted, stable, and never reused for something else —
	// "billing.revenue_by_month".
	Key() string

	// App is the module id this report belongs to, e.g.
	// "io.gerege.nexus.billing". A tenant that has not installed that app does
	// not see the report at all.
	App() string

	// Titles is the report's name per locale. "mn" is required; the resolver
	// falls back to "en" and then to the key.
	Titles() map[string]string

	// Params declares what the caller may pass. Anything not declared is
	// rejected before Run is reached.
	Params() []ParamSpec

	// Columns describes the result, for the table, the chart and the export
	// header. Run must return rows matching it.
	Columns() []ColumnSpec

	// Run executes the report. It must use only the Querier it is handed, and
	// only the parameters in p.
	Run(ctx context.Context, q Querier, p Params) (Result, error)
}

// Querier is the read surface a report gets. Deliberately narrower than
// pgxpool.Pool: a report reads. It cannot Exec, cannot open a transaction and
// cannot reach for a connection of its own — which would be a connection
// outside the tenant binding.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
}

// Rows is the cursor a report iterates. It mirrors the part of pgx.Rows a
// report needs, so that reports do not depend on the driver.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

// ParamKind is how a parameter is rendered and validated.
type ParamKind string

const (
	// ParamDateRange is a from/to pair. It arrives as two keys — `<key>_from`
	// and `<key>_to` — and is the parameter almost every report has.
	ParamDateRange ParamKind = "date_range"
	// ParamUUID is a reference to a row the tenant owns: a warehouse, a
	// contact. Options are filled by OptionsQuery.
	ParamUUID ParamKind = "uuid"
	// ParamSelect is a closed list declared in code.
	ParamSelect ParamKind = "select"
	// ParamText is free text. Always passed as a query parameter, never
	// interpolated — see params.go.
	ParamText ParamKind = "text"
	// ParamBool is a checkbox.
	ParamBool ParamKind = "bool"
)

// ParamSpec declares one parameter.
type ParamSpec struct {
	Key      string            `json:"key"`
	Kind     ParamKind         `json:"kind"`
	Titles   map[string]string `json:"titles"`
	Required bool              `json:"required"`
	// Options is the closed list for ParamSelect.
	Options []ParamOption `json:"options,omitempty"`
	// OptionsQuery is the SQL that fills a ParamUUID's dropdown. It must select
	// exactly two columns, id and label, and it runs under the caller's tenant
	// binding like everything else.
	OptionsQuery string `json:"-"`
	// Default is used when the caller omits the parameter. For a date range it
	// is a duration back from today, e.g. 30 * 24 * time.Hour.
	Default       any           `json:"default,omitempty"`
	DefaultWindow time.Duration `json:"-"`
}

// ParamOption is one entry of a ParamSelect.
type ParamOption struct {
	Value  string            `json:"value"`
	Titles map[string]string `json:"titles"`
}

// ColumnKind decides formatting in the table, in the chart and in the export.
type ColumnKind string

const (
	ColumnText    ColumnKind = "text"
	ColumnNumber  ColumnKind = "number"
	ColumnMoney   ColumnKind = "money"
	ColumnDate    ColumnKind = "date"
	ColumnMonth   ColumnKind = "month"
	ColumnPercent ColumnKind = "percent"
)

// ChartRole is the hint the frontend uses to draw a report without being told
// about it. A report with exactly one Category column and at least one Value
// column gets a chart; anything else gets a table only.
type ChartRole string

const (
	ChartNone     ChartRole = ""
	ChartCategory ChartRole = "category"
	ChartValue    ChartRole = "value"
)

// ColumnSpec describes one output column.
type ColumnSpec struct {
	Key    string            `json:"key"`
	Titles map[string]string `json:"titles"`
	Kind   ColumnKind        `json:"kind"`
	Chart  ChartRole         `json:"chart,omitempty"`
	// Total asks the engine to sum this column across the rows and put the
	// figure in Result.Totals. Only meaningful for number and money.
	Total bool `json:"total,omitempty"`
}

// Result is what a report returns.
type Result struct {
	Columns []ColumnSpec     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	// Totals holds the sums for columns declared with Total. Computed by the
	// engine rather than by each report, so a report cannot return a total that
	// disagrees with its own rows.
	Totals map[string]float64 `json:"totals,omitempty"`
	// Notes carries anything the reader has to know to read the numbers
	// correctly — most importantly, in a consolidated run, which organisations
	// failed and are therefore missing from the totals.
	Notes []Note `json:"notes,omitempty"`
}

// Note is a per-result remark, in the caller's locale where the engine writes
// it and in Mongolian where a report does.
type Note struct {
	Level   string `json:"level"` // "info" | "warning"
	Message string `json:"message"`
}

// LocalizedTitle resolves a title map against a locale, falling back to
// Mongolian, then English, then the fallback string.
//
// Mongolian first because it is this platform's source language: a report added
// with only `mn` filled in is readable to the people it was written for, and a
// missing translation should not make a report anonymous.
func LocalizedTitle(titles map[string]string, locale, fallback string) string {
	if title, ok := titles[locale]; ok && title != "" {
		return title
	}
	if title, ok := titles["mn"]; ok && title != "" {
		return title
	}
	if title, ok := titles["en"]; ok && title != "" {
		return title
	}
	return fallback
}
