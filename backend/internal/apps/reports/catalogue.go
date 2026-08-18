/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The reporting engine, answering the questions the app's rules ask of it.
 */

package reports

import (
	domain "github.com/gerege-systems/open-gerege-nexus/backend/domain/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/reporting"
)

// catalogue is domain/reports.Catalogue over internal/platform/reporting.
//
// It is a translation and nothing else: a registered report becomes the four
// facts the rules need — its key, its app, its names and the sharing scopes it
// was written for — and the three validators forward the engine's own refusal.
// Everything the engine does that is not answering a question about a report —
// running one, exporting one, mailing one — stays out of the domain's reach on
// purpose, because none of it is this app's decision.
type catalogue struct{}

func (catalogue) Report(key string) (domain.Report, bool) {
	report, found := reporting.Get(key)
	if !found {
		return domain.Report{}, false
	}
	return describe(report), true
}

func (catalogue) ForApps(installed map[string]bool) []domain.Report {
	permitted := reporting.ForApps(installed)
	described := make([]domain.Report, 0, len(permitted))
	for _, report := range permitted {
		described = append(described, describe(report))
	}
	return described
}

func (catalogue) Title(report domain.Report, locale string) string {
	return reporting.LocalizedTitle(report.Titles, locale, report.Key)
}

func (catalogue) ValidateParams(key string, params map[string]string, locale string) error {
	report, found := reporting.Get(key)
	if !found {
		return nil
	}
	_, err := reporting.Bind(report, params, locale)
	return err
}

func (catalogue) ValidateCron(expression string) error {
	_, err := reporting.ParseCron(expression)
	return err
}

func (catalogue) NormalizeFormat(raw string) (string, error) {
	format, err := reporting.ParseFormat(raw)
	return string(format), err
}

// describe is the one place a registered report becomes a value.
//
// Scopes come from the Shareable marker rather than from a field: sharing is
// something a report has to be written for, and a report that does not
// implement the marker cannot be named in a grant at all.
func describe(report reporting.Report) domain.Report {
	described := domain.Report{
		Key:    report.Key(),
		App:    report.App(),
		Titles: report.Titles(),
	}
	if shareable, ok := report.(reporting.Shareable); ok {
		described.Scopes = shareable.Scopes()
	}
	return described
}
