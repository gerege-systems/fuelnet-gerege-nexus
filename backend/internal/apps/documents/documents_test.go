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

// What the citizen reads on their own device is the only thing telling them what
// they are approving, so the words saying a signature is being given have to
// survive — and they only survive if we cut the text ourselves. eID's field is
// displayText60 and the core client slices it to 60 BYTES, which for Cyrillic is
// about 30 letters and can land mid-letter.
func TestSignatureDisplayText(t *testing.T) {
	short := signatureDisplayText("Гэрээ 2026")
	if !strings.Contains(short, "Гэрээ 2026") {
		t.Errorf("got %q, want a title that fits to be shown whole", short)
	}
	if !strings.Contains(short, "Гарын үсэг") {
		t.Errorf("got %q, want it to say a signature is being asked for", short)
	}

	// The case that used to break: a long Cyrillic title. The purpose has to
	// survive it, because the purpose is the whole point of the prompt.
	long := signatureDisplayText("Улаанбаатар хотын Захирагчийн ажлын албатай хамтран ажиллах гурван талт гэрээ, 2026 оны 1 дүгээр сарын 15-ны өдөр")
	if !strings.Contains(long, "Гарын үсэг") {
		t.Errorf("got %q, want the purpose to survive a long title", long)
	}
	if !strings.HasSuffix(long, "…") {
		t.Errorf("got %q, want a cut title marked as cut", long)
	}

	for name, text := range map[string]string{"short": short, "long": long} {
		if len(text) > eidDisplayTextBytes {
			t.Errorf("%s: %d bytes, want at most %d — core would slice it mid-letter",
				name, len(text), eidDisplayTextBytes)
		}
		if !utf8.ValidString(text) {
			t.Errorf("%s: the cut left invalid UTF-8 behind: %q", name, text)
		}
	}

	// A document with no title still has to say what is being approved.
	if got := signatureDisplayText("   "); !strings.Contains(got, "гарын үсэг") {
		t.Errorf("got %q, want an empty title to still state the purpose", got)
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

// A signature is held to whoever the document's next step names, and held back
// from anyone the chain still needs further along.
func TestCheckSigner(t *testing.T) {
	if err := checkSigner(nil, "CONTRACT", "AA90010111"); err != nil {
		t.Errorf("a document with no chain must accept any authorised signer: %v", err)
	}
	if err := checkSigner(&approvalPosition{}, "CONTRACT", "AA90010111"); err != nil {
		t.Errorf("a position with no next step must accept any authorised signer: %v", err)
	}

	open := &approvalPosition{Next: &ApprovalStep{Order: 1, Name: "Хэн ч"}}
	if err := checkSigner(open, "CONTRACT", "AA90010111"); err != nil {
		t.Errorf("an open step must accept any authorised signer: %v", err)
	}

	named := &approvalPosition{Next: &ApprovalStep{Order: 2, Name: "Захирал", SignerRegNumber: "CC90010111"}}
	if err := checkSigner(named, "CONTRACT", "CC90010111"); err != nil {
		t.Errorf("the named signer must be accepted: %v", err)
	}
	err := checkSigner(named, "CONTRACT", "AA90010111")
	if !errors.Is(err, ErrSignatureRejected) {
		t.Fatalf("got %v, want ErrSignatureRejected", err)
	}
	for _, want := range []string{"AA90010111", "CC90010111", "Захирал"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("got %q, want it to mention %q so an operator can tell why", err, want)
		}
	}

	// An open step is open — but not to somebody a later step names. Spending their
	// one signature here would leave their own step unfillable for ever.
	reserved := &approvalPosition{
		Next:     &ApprovalStep{Order: 1, Name: "Хянагч"},
		Reserved: []string{"CC90010111"},
	}
	if err := checkSigner(reserved, "CONTRACT", "AA90010111"); err != nil {
		t.Errorf("anyone else may still take the open step: %v", err)
	}
	err = checkSigner(reserved, "CONTRACT", "CC90010111")
	if !errors.Is(err, ErrSignatureRejected) {
		t.Fatalf("a citizen a later step names must be held back: got %v", err)
	}
	if !strings.Contains(err.Error(), "CC90010111") {
		t.Errorf("got %q, want it to name the signer being held back", err)
	}
}

// A chain that could never be completed under a named-signer policy must be
// refused when it is saved, not discovered when a document sticks.
func TestStepsCanRequireNamedSigners(t *testing.T) {
	ok := []WorkflowStep{
		{Order: 1, Name: "Дарга", SignerRegNumber: "AA90010111"},
		{Order: 2, Name: "Захирал", SignerRegNumber: "BB90010111"},
	}
	if err := stepsCanRequireNamedSigners("CONTRACT", ok); err != nil {
		t.Errorf("a chain naming a distinct signer per step is fine: %v", err)
	}

	for name, steps := range map[string][]WorkflowStep{
		"no steps at all": {},
		"an open step nobody could fill": {
			{Order: 1, Name: "Дарга", SignerRegNumber: "AA90010111"},
			{Order: 2, Name: "Хэн ч"},
		},
		"one citizen named twice, who signs once": {
			{Order: 1, Name: "Дарга", SignerRegNumber: "AA90010111"},
			{Order: 2, Name: "Дарга дахин", SignerRegNumber: "AA90010111"},
		},
	} {
		if err := stepsCanRequireNamedSigners("CONTRACT", steps); !errors.Is(err, ErrInvalidConfiguration) {
			t.Errorf("%s: got %v, want ErrInvalidConfiguration", name, err)
		}
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
