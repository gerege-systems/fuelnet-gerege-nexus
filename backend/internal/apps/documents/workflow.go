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

// AppliedSignature is one row of a document's signature ledger.
type AppliedSignature struct {
	SignerName      string    `json:"signer_name"`
	SignerRegNumber string    `json:"signer_reg_number"`
	SignerMethod    string    `json:"signer_method"`
	SignatureHash   string    `json:"signature_hash"`
	SignedAt        time.Time `json:"signed_at"`
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
		return nil, fmt.Errorf("invalid doc_type %q", docType)
	}
	if len(steps) > 10 {
		return nil, errors.New("an approval chain is limited to 10 steps")
	}

	cleaned := make([]WorkflowStep, 0, len(steps))
	for i, step := range steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			return nil, fmt.Errorf("step %d needs a name", i+1)
		}
		cleaned = append(cleaned, WorkflowStep{
			Order:           i + 1,
			Name:            name,
			SignerRegNumber: strings.ToUpper(strings.TrimSpace(step.SignerRegNumber)),
		})
	}

	// The signature policy may insist that only a named signer approves this
	// type. Saving a chain that names nobody would then lock the type out, so the
	// two screens guard the same setting from both sides.
	policy, err := m.SignaturePolicyFor(ctx, tenantID, docType)
	if err != nil {
		return nil, err
	}
	if policy.RequireNamedSigner && !slices.ContainsFunc(cleaned, func(s WorkflowStep) bool { return s.SignerRegNumber != "" }) {
		return nil, fmt.Errorf("the signature policy for %s requires a named signer, so at least one step must carry a registration number", docType)
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin workflow update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
	return &DocumentWorkflow{DocType: docType, Steps: cleaned}, nil
}

// approvalChain reports how many signatures a type needs and which registration
// numbers its steps name. A type with no chain needs one signature and names
// nobody, so signing keeps working for a tenant that never opens the screen.
func (m *DocumentsModule) approvalChain(ctx context.Context, tenantID, docType string) (required int, named []string, err error) {
	rows, err := m.db.Query(ctx,
		`SELECT signer_reg_number FROM document_workflow_steps
		  WHERE tenant_id = $1 AND doc_type = $2 ORDER BY step_order`, tenantID, docType)
	if err != nil {
		return 0, nil, fmt.Errorf("query approval chain: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reg string
		if err := rows.Scan(&reg); err != nil {
			return 0, nil, fmt.Errorf("scan approval chain: %w", err)
		}
		required++
		if reg != "" {
			named = append(named, reg)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	if required == 0 {
		required = 1
	}
	return required, named, nil
}

// RouteDocument sends a draft for approval. Nothing the app creates is a draft
// today — CreateDocument writes PENDING_APPROVAL — so this exists for rows that
// arrived any other way, and for the day drafting is added.
func (m *DocumentsModule) RouteDocument(ctx context.Context, tenantID, docID string) (*Document, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotRoutable
	}

	tag, err := m.db.Exec(ctx,
		`UPDATE document_records SET status = $1
		  WHERE id = $2 AND tenant_id = $3 AND status = $4`,
		StatusPending, docID, tenantID, StatusDraft)
	if err != nil {
		return nil, fmt.Errorf("route document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotRoutable
	}
	return m.getDocument(ctx, tenantID, docID)
}

// ListSignatures returns a document's signature ledger, oldest first.
func (m *DocumentsModule) ListSignatures(ctx context.Context, tenantID, docID string) ([]AppliedSignature, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	rows, err := m.db.Query(ctx,
		`SELECT signer_name, signer_reg_number, signer_method, signature_hash, signed_at,
		        COALESCE(certificate_serial, ''), COALESCE(certificate_issuer, '')
		   FROM document_signatures
		  WHERE tenant_id = $1 AND document_id = $2 ORDER BY signed_at`, tenantID, docID)
	if err != nil {
		return nil, fmt.Errorf("query signatures: %w", err)
	}
	defer rows.Close()

	list := make([]AppliedSignature, 0)
	for rows.Next() {
		var sig AppliedSignature
		if err := rows.Scan(&sig.SignerName, &sig.SignerRegNumber, &sig.SignerMethod,
			&sig.SignatureHash, &sig.SignedAt,
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
		writeError(w, http.StatusBadRequest, err.Error())
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
