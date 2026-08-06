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

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
)

// SignaturePolicy says how a document type may be signed. Every type has an
// effective policy: a type nobody has configured allows both national channels
// and names no signer, which is how the app behaved before the table existed.
type SignaturePolicy struct {
	DocType            string     `json:"doc_type"`
	AllowEID           bool       `json:"allow_eid"`
	AllowDAN           bool       `json:"allow_dan"`
	RequireNamedSigner bool       `json:"require_named_signer"`
	Configured         bool       `json:"configured"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

// defaultSignaturePolicy is the policy a type falls back to while no row exists.
func defaultSignaturePolicy(docType string) SignaturePolicy {
	return SignaturePolicy{DocType: docType, AllowEID: true, AllowDAN: true}
}

func (p SignaturePolicy) allows(method string) bool {
	switch method {
	case SignerEID:
		return p.AllowEID
	case SignerDAN:
		return p.AllowDAN
	default:
		return false
	}
}

// SignaturePolicyFor reads the stored policy for a type, or the default when the
// tenant has not configured one.
func (m *DocumentsModule) SignaturePolicyFor(ctx context.Context, tenantID, docType string) (SignaturePolicy, error) {
	policy := defaultSignaturePolicy(docType)
	var updatedAt time.Time

	err := m.db.QueryRow(ctx,
		`SELECT allow_eid, allow_dan, require_named_signer, updated_at
		   FROM document_signature_policies
		  WHERE tenant_id = $1 AND doc_type = $2`, tenantID, docType).
		Scan(&policy.AllowEID, &policy.AllowDAN, &policy.RequireNamedSigner, &updatedAt)
	if err != nil {
		if isNoRows(err) {
			return policy, nil
		}
		return policy, fmt.Errorf("load signature policy: %w", err)
	}

	policy.Configured = true
	policy.UpdatedAt = &updatedAt
	return policy, nil
}

// ListSignaturePolicies returns the effective policy for every document type, so
// the screen shows what actually applies rather than only the rows that exist.
func (m *DocumentsModule) ListSignaturePolicies(ctx context.Context, tenantID string) ([]SignaturePolicy, error) {
	stored := map[string]SignaturePolicy{}

	rows, err := m.db.Query(ctx,
		`SELECT doc_type, allow_eid, allow_dan, require_named_signer, updated_at
		   FROM document_signature_policies WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query signature policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var policy SignaturePolicy
		var updatedAt time.Time
		if err := rows.Scan(&policy.DocType, &policy.AllowEID, &policy.AllowDAN,
			&policy.RequireNamedSigner, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan signature policy: %w", err)
		}
		policy.Configured = true
		policy.UpdatedAt = &updatedAt
		stored[policy.DocType] = policy
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	list := make([]SignaturePolicy, 0, len(DocTypes))
	for _, docType := range DocTypes {
		if policy, ok := stored[docType]; ok {
			list = append(list, policy)
			continue
		}
		list = append(list, defaultSignaturePolicy(docType))
	}
	return list, nil
}

// SaveSignaturePolicy upserts the policy for one document type.
func (m *DocumentsModule) SaveSignaturePolicy(ctx context.Context, tenantID string, policy SignaturePolicy) (*SignaturePolicy, error) {
	docType := strings.ToUpper(strings.TrimSpace(policy.DocType))
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("invalid doc_type %q", docType)
	}
	if !policy.AllowEID && !policy.AllowDAN {
		return nil, errors.New("a policy must allow at least one of E-ID or DAN, otherwise the type cannot be signed")
	}

	// Requiring a named signer while the approval chain names nobody would leave
	// the type unsignable by anyone. The two screens are one setting in practice,
	// so the check lives on both sides of it.
	if policy.RequireNamedSigner {
		_, named, err := m.approvalChain(ctx, tenantID, docType)
		if err != nil {
			return nil, err
		}
		if len(named) == 0 {
			return nil, fmt.Errorf("the approval chain for %s names no signer, so requiring a named signer would make the type unsignable", docType)
		}
	}

	saved := SignaturePolicy{DocType: docType, Configured: true}
	var updatedAt time.Time
	err := m.db.QueryRow(ctx,
		`INSERT INTO document_signature_policies
		        (tenant_id, doc_type, allow_eid, allow_dan, require_named_signer, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (tenant_id, doc_type) DO UPDATE
		    SET allow_eid = EXCLUDED.allow_eid,
		        allow_dan = EXCLUDED.allow_dan,
		        require_named_signer = EXCLUDED.require_named_signer,
		        updated_at = NOW()
		 RETURNING allow_eid, allow_dan, require_named_signer, updated_at`,
		tenantID, docType, policy.AllowEID, policy.AllowDAN, policy.RequireNamedSigner).
		Scan(&saved.AllowEID, &saved.AllowDAN, &saved.RequireNamedSigner, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("save signature policy: %w", err)
	}

	saved.UpdatedAt = &updatedAt

	audit.Record(ctx, tenantID, actorFor(ctx), "documents.signature_policy_changed", docType, map[string]any{
		"allow_eid":            saved.AllowEID,
		"allow_dan":            saved.AllowDAN,
		"require_named_signer": saved.RequireNamedSigner,
	})

	return &saved, nil
}

func (m *DocumentsModule) listSignaturePoliciesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.ListSignaturePolicies(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch signature policies")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (m *DocumentsModule) saveSignaturePolicyHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		AllowEID           bool `json:"allow_eid"`
		AllowDAN           bool `json:"allow_dan"`
		RequireNamedSigner bool `json:"require_named_signer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid signature policy payload")
		return
	}

	saved, err := m.SaveSignaturePolicy(r.Context(), tenantID, SignaturePolicy{
		DocType:            chi.URLParam(r, "docType"),
		AllowEID:           req.AllowEID,
		AllowDAN:           req.AllowDAN,
		RequireNamedSigner: req.RequireNamedSigner,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}
