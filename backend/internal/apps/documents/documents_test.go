package documents

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
)

// The HTTP handlers turn these three error classes into 409, 400 and 500, so the
// class a failure lands in is what decides whether the caller is told to fix
// something or told the server broke. Every case below is settled before a query
// runs, which is why a nil pool is safe here.
func TestSignAndRejectClassifyFailuresBeforeTouchingStorage(t *testing.T) {
	const validID = "3f1b9c62-2f1a-4a1c-9d3e-8b7a5c4e1d20"
	ctx := context.Background()
	module := &DocumentsModule{}

	t.Run("a malformed id is not a document we hold", func(t *testing.T) {
		if _, err := module.SignDocument(ctx, "tenant", "not-a-uuid", SignerEID, "AA90010111", "123456"); !errors.Is(err, ErrNotSignable) {
			t.Errorf("sign: got %v, want ErrNotSignable", err)
		}
		if _, err := module.RejectDocument(ctx, "tenant", "not-a-uuid"); !errors.Is(err, ErrNotSignable) {
			t.Errorf("reject: got %v, want ErrNotSignable", err)
		}
	})

	t.Run("a channel this module does not speak is the caller's mistake", func(t *testing.T) {
		_, err := module.SignDocument(ctx, "tenant", validID, "SMARTCARD", "AA90010111", "123456")
		if !errors.Is(err, ErrSignatureRejected) {
			t.Errorf("got %v, want ErrSignatureRejected", err)
		}
	})

	t.Run("an identity the provider refuses is the caller's mistake", func(t *testing.T) {
		// The E-ID client rejects a registration number under eight characters
		// before it decides between live and mock mode, so this holds either way.
		signer := &DocumentsModule{eidSvc: eid.NewEIDService()}
		_, err := signer.SignDocument(ctx, "tenant", validID, SignerEID, "AA1", "123456")
		if !errors.Is(err, ErrSignatureRejected) {
			t.Errorf("got %v, want ErrSignatureRejected", err)
		}
	})
}

// An empty method is a convenience, not an error: E-ID is the default channel.
func TestEmptySignerMethodDefaultsToEID(t *testing.T) {
	module := &DocumentsModule{eidSvc: eid.NewEIDService()}
	_, err := module.SignDocument(context.Background(), "tenant",
		"3f1b9c62-2f1a-4a1c-9d3e-8b7a5c4e1d20", "  ", "AA1", "123456")

	// "AA1" is refused by the E-ID client, which is how we know the request
	// reached it instead of falling into the unsupported-method branch.
	if !errors.Is(err, ErrSignatureRejected) {
		t.Fatalf("got %v, want ErrSignatureRejected", err)
	}
	if got := err.Error(); !strings.Contains(got, "E-ID") {
		t.Errorf("got %q, want the E-ID channel to have handled it", got)
	}
}
