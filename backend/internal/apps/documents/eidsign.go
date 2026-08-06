package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

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
// The provider mints the session id, so the pairing of session to document can
// only be written after the push. It is written with a context that outlives the
// caller's, because that pairing is what stops an approval given for one document
// from being presented as a signature on another — a citizen must never be able to
// approve something nothing can attach their signature to. It is single-use, and
// it is spent in the same transaction as the signature it produces.

// signSessionGrace is how far past eID's own deadline a session is still
// accepted, so a small clock difference between us and the provider does not
// throw away an approval the citizen gave in time.
const signSessionGrace = 2 * time.Minute

// parseExpiry reads the provider's deadline. eID returns RFC3339; anything it
// cannot parse falls back to the two minutes an eID request is normally given,
// which is safer than storing nothing and never expiring the session.
func parseExpiry(value string) time.Time {
	if at, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return at
	}
	return time.Now().Add(2 * time.Minute)
}

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

// eID shows the display text as Smart-ID's displayText60, and the core client
// truncates it to 60 BYTES with a raw slice (pkg/eid newAuthBody). Cyrillic is two
// bytes a letter, so a Mongolian sentence is cut at about half its length and can
// be cut mid-letter, leaving invalid UTF-8 in the request body.
//
// Two consequences shape signatureDisplayText: the purpose has to come FIRST,
// because it is the tail that gets lost, and the cut has to be ours, on rune
// boundaries, so core's byte slice never fires. An earlier version put the title
// first and budgeted 120 — so the words saying a signature was being given were
// exactly what disappeared, and the citizen was asked to approve a bare fragment
// of a title.
const eidDisplayTextBytes = 60

// signatureDisplayText is what the citizen reads on their own device. It names the
// document, because approving "Gerege ERP-д нэвтрэх" is not consent to sign a
// contract.
func signatureDisplayText(title string) string {
	const prefix = "Гарын үсэг: "
	const ellipsis = "…"

	title = strings.TrimSpace(title)
	if title == "" {
		return "Баримтад гарын үсэг зурах"
	}
	if len(prefix)+len(title) <= eidDisplayTextBytes {
		return prefix + title
	}

	budget := eidDisplayTextBytes - len(prefix) - len(ellipsis)
	if budget <= 0 {
		// Cannot happen with the prefix above, but a shorter budget than the
		// marker must not produce a negative slice.
		return prefix
	}

	var b strings.Builder
	b.WriteString(prefix)
	used := 0
	for _, r := range title {
		size := utf8.RuneLen(r)
		if used+size > budget {
			break
		}
		b.WriteRune(r)
		used += size
	}
	b.WriteString(ellipsis)
	return b.String()
}

// StartEIDSignature pushes an approval request for this document to the citizen's
// eID app and remembers which document it was for.
func (m *DocumentsModule) StartEIDSignature(ctx context.Context, tenantID, docID, regNumber string) (*EIDSignSession, error) {
	// What this module can see is wrong is refused before any work is done on the
	// caller's behalf — and so that anything the provider refuses later can be
	// treated as the provider's trouble, which a polling client must retry rather
	// than abandon a ceremony over.
	regNumber = strings.ToUpper(strings.TrimSpace(regNumber))
	if len(regNumber) < 8 {
		return nil, fmt.Errorf("%w: %q is not a registration number", ErrSignatureRejected, regNumber)
	}

	pre, err := m.preflightSignature(ctx, tenantID, docID, SignerEID)
	if err != nil {
		return nil, err
	}
	// Refusing here saves pushing a request that could never be turned into a
	// signature, and tells the operator why before the citizen is involved.
	if err := checkSigner(pre.Position, pre.DocType, regNumber); err != nil {
		return nil, err
	}
	// Asking somebody to approve on their own device and then discarding it because
	// they had already signed is worse than not asking.
	signed, err := m.alreadySigned(ctx, m.db, tenantID, docID, regNumber)
	if err != nil {
		return nil, err
	}
	if signed {
		return nil, ErrAlreadySigned
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

	// Housekeeping happens BEFORE the citizen is troubled. An approval nobody
	// answers is never polled again, so its row would sit here for good; but a
	// failure to tidy up must not cost a live approval prompt, which is what
	// happened when this ran after the push and returned its error.
	if _, err := m.db.Exec(ctx,
		`DELETE FROM document_eid_sign_sessions
		  WHERE tenant_id = $1 AND document_id = $2 AND consumed_at IS NULL
		    AND (expires_at < NOW() - INTERVAL '1 hour'
		         OR created_at < NOW() - INTERVAL '1 hour')`,
		tenantID, docID); err != nil {
		slog.WarnContext(ctx, "could not clear stale document signature sessions",
			"tenant_id", tenantID, "document_id", docID, "error", err)
	}

	displayText := signatureDisplayText(title)
	started, err := m.eidSvc.StartSignature(ctx, regNumber, displayText, "")
	if err != nil {
		return nil, fmt.Errorf("%w: E-ID could not reach the signer: %w", ErrProviderUnavailable, err)
	}

	// The citizen has been asked by this point, so that is recorded before anything
	// else can fail. Who asked for a signature is not who gave it, and the ledger
	// only ever knows the latter — this is the other half of the trail, and it must
	// not go missing because the bookkeeping below did.
	audit.Record(ctx, tenantID, actorFor(ctx), "documents.signature_requested", docID, map[string]any{
		"signer_reg_number": regNumber,
		"display_text":      displayText,
		"session_id":        started.SessionID,
	})

	// The citizen's phone is now showing the request, so the pairing that makes it
	// redeemable has to be written even if the caller has gone away — otherwise
	// they could approve a document nothing can attach their signature to.
	expiresAt := parseExpiry(started.ExpiresAt)
	if _, err := m.db.Exec(context.WithoutCancel(ctx),
		`INSERT INTO document_eid_sign_sessions
		        (session_id, tenant_id, document_id, reg_number, display_text, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		started.SessionID, tenantID, docID, regNumber, displayText, expiresAt); err != nil {
		return nil, fmt.Errorf("record signature session: %w", err)
	}

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
	var consumed, expired bool
	err := m.db.QueryRow(ctx,
		`SELECT reg_number, consumed_at IS NOT NULL,
		        (expires_at IS NOT NULL AND expires_at < NOW() - $4::interval)
		   FROM document_eid_sign_sessions
		  WHERE session_id = $1 AND tenant_id = $2 AND document_id = $3`,
		sessionID, tenantID, docID, signSessionGrace.String()).Scan(&regNumber, &consumed, &expired)
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

	// An approval the citizen gave days ago is not one to turn into a signature
	// dated today. eID's own deadline decides, with a little grace for clock skew.
	if expired {
		if _, err := m.db.Exec(ctx,
			`DELETE FROM document_eid_sign_sessions WHERE session_id = $1 AND tenant_id = $2`,
			sessionID, tenantID); err != nil {
			slog.WarnContext(ctx, "could not delete expired document signature session",
				"session_id", sessionID, "error", err)
		}
		return &EIDSignProgress{State: ApprovalExpired}, nil
	}

	result, err := m.eidSvc.Poll(ctx, sessionID)
	if err != nil {
		// A dropped connection to eID is not an answer. Reporting it as the caller's
		// mistake made the dialog give up on a ceremony the citizen was still holding
		// their phone for.
		return nil, fmt.Errorf("%w: E-ID could not be reached: %w", ErrProviderUnavailable, err)
	}

	if result.State != ApprovalComplete {
		// A refused or expired request is over; dropping the row keeps a dead
		// session from being polled forever. The citizen's answer is the one thing
		// the operator needs, so it is returned whether or not the tidy-up worked —
		// making it contingent on housekeeping turned "they declined" into a 500.
		if result.State == ApprovalRefused || result.State == ApprovalExpired {
			if _, err := m.db.Exec(ctx,
				`DELETE FROM document_eid_sign_sessions WHERE session_id = $1 AND tenant_id = $2`,
				sessionID, tenantID); err != nil {
				slog.WarnContext(ctx, "could not clear finished document signature session",
					"session_id", sessionID, "error", err)
			}
		}
		return &EIDSignProgress{State: result.State}, nil
	}

	if result.Identity == nil {
		return nil, fmt.Errorf("%w: E-ID reported an approval without an identity", ErrProviderUnavailable)
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
	if err := checkSigner(pre.Position, pre.DocType, approved); err != nil {
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

	// The session is spent in the same transaction as the signature, so a recorded
	// signature and an unspent session can never disagree. Doing it afterwards let
	// a dropped connection report a 500 for a signature that had in fact landed,
	// and left the session redeemable a second time.
	doc, err := m.recordSignature(ctx, tenantID, docID, SignerEID, signature, sessionID)
	if err != nil {
		return nil, err
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
	case errors.Is(err, ErrNotSignable), errors.Is(err, ErrAlreadySigned):
		writeError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, ErrProviderUnavailable):
		writeError(w, http.StatusServiceUnavailable, err.Error())
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
	case errors.Is(err, ErrProviderUnavailable):
		writeError(w, http.StatusServiceUnavailable, err.Error())
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
