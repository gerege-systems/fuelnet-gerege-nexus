/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * A task's states, and the request-code vocabulary tasks are drawn from.
 */

package urtuu

import (
	"encoding/json"
	"strings"
	"time"
)

// TaskStatus is where a task has got to.
//
// The values are Latin and upper case because every other status on this
// platform is — gov_tasks, esign sessions, integrations — and because they are
// written into a CHECK constraint, read by a report engine and matched by a
// Prometheus label. The Mongolian name each one is presented under is in the
// comment beside it and lives in the dictionary
// (frontend/lib/i18n/addons/urtuu.ts), which is where the other six languages
// are as well. A status value is an identifier, not a label.
type TaskStatus string

const (
	// StatusReceived — ИРСЭН. The envelope arrived and a row exists. Nobody
	// here has looked at it yet.
	StatusReceived TaskStatus = "RECEIVED"
	// StatusAccepted — ХҮЛЭЭН АВСАН. This installation has taken responsibility
	// for it. The distinction from RECEIVED is the whole point of the two:
	// arriving is something the sender did, accepting is something the receiver
	// did, and only the second is a commitment.
	StatusAccepted TaskStatus = "ACCEPTED"
	// StatusInProgress — ХИЙГДЭЖ БАЙГАА. Somebody here is working on it.
	StatusInProgress TaskStatus = "IN_PROGRESS"
	// StatusDelegated — ЗАДАЛСАН. Broken out to children of this installation;
	// the child rows carry parent_task_id back to this one.
	StatusDelegated TaskStatus = "DELEGATED"
	// StatusCompleted — БИЕЛСЭН. Done here. A delegated task reaches this by
	// itself once every child task has.
	StatusCompleted TaskStatus = "COMPLETED"
	// StatusReturned — БУЦААСАН. Refused, with a reason, back up. Not a
	// failure state to be retried silently: somebody upstream has to read the
	// reason and decide.
	StatusReturned TaskStatus = "RETURNED"
	// StatusClosed — ХААГДСАН. The originator has accepted the outcome. Only
	// the side that sent the task may close it, which is why COMPLETED and
	// CLOSED are two states and not one.
	StatusClosed TaskStatus = "CLOSED"
)

// transitions is the whole state machine, as "from → what may follow".
//
// A map rather than a switch because the transport, the app and the migration's
// CHECK constraint all have to agree about it, and one table they can each read
// is the only version of that agreement that cannot drift.
var transitions = map[TaskStatus][]TaskStatus{
	StatusReceived:   {StatusAccepted, StatusReturned},
	StatusAccepted:   {StatusInProgress, StatusDelegated, StatusCompleted, StatusReturned},
	StatusInProgress: {StatusDelegated, StatusCompleted, StatusReturned},
	// A delegated task is not worked on here any more; it is waiting on its
	// children. It completes when they all do, or is returned if the tree
	// cannot be finished.
	StatusDelegated: {StatusCompleted, StatusReturned},
	StatusCompleted: {StatusClosed},
	// A returned task is closed by whoever sent it, after they have read why.
	StatusReturned: {StatusClosed},
	StatusClosed:   nil,
}

// KnownStatus reports whether a value is a status at all. Used where a status
// arrives from outside — a filter in a query string, a field in an envelope.
func KnownStatus(status TaskStatus) bool {
	_, ok := transitions[status]
	return ok
}

// CanMoveTo reports whether one status may follow another.
func (s TaskStatus) CanMoveTo(next TaskStatus) bool {
	for _, allowed := range transitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// Next lists what may follow this status, for a screen that has to offer the
// buttons rather than guess at them.
func (s TaskStatus) Next() []TaskStatus { return transitions[s] }

// Final reports whether a task has stopped moving. CLOSED alone: COMPLETED is
// an answer waiting to be accepted, and RETURNED is a refusal waiting to be
// read.
func (s TaskStatus) Final() bool { return s == StatusClosed }

// Overdue is the ХОЦОРСОН flag.
//
// A flag and not a status, because it is true *of* a status rather than instead
// of one: a task can be in progress and late, and collapsing the two would lose
// whichever of the facts was written second. It is derived on every read rather
// than stored, so a deadline that is edited does not leave a stale mark behind
// — the one thing housekeeping persists is the metric, not the truth.
func Overdue(status TaskStatus, deadline *time.Time, now time.Time) bool {
	if deadline == nil || status.Final() {
		return false
	}
	// COMPLETED is deliberately still eligible. Work finished after the
	// deadline was finished late, and the parent closing it is what settles
	// the question — hiding that between COMPLETED and CLOSED would let a late
	// task be laundered by being finished.
	return now.After(*deadline)
}

// ---------------------------------------------------------- request codes

// Where a request code came from. A code is never free text: a task is created
// by choosing one, and the vocabulary is somebody's register rather than this
// platform's invention.
const (
	// SourceRing — ring.dgov.mn, the state's register of service processes.
	// This platform is a consumer of it and does not author codes there.
	SourceRing = "ring"
	// SourceLink — announced by a parent on a link and synced down. A child
	// cannot edit these; the parent decides what it will accept.
	SourceLink = "link"
	// SourceLocal — an organisation's own, for work that has no entry in the
	// national register. Namespaced so the two can never be confused.
	SourceLocal = "local"
)

// LocalPrefix is what a locally-authored code must start with. Without a
// namespace, a local code invented today would collide with a ring code
// published tomorrow, and the collision would look like a rename rather than
// like two different things.
const LocalPrefix = "local."

// RequestCode is one entry in the vocabulary a task may be raised under.
//
// It is in the contract package rather than in the app because a code travels:
// a parent announces its open codes over a link and a child stores what it was
// told, so both ends need the same idea of what one is.
type RequestCode struct {
	Code string `json:"code"`
	// Names is the code's label per locale, keyed by ISO 639-1. Server-owned
	// content, so the seven translations travel with the code rather than
	// needing to exist in every consumer's dictionary.
	Names map[string]string `json:"names"`
	// Schema is a JSON Schema for the task body. The form a person fills in is
	// generated from it, and what they filled in is validated against it, so
	// the two cannot disagree.
	Schema json.RawMessage `json:"schema,omitempty"`
	// DefaultSLA is how long the work is normally allowed, from the envelope's
	// created_at. Zero means the code names no norm and a deadline has to be
	// set by hand.
	DefaultSLA time.Duration `json:"default_sla"`
	Source     string        `json:"source"`
	// RingProcessRef is the id of the process in ring.dgov.mn this was imported
	// from. Kept so a re-import can update rather than duplicate, and so a
	// question about the norm has somewhere to be asked.
	RingProcessRef string `json:"ring_process_ref,omitempty"`
	Version        int    `json:"version"`
	Active         bool   `json:"active"`
}

// LocalizedName resolves a code's label, falling back through Mongolian — the
// source language of the register — to the code itself. A code with no label at
// all is still better rendered as its code than as an empty cell.
func (c RequestCode) LocalizedName(locale string) string {
	if name := strings.TrimSpace(c.Names[locale]); name != "" {
		return name
	}
	if name := strings.TrimSpace(c.Names["mn"]); name != "" {
		return name
	}
	return c.Code
}

// KnownSource reports whether a value is one of the three origins.
func KnownSource(source string) bool {
	return source == SourceRing || source == SourceLink || source == SourceLocal
}
