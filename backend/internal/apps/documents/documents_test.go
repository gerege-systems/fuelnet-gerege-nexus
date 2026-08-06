package documents

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
)

// The HTTP handlers turn these error classes into 409, 400 and 500, so the class
// a failure lands in decides whether the caller is told to fix something or told
// the server broke. A nil pool is safe here: every case is settled before a query.
func TestSignAndRejectRefuseAMalformedID(t *testing.T) {
	ctx := context.Background()
	module := &DocumentsModule{}

	if _, err := module.SignDocument(ctx, "tenant", "not-a-uuid", SignerEID, "AA90010111", "123456"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("sign: got %v, want ErrNotSignable", err)
	}
	if _, err := module.RejectDocument(ctx, "tenant", "not-a-uuid"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("reject: got %v, want ErrNotSignable", err)
	}
	if _, err := module.RouteDocument(ctx, "tenant", "not-a-uuid"); !errors.Is(err, ErrNotRoutable) {
		t.Errorf("route: got %v, want ErrNotRoutable", err)
	}
	if _, err := module.ListSignatures(ctx, "tenant", "not-a-uuid"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("signatures: got %v, want ErrNotSignable", err)
	}
	if err := module.DeleteTemplate(ctx, "tenant", "not-a-uuid"); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("delete template: got %v, want ErrTemplateNotFound", err)
	}
	if _, err := module.CreateDocumentFromTemplate(ctx, "tenant", "not-a-uuid"); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("use template: got %v, want ErrTemplateNotFound", err)
	}
}

func TestNormalizeSignerMethod(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"", SignerEID}, // an unstated channel means E-ID
		{"  ", SignerEID},
		{"eid", SignerEID},
		{" DAN ", SignerDAN},
		{"dan", SignerDAN},
	} {
		got, err := normalizeSignerMethod(tc.in)
		if err != nil {
			t.Errorf("%q: unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.in, got, tc.want)
		}
	}

	for _, in := range []string{"SMARTCARD", "pki", "e-id"} {
		if _, err := normalizeSignerMethod(in); !errors.Is(err, ErrSignatureRejected) {
			t.Errorf("%q: got %v, want ErrSignatureRejected", in, err)
		}
	}
}

// An identity the provider will not vouch for is the caller's problem, not a
// server fault, whichever channel refused it.
func TestVerifySignerReportsARefusalAsRejected(t *testing.T) {
	module := &DocumentsModule{eidSvc: eid.NewEIDService()}
	ctx := context.Background()

	// The E-ID client refuses a registration number under eight characters
	// before it chooses between live and mock mode, so this holds either way.
	_, err := module.verifySigner(ctx, SignerEID, "AA1", "123456")
	if !errors.Is(err, ErrSignatureRejected) {
		t.Fatalf("got %v, want ErrSignatureRejected", err)
	}
	if got := err.Error(); !strings.Contains(got, "E-ID") {
		t.Errorf("got %q, want the message to name the channel that refused", got)
	}

	if _, err := module.verifySigner(ctx, "SMARTCARD", "AA90010111", ""); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("unsupported channel: got %v, want ErrSignatureRejected", err)
	}
}

func TestSignaturePolicyDefaultsToBothChannels(t *testing.T) {
	policy := defaultSignaturePolicy("CONTRACT")

	if !policy.allows(SignerEID) || !policy.allows(SignerDAN) {
		t.Error("an unconfigured type must accept both national channels, as it did before policies existed")
	}
	if policy.RequireNamedSigner {
		t.Error("an unconfigured type must not require a named signer")
	}
	if policy.Configured {
		t.Error("a default policy is not a stored one")
	}
	if policy.allows("SMARTCARD") {
		t.Error("a channel the module does not speak must never be allowed")
	}

	only := SignaturePolicy{DocType: "CONTRACT", AllowEID: true}
	if only.allows(SignerDAN) {
		t.Error("DAN must be refused when the policy allows E-ID only")
	}
}

func TestResolveTitlePattern(t *testing.T) {
	at := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	if got, want := resolveTitlePattern("Гэрээ {year}", at), "Гэрээ 2026"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := resolveTitlePattern("{date} · {month}", at), "2026-08-06 · 08"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// An unknown token is left in place rather than silently dropped, so a typo
	// shows up in the document instead of disappearing.
	if got, want := resolveTitlePattern("Гэрээ {quarter}", at), "Гэрээ {quarter}"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
