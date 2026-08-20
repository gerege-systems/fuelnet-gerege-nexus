/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Signing what the document carries.
 *
 * The other ceremony — eidsign.go — asks a citizen to approve a prompt naming
 * the document, and records that they did. It is the honest thing to record for
 * a document with no content, and until ADR 0003 it was the only thing this app
 * could record at all.
 *
 * This one signs the file's digest through nexus.Signer, and confirms
 * afterwards that what the rail signed is what was sent. That confirmation is
 * the difference: without it a caller has a session id and a promise, which is
 * what the old path had.
 */

package documents

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	domain "github.com/gerege-systems/open-gerege-nexus/backend/domain/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// startContentSignature asks the citizen to sign the document's file.
//
// The guards are the approval path's, in the same order and for the same
// reasons: what this module can see is wrong is refused before a citizen is
// troubled, and the authority check runs again under the lock when the
// signature is recorded.
func (m *DocumentsModule) startContentSignature(ctx context.Context, tenantID, docID, regNumber string,
	artifact domain.Artifact, pre *signaturePreflight, displayText string) (*EIDSignSession, error) {

	if m.signer == nil || !m.signer.Enabled() {
		return nil, fmt.Errorf("%w: this installation has no signing rail", ErrProviderUnavailable)
	}

	// The format the document's own content dictates. pdfRail is false because
	// the PAdES rail is esign's and is reached through its own routes and its
	// own store; a PDF filed here is signed detached, which covers the same
	// bytes and is what this app can produce today. See ADR 0003.
	format := domain.FormatFor(artifact, false)

	started, err := m.signer.SignDigest(ctx, nexus.SignatureRequest{
		RegNumber:    regNumber,
		DigestHex:    artifact.SHA256,
		DisplayText:  displayText,
		DocumentName: artifact.FileName,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: the signing rail could not reach the signer: %w",
			ErrProviderUnavailable, err)
	}

	nexus.Audit(ctx, tenantID, actorFor(ctx), "documents.signature_requested", docID, map[string]any{
		"signer_reg_number": regNumber,
		"display_text":      displayText,
		"session_id":        started.SessionID,
		"format":            string(format),
		"digest":            artifact.SHA256,
	})

	// The citizen's device is showing the request by now, so the pairing that
	// makes it redeemable is written even if the caller has gone away — and
	// with the digest, so that what comes back can be held to what was sent.
	if _, err := m.db.Exec(ctx, `
		INSERT INTO document_eid_sign_sessions
		    (session_id, tenant_id, document_id, reg_number, display_text, requested_digest, format)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		started.SessionID, tenantID, docID, regNumber, displayText,
		artifact.SHA256, string(format)); err != nil {
		return nil, fmt.Errorf("record signing session: %w", err)
	}

	return &EIDSignSession{
		SessionID:        started.SessionID,
		VerificationCode: started.VerificationCode,
		DisplayText:      displayText,
	}, nil
}

// pollContentSignature waits for the citizen and, when they have approved,
// checks that the rail signed what this document sent before anything is
// recorded.
func (m *DocumentsModule) pollContentSignature(ctx context.Context, tenantID, docID, sessionID string,
	session signSession) (*EIDSignProgress, error) {

	if m.signer == nil || !m.signer.Enabled() {
		return nil, fmt.Errorf("%w: this installation has no signing rail", ErrProviderUnavailable)
	}

	state, err := m.signer.PollSignature(ctx, session.RegNumber, sessionID)
	if err != nil {
		// A dropped connection is not an answer, and reporting it as the
		// caller's mistake makes a dialog give up on a ceremony the citizen is
		// still holding their phone for.
		return nil, fmt.Errorf("%w: the signing rail could not be reached: %w", ErrProviderUnavailable, err)
	}
	if state != nexus.SignatureCompleted {
		if state == nexus.SignatureRejected || state == nexus.SignatureExpired || state == nexus.SignatureFailed {
			m.forgetSession(ctx, tenantID, sessionID)
		}
		return &EIDSignProgress{State: approvalStateOf(state)}, nil
	}

	// The half that makes this a signature rather than a session id. A rail
	// that answers with a different digest signed something else — a stale
	// session, a mixed-up id — and recording it would put a signature for one
	// document on another.
	signed, err := m.signer.VerifiedDigest(ctx, session.RegNumber, sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: the signing rail would not say what was signed: %w",
			ErrProviderUnavailable, err)
	}
	if !sameDigest(signed, session.RequestedDigest) {
		return nil, fmt.Errorf("%w: the rail signed %s, this document asked for %s",
			ErrSignatureRejected, signed, session.RequestedDigest)
	}

	// The policy and the chain are read again here rather than trusted from
	// start time: minutes of a citizen's ceremony may have passed, and it is
	// this moment that decides.
	pre, err := m.preflightSignature(ctx, tenantID, docID, SignerEID)
	if err != nil {
		return nil, err
	}
	if err := checkSigner(pre.Position, pre.DocType, session.RegNumber, pre.Policy.RequireNamedSigner); err != nil {
		return nil, err
	}

	doc, err := m.recordSignature(ctx, tenantID, docID, SignerEID, &verifiedSignature{
		SignerName: session.RegNumber + " (E-ID баталгаажсан)",
		RegNumber:  session.RegNumber,
		// The signature references the ceremony, as before. What it COVERS is
		// the digest below, and that is the field a dispute turns on.
		Hash:          "eid_session_" + sessionID,
		Format:        domain.Format(session.Format),
		CoveredDigest: session.RequestedDigest,
	}, sessionID)
	if err != nil {
		return nil, err
	}
	return &EIDSignProgress{State: ApprovalComplete, Document: doc}, nil
}

// signSession is the pairing row: which document a ceremony belongs to, who it
// was addressed to, and what it was asked to sign.
type signSession struct {
	RegNumber       string
	DisplayText     string
	RequestedDigest string
	Format          string
	Consumed        bool
}

func (m *DocumentsModule) loadSignSession(ctx context.Context, tenantID, docID, sessionID string) (signSession, error) {
	var session signSession
	err := m.db.QueryRow(ctx, `
		SELECT reg_number, display_text, COALESCE(requested_digest, ''), COALESCE(format, ''),
		       consumed_at IS NOT NULL
		  FROM document_eid_sign_sessions
		 WHERE session_id = $1 AND tenant_id = $2 AND document_id = $3`,
		sessionID, tenantID, docID).Scan(&session.RegNumber, &session.DisplayText,
		&session.RequestedDigest, &session.Format, &session.Consumed)
	if isNoRows(err) {
		return signSession{}, ErrNoSuchSession
	}
	return session, err
}

// ErrNoSuchSession is a ceremony this document does not have.
var ErrNoSuchSession = errors.New("no such signing session for this document")

func (m *DocumentsModule) forgetSession(ctx context.Context, tenantID, sessionID string) {
	if _, err := m.db.Exec(ctx,
		`DELETE FROM document_eid_sign_sessions WHERE session_id = $1 AND tenant_id = $2`,
		sessionID, tenantID); err != nil {
		// A finished ceremony that could not be tidied away is not worth
		// failing the citizen's answer over.
		_ = err
	}
}

// approvalStateOf maps the rail's words onto the ones this app's clients have
// always polled for.
func approvalStateOf(state nexus.SignatureState) string {
	switch state {
	case nexus.SignatureCompleted:
		return ApprovalComplete
	case nexus.SignatureRejected:
		return ApprovalRefused
	case nexus.SignatureExpired, nexus.SignatureFailed:
		return ApprovalExpired
	default:
		return ApprovalRunning
	}
}

// sameDigest compares what the rail says it signed with what was sent.
//
// The two arrive in different encodings — the rail answers base64, this app
// stores hex — so they are compared as bytes. A string comparison would fail
// for every correct signature, which is the kind of check that gets deleted
// rather than fixed.
func sameDigest(railAnswer, requestedHex string) bool {
	wanted, err := hex.DecodeString(strings.TrimSpace(requestedHex))
	if err != nil || len(wanted) == 0 {
		return false
	}
	if got, err := base64.StdEncoding.DecodeString(strings.TrimSpace(railAnswer)); err == nil {
		return string(got) == string(wanted)
	}
	// Some rails answer hex. Accepting both is not laxness: what matters is
	// that the bytes are the same bytes, and refusing a correct answer because
	// of its encoding would refuse a citizen's real signature.
	if got, err := hex.DecodeString(strings.TrimSpace(railAnswer)); err == nil {
		return string(got) == string(wanted)
	}
	return false
}
