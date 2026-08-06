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
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/dan"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
)

// DocTypes enumerates the accepted document classifications. The column has no
// CHECK constraint, so validation happens here.
var DocTypes = []string{"CONTRACT", "REQUEST", "APPROVAL"}

// Document status lifecycle: PENDING_APPROVAL -> APPROVED (via e-signature) or
// REJECTED. A document can only be signed or rejected while it is pending.
//
// document_records.status still defaults to DRAFT at the column level, but
// CreateDocument always writes PENDING_APPROVAL, so nothing this app creates is
// ever a draft. Whoever builds the "Document workflows" screen owns the
// DRAFT -> PENDING_APPROVAL transition; until then DRAFT only reaches the table
// through direct SQL.
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

// ErrSignatureRejected is returned when the signer could not be verified: a
// registration number or one-time code the identity provider refused, or a
// channel this module does not speak. It is the caller's problem to fix, so the
// wrapped message is safe to show them — unlike a storage failure, which is
// reported as a plain server error.
var ErrSignatureRejected = errors.New("signature rejected")

type Document struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	Title           string     `json:"title"`
	DocType         string     `json:"doc_type"` // CONTRACT, REQUEST, APPROVAL
	Status          string     `json:"status"`   // DRAFT, PENDING_APPROVAL, APPROVED, REJECTED
	SignedBy        string     `json:"signed_by,omitempty"`
	SignatureHash   string     `json:"signature_hash,omitempty"`
	SignerRegNumber string     `json:"signer_reg_number,omitempty"`
	SignerMethod    string     `json:"signer_method,omitempty"`
	SignedAt        *time.Time `json:"signed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
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
		{Code: "documents.manage", Name: "Manage Documents", Description: "Create documents and route them for approval"},
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
		dr.Post("/{id}/sign", m.signDocumentHandler)
		dr.Post("/{id}/reject", m.rejectDocumentHandler)
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, doc)
}

func (m *DocumentsModule) signDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	docID := chi.URLParam(r, "id")
	var req struct {
		Method    string `json:"method"`     // "EID" or "DAN"
		RegNumber string `json:"reg_number"` // Регистрийн дугаар
		OTPCode   string `json:"otp_code"`   // Нэг удаагийн нууц код
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RegNumber == "" {
		writeError(w, http.StatusBadRequest, "invalid signature request: reg_number is required")
		return
	}

	doc, err := m.SignDocument(r.Context(), tenantID, docID, req.Method, req.RegNumber, req.OTPCode)
	switch {
	case errors.Is(err, ErrNotSignable):
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

func (m *DocumentsModule) CreateDocument(ctx context.Context, tenantID, title, docType string) (*Document, error) {
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}
	if docType == "" {
		docType = "CONTRACT"
	}
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("unsupported doc_type %q", docType)
	}

	var doc Document
	const query = `
		INSERT INTO document_records (tenant_id, title, doc_type, status)
		VALUES ($1, $2, $3, 'PENDING_APPROVAL')
		RETURNING id, tenant_id, title, doc_type, status, created_at
	`
	// The old code answered a failed INSERT with a canned "doc_demo_200"
	// record, reporting success for a document that was never stored.
	err := m.db.QueryRow(ctx, query, tenantID, title, docType).Scan(
		&doc.ID, &doc.TenantID, &doc.Title, &doc.DocType, &doc.Status, &doc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return &doc, nil
}

// SignDocument verifies the signer through the requested national identity
// channel (E-ID or DAN) and, on success, records the digital signature and
// moves the document to APPROVED. The status guard in the UPDATE makes this
// idempotent-safe: a document that is not pending is never re-signed.
func (m *DocumentsModule) SignDocument(ctx context.Context, tenantID, docID, method, regNumber, otpCode string) (*Document, error) {
	// document_records.id is a uuid column, so a malformed path parameter would
	// otherwise surface as a storage error. It is simply not a document we hold.
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = SignerEID
	}

	var signedBy, sigHash, signerReg string
	switch method {
	case SignerEID:
		identity, err := m.eidSvc.AuthenticateWithMethod(ctx, regNumber, otpCode, eid.AuthMethodMobileOTP)
		if err != nil {
			return nil, fmt.Errorf("%w: E-ID verification failed: %w", ErrSignatureRejected, err)
		}
		signedBy = strings.TrimSpace(identity.FirstName+" "+identity.LastName) + " (E-ID баталгаажсан)"
		sigHash = identity.SignatureHash
		signerReg = identity.RegNumber
	case SignerDAN:
		profile, err := m.danSvc.AuthenticateDANCitizen(ctx, regNumber, otpCode)
		if err != nil {
			return nil, fmt.Errorf("%w: DAN verification failed: %w", ErrSignatureRejected, err)
		}
		signedBy = strings.TrimSpace(profile.FirstName+" "+profile.LastName) + " (DAN баталгаажсан)"
		sigHash = "dan_sig_" + profile.DANSessionID
		signerReg = profile.RegNumber
	default:
		return nil, fmt.Errorf("%w: unsupported signer method %q (expected EID or DAN)", ErrSignatureRejected, method)
	}

	signedAt := time.Now()
	const query = `
		UPDATE document_records
		   SET status = 'APPROVED', signed_by = $1, signature_hash = $2,
		       signer_reg_number = $3, signer_method = $4, signed_at = $5
		 WHERE id = $6 AND tenant_id = $7 AND status = 'PENDING_APPROVAL'
		RETURNING id, tenant_id, title, doc_type, status,
		          COALESCE(signed_by, ''), COALESCE(signature_hash, ''),
		          COALESCE(signer_reg_number, ''), COALESCE(signer_method, ''),
		          signed_at, created_at
	`
	doc, err := m.scanDocument(m.db.QueryRow(ctx, query,
		signedBy, sigHash, signerReg, method, signedAt, docID, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotSignable
	}
	if err != nil {
		return nil, fmt.Errorf("sign document: %w", err)
	}
	return doc, nil
}

// RejectDocument moves a pending document to REJECTED. Like signing, it is a
// no-op on anything that is not currently pending.
func (m *DocumentsModule) RejectDocument(ctx context.Context, tenantID, docID string) (*Document, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrNotSignable
	}

	const query = `
		UPDATE document_records
		   SET status = 'REJECTED'
		 WHERE id = $1 AND tenant_id = $2 AND status = 'PENDING_APPROVAL'
		RETURNING id, tenant_id, title, doc_type, status,
		          COALESCE(signed_by, ''), COALESCE(signature_hash, ''),
		          COALESCE(signer_reg_number, ''), COALESCE(signer_method, ''),
		          signed_at, created_at
	`
	doc, err := m.scanDocument(m.db.QueryRow(ctx, query, docID, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotSignable
	}
	if err != nil {
		return nil, fmt.Errorf("reject document: %w", err)
	}
	return doc, nil
}

func (m *DocumentsModule) ListDocuments(ctx context.Context, tenantID string) ([]Document, error) {
	const query = `SELECT id, tenant_id, title, doc_type, status,
	                      COALESCE(signed_by, ''), COALESCE(signature_hash, ''),
	                      COALESCE(signer_reg_number, ''), COALESCE(signer_method, ''),
	                      signed_at, created_at
	                 FROM document_records WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := m.db.Query(ctx, query, tenantID)
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
		&doc.SignedAt, &doc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
