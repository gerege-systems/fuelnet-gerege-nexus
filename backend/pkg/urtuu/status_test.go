package urtuu_test

import (
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/urtuu"
)

func TestTheStateMachineOnlyAllowsWhatItDeclares(t *testing.T) {
	allowed := []struct{ from, to urtuu.TaskStatus }{
		{urtuu.StatusReceived, urtuu.StatusAccepted},
		{urtuu.StatusReceived, urtuu.StatusReturned},
		{urtuu.StatusAccepted, urtuu.StatusDelegated},
		{urtuu.StatusInProgress, urtuu.StatusCompleted},
		{urtuu.StatusDelegated, urtuu.StatusCompleted},
		{urtuu.StatusCompleted, urtuu.StatusClosed},
		{urtuu.StatusReturned, urtuu.StatusClosed},
	}
	for _, move := range allowed {
		if !move.from.CanMoveTo(move.to) {
			t.Errorf("%s → %s was refused", move.from, move.to)
		}
	}

	refused := []struct{ from, to urtuu.TaskStatus }{
		// Arriving is not accepting: a task cannot be worked on before somebody
		// here has taken it.
		{urtuu.StatusReceived, urtuu.StatusInProgress},
		{urtuu.StatusReceived, urtuu.StatusCompleted},
		// Only the originator closes, and only after an outcome.
		{urtuu.StatusReceived, urtuu.StatusClosed},
		{urtuu.StatusAccepted, urtuu.StatusClosed},
		// Closed is the end of it.
		{urtuu.StatusClosed, urtuu.StatusAccepted},
		{urtuu.StatusClosed, urtuu.StatusReturned},
		// A completed task is not reopened; a new one is raised.
		{urtuu.StatusCompleted, urtuu.StatusInProgress},
		{urtuu.StatusReturned, urtuu.StatusAccepted},
	}
	for _, move := range refused {
		if move.from.CanMoveTo(move.to) {
			t.Errorf("%s → %s was allowed", move.from, move.to)
		}
	}
}

func TestOnlyClosedIsFinal(t *testing.T) {
	for _, status := range []urtuu.TaskStatus{
		urtuu.StatusReceived, urtuu.StatusAccepted, urtuu.StatusInProgress,
		urtuu.StatusDelegated, urtuu.StatusCompleted, urtuu.StatusReturned,
	} {
		if status.Final() {
			t.Errorf("%s reports itself final; nothing would ever close it", status)
		}
	}
	if !urtuu.StatusClosed.Final() {
		t.Error("CLOSED is not final")
	}
}

func TestKnownStatusRefusesAnythingInvented(t *testing.T) {
	if urtuu.KnownStatus("DONE") {
		t.Error("an invented status was accepted; a query filter could then match nothing silently")
	}
	if !urtuu.KnownStatus(urtuu.StatusInProgress) {
		t.Error("a real status was refused")
	}
}

func TestOverdueIsAFlagAndNotAState(t *testing.T) {
	now := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if !urtuu.Overdue(urtuu.StatusInProgress, &past, now) {
		t.Error("work in progress past its deadline is not flagged late")
	}
	if urtuu.Overdue(urtuu.StatusInProgress, &future, now) {
		t.Error("work inside its deadline is flagged late")
	}
	if urtuu.Overdue(urtuu.StatusInProgress, nil, now) {
		t.Error("a task with no deadline is flagged late")
	}
	// Finishing late is still late. Only the originator accepting the outcome
	// settles it.
	if !urtuu.Overdue(urtuu.StatusCompleted, &past, now) {
		t.Error("a task finished after its deadline stopped being late by being finished")
	}
	if urtuu.Overdue(urtuu.StatusClosed, &past, now) {
		t.Error("a closed task is still reported late")
	}
}

func TestRequestCodeNamesFallBackRatherThanBlank(t *testing.T) {
	code := urtuu.RequestCode{
		Code:  "D-101",
		Names: map[string]string{"mn": "Хагас жилийн тооллого", "en": "Half-year count"},
	}
	if got := code.LocalizedName("en"); got != "Half-year count" {
		t.Errorf("en = %q", got)
	}
	// Not yet translated into Arabic: Mongolian is the source language of the
	// register, so it is what an untranslated code is shown as.
	if got := code.LocalizedName("ar"); got != "Хагас жилийн тооллого" {
		t.Errorf("ar = %q, want the Mongolian source", got)
	}
	if got := (urtuu.RequestCode{Code: "local.audit"}).LocalizedName("mn"); got != "local.audit" {
		t.Errorf("unlabelled = %q, want the code itself", got)
	}
}

func TestKnownSourceIsClosed(t *testing.T) {
	for _, source := range []string{urtuu.SourceRing, urtuu.SourceLink, urtuu.SourceLocal} {
		if !urtuu.KnownSource(source) {
			t.Errorf("%s was refused", source)
		}
	}
	if urtuu.KnownSource("manual") {
		t.Error("an invented source was accepted")
	}
}
