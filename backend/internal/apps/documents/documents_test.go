package documents

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// The HTTP handlers turn these error classes into 409, 404, 400 and 500, so the
// class a failure lands in decides whether the caller is told to fix something or
// told the server broke. A nil pool is safe here: every case is settled before a
// query, and before any citizen is troubled.
func TestSigningRefusesAMalformedID(t *testing.T) {
	ctx := context.Background()
	module := &DocumentsModule{}

	if _, err := module.SignWithDAN(ctx, "tenant", "not-a-uuid", "AA90010111", "123456"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("dan: got %v, want ErrNotSignable", err)
	}
	if _, err := module.StartEIDSignature(ctx, "tenant", "not-a-uuid", "AA90010111"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("eid start: got %v, want ErrNotSignable", err)
	}
	if _, err := module.PollEIDSignature(ctx, "tenant", "not-a-uuid", "some-session"); !errors.Is(err, ErrSignSessionUnknown) {
		t.Errorf("eid poll: got %v, want ErrSignSessionUnknown", err)
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

// What the citizen reads on their phone is the only thing telling them what they
// are approving, so it has to name the document — approving a sign-in prompt is
// not consent to sign a contract.
func TestSignatureDisplayTextNamesTheDocument(t *testing.T) {
	got := signatureDisplayText("Хамтран ажиллах гэрээ 2026")
	if !strings.Contains(got, "Хамтран ажиллах гэрээ 2026") {
		t.Errorf("got %q, want the document title in it", got)
	}
	if !strings.Contains(got, "гарын үсэг") {
		t.Errorf("got %q, want it to say what is being asked", got)
	}

	// eID limits the text, so a long title is cut rather than dropped — measured
	// in runes, because a Cyrillic title would otherwise be sliced mid-character.
	long := signatureDisplayText(strings.Repeat("Урт нэр ", 40))
	if runes := []rune(long); len(runes) > 120 {
		t.Errorf("display text is %d runes, want at most 120", len(runes))
	}
	if !strings.HasSuffix(long, "…") {
		t.Errorf("got %q, want a cut title to be marked as cut", long)
	}
	if !utf8.ValidString(long) {
		t.Error("the cut left invalid UTF-8 behind")
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

// The named-signer rule is checked against the registration number a provider
// vouched for, so the check itself has to be exact about what it accepts.
func TestCheckNamedSigner(t *testing.T) {
	open := &signaturePreflight{DocType: "CONTRACT", Policy: SignaturePolicy{AllowEID: true}}
	if err := open.checkNamedSigner("AA90010111"); err != nil {
		t.Errorf("a policy that names nobody must accept anyone: %v", err)
	}

	strict := &signaturePreflight{
		DocType: "CONTRACT",
		Policy:  SignaturePolicy{AllowEID: true, RequireNamedSigner: true},
		Named:   []string{"CC90010111"},
	}
	if err := strict.checkNamedSigner("CC90010111"); err != nil {
		t.Errorf("the named signer must be accepted: %v", err)
	}
	err := strict.checkNamedSigner("AA90010111")
	if !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("got %v, want ErrSignatureRejected", err)
	}
	if !strings.Contains(err.Error(), "AA90010111") {
		t.Errorf("got %q, want the refused signer named so an operator can tell why", err)
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
