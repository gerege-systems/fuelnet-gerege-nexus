/*
 * Gerege Template Platform
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package documents implements Digital Documents & E-Signatures Go module (io.example.documents).
 */

package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/dan"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
)

// DocTypes enumerates the accepted document classifications. The column has no
// CHECK constraint, so validation happens here.
var DocTypes = []string{"CONTRACT", "REQUEST", "APPROVAL"}

// Document status lifecycle: DRAFT -> PENDING_APPROVAL (RouteDocument) ->
// APPROVED once the approval chain is complete, or REJECTED. A document can only
// be signed or rejected while it is pending.
//
// CreateDocument writes PENDING_APPROVAL directly, so nothing the app creates is
// a draft; DRAFT is the column default and RouteDocument is what moves a row that
// arrived any other way.
const (
	StatusDraft    = "DRAFT"
	StatusPending  = "PENDING_APPROVAL"
	StatusApproved = "APPROVED"
	StatusRejected = "REJECTED"
)

// SignerMethod is the national identity channel used to apply the signature.
const (
	SignerEID = "EID" // eidmongolia.mn / sso.gov.mn digital signature
	SignerDAN = "DAN" // dan.gerege.mn SSO gateway
)

// ErrNotSignable is returned when a sign/reject is attempted on a document that
// is missing, belongs to another tenant, or is no longer pending. The three
// cases are deliberately indistinguishable so the endpoint cannot be used to
// discover which documents exist in someone else's tenant.
var ErrNotSignable = errors.New("document not found or is not awaiting approval")

// ErrSignatureRejected is returned when the signature was refused: an identity
// the provider would not verify, a channel the type's policy does not allow, or a
// signer its approval chain does not name. It is the caller's problem to fix, so
// the wrapped message is safe to show them — unlike a storage failure, which is
// reported as a plain server error.
var ErrSignatureRejected = errors.New("signature rejected")

// ErrInvalidDocument is returned when what the caller sent cannot make a
// document: an empty or over-long title, an unknown type. It is theirs to fix, so
// the message is safe to show them — a storage failure is not, and is reported as
// a plain server error instead.
var ErrInvalidDocument = errors.New("invalid document")

// ErrInvalidConfiguration is returned when a template, approval chain, signature
// policy or retention rule the caller sent cannot be stored as asked. Like
// ErrInvalidDocument its message is meant to be read by whoever sent it.
var ErrInvalidConfiguration = errors.New("invalid configuration")

// ErrProviderUnavailable is returned when the identity provider could not be
// reached or answered in a way we could not use. It is nobody's mistake and it may
// work in a moment, so it is reported as a temporary server-side failure — a
// polling client must retry it rather than abandon a ceremony the citizen may be
// about to complete.
var ErrProviderUnavailable = errors.New("identity provider unavailable")

type Document struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Title    string `json:"title"`
	DocType  string `json:"doc_type"` // CONTRACT, REQUEST, APPROVAL
	Status   string `json:"status"`   // DRAFT, PENDING_APPROVAL, APPROVED, REJECTED
	// The newest signature, mirrored onto the record so the list stays one query.
	// The full history is in document_signatures.
	SignedBy        string     `json:"signed_by,omitempty"`
	SignatureHash   string     `json:"signature_hash,omitempty"`
	SignerRegNumber string     `json:"signer_reg_number,omitempty"`
	SignerMethod    string     `json:"signer_method,omitempty"`
	SignedAt        *time.Time `json:"signed_at,omitempty"`
	// Progress through the approval chain: how many signatures the document carries
	// and how many it asked for.
	SignatureCount     int `json:"signature_count"`
	RequiredSignatures int `json:"required_signatures"`
	// OutstandingSteps is how many steps of the document's own chain no signature has
	// filled. Completion needs this to be zero as well as the count to be met, so a
	// screen that only compared the two numbers could paint "complete" over a document
	// whose named step was still owed — a signature from somebody no step names counts
	// toward the total and satisfies none of the chain.
	OutstandingSteps int       `json:"outstanding_steps"`
	CreatedAt        time.Time `json:"created_at"`
}

type DocumentsModule struct {
	db     *pgxpool.Pool
	eidSvc *eid.EIDService
	danSvc *dan.DANService
}

// New builds the module and registers it in the compile-time app registry so
// the app store can resolve and install io.example.documents. The module owns
// its own E-ID and DAN clients so signing stays self-contained; both read their
// configuration from the environment and default to mock mode.
func New(db *pgxpool.Pool) *DocumentsModule {
	m := &DocumentsModule{
		db:     db,
		eidSvc: eid.NewEIDService(),
		danSvc: dan.NewDANService(),
	}
	appregistry.Register(m)
	return m
}

func (m *DocumentsModule) ID() string      { return "io.example.documents" }
func (m *DocumentsModule) Name() string    { return "Digital Documents & Signatures" }
func (m *DocumentsModule) Version() string { return "1.0.0" }

func (m *DocumentsModule) Dependencies() []internal.Dependency { return nil }

func (m *DocumentsModule) Permissions() []internal.PermissionDefinition {
	return []internal.PermissionDefinition{
		{Code: "documents.read", Name: "Read Documents", Description: "View documents and signature status"},
		{Code: "documents.manage", Name: "Manage Documents", Description: "Create documents, route them for approval, and configure templates, approval chains, signature policies and retention rules"},
		{Code: "documents.sign", Name: "Sign Documents", Description: "Apply an E-ID / DAN digital signature or reject a document"},
	}
}

func (m *DocumentsModule) Menus() []internal.MenuDefinition {
	return []internal.MenuDefinition{
		{ID: "documents", ParentID: "operations", Label: "Documents & E-Sign", Path: "/documents", Icon: "file-text", Order: 30, Labels: map[string]string{"mn": "Баримт ба цахим гарын үсэг"}},
	}
}

func (m *DocumentsModule) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/documents", func(dr chi.Router) {
		dr.Use(tenantAuthMiddleware)
		dr.Get("/", m.listDocumentsHandler)
		dr.Post("/", m.createDocumentHandler)

		// Templates a document is started from.
		dr.Get("/templates", m.listTemplatesHandler)
		dr.Post("/templates", m.createTemplateHandler)
		dr.Put("/templates/{id}", m.updateTemplateHandler)
		dr.Delete("/templates/{id}", m.deleteTemplateHandler)
		dr.Post("/templates/{id}/use", m.useTemplateHandler)

		// How a document type may be signed.
		dr.Get("/policies", m.listSignaturePoliciesHandler)
		dr.Put("/policies/{docType}", m.saveSignaturePolicyHandler)

		// Who must sign it, in order.
		dr.Get("/workflows", m.listWorkflowsHandler)
		dr.Put("/workflows/{docType}", m.saveWorkflowHandler)

		// How long it is kept.
		dr.Get("/retention", m.listRetentionRulesHandler)
		dr.Put("/retention/{docType}", m.saveRetentionRuleHandler)

		// A single document. Static segments above win over {id} in chi's trie,
		// so "templates" and "policies" are never read as document ids.
		dr.Get("/{id}/signatures", m.listSignaturesHandler)
		dr.Get("/{id}/steps", m.listDocumentStepsHandler)
		dr.Post("/{id}/route", m.routeDocumentHandler)
		dr.Post("/{id}/reject", m.rejectDocumentHandler)

		// Signing is per channel, because the channels are not the same shape.
		// E-ID is an approval the citizen gives on their own device, so it takes
		// two calls; DAN is a code they read out, so it takes one.
		dr.Post("/{id}/sign/eid/start", m.startEIDSignatureHandler)
		dr.Post("/{id}/sign/eid/poll", m.pollEIDSignatureHandler)
		dr.Post("/{id}/sign/dan", m.signWithDANHandler)
	})
}

func (m *DocumentsModule) listDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.ListDocuments(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch documents")
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (m *DocumentsModule) createDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Title   string `json:"title"`
		DocType string `json:"doc_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeError(w, http.StatusBadRequest, "invalid document parameters: title is required")
		return
	}

	doc, err := m.CreateDocument(r.Context(), tenantID, req.Title, req.DocType)
	if err != nil {
		writeWriteFailure(r.Context(), w, err, "failed to create document")
		return
	}

	writeJSON(w, http.StatusCreated, doc)
}

func (m *DocumentsModule) signWithDANHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	docID := chi.URLParam(r, "id")
	var req struct {
		RegNumber string `json:"reg_number"` // Регистрийн дугаар
		OTPCode   string `json:"otp_code"`   // Нэг удаагийн нууц код
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RegNumber == "" {
		writeError(w, http.StatusBadRequest, "invalid signature request: reg_number is required")
		return
	}

	doc, err := m.SignWithDAN(r.Context(), tenantID, docID, req.RegNumber, req.OTPCode)
	switch {
	case errors.Is(err, ErrNotSignable):
		writeError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, ErrAlreadySigned):
		writeError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, ErrSignatureRejected):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		// A storage failure is ours, not the caller's: report it as one and keep
		// the driver's message out of the response.
		writeError(w, http.StatusInternalServerError, "failed to sign document")
		return
	}

	writeJSON(w, http.StatusOK, doc)
}

func (m *DocumentsModule) rejectDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	docID := chi.URLParam(r, "id")
	doc, err := m.RejectDocument(r.Context(), tenantID, docID)
	switch {
	case errors.Is(err, ErrNotSignable):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to reject document")
		return
	}

	writeJSON(w, http.StatusOK, doc)
}

// TitleLimit is what document_records.title holds. Checking it here turns a
// storage error the caller cannot read into one they can act on.
const TitleLimit = 255

func (m *DocumentsModule) CreateDocument(ctx context.Context, tenantID, title, docType string) (*Document, error) {
	title = strings.TrimSpace(title)
	docType = strings.ToUpper(strings.TrimSpace(docType))

	if title == "" {
		return nil, fmt.Errorf("%w: title cannot be empty", ErrInvalidDocument)
	}
	if len([]rune(title)) > TitleLimit {
		return nil, fmt.Errorf("%w: a title is at most %d characters, this one is %d",
			ErrInvalidDocument, TitleLimit, len([]rune(title)))
	}
	if docType == "" {
		docType = DocTypes[0]
	}
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("%w: unsupported doc_type %q", ErrInvalidDocument, docType)
	}

	// A document and the chain it will be approved under are created together: a
	// document that started waiting keeps the rules it started under, whatever the
	// type's chain becomes later.
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create document: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The old code answered a failed INSERT with a canned "doc_demo_200"
	// record, reporting success for a document that was never stored.
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO document_records (tenant_id, title, doc_type, status)
		      VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, title, docType, StatusPending).Scan(&id); err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	if err := m.snapshotApprovalChain(ctx, tx, tenantID, id, docType); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create document: %w", err)
	}
	return m.getDocument(ctx, tenantID, id)
}

// verifiedSignature is what a national identity channel hands back once it has
// vouched for the signer.
type verifiedSignature struct {
	SignerName string
	RegNumber  string
	// Hash references the act of approval: the DAN session for DAN, the eID
	// session for E-ID.
	Hash string
	// CertificateSerial and CertificateIssuer are the citizen's eID certificate,
	// when the provider returned one. They say whose key gave the approval, which
	// is what a dispute turns on; a session id alone only says one happened.
	CertificateSerial string
	CertificateIssuer string
}

// signaturePreflight is what the document's own configuration decides before a
// citizen is asked to approve anything.
type signaturePreflight struct {
	DocType  string
	Policy   SignaturePolicy
	Required int
	Position *approvalPosition
}

// preflightSignature checks that the document is still awaiting approval, that
// the tenant's policy allows this channel, and which step of the document's own
// chain comes next. Both signing paths run it: E-ID before pushing a request to
// somebody's phone, DAN before asking for a code — there is no point troubling a
// citizen for a document their signature cannot land on.
func (m *DocumentsModule) preflightSignature(ctx context.Context, tenantID, docID, method string) (*signaturePreflight, error) {
	// document_records.id is a uuid column, so a malformed path parameter would
	// otherwise surface as a storage error. It is simply not a document we hold.
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	var docType string
	var required int
	err := m.db.QueryRow(ctx,
		`SELECT doc_type, required_signatures FROM document_records
		  WHERE id = $1 AND tenant_id = $2 AND status = $3`,
		docID, tenantID, StatusPending).Scan(&docType, &required)
	if isNoRows(err) {
		return nil, ErrNotSignable
	}
	if err != nil {
		return nil, fmt.Errorf("load document for signing: %w", err)
	}

	policy, err := m.SignaturePolicyFor(ctx, tenantID, docType)
	if err != nil {
		return nil, err
	}
	if !policy.allows(method) {
		return nil, fmt.Errorf("%w: %s documents may not be signed through %s under this tenant's policy",
			ErrSignatureRejected, docType, method)
	}

	position, err := m.nextApprovalStep(ctx, m.db, tenantID, docID)
	if err != nil {
		return nil, err
	}
	return &signaturePreflight{DocType: docType, Policy: policy, Required: required, Position: position}, nil
}

// alreadySigned reports whether this citizen has already signed this document. The
// unique constraint would catch it either way, but only after the citizen had been
// asked to approve on their own device and their approval discarded.
func (m *DocumentsModule) alreadySigned(ctx context.Context, q querier, tenantID, docID, regNumber string) (bool, error) {
	var signed bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM document_signatures
		                 WHERE tenant_id = $1 AND document_id = $2 AND signer_reg_number = $3)`,
		tenantID, docID, regNumber).Scan(&signed); err != nil {
		return false, fmt.Errorf("check whether the signer already signed: %w", err)
	}
	return signed, nil
}

// checkSigner holds the signature to whoever the document's next step names, and
// holds back anyone a later step names. The check runs against the identity a
// provider vouched for, never against what the caller typed.
func checkSigner(pos *approvalPosition, docType, regNumber string) error {
	if pos == nil || pos.Next == nil {
		return nil // no chain: one open signature approves
	}

	if named := pos.Next.SignerRegNumber; named != "" {
		if named == regNumber {
			return nil
		}
		return fmt.Errorf("%w: step %d of the %s chain (%s) is reserved for %s, not %s",
			ErrSignatureRejected, pos.Next.Order, docType, pos.Next.Name, named, regNumber)
	}

	// The step is open — but not to somebody the chain still needs further along.
	// A citizen signs a document once, so letting them spend their signature here
	// would leave their own step unfillable and the document unapprovable.
	if slices.Contains(pos.Reserved, regNumber) {
		return fmt.Errorf("%w: %s is named by a later step of the %s chain, so their signature is held for it — step %d (%s) is open to anyone else",
			ErrSignatureRejected, regNumber, docType, pos.Next.Order, pos.Next.Name)
	}
	return nil
}

// SignWithDAN signs through dan.gerege.mn's registration-number and one-time-code
// gateway. DAN exposes no approval-push flow in this platform, so unlike E-ID it
// is still a single call — and it only succeeds while DAN_MOCK_MODE is on, because
// the live DAN client has no implementation for it.
func (m *DocumentsModule) SignWithDAN(ctx context.Context, tenantID, docID, regNumber, otpCode string) (*Document, error) {
	pre, err := m.preflightSignature(ctx, tenantID, docID, SignerDAN)
	if err != nil {
		return nil, err
	}

	profile, err := m.danSvc.AuthenticateDANCitizen(ctx, regNumber, otpCode)
	if err != nil {
		return nil, fmt.Errorf("%w: DAN verification failed: %w", ErrSignatureRejected, err)
	}
	signature := &verifiedSignature{
		SignerName: strings.TrimSpace(profile.FirstName+" "+profile.LastName) + " (DAN баталгаажсан)",
		RegNumber:  profile.RegNumber,
		Hash:       "dan_sig_" + profile.DANSessionID,
	}

	if err := checkSigner(pre.Position, pre.DocType, signature.RegNumber); err != nil {
		return nil, err
	}
	signed, err := m.alreadySigned(ctx, m.db, tenantID, docID, signature.RegNumber)
	if err != nil {
		return nil, err
	}
	if signed {
		return nil, ErrAlreadySigned
	}
	return m.recordSignature(ctx, tenantID, docID, SignerDAN, signature, "")
}

// recordSignature writes the signature and decides whether the chain is complete.
//
// Everything the decision rests on is read inside one transaction with the
// document row locked: the requirement from the row itself, the count of
// signatures already applied, and which step comes next. An earlier version took
// the requirement from a pool read before the lock, which let a chain edited in
// between approve a document on fewer signatures than it asked for.
//
// sessionID, when given, is the eID session this signature came from; it is
// marked spent in the same transaction so a recorded signature and an unspent
// session can never disagree.
func (m *DocumentsModule) recordSignature(ctx context.Context, tenantID, docID, method string, signature *verifiedSignature, sessionID string) (*Document, error) {
	// The citizen has already approved by the time we get here, so an operator
	// closing the dialog must not destroy their signature. The write runs on a
	// context the caller cannot cancel — with its own deadline, so a stalled
	// database still lets go.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin signature: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var docType string
	var required int
	err = tx.QueryRow(ctx,
		`SELECT doc_type, required_signatures FROM document_records
		  WHERE id = $1 AND tenant_id = $2 AND status = $3 FOR UPDATE`,
		docID, tenantID, StatusPending).Scan(&docType, &required)
	if isNoRows(err) {
		return nil, ErrNotSignable
	}
	if err != nil {
		return nil, fmt.Errorf("lock document for signing: %w", err)
	}

	position, err := m.nextApprovalStep(ctx, tx, tenantID, docID)
	if err != nil {
		return nil, err
	}
	// The authority check that counts is this one: the preflight's was for a good
	// error message before a citizen was troubled, this one is under the lock.
	if err := checkSigner(position, docType, signature.RegNumber); err != nil {
		return nil, err
	}

	// The signature fills the step it was checked against — not "the next number",
	// which is not the same thing once a signature can sit on a later step than an
	// unfilled earlier one.
	step := position.Applied + 1
	if position.Next != nil {
		step = position.Next.Order
	}
	signedAt := time.Now()
	_, err = tx.Exec(ctx,
		`INSERT INTO document_signatures
		        (tenant_id, document_id, signer_name, signer_reg_number,
		         signer_method, signature_hash, signed_at,
		         certificate_serial, certificate_issuer, step_order)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''), $10)`,
		tenantID, docID, signature.SignerName, signature.RegNumber,
		method, signature.Hash, signedAt,
		signature.CertificateSerial, signature.CertificateIssuer, step)
	if isUniqueViolation(err) {
		return nil, ErrAlreadySigned
	}
	if err != nil {
		return nil, fmt.Errorf("record signature: %w", err)
	}

	// Complete means every step of the document's own chain is filled AND it carries
	// at least as many signatures as it asked for. A document with no chain has no
	// steps to fill, so the count alone decides — which is how a type without a
	// chain has always behaved.
	left, err := m.unfilledSteps(ctx, tx, docID)
	if err != nil {
		return nil, err
	}
	status := StatusPending
	if left == 0 && position.Applied+1 >= required {
		status = StatusApproved
	}

	if _, err := tx.Exec(ctx,
		`UPDATE document_records
		    SET status = $1, signed_by = $2, signature_hash = $3,
		        signer_reg_number = $4, signer_method = $5, signed_at = $6
		  WHERE id = $7 AND tenant_id = $8`,
		status, signature.SignerName, signature.Hash,
		signature.RegNumber, method, signedAt, docID, tenantID); err != nil {
		return nil, fmt.Errorf("apply signature to document: %w", err)
	}

	if sessionID != "" {
		if _, err := tx.Exec(ctx,
			`UPDATE document_eid_sign_sessions SET consumed_at = NOW()
			  WHERE session_id = $1 AND tenant_id = $2`, sessionID, tenantID); err != nil {
			return nil, fmt.Errorf("mark signature session spent: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit signature: %w", err)
	}

	// Both channels land here, so one record covers every signature the module
	// applies. It carries what the signature was, not only that one happened:
	// which citizen, through which channel, on whose certificate, which step of
	// the chain it filled, and whether it completed it.
	audit.Record(ctx, tenantID, actorFor(ctx), "documents.signed", docID, map[string]any{
		"method":              method,
		"signer_reg_number":   signature.RegNumber,
		"certificate_serial":  signature.CertificateSerial,
		"certificate_issuer":  signature.CertificateIssuer,
		"step_order":          step,
		"signatures_required": required,
		"status":              status,
	})

	return m.getDocument(ctx, tenantID, docID)
}

// RejectDocument moves a pending document to REJECTED. Like signing, it is a
// no-op on anything that is not currently pending.
func (m *DocumentsModule) RejectDocument(ctx context.Context, tenantID, docID string) (*Document, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	tag, err := m.db.Exec(ctx,
		`UPDATE document_records SET status = $1
		  WHERE id = $2 AND tenant_id = $3 AND status = $4`,
		StatusRejected, docID, tenantID, StatusPending)
	if err != nil {
		return nil, fmt.Errorf("reject document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotSignable
	}

	audit.Record(ctx, tenantID, actorFor(ctx), "documents.rejected", docID, nil)

	return m.getDocument(ctx, tenantID, docID)
}

// documentColumns is the shape every document read returns. The progress is how
// many signatures the document carries against how many it asked for — read from
// the document's own row, not counted from the type's current chain, so a later
// configuration change cannot rewrite what a decided document required.
const documentColumns = `
	d.id, d.tenant_id, d.title, d.doc_type, d.status,
	COALESCE(d.signed_by, ''), COALESCE(d.signature_hash, ''),
	COALESCE(d.signer_reg_number, ''), COALESCE(d.signer_method, ''),
	d.signed_at, d.created_at,
	(SELECT count(*) FROM document_signatures s WHERE s.document_id = d.id),
	GREATEST(1, d.required_signatures),
	(SELECT count(*) FROM document_approval_steps st
	  WHERE st.document_id = d.id
	    AND NOT EXISTS (SELECT 1 FROM document_signatures s
	                     WHERE s.document_id = st.document_id
	                       AND s.step_order = st.step_order))`

func (m *DocumentsModule) ListDocuments(ctx context.Context, tenantID string) ([]Document, error) {
	rows, err := m.db.Query(ctx,
		`SELECT `+documentColumns+`
		   FROM document_records d
		  WHERE d.tenant_id = $1 ORDER BY d.created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	list := make([]Document, 0)
	for rows.Next() {
		doc, err := m.scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		list = append(list, *doc)
	}
	return list, rows.Err()
}

// getDocument re-reads one document through the same column list as the list, so
// a write endpoint answers with exactly what a later refresh will show.
func (m *DocumentsModule) getDocument(ctx context.Context, tenantID, docID string) (*Document, error) {
	doc, err := m.scanDocument(m.db.QueryRow(ctx,
		`SELECT `+documentColumns+`
		   FROM document_records d
		  WHERE d.tenant_id = $1 AND d.id = $2`, tenantID, docID))
	if isNoRows(err) {
		return nil, ErrNotSignable
	}
	if err != nil {
		return nil, fmt.Errorf("read document: %w", err)
	}
	return doc, nil
}

// rowScanner is satisfied by both pgx.Row (QueryRow) and pgx.Rows (Query), so
// the column layout lives in exactly one place.
type rowScanner interface {
	Scan(dest ...any) error
}

func (m *DocumentsModule) scanDocument(row rowScanner) (*Document, error) {
	var doc Document
	err := row.Scan(
		&doc.ID, &doc.TenantID, &doc.Title, &doc.DocType, &doc.Status,
		&doc.SignedBy, &doc.SignatureHash, &doc.SignerRegNumber, &doc.SignerMethod,
		&doc.SignedAt, &doc.CreatedAt, &doc.SignatureCount, &doc.RequiredSignatures,
		&doc.OutstandingSteps,
	)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// actorFor names who did it in the audit log, preferring the email an
// administrator reading the log would recognise. Calls that arrive outside a
// request — a migration, a test — have no claims and are recorded as the system.
func actorFor(ctx context.Context) string {
	claims, err := auth.UserFromContext(ctx)
	if err != nil {
		return "system"
	}
	if claims.Email != "" {
		return claims.Email
	}
	return claims.UserID
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// isUniqueViolation reports a Postgres 23505, which is how a duplicate reaches Go.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeWriteFailure sorts a failed write into the class its caller can act on:
// what they sent (400 with the reason), a collision with somebody else's change
// (409, retryable), or ours (500 with a fixed message, the driver's own text
// logged rather than returned). Answering everything with 400 and the wrapped
// error told operators their request was invalid when the database was down, and
// put connection strings and SQLSTATEs on screen.
func writeWriteFailure(ctx context.Context, w http.ResponseWriter, err error, whatFailed string) {
	switch {
	case errors.Is(err, ErrInvalidDocument), errors.Is(err, ErrInvalidConfiguration):
		writeError(w, http.StatusBadRequest, err.Error())
	case isUniqueViolation(err):
		writeError(w, http.StatusConflict, "this was changed by someone else at the same time — reload and try again")
	default:
		slog.ErrorContext(ctx, whatFailed, "error", err)
		writeError(w, http.StatusInternalServerError, whatFailed)
	}
}
