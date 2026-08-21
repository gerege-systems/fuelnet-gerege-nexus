/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package reporting

import (
	"context"
	"fmt"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// AsEngine presents the report engine as the SDK's nexus.ReportEngine.
//
// Six methods over fifteen package functions, and the reduction is the point.
// The app that shows reports was calling Get, Bind, Run, Export, Filename,
// ParseFormat, ParseCron, ForApps and LocalizedTitle directly — every one of
// them under internal/, which is what kept the screens in this repository.
//
// Where three calls become one, it is because the three had an order: bind
// before run, parse the format before exporting. A contract that offered them
// separately would be a contract a caller could use in the wrong order, and
// the wrong order here runs a report with parameters nothing checked.
func AsEngine(e *Engine) nexus.ReportEngine {
	if e == nil {
		return nil
	}
	return engineAdapter{e}
}

type engineAdapter struct{ engine *Engine }

func (a engineAdapter) Available(installed map[string]bool) []nexus.ReportDescription {
	permitted := ForApps(installed)
	described := make([]nexus.ReportDescription, 0, len(permitted))
	for _, report := range permitted {
		described = append(described, describe(report))
	}
	return described
}

func (a engineAdapter) Describe(key string) (nexus.ReportDescription, bool) {
	report, found := Get(key)
	if !found {
		return nexus.ReportDescription{}, false
	}
	return describe(report), true
}

func (a engineAdapter) Run(ctx context.Context, tenantID, key string, raw map[string]string, locale string) (*nexus.ReportRun, error) {
	report, params, err := a.bind(key, raw, locale)
	if err != nil {
		return nil, err
	}
	result, err := a.engine.Run(ctx, tenantID, report, params)
	if err != nil {
		return nil, err
	}
	return &nexus.ReportRun{
		Key:    report.Key(),
		Title:  LocalizedTitle(report.Titles(), locale, report.Key()),
		Result: result,
	}, nil
}

func (a engineAdapter) Export(ctx context.Context, tenantID, key string, raw map[string]string, locale, rawFormat string) (*nexus.ReportExport, error) {
	format, err := ParseFormat(rawFormat)
	if err != nil {
		return nil, err
	}
	report, params, err := a.bind(key, raw, locale)
	if err != nil {
		return nil, err
	}
	result, err := a.engine.Run(ctx, tenantID, report, params)
	if err != nil {
		return nil, err
	}
	title := LocalizedTitle(report.Titles(), locale, report.Key())
	payload, err := Export(format, title, result, locale)
	if err != nil {
		return nil, err
	}
	return &nexus.ReportExport{
		Filename:    Filename(report.Key(), format),
		ContentType: format.ContentType(),
		Bytes:       payload,
		Rows:        len(result.Rows),
	}, nil
}

func (a engineAdapter) ValidateSchedule(key string, raw map[string]string, locale, cron, format string) error {
	if _, _, err := a.bind(key, raw, locale); err != nil {
		return err
	}
	if _, err := ParseCron(cron); err != nil {
		return err
	}
	if _, err := ParseFormat(format); err != nil {
		return err
	}
	return nil
}

func (a engineAdapter) Deliverable() bool { return NewSMTPDeliverer() != nil }

// bind is the step a caller cannot skip: it refuses a parameter the report did
// not declare, which is the difference between running a report and running
// whatever a browser sent.
func (a engineAdapter) bind(key string, raw map[string]string, locale string) (Report, Params, error) {
	report, found := Get(key)
	if !found {
		return nil, Params{}, fmt.Errorf("no report is registered as %q", key)
	}
	params, err := Bind(report, raw, locale)
	if err != nil {
		return nil, Params{}, err
	}
	return report, params, nil
}

// describe turns a registered report into a catalogue entry.
//
// Scopes come from the Shareable marker rather than from a field: sharing is
// something a report has to be written for, and a report that does not
// implement the marker cannot be named in a grant at all.
func describe(report Report) nexus.ReportDescription {
	described := nexus.ReportDescription{
		Key: report.Key(), App: report.App(), Titles: report.Titles(),
	}
	if shareable, ok := report.(Shareable); ok {
		described.Scopes = shareable.Scopes()
	}
	return described
}
