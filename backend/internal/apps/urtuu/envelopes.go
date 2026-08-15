/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * What crosses the boundary: work going down, and news coming back up.
 *
 * Two kinds and no more. A task assignment goes from a parent to one child; a
 * task update goes from a child back to the parent for one task. Everything
 * else the board does — assigning somebody, editing a note, closing a
 * completed task — is local, and deliberately: an organisation's internal
 * arrangements are its own (§2.4), and only the state of the work travels.
 */

package urtuu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	transport "github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/urtuu"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	contract "github.com/gerege-systems/open-gerege-nexus/backend/pkg/urtuu"
	"github.com/jackc/pgx/v5"
)

// assignment is the payload of a task.assigned envelope.
type assignment struct {
	// TaskID is the *sender's* row id — the mirror it keeps of this work. It
	// comes back on every update, which is how the two sides stay matched
	// without either of them having to know the other's database.
	TaskID  string          `json:"task_id"`
	Code    string          `json:"code"`
	Title   string          `json:"title"`
	Payload json.RawMessage `json:"payload"`
	// Deadline is the sender's, and it is absolute rather than a duration:
	// the two clocks disagree, and a duration would be measured from whichever
	// of them happened to be reading.
	Deadline *time.Time `json:"deadline,omitempty"`
	// OriginChain is every installation this work has passed through. It is the
	// cycle guard (§9) and it is why a graph rather than a tree is safe.
	OriginChain []string `json:"origin_chain"`
}

// update is the payload of a task.update envelope.
type update struct {
	// TaskID is the *recipient's* row id, taken from the assignment that
	// started this. The sender of an update is naming a row in the receiver's
	// database, which is the only way an answer can find its question.
	TaskID   string          `json:"task_id"`
	Status   string          `json:"status"`
	Note     string          `json:"note,omitempty"`
	Evidence json.RawMessage `json:"evidence,omitempty"`
}

// -------------------------------------------------------------- sending down

// sendDown writes the mirror row for one subordinate and queues the work.
//
// The row is written first and inside the caller's transaction: if the enqueue
// fails, everything unwinds, because a mirror with nothing behind it would show
// on the tree as work in progress that nobody was ever asked to do.
func (m *Module) sendDown(ctx context.Context, tx pgx.Tx, tenantID, actorUserID string,
	parent Task, peerID string, deadline *time.Time) error {

	open, err := m.codeOpenOn(ctx, tenantID, peerID, parent.Code)
	if err != nil {
		return err
	}
	if !open {
		return fmt.Errorf("the code %s is not open on that link", parent.Code)
	}

	var mirrorID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO urtuu_tasks
		    (tenant_id, code, title, payload, target_peer_id, parent_task_id,
		     origin_chain, status, deadline, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'RECEIVED', $8, NULLIF($9, '')::uuid)
		RETURNING id`,
		tenantID, parent.Code, parent.Title, parent.Payload, peerID, parent.ID,
		parent.OriginChain, deadline, actorUserID).Scan(&mirrorID); err != nil {
		return err
	}
	if err := m.record(ctx, tx, tenantID, mirrorID, string(contract.StatusReceived), actorUserID, "", ""); err != nil {
		return err
	}

	_, err = m.link.EnqueueTx(ctx, tx, tenantID, contract.KindTaskAssigned, assignment{
		TaskID:      mirrorID,
		Code:        parent.Code,
		Title:       parent.Title,
		Payload:     parent.Payload,
		Deadline:    deadline,
		OriginChain: parent.OriginChain,
	}, peerID)
	return err
}

// reportUp tells the installation that gave us this task where it has got to.
//
// Best effort in the sense that a failure here does not undo the local move:
// the work really has been accepted, and the enqueue is the one part that gets
// retried by the transport anyway. What it must not do is silently succeed —
// hence the log line, and hence the undelivered count on the links screen.
func (m *Module) reportUp(ctx context.Context, tenantID string, task Task, note string) {
	if task.OriginPeerID == "" || task.OriginTaskRef() == "" {
		return
	}
	if _, err := m.link.Enqueue(ctx, tenantID, contract.KindTaskUpdate, update{
		TaskID:   task.OriginTaskRef(),
		Status:   task.Status,
		Note:     note,
		Evidence: task.Evidence,
	}, task.OriginPeerID); err != nil {
		slog.Warn("urtuu: could not report a task's state upward",
			"task_id", task.ID, "status", task.Status, "error", err)
	}
}

// ------------------------------------------------------------ receiving down

// receiveAssignment turns an envelope into a task.
//
// Two refusals happen here and both answer upward rather than dropping the
// message. A cycle is refused because the work has already been through this
// installation and going round again would never terminate; an unknown code is
// refused because a task that cannot be read is a task nobody can do. In both
// cases the parent is told, with a reason, which is the whole difference
// between a refusal and a silence.
func (m *Module) receiveAssignment(ctx context.Context, message transport.Received) error {
	var work assignment
	if err := json.Unmarshal(message.Payload, &work); err != nil {
		// From a verified sender, so retrying will not make it parse. Marked
		// read rather than left to fail in the log for ever.
		slog.Warn("urtuu: an assignment could not be read", "peer_id", message.PeerID, "error", err)
		return nil
	}

	ctx = nexus.WithTenantID(ctx, message.TenantID)

	if refusal := m.refuseAssignment(ctx, message, work); refusal != "" {
		if _, err := m.link.Enqueue(ctx, message.TenantID, contract.KindTaskUpdate, update{
			TaskID: work.TaskID,
			Status: string(contract.StatusReturned),
			Note:   refusal,
		}, message.PeerID); err != nil {
			return err
		}
		return nil
	}

	code, err := m.lookupCode(ctx, message.TenantID, work.Code)
	if err != nil {
		return err
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The deadline the sender set, or this code's own norm measured from the
	// envelope's stamp — the sender's clock, because that is the moment the
	// work was promised against.
	deadline := deadlineFor(code, work.Deadline, message.CreatedAt)

	var taskID string
	err = tx.QueryRow(ctx, `
		INSERT INTO urtuu_tasks
		    (tenant_id, code, title, payload, origin_peer_id, origin_task_id,
		     origin_chain, status, deadline)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'RECEIVED', $8)
		-- The unique index on (origin_peer_id, origin_task_id). A reader has to
		-- be safe to repeat: the envelope is идемпотент by message id, but the
		-- write that marks it read can fail after this one succeeded.
		ON CONFLICT (origin_peer_id, origin_task_id) WHERE origin_peer_id IS NOT NULL DO NOTHING
		RETURNING id`,
		message.TenantID, work.Code, work.Title, payloadOrEmpty(work.Payload),
		message.PeerID, work.TaskID, append(work.OriginChain, m.link.InstallationID()),
		deadline).Scan(&taskID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already here. Nothing to do and nothing wrong.
		return nil
	}
	if err != nil {
		return err
	}
	if err := m.record(ctx, tx, message.TenantID, taskID,
		string(contract.StatusReceived), "", message.PeerID, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// refuseAssignment returns the reason this task cannot be taken, or empty.
func (m *Module) refuseAssignment(ctx context.Context, message transport.Received, work assignment) string {
	// The cycle guard. A→Б→А is a legitimate shape for a peer graph — two
	// ministries can each be above the other for different kinds of work — so
	// it is the *task* that must not go round, not the link that must not
	// exist.
	for _, installation := range work.OriginChain {
		if installation == m.link.InstallationID() {
			return "this task has already passed through this installation (origin chain)"
		}
	}

	code, err := m.lookupCode(ctx, message.TenantID, work.Code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "this installation has no request code " + work.Code
	}
	if err != nil {
		// A database failure is not a refusal. Returning empty here would have
		// the caller create the task; returning a reason would refuse work for
		// a transient fault, and neither is right — so it is reported as a
		// refusal only after the lookup has actually answered.
		slog.Warn("urtuu: could not check an incoming task's code", "code", work.Code, "error", err)
		return "this installation could not check the request code " + work.Code
	}
	if !code.Active {
		return "the request code " + work.Code + " is not in use at this installation"
	}
	return ""
}

// ------------------------------------------------------------- receiving back

// receiveUpdate applies a subordinate's news to the mirror this side keeps.
func (m *Module) receiveUpdate(ctx context.Context, message transport.Received) error {
	var news update
	if err := json.Unmarshal(message.Payload, &news); err != nil {
		slog.Warn("urtuu: an update could not be read", "peer_id", message.PeerID, "error", err)
		return nil
	}
	if !contract.KnownStatus(contract.TaskStatus(news.Status)) {
		slog.Warn("urtuu: an update carried an unknown status",
			"peer_id", message.PeerID, "status", news.Status)
		return nil
	}

	ctx = nexus.WithTenantID(ctx, message.TenantID)
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// `target_peer_id = $2` is the whole authorization: a link may move the
	// mirrors of the work it was actually given and nothing else.
	var from, parentID string
	err = tx.QueryRow(ctx, `
		SELECT status, coalesce(parent_task_id::text, '')
		  FROM urtuu_tasks
		 WHERE id = $1 AND tenant_id = $3 AND target_peer_id = $2
		 FOR UPDATE`, news.TaskID, message.PeerID, message.TenantID).Scan(&from, &parentID)
	if errors.Is(err, pgx.ErrNoRows) {
		slog.Warn("urtuu: an update named a task this link was never given",
			"peer_id", message.PeerID, "task_id", news.TaskID)
		return nil
	}
	if err != nil {
		return err
	}

	// A mirror is not a state machine this side owns — it is somebody else's
	// state copied here — so the transition table is not applied to it. What is
	// applied is monotonicity: deliveries retry and can arrive out of order,
	// and an older update must not walk a finished task backwards.
	if rank(news.Status) < rank(from) {
		return nil
	}
	if news.Status == from {
		return nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE urtuu_tasks
		   SET status = $2, note = CASE WHEN $3 = '' THEN note ELSE $3 END,
		       evidence = coalesce($4, evidence), updated_at = NOW()
		 WHERE id = $1`, news.TaskID, news.Status, news.Note, evidenceOrNil(news.Evidence)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO urtuu_task_events
		    (tenant_id, task_id, from_status, to_status, actor_peer_id, note)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		message.TenantID, news.TaskID, from, news.Status, message.PeerID, news.Note); err != nil {
		return err
	}
	tasksTotal.WithLabelValues(news.Status).Inc()
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if parentID != "" {
		m.rollUp(ctx, message.TenantID, parentID)
	}
	return nil
}

// rollUp completes a delegated task once every branch of it has finished.
//
// This is what makes a fan-out a single piece of work rather than a list of
// them: the ministry does not have to watch twenty-one provinces and decide for
// itself when the count is done.
//
// A returned branch does not complete the parent. Somebody has to read the
// reason and decide, which is the difference between a refusal and a delay.
func (m *Module) rollUp(ctx context.Context, tenantID, parentID string) {
	ctx = nexus.WithTenantID(ctx, tenantID)

	var outstanding int
	if err := m.db.QueryRow(ctx, `
		SELECT count(*) FROM urtuu_tasks
		 WHERE parent_task_id = $1 AND target_peer_id IS NOT NULL AND status <> 'COMPLETED'`,
		parentID).Scan(&outstanding); err != nil {
		slog.Warn("urtuu: could not check a delegated task's branches", "task_id", parentID, "error", err)
		return
	}
	if outstanding > 0 {
		return
	}

	parent, err := m.move(ctx, tenantID, parentID, contract.StatusCompleted, "", "",
		"every delegated branch is complete")
	if err != nil {
		// Ordinary when a parent has already been completed or returned by
		// hand, which is why this is a debug line and not a warning.
		slog.Debug("urtuu: a delegated task did not roll up", "task_id", parentID, "error", err)
		return
	}
	m.reportUp(ctx, tenantID, parent, "every delegated branch is complete")
	// And upward again: a chain four installations deep completes from the
	// bottom, one link at a time.
	if parent.ParentTaskID != "" {
		m.rollUp(ctx, tenantID, parent.ParentTaskID)
	}
}

// rank orders the statuses for the monotonicity check above. It is not part of
// the contract's state machine and must not be mistaken for one: it exists only
// so that two updates arriving out of order settle the same way round.
func rank(status string) int {
	switch contract.TaskStatus(status) {
	case contract.StatusReceived:
		return 0
	case contract.StatusAccepted:
		return 1
	case contract.StatusInProgress, contract.StatusDelegated:
		return 2
	case contract.StatusCompleted, contract.StatusReturned:
		return 3
	case contract.StatusClosed:
		return 4
	default:
		return -1
	}
}

// OriginTaskRef is the id this task has on the installation that sent it.
//
// A method rather than a field on Task because it is never shown: the id means
// nothing here, it is only ever quoted back. Keeping it off the JSON keeps
// another installation's internal identifier out of this one's API.
func (t Task) OriginTaskRef() string { return t.originTaskID }

func payloadOrEmpty(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func evidenceOrNil(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return []byte(raw)
}
