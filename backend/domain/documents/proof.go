/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package documents

// Proof is what a signature record actually establishes.
//
// The distinction is not pedantry, it is the difference between two legal
// artifacts that this app had been recording under one word. See
// docs/adr/0002-one-signing-rail.md.
type Proof string

const (
	// ProofApproval — a citizen proved who they were and approved a prompt
	// naming this document, at a moment recorded here.
	//
	// It establishes identity, intent and time. It does NOT bind to the
	// document's content: nothing in the record covers the bytes, so a document
	// edited afterwards is not contradicted by anything stored. This is what
	// both national channels give this app, because a document_records row
	// holds no content to sign — the title, the type and the status, and that
	// is all.
	ProofApproval Proof = "approval"

	// ProofSignature — a signature over the document's content, verifiable
	// against it afterwards.
	//
	// Nothing in this app produces one today and the record must not claim it.
	// It is declared because the platform can: internal/platform/esign signs
	// PDFs it holds, through the rail nexus.Signer publishes. The day a
	// document here carries an artifact, this is what its record would say.
	ProofSignature Proof = "signature"

	// ProofUnknown is what an unrecognised method yields, so that a rail added
	// without deciding this question is visible rather than silently filed as
	// one of the two.
	ProofUnknown Proof = ""
)

// ProofOf reports what a signing method establishes on this platform.
//
// A table rather than a default, because the answer is a decision about a
// national channel and not something to be guessed from a string: adding a rail
// should mean answering this, and an unanswered one shows up as unknown instead
// of being quietly promoted.
func ProofOf(method string) Proof {
	switch method {
	case SignerEID, SignerDAN:
		// Both authenticate a citizen and record their approval. eID has no
		// separate document-signing endpoint on this path — the approval a
		// citizen gives with their own credentials is what is recorded — and
		// ДАН has no signing product on this platform at all.
		return ProofApproval
	default:
		return ProofUnknown
	}
}
