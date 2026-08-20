package documents

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// fakeSigner is nexus.Signer for the tests: a rail that always says yes, and
// remembers what it was asked to sign.
//
// The remembering is the point. What separates a signature from a session id is
// that the caller can hold the rail's answer against what it sent, and a fake
// that answered a fixed digest would let that check pass while never testing
// it — see badDigest below, which is the same rail signing something else.
type fakeSigner struct {
	mu sync.Mutex
	// asked is every digest this rail was handed, newest last.
	asked map[string]string
	// badDigest makes the rail answer with a digest it was never given, which
	// is what a stale or mixed-up session looks like from here.
	badDigest string
	// refuse makes every ceremony come back rejected, the way a citizen
	// declining does.
	refuse bool
	off    bool
}

func newFakeSigner() *fakeSigner { return &fakeSigner{asked: map[string]string{}} }

func (f *fakeSigner) Enabled() bool { return !f.off }

func (f *fakeSigner) SignDigest(_ context.Context, request nexus.SignatureRequest) (nexus.SignatureSession, error) {
	if f.off {
		return nexus.SignatureSession{}, nexus.ErrSigningUnavailable
	}
	if strings.TrimSpace(request.DigestHex) == "" {
		return nexus.SignatureSession{}, errors.New("a ceremony needs something to sign")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	session := "fake-sign-" + request.DigestHex[:8]
	f.asked[session] = request.DigestHex
	return nexus.SignatureSession{SessionID: session, VerificationCode: "2026"}, nil
}

func (f *fakeSigner) PollSignature(_ context.Context, _, sessionID string) (nexus.SignatureState, error) {
	if f.off {
		return "", nexus.ErrSigningUnavailable
	}
	if f.refuse {
		return nexus.SignatureRejected, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, known := f.asked[sessionID]; !known {
		return nexus.SignatureExpired, nil
	}
	return nexus.SignatureCompleted, nil
}

func (f *fakeSigner) VerifiedDigest(_ context.Context, _, sessionID string) (string, error) {
	if f.off {
		return "", nexus.ErrSigningUnavailable
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	answer := f.asked[sessionID]
	if f.badDigest != "" {
		answer = f.badDigest
	}
	raw, err := hex.DecodeString(answer)
	if err != nil {
		return "", err
	}
	// Base64, the way the real rail answers.
	return base64.StdEncoding.EncodeToString(raw), nil
}

var _ nexus.Signer = (*fakeSigner)(nil)
