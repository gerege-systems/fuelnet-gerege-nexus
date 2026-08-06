package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
)

// ErrNotRoutable is returned when a document cannot be sent for approval: it is
// missing, belongs to another tenant, or has already left DRAFT.
var ErrNotRoutable = errors.New("document not found or is not a draft")

// ErrAlreadySigned is returned when a citizen signs a document they have already
// signed. One approval per person is not progress through the workflow.
var ErrAlreadySigned = errors.New("this signer has already signed the document")

// WorkflowStep is one approval a document type needs, in order. An empty
// SignerRegNumber means the step counts but names nobody for it.
type WorkflowStep struct {
	Order           int    `json:"order"`
	Name            string `json:"name"`
	SignerRegNumber string `json:"signer_reg_number"`
}

// DocumentWorkflow is the ordered approval chain for one document type. No steps
// means a single signature approves, which is how the app behaved before.
type DocumentWorkflow struct {
	DocType string         `json:"doc_type"`
	Steps   []WorkflowStep `json:"steps"`
}

// ApprovalStep is one step of a document's OWN chain — the copy taken when it
// started waiting for approval, which no later configuration change touches.
type ApprovalStep struct {
	Order int    `json:"order"`
	Name  string `json:"name"`
	// Empty means the step is open: anyone allowed to sign may take it.
	SignerRegNumber string `json:"signer_reg_number"`
}

// querier is satisfied by both the pool and a transaction, so a read that has to
// happen inside somebody else's transaction does not need its own copy.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// snapshotApprovalChain copies the type's chain onto the document and records how
// many signatures that comes to. A type with no chain leaves no steps and a
// requirement of one, which is how a document of such a type has always behaved.
//
// It must run inside the transaction that starts the document waiting, so a
// document can never exist in a state where its requirement and its steps
// disagree.
func (m *DocumentsModule) snapshotApprovalChain(ctx context.Context, tx pgx.Tx, tenantID, docID, docType string) error {
	// A step nobody could fill is copied open. Three ways a stored chain can carry one:
	// a citizen named twice, who signs a document once; a name too short to be a
	// registration number any provider would accept; and a number in a shape no signer
	// will ever present, since both providers' answers are upper-cased and trimmed
	// before they are compared. ReplaceWorkflow refuses all three now, but chains saved
	// before it did still exist — and the path that decides who may sign must never
	// snapshot a chain the path that saves them would reject.
	//
	// The comparison is on the normalised number for the same reason: 'AA90010111' and
	// 'aa90010111' in one chain are one citizen named twice, and the raw strings would
	// not notice. An empty number normalises to empty, which is shorter than the limit,
	// so an open step stays open.
	if _, err := tx.Exec(ctx,
		`INSERT INTO document_approval_steps (tenant_id, document_id, step_order, name, signer_reg_number)
		 SELECT $1, $2, w.step_order, w.name,
		        CASE WHEN length(upper(btrim(w.signer_reg_number))) < $4 THEN ''
		             WHEN w.step_order = min(w.step_order)
		                    OVER (PARTITION BY upper(btrim(w.signer_reg_number)))
		                  THEN upper(btrim(w.signer_reg_number))
		             ELSE '' END
		   FROM document_workflow_steps w
		  WHERE w.tenant_id = $1 AND w.doc_type = $3
		     ON CONFLICT (document_id, step_order) DO NOTHING`,
		tenantID, docID, docType, RegNumberLimit); err != nil {
		return fmt.Errorf("copy approval chain onto document: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE document_records
		    SET required_signatures = GREATEST(1, (
		          SELECT count(*) FROM document_approval_steps s WHERE s.document_id = $1))
		  WHERE id = $1 AND tenant_id = $2`, docID, tenantID); err != nil {
		return fmt.Errorf("pin approval requirement to document: %w", err)
	}
	return nil
}

// approvalPosition is where a document stands in its own chain: how many
// signatures it carries, which step comes next, and which citizens are held back
// for a step further along.
type approvalPosition struct {
	Applied int
	// Next is the step this signature would fill. Nil means the document carries
	// no chain and one open signature approves it.
	Next *ApprovalStep
	// Reserved are the registration numbers a LATER unfilled step names. They may
	// not fill an open step now: a citizen signs a document once, so spending their
	// signature early would leave their own step unfillable and the document
	// unapprovable.
	Reserved []string
	// Unfilled is how many steps of the chain no signature has filled.
	Unfilled int
}

// nextApprovalStep reads a document's position in its own chain: the next step is
// the LOWEST one no signature has filled.
//
// Counting signatures and adding one would be wrong. Signatures made before the
// chain became per-document were matched to the step that names their signer, not
// to their place in time, so a document can legitimately hold a signature on step
// two while step one is still open. Asking such a document for step two again would
// ask a citizen who has already signed, and stick for ever.
func (m *DocumentsModule) nextApprovalStep(ctx context.Context, q querier, tenantID, docID string) (*approvalPosition, error) {
	pos := &approvalPosition{}
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM document_signatures WHERE tenant_id = $1 AND document_id = $2`,
		tenantID, docID).Scan(&pos.Applied); err != nil {
		return nil, fmt.Errorf("count applied signatures: %w", err)
	}

	rows, err := q.Query(ctx,
		`SELECT st.step_order, st.name, st.signer_reg_number
		   FROM document_approval_steps st
		  WHERE st.tenant_id = $1 AND st.document_id = $2
		    AND NOT EXISTS (SELECT 1 FROM document_signatures s
		                     WHERE s.document_id = st.document_id
		                       AND s.step_order = st.step_order)
		  ORDER BY st.step_order`,
		tenantID, docID)
	if err != nil {
		return nil, fmt.Errorf("read unfilled approval steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var step ApprovalStep
		if err := rows.Scan(&step.Order, &step.Name, &step.SignerRegNumber); err != nil {
			return nil, fmt.Errorf("scan approval step: %w", err)
		}
		if pos.Next == nil {
			next := step
			pos.Next = &next
			continue
		}
		if step.SignerRegNumber != "" {
			pos.Reserved = append(pos.Reserved, step.SignerRegNumber)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pos.Unfilled = len(pos.Reserved)
	if pos.Next != nil {
		pos.Unfilled++ // Reserved only counts the named ones after Next
	}
	return pos, nil
}

// unfilledSteps counts the steps of a document's chain that no signature has
// filled. It is asked again after an insert, because "how many signatures" is not
// the same question as "is the chain complete" once a signature can land on a step
// other than the next one.
func (m *DocumentsModule) unfilledSteps(ctx context.Context, q querier, docID string) (int, error) {
	var left int
	if err := q.QueryRow(ctx,
		`SELECT count(*) FROM document_approval_steps st
		  WHERE st.document_id = $1
		    AND NOT EXISTS (SELECT 1 FROM document_signatures s
		                     WHERE s.document_id = st.document_id
		                       AND s.step_order = st.step_order)`, docID).Scan(&left); err != nil {
		return 0, fmt.Errorf("count unfilled approval steps: %w", err)
	}
	return left, nil
}

// DocumentSteps returns a document's own chain, so a screen can show who each
// remaining approval is waiting on.
func (m *DocumentsModule) DocumentSteps(ctx context.Context, tenantID, docID string) ([]ApprovalStep, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	rows, err := m.db.Query(ctx,
		`SELECT step_order, name, signer_reg_number
		   FROM document_approval_steps
		  WHERE tenant_id = $1 AND document_id = $2 ORDER BY step_order`, tenantID, docID)
	if err != nil {
		return nil, fmt.Errorf("query document approval steps: %w", err)
	}
	defer rows.Close()

	list := make([]ApprovalStep, 0)
	for rows.Next() {
		var step ApprovalStep
		if err := rows.Scan(&step.Order, &step.Name, &step.SignerRegNumber); err != nil {
			return nil, fmt.Errorf("scan document approval step: %w", err)
		}
		list = append(list, step)
	}
	return list, rows.Err()
}

// AppliedSignature is one row of a document's signature ledger.
type AppliedSignature struct {
	SignerName      string    `json:"signer_name"`
	SignerRegNumber string    `json:"signer_reg_number"`
	SignerMethod    string    `json:"signer_method"`
	SignatureHash   string    `json:"signature_hash"`
	SignedAt        time.Time `json:"signed_at"`
	// StepOrder is which approval of the document's chain this signature filled.
	StepOrder int `json:"step_order"`
	// The eID certificate the citizen approved with, when the provider returned
	// one. Empty for DAN, and for E-ID approvals whose certificate eID could not
	// parse — a signature is still valid without it, it just carries less proof.
	CertificateSerial string `json:"certificate_serial,omitempty"`
	CertificateIssuer string `json:"certificate_issuer,omitempty"`
}

// ListWorkflows returns the chain for every document type, including the types
// nobody has configured, so the screen shows the whole picture.
func (m *DocumentsModule) ListWorkflows(ctx context.Context, tenantID string) ([]DocumentWorkflow, error) {
	rows, err := m.db.Query(ctx,
		`SELECT doc_type, step_order, name, signer_reg_number
		   FROM document_workflow_steps
		  WHERE tenant_id = $1 ORDER BY doc_type, step_order`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query workflow steps: %w", err)
	}
	defer rows.Close()

	steps := map[string][]WorkflowStep{}
	for rows.Next() {
		var docType string
		var step WorkflowStep
		if err := rows.Scan(&docType, &step.Order, &step.Name, &step.SignerRegNumber); err != nil {
			return nil, fmt.Errorf("scan workflow step: %w", err)
		}
		steps[docType] = append(steps[docType], step)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	list := make([]DocumentWorkflow, 0, len(DocTypes))
	for _, docType := range DocTypes {
		chain := steps[docType]
		if chain == nil {
			chain = []WorkflowStep{}
		}
		list = append(list, DocumentWorkflow{DocType: docType, Steps: chain})
	}
	return list, nil
}

// ReplaceWorkflow swaps the whole chain for one document type. Editing a chain
// step by step would let a half-applied edit decide who may approve, so the
// delete and the inserts share one transaction.
func (m *DocumentsModule) ReplaceWorkflow(ctx context.Context, tenantID, docType string, steps []WorkflowStep) (*DocumentWorkflow, error) {
	docType = strings.ToUpper(strings.TrimSpace(docType))
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("%w: invalid doc_type %q", ErrInvalidConfiguration, docType)
	}
	if len(steps) > 10 {
		return nil, fmt.Errorf("%w: an approval chain is limited to 10 steps", ErrInvalidConfiguration)
	}

	cleaned := make([]WorkflowStep, 0, len(steps))
	namedAt := map[string]int{}
	for i, step := range steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: step %d needs a name", ErrInvalidConfiguration, i+1)
		}
		reg := strings.ToUpper(strings.TrimSpace(step.SignerRegNumber))
		// A step naming something no citizen could present is a step nobody can
		// fill: signing checks the identity a provider vouched for, and both
		// providers refuse a registration number under eight characters.
		if reg != "" && len(reg) < RegNumberLimit {
			return nil, fmt.Errorf("%w: step %d names %q, which is not a registration number",
				ErrInvalidConfiguration, i+1, reg)
		}
		if len(reg) > 64 {
			return nil, fmt.Errorf("%w: step %d names a registration number of %d characters",
				ErrInvalidConfiguration, i+1, len(reg))
		}
		// One citizen signs a document once, so naming the same person at two
		// steps builds a chain that can never be completed — whatever the
		// signature policy says. This has to be refused when it is saved, not
		// discovered when a document sticks halfway.
		if reg != "" {
			if first, repeated := namedAt[reg]; repeated {
				return nil, fmt.Errorf("%w: steps %d and %d both name %s, and one citizen signs a document once",
					ErrInvalidConfiguration, first, i+1, reg)
			}
			namedAt[reg] = i + 1
		}
		cleaned = append(cleaned, WorkflowStep{Order: i + 1, Name: name, SignerRegNumber: reg})
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin workflow update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// One lock per (tenant, type), taken by this screen and by the signature-policy
	// screen. It settles two races. Two admins saving the same chain: under READ
	// COMMITTED the second DELETE cannot see the rows the first inserted, so it
	// clears nothing and then either collides on step_order or silently discards the
	// caller's chain while reporting success. And a chain saved against a policy
	// read outside a transaction: this screen and the policy screen could each pass
	// their guard on a stale view of the other and commit the state both forbid.
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 0))`,
		tenantID, docType); err != nil {
		return nil, fmt.Errorf("lock approval chain: %w", err)
	}

	// The signature policy may insist that only named signers approve this type.
	// Saving a chain that could not satisfy it would lock the type out.
	var requireNamed bool
	if err := tx.QueryRow(ctx,
		`SELECT require_named_signer FROM document_signature_policies
		  WHERE tenant_id = $1 AND doc_type = $2`, tenantID, docType).Scan(&requireNamed); err != nil {
		if !isNoRows(err) {
			return nil, fmt.Errorf("read signature policy: %w", err)
		}
		requireNamed = false // an unconfigured type allows open steps
	}
	if requireNamed {
		if err := stepsCanRequireNamedSigners(docType, cleaned); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM document_workflow_steps WHERE tenant_id = $1 AND doc_type = $2`,
		tenantID, docType); err != nil {
		return nil, fmt.Errorf("clear workflow steps: %w", err)
	}

	for _, step := range cleaned {
		if _, err := tx.Exec(ctx,
			`INSERT INTO document_workflow_steps (tenant_id, doc_type, step_order, name, signer_reg_number)
			      VALUES ($1, $2, $3, $4, $5)`,
			tenantID, docType, step.Order, step.Name, step.SignerRegNumber); err != nil {
			return nil, fmt.Errorf("insert workflow step %d: %w", step.Order, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit workflow update: %w", err)
	}

	// Who must sign a document type is an authority decision, so a change to it
	// is recorded alongside the signatures it will govern.
	audit.Record(ctx, tenantID, actorFor(ctx), "documents.approval_chain_changed", docType, map[string]any{
		"steps": len(cleaned),
	})

	return &DocumentWorkflow{DocType: docType, Steps: cleaned}, nil
}

// stepsCanRequireNamedSigners reports whether a chain could ever be completed
// under a policy that only accepts named signers: every step has to name one, and
// no two steps may name the same citizen, because one citizen signs a document
// once. A chain that fails this would leave the type unapprovable by anybody.
func stepsCanRequireNamedSigners(docType string, steps []WorkflowStep) error {
	if len(steps) == 0 {
		return fmt.Errorf("%w: the %s chain has no steps, so requiring a named signer would leave nobody able to sign", ErrInvalidConfiguration, docType)
	}

	seen := map[string]int{}
	for _, step := range steps {
		if step.SignerRegNumber == "" {
			return fmt.Errorf("%w: step %d of the %s chain (%s) names no signer, so requiring a named signer would leave it unfillable",
				ErrInvalidConfiguration, step.Order, docType, step.Name)
		}
		if first, repeated := seen[step.SignerRegNumber]; repeated {
			return fmt.Errorf("%w: steps %d and %d of the %s chain both name %s, and one citizen signs a document once",
				ErrInvalidConfiguration, first, step.Order, docType, step.SignerRegNumber)
		}
		seen[step.SignerRegNumber] = step.Order
	}
	return nil
}

// workflowStepsTx reads one type's chain inside somebody else's transaction, so a
// guard and the write it protects see the same state.
func (m *DocumentsModule) workflowStepsTx(ctx context.Context, q querier, tenantID, docType string) ([]WorkflowStep, error) {
	rows, err := q.Query(ctx,
		`SELECT step_order, name, signer_reg_number
		   FROM document_workflow_steps
		  WHERE tenant_id = $1 AND doc_type = $2 ORDER BY step_order`, tenantID, docType)
	if err != nil {
		return nil, fmt.Errorf("query workflow steps: %w", err)
	}
	defer rows.Close()

	steps := make([]WorkflowStep, 0)
	for rows.Next() {
		var step WorkflowStep
		if err := rows.Scan(&step.Order, &step.Name, &step.SignerRegNumber); err != nil {
			return nil, fmt.Errorf("scan workflow step: %w", err)
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

// RouteDocument sends a draft for approval. Nothing the app creates is a draft
// today — CreateDocument writes PENDING_APPROVAL — so this exists for rows that
// arrived any other way, and for the day drafting is added.
func (m *DocumentsModule) RouteDocument(ctx context.Context, tenantID, docID string) (*Document, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotRoutable
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin route document: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var docType string
	err = tx.QueryRow(ctx,
		`UPDATE document_records SET status = $1
		  WHERE id = $2 AND tenant_id = $3 AND status = $4
		  RETURNING doc_type`,
		StatusPending, docID, tenantID, StatusDraft).Scan(&docType)
	if isNoRows(err) {
		return nil, ErrNotRoutable
	}
	if err != nil {
		return nil, fmt.Errorf("route document: %w", err)
	}

	// The document takes the chain as it stands at the moment it starts waiting.
	if err := m.snapshotApprovalChain(ctx, tx, tenantID, docID, docType); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit route document: %w", err)
	}

	audit.Record(ctx, tenantID, actorFor(ctx), "documents.routed", docID, nil)

	return m.getDocument(ctx, tenantID, docID)
}

// ListSignatures returns a document's signature ledger, oldest first.
func (m *DocumentsModule) ListSignatures(ctx context.Context, tenantID, docID string) ([]AppliedSignature, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	rows, err := m.db.Query(ctx,
		`SELECT signer_name, signer_reg_number, signer_method, signature_hash, signed_at,
		        COALESCE(step_order, 0),
		        COALESCE(certificate_serial, ''), COALESCE(certificate_issuer, '')
		   FROM document_signatures
		  WHERE tenant_id = $1 AND document_id = $2
		  ORDER BY COALESCE(step_order, 0), signed_at`, tenantID, docID)
	if err != nil {
		return nil, fmt.Errorf("query signatures: %w", err)
	}
	defer rows.Close()

	list := make([]AppliedSignature, 0)
	for rows.Next() {
		var sig AppliedSignature
		if err := rows.Scan(&sig.SignerName, &sig.SignerRegNumber, &sig.SignerMethod,
			&sig.SignatureHash, &sig.SignedAt, &sig.StepOrder,
			&sig.CertificateSerial, &sig.CertificateIssuer); err != nil {
			return nil, fmt.Errorf("scan signature: %w", err)
		}
		list = append(list, sig)
	}
	return list, rows.Err()
}

func (m *DocumentsModule) listWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.ListWorkflows(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch approval chains")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (m *DocumentsModule) saveWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Steps []WorkflowStep `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid approval chain payload")
		return
	}

	saved, err := m.ReplaceWorkflow(r.Context(), tenantID, chi.URLParam(r, "docType"), req.Steps)
	if err != nil {
		writeWriteFailure(r.Context(), w, err, "failed to save the approval chain")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (m *DocumentsModule) routeDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	doc, err := m.RouteDocument(r.Context(), tenantID, chi.URLParam(r, "id"))
	switch {
	case errors.Is(err, ErrNotRoutable):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to route document")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (m *DocumentsModule) listDocumentStepsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.DocumentSteps(r.Context(), tenantID, chi.URLParam(r, "id"))
	switch {
	case errors.Is(err, ErrNotSignable):
		writeError(w, http.StatusNotFound, "document not found")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to fetch the approval chain")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (m *DocumentsModule) listSignaturesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.ListSignatures(r.Context(), tenantID, chi.URLParam(r, "id"))
	switch {
	case errors.Is(err, ErrNotSignable):
		writeError(w, http.StatusNotFound, "document not found")
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to fetch signatures")
		return
	}
	writeJSON(w, http.StatusOK, list)
}
