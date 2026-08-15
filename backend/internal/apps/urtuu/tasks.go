/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * A task's life: raising one, moving it, and splitting it downward.
 *
 * Three shapes of row live in urtuu_tasks and the migration explains them; the
 * one to keep in mind while reading this file is the *mirror*. When work is
 * sent to a subordinate installation, a row is written here that stands for
 * that work over there. It is never moved by anybody on this side — its status
 * changes only when a task_update arrives from the installation actually doing
 * it — and it is what makes a fan-out visible as a tree rather than as a
 * message that left.
 */

package urtuu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	contract "github.com/gerege-systems/open-gerege-nexus/backend/pkg/urtuu"
	"github.com/jackc/pgx/v5"
)

// Task is one row as the API returns it.
type Task struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	// Line is which of Өртөө's two promises this task is under — a state
	// service somebody applied for, or an assignment a superior organisation
	// gave. Copied from the code when the task is raised, so that a code being
	// withdrawn later cannot change what a task in flight was.
	Line    string          `json:"line"`
	Title   string          `json:"title"`
	Payload json.RawMessage `json:"payload"`
	// Applicant is who asked, on the service line. Empty on the assignment
	// line, where there is nobody outside the platform waiting.
	Applicant json.RawMessage `json:"applicant,omitempty"`
	// Answer is what is being told back to the applicant. A service task
	// cannot be completed without one — the schema itself refuses it.
	Answer string `json:"answer,omitempty"`
	// Direction is derived rather than stored: "incoming" is work this
	// organisation owes somebody, "outgoing" is work it is owed, and "local" is
	// its own. One field the screens can filter on instead of three nullable
	// ids they would each have to interpret.
	Direction      string     `json:"direction"`
	OriginPeerID   string     `json:"origin_peer_id,omitempty"`
	OriginPeerName string     `json:"origin_peer_name,omitempty"`
	TargetPeerID   string     `json:"target_peer_id,omitempty"`
	TargetPeerName string     `json:"target_peer_name,omitempty"`
	ParentTaskID   string     `json:"parent_task_id,omitempty"`
	OriginChain    []string   `json:"origin_chain"`
	Status         string     `json:"status"`
	Deadline       *time.Time `json:"deadline,omitempty"`
	// Overdue is computed on every read. Storing it would leave a stale mark
	// behind the moment a deadline is edited.
	Overdue        bool            `json:"overdue"`
	AssignedUserID string          `json:"assigned_user_id,omitempty"`
	AssignedName   string          `json:"assigned_name,omitempty"`
	Note           string          `json:"note,omitempty"`
	Evidence       json.RawMessage `json:"evidence,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	// originTaskID is this task's id on the installation that sent it, quoted
	// back on every update. Unexported and unserialised: another installation's
	// internal identifier has no business in this one's API, and nothing on a
	// screen could do anything with it. Read it through OriginTaskRef.
	originTaskID string
}

// TaskEvent is one transition.
type TaskEvent struct {
	FromStatus string    `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	ActorName  string    `json:"actor_name,omitempty"`
	PeerName   string    `json:"peer_name,omitempty"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// taskColumns is the one place the select list lives. Three queries read tasks
// — the list, the detail and the tree — and a column added to one of them and
// not the others is how a screen quietly stops showing something.
const taskColumns = `
	t.id::text, t.code, t.line, t.title, t.payload, t.applicant, t.answer,
	coalesce(t.origin_peer_id::text, ''), coalesce(op.name, ''), t.origin_task_id,
	coalesce(t.target_peer_id::text, ''), coalesce(tp.name, ''),
	coalesce(t.parent_task_id::text, ''), t.origin_chain, t.status, t.deadline,
	coalesce(t.assigned_user_id::text, ''), coalesce(u.name, ''),
	t.note, t.evidence, t.created_at, t.updated_at`

const taskFrom = `
	FROM urtuu_tasks t
	LEFT JOIN urtuu_peers op ON op.id = t.origin_peer_id
	LEFT JOIN urtuu_peers tp ON tp.id = t.target_peer_id
	LEFT JOIN users u ON u.id = t.assigned_user_id`

func scanTask(rows pgx.Rows, now time.Time) (Task, error) {
	var task Task
	if err := rows.Scan(&task.ID, &task.Code, &task.Line, &task.Title, &task.Payload,
		&task.Applicant, &task.Answer,
		&task.OriginPeerID, &task.OriginPeerName, &task.originTaskID,
		&task.TargetPeerID, &task.TargetPeerName,
		&task.ParentTaskID, &task.OriginChain, &task.Status, &task.Deadline,
		&task.AssignedUserID, &task.AssignedName, &task.Note, &task.Evidence,
		&task.CreatedAt, &task.UpdatedAt); err != nil {
		return Task{}, err
	}
	task.Direction = directionOf(task)
	task.Overdue = contract.Overdue(contract.TaskStatus(task.Status), task.Deadline, now)
	return task, nil
}

func directionOf(task Task) string {
	switch {
	case task.OriginPeerID != "":
		return "incoming"
	case task.TargetPeerID != "":
		return "outgoing"
	default:
		return "local"
	}
}

// taskFilter is what the two queue screens ask for.
type taskFilter struct {
	Direction string
	Line      string
	Status    string
	Code      string
	Overdue   bool
	ParentID  string
}

func (m *Module) listTasks(ctx context.Context, tenantID string, filter taskFilter) ([]Task, error) {
	// The clauses are built rather than a single query with five OR'd
	// "$n IS NULL" conditions, because that form stops the planner using the
	// partial indexes the migration created for exactly these two screens.
	where := []string{"t.tenant_id = $1"}
	args := []any{tenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}

	switch filter.Direction {
	case "incoming":
		where = append(where, "t.origin_peer_id IS NOT NULL")
	case "outgoing":
		where = append(where, "t.target_peer_id IS NOT NULL")
	case "local":
		where = append(where, "t.origin_peer_id IS NULL AND t.target_peer_id IS NULL")
	}
	if filter.Line != "" {
		add("t.line = $%d", filter.Line)
	}
	if filter.Status != "" {
		add("t.status = $%d", filter.Status)
	}
	if filter.Code != "" {
		add("t.code = $%d", filter.Code)
	}
	if filter.ParentID != "" {
		add("t.parent_task_id = $%d", filter.ParentID)
	}
	if filter.Overdue {
		// The same rule as contract.Overdue, in SQL. Two expressions of one
		// rule is a risk, and the alternative — reading every task and
		// filtering in Go — is worse for a screen whose whole job is to be a
		// short list drawn from a long table.
		where = append(where, "t.deadline IS NOT NULL AND t.deadline < NOW() AND t.status <> 'CLOSED'")
	}

	rows, err := m.db.Query(nexus.WithTenantID(ctx, tenantID),
		`SELECT`+taskColumns+taskFrom+` WHERE `+strings.Join(where, " AND ")+
			` ORDER BY t.created_at DESC LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	tasks := make([]Task, 0, 64)
	for rows.Next() {
		task, err := scanTask(rows, now)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (m *Module) getTask(ctx context.Context, tenantID, id string) (Task, error) {
	rows, err := m.db.Query(nexus.WithTenantID(ctx, tenantID),
		`SELECT`+taskColumns+taskFrom+` WHERE t.tenant_id = $1 AND t.id = $2`, tenantID, id)
	if err != nil {
		return Task{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Task{}, pgx.ErrNoRows
	}
	return scanTask(rows, time.Now())
}

func (m *Module) taskEvents(ctx context.Context, tenantID, id string) ([]TaskEvent, error) {
	rows, err := m.db.Query(nexus.WithTenantID(ctx, tenantID), `
		SELECT e.from_status, e.to_status, coalesce(u.name, ''), coalesce(p.name, ''),
		       e.note, e.created_at
		  FROM urtuu_task_events e
		  LEFT JOIN users u ON u.id = e.actor_user_id
		  LEFT JOIN urtuu_peers p ON p.id = e.actor_peer_id
		 WHERE e.task_id = $1
		 ORDER BY e.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]TaskEvent, 0, 16)
	for rows.Next() {
		var event TaskEvent
		if err := rows.Scan(&event.FromStatus, &event.ToStatus, &event.ActorName,
			&event.PeerName, &event.Note, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// ---------------------------------------------------------------- moving one

// ErrTransition is refusing a move the state machine does not allow.
var ErrTransition = errors.New("that is not a move this task can make")

// move applies one transition and records it, inside one transaction.
//
// The guard is contract's table rather than a switch here, because the same
// table is what the transport, the app and the migration's CHECK are all held
// against. Two expressions of a state machine drift; one does not.
//
// It returns the row as it now stands, so the caller can decide what to report
// upward without a second read.
func (m *Module) move(ctx context.Context, tenantID, id string, to contract.TaskStatus,
	actorUserID, actorPeerID, note string) (Task, error) {

	ctx = nexus.WithTenantID(ctx, tenantID)
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// FOR UPDATE, because two things move a task: a person pressing a button
	// and an envelope arriving from a peer. Without the lock a child's
	// completion and an operator's return can both read RECEIVED and both
	// write, and the loser's event row would describe a transition that never
	// happened.
	var from string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM urtuu_tasks WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		id, tenantID).Scan(&from); err != nil {
		return Task{}, err
	}
	if !contract.TaskStatus(from).CanMoveTo(to) {
		return Task{}, fmt.Errorf("%w: %s → %s", ErrTransition, from, to)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE urtuu_tasks
		   SET status = $3, note = CASE WHEN $4 = '' THEN note ELSE $4 END, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`, id, tenantID, string(to), note); err != nil {
		return Task{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO urtuu_task_events
		    (tenant_id, task_id, from_status, to_status, actor_user_id, actor_peer_id, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, NULLIF($6, '')::uuid, $7)`,
		tenantID, id, from, string(to), actorUserID, actorPeerID, note); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Task{}, err
	}
	tasksTotal.WithLabelValues(string(to)).Inc()
	return m.getTask(ctx, tenantID, id)
}

// record writes an event with no transition behind it — the birth of a task.
func (m *Module) record(ctx context.Context, tx pgx.Tx, tenantID, taskID, status, actorUserID, actorPeerID, note string) error {
	// Counted with the transitions: a task being raised is a task reaching
	// RECEIVED, and a series that only counted moves would undercount the
	// intake by exactly the tasks that were never moved.
	tasksTotal.WithLabelValues(status).Inc()
	_, err := tx.Exec(ctx, `
		INSERT INTO urtuu_task_events
		    (tenant_id, task_id, from_status, to_status, actor_user_id, actor_peer_id, note)
		VALUES ($1, $2, '', $3, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid, $6)`,
		tenantID, taskID, status, actorUserID, actorPeerID, note)
	return err
}

// ------------------------------------------------------------- the vocabulary

// requestCode is what the app needs to know about a code before raising work
// under it.
//
// Read straight out of the transport's table rather than through an accessor.
// The two packages are one product split by layer, sharing one schema and one
// tenant binding; an interface between them would be a second description of
// the same three columns.
type requestCode struct {
	Code     string
	Names    map[string]string
	SLA      *int64
	Line     string
	Active   bool
	SourceOf string
}

func (m *Module) lookupCode(ctx context.Context, tenantID, code string) (requestCode, error) {
	var found requestCode
	err := m.db.QueryRow(nexus.WithTenantID(ctx, tenantID), `
		SELECT code, names, EXTRACT(EPOCH FROM default_sla)::bigint, line, active, source
		  FROM urtuu_request_codes WHERE tenant_id = $1 AND code = $2`,
		tenantID, code).Scan(&found.Code, &found.Names, &found.SLA, &found.Line,
		&found.Active, &found.SourceOf)
	return found, err
}

// codeOpenOn reports whether a code has been announced on a link. A parent that
// has not opened a code on a link must not be able to send work under it: the
// child would receive a task naming a code it has never been told about, and
// the whole point of announcing the vocabulary is that it does not have to
// guess.
func (m *Module) codeOpenOn(ctx context.Context, tenantID, peerID, code string) (bool, error) {
	var open bool
	err := m.db.QueryRow(nexus.WithTenantID(ctx, tenantID), `
		SELECT EXISTS (SELECT 1 FROM urtuu_peer_codes
		                WHERE tenant_id = $1 AND peer_id = $2 AND code = $3)`,
		tenantID, peerID, code).Scan(&open)
	return open, err
}

// titleFor is what a task is called, decided once when it is raised.
//
// Copied rather than looked up on every read: a code can be withdrawn or
// retranslated, and what was asked for in March has to still read as what was
// asked for in March.
func titleFor(code requestCode, given, locale string) string {
	if title := strings.TrimSpace(given); title != "" {
		return title
	}
	if name := strings.TrimSpace(code.Names[locale]); name != "" {
		return name
	}
	if name := strings.TrimSpace(code.Names["mn"]); name != "" {
		return name
	}
	return code.Code
}

// deadlineFor resolves when the work is due.
//
// From the code's norm when nobody said otherwise, and measured from the moment
// the task is raised — which for received work is the sender's stamp, not this
// installation's clock (§9).
func deadlineFor(code requestCode, given *time.Time, from time.Time) *time.Time {
	if given != nil {
		return given
	}
	if code.SLA == nil || *code.SLA <= 0 {
		return nil
	}
	due := from.Add(time.Duration(*code.SLA) * time.Second)
	return &due
}
