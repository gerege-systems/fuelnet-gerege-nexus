package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	coreeid "github.com/gerege-systems/open-gerege-core/pkg/eid"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
)

// E-ID signing is not a form the platform fills in on the citizen's behalf. eID
// Mongolia has no document-signing endpoint: what it has is an approval a citizen
// gives on their own registered device, with their own credentials, against a
// display text we choose. That approval *is* the signature, so the ceremony is
//
//	start  — push "sign <document>" to the citizen's eID app
//	poll   — wait for them to approve, refuse, or let it expire
//
// and the signature is recorded the moment the approval comes back COMPLETE.
//
// The session is written to document_eid_sign_sessions before the citizen is
// troubled, because the pairing of session to document is what stops an approval
// given for one document from being presented as a signature on another. It is
// single-use for the same reason.

// ErrSignSessionUnknown is returned for a session this tenant did not start for
// this document. Whether it never existed, belongs elsewhere, or was already
// spent is deliberately not distinguished.
var ErrSignSessionUnknown = errors.New("signature session not found for this document")

// Approval states, as the caller sees them. They mirror eID's own vocabulary so a
// screen can report exactly what happened.
const (
	ApprovalRunning  = coreeid.StateRunning  // pushed, waiting for the citizen
	ApprovalComplete = coreeid.StateComplete // approved and signed
	ApprovalRefused  = coreeid.StateRefused  // the citizen declined
	ApprovalExpired  = coreeid.StateExpired  // nobody answered in time
)

// EIDSignSession is what the citizen's device needs in order to be found, plus
// what the operator reads out to them so both sides know it is the same request.
type EIDSignSession struct {
	SessionID        string `json:"session_id"`
	VerificationCode string `json:"verification_code"`
	ExpiresAt        string `json:"expires_at"`
	DeviceLinkURL    string `json:"device_link_url,omitempty"`
	// DisplayText is what the citizen is being shown. Returned so the screen can
	// display the same words rather than a paraphrase of them.
	DisplayText string `json:"display_text"`
}

// EIDSignProgress is one answer to "has the citizen approved yet?". Document is
// filled in only once the signature has been recorded.
type EIDSignProgress struct {
	State    string    `json:"state"`
	Document *Document `json:"document,omitempty"`
}

// signatureDisplayText is what the citizen reads on their phone. It names the
// document, because approving "Gerege ERP-д нэвтрэх" is not consent to sign a
// contract. eID limits the text, so a long title is cut rather than dropped.
func signatureDisplayText(title string) string {
	const limit = 120
	text := fmt.Sprintf("«%s» баримтад гарын үсэг зурах", strings.TrimSpace(title))
	if len(text) <= limit {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}

// StartEIDSignature pushes an approval request for this document to the citizen's
// eID app and remembers which document it was for.
func (m *DocumentsModule) StartEIDSignature(ctx context.Context, tenantID, docID, regNumber string) (*EIDSignSession, error) {
	pre, err := m.preflightSignature(ctx, tenantID, docID, SignerEID)
	if err != nil {
		return nil, err
	}

	regNumber = strings.ToUpper(strings.TrimSpace(regNumber))
	// Refusing here saves pushing a request that could never be turned into a
	// signature, and tells the operator why before the citizen is involved.
	if err := pre.checkNamedSigner(regNumber); err != nil {
		return nil, err
	}

	var title string
	if err := m.db.QueryRow(ctx,
		`SELECT title FROM document_records WHERE id = $1 AND tenant_id = $2`,
		docID, tenantID).Scan(&title); err != nil {
		if isNoRows(err) {
			return nil, ErrNotSignable
		}
		return nil, fmt.Errorf("read document title: %w", err)
	}

	displayText := signatureDisplayText(title)
	started, err := m.eidSvc.StartSignature(ctx, regNumber, displayText, "")
	if err != nil {
		return nil, fmt.Errorf("%w: E-ID could not reach the signer: %w", ErrSignatureRejected, err)
	}

	// An approval nobody ever answers is never polled again, so it would sit here
	// for good. Clearing this document's stale attempts as a new one starts keeps
	// the table bounded without a job to forget about.
	if _, err := m.db.Exec(ctx,
		`DELETE FROM document_eid_sign_sessions
		  WHERE tenant_id = $1 AND document_id = $2
		    AND consumed_at IS NULL AND created_at < NOW() - INTERVAL '1 hour'`,
		tenantID, docID); err != nil {
		return nil, fmt.Errorf("clear stale signature sessions: %w", err)
	}

	if _, err := m.db.Exec(ctx,
		`INSERT INTO document_eid_sign_sessions
		        (session_id, tenant_id, document_id, reg_number, display_text)
		 VALUES ($1, $2, $3, $4, $5)`,
		started.SessionID, tenantID, docID, regNumber, displayText); err != nil {
		return nil, fmt.Errorf("record signature session: %w", err)
	}

	// Who asked for the signature is not who gave it, and the ledger only records
	// the latter. This is the other half of the trail.
	audit.Record(ctx, tenantID, actorFor(ctx), "documents.signature_requested", docID, map[string]any{
		"signer_reg_number": regNumber,
		"display_text":      displayText,
	})

	return &EIDSignSession{
		SessionID:        started.SessionID,
		VerificationCode: started.VerificationCode,
		ExpiresAt:        started.ExpiresAt,
		DeviceLinkURL:    started.DeviceLinkURL,
		DisplayText:      displayText,
	}, nil
}

// PollEIDSignature asks eID whether the citizen has approved, and records the
// signature the moment they have. It is safe to call repeatedly: the session is
// spent on the first completion, and a spent session answers with the document it
// already produced rather than signing twice.
func (m *DocumentsModule) PollEIDSignature(ctx context.Context, tenantID, docID, sessionID string) (*EIDSignProgress, error) {
	if uuid.Validate(docID) != nil {
		return nil, ErrSignSessionUnknown
	}

	var regNumber string
	var consumed bool
	err := m.db.QueryRow(ctx,
		`SELECT reg_number, consumed_at IS NOT NULL
		   FROM document_eid_sign_sessions
		  WHERE session_id = $1 AND tenant_id = $2 AND document_id = $3`,
		sessionID, tenantID, docID).Scan(&regNumber, &consumed)
	if isNoRows(err) {
		return nil, ErrSignSessionUnknown
	}
	if err != nil {
		return nil, fmt.Errorf("load signature session: %w", err)
	}

	if consumed {
		doc, err := m.getDocument(ctx, tenantID, docID)
		if err != nil {
			return nil, err
		}
		return &EIDSignProgress{State: ApprovalComplete, Document: doc}, nil
	}

	result, err := m.eidSvc.Poll(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: E-ID could not be reached: %w", ErrSignatureRejected, err)
	}

	if result.State != ApprovalComplete {
		// A refused or expired request is over; dropping the row keeps a dead
		// session from being polled forever.
		if result.State == ApprovalRefused || result.State == ApprovalExpired {
			if _, err := m.db.Exec(ctx,
				`DELETE FROM document_eid_sign_sessions WHERE session_id = $1 AND tenant_id = $2`,
				sessionID, tenantID); err != nil {
				return nil, fmt.Errorf("clear finished signature session: %w", err)
			}
		}
		return &EIDSignProgress{State: result.State}, nil
	}

	if result.Identity == nil {
		return nil, fmt.Errorf("%w: E-ID reported an approval without an identity", ErrSignatureRejected)
	}

	// The approval has to come from the citizen the request was addressed to.
	approved := strings.ToUpper(strings.TrimSpace(result.Identity.RegNumber))
	if approved != regNumber {
		return nil, fmt.Errorf("%w: the request was sent to %s but %s approved it",
			ErrSignatureRejected, regNumber, approved)
	}

	// The policy and chain are re-read here rather than trusted from start time:
	// minutes may have passed, and it is this moment that decides.
	pre, err := m.preflightSignature(ctx, tenantID, docID, SignerEID)
	if err != nil {
		return nil, err
	}
	if err := pre.checkNamedSigner(approved); err != nil {
		return nil, err
	}

	identity := result.Identity
	signature := &verifiedSignature{
		SignerName:        strings.TrimSpace(identity.FirstName+" "+identity.LastName) + " (E-ID баталгаажсан)",
		RegNumber:         approved,
		Hash:              "eid_session_" + sessionID,
		CertificateSerial: identity.CertificateSerial,
		CertificateIssuer: identity.CertificateIssuer,
	}

	doc, err := m.recordSignature(ctx, tenantID, docID, SignerEID, pre.Required, signature)
	if err != nil {
		return nil, err
	}

	if _, err := m.db.Exec(ctx,
		`UPDATE document_eid_sign_sessions SET consumed_at = NOW()
		  WHERE session_id = $1 AND tenant_id = $2`, sessionID, tenantID); err != nil {
		return nil, fmt.Errorf("mark signature session spent: %w", err)
	}

	return &EIDSignProgress{State: ApprovalComplete, Document: doc}, nil
}

func (m *DocumentsModule) startEIDSignatureHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		RegNumber string `json:"reg_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.RegNumber) == "" {
		writeError(w, http.StatusBadRequest, "invalid signature request: reg_number is required")
		return
	}

	session, err := m.StartEIDSignature(r.Context(), tenantID, chi.URLParam(r, "id"), req.RegNumber)
	switch {
	case errors.Is(err, ErrNotSignable):
		writeError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, ErrSignatureRejected):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to start the signature request")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (m *DocumentsModule) pollEIDSignatureHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.SessionID) == "" {
		writeError(w, http.StatusBadRequest, "invalid poll request: session_id is required")
		return
	}

	progress, err := m.PollEIDSignature(r.Context(), tenantID, chi.URLParam(r, "id"), req.SessionID)
	switch {
	case errors.Is(err, ErrSignSessionUnknown):
		writeError(w, http.StatusNotFound, err.Error())
		return
	case errors.Is(err, ErrNotSignable), errors.Is(err, ErrAlreadySigned):
		writeError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, ErrSignatureRejected):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to complete the signature")
		return
	}
	writeJSON(w, http.StatusOK, progress)
}
