package documents_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/domain/documents"
)

var docTypes = []string{"CONTRACT", "REQUEST", "APPROVAL"}

// One citizen signs a document once. Every rule below follows from that
// sentence, and every one of them used to need a migrated database and a
// signing provider to observe.
func TestAChainThatCouldNeverBeCompletedIsRefused(t *testing.T) {
	refusals := []struct {
		name  string
		steps []documents.WorkflowStep
		want  string
	}{
		{"a step with no name",
			[]documents.WorkflowStep{{Name: "  "}},
			"invalid configuration: step 1 needs a name"},
		{"the same citizen twice",
			[]documents.WorkflowStep{
				{Name: "Эхний", SignerRegNumber: "УБ99010111"},
				{Name: "Хоёр дахь", SignerRegNumber: "уб99010111"},
			},
			"invalid configuration: steps 1 and 2 both name УБ99010111, and one citizen signs a document once"},
		{"a signer nobody could present",
			[]documents.WorkflowStep{{Name: "Эхний", SignerRegNumber: "УБ9901"}},
			`invalid configuration: step 1 names "УБ9901", which is 6 characters — a registration number is 8 to 64`},
		{"a name that cannot be stored",
			[]documents.WorkflowStep{{Name: "Эхний\x00"}},
			"invalid configuration: step 1's name cannot be stored — it contains a NUL character"},
	}

	for _, refusal := range refusals {
		t.Run(refusal.name, func(t *testing.T) {
			_, _, err := documents.ValidateChain("CONTRACT", docTypes, refusal.steps)
			if err == nil {
				t.Fatal("expected a refusal")
			}
			if !errors.Is(err, documents.ErrInvalidConfiguration) {
				t.Fatalf("a refusal must wrap ErrInvalidConfiguration: %v", err)
			}
			if err.Error() != refusal.want {
				t.Fatalf("got %q, want %q", err, refusal.want)
			}
		})
	}

	// A type nothing produces, and a chain longer than anybody approves.
	if _, _, err := documents.ValidateChain("INVOICE", docTypes, nil); err == nil ||
		err.Error() != `invalid configuration: invalid doc_type "INVOICE"` {
		t.Fatalf("an unknown type: %v", err)
	}
	long := make([]documents.WorkflowStep, documents.MaxChainSteps+1)
	for i := range long {
		long[i] = documents.WorkflowStep{Name: "Алхам"}
	}
	if _, _, err := documents.ValidateChain("contract", docTypes, long); err == nil ||
		err.Error() != "invalid configuration: an approval chain is limited to 10 steps" {
		t.Fatalf("too many steps: %v", err)
	}
}

// A policy that only accepts named signers, over a chain with an open step,
// would leave the type unapprovable by anybody.
func TestRequiringNamedSignersNeedsAChainThatNamesThem(t *testing.T) {
	if err := documents.StepsCanRequireNamedSigners("CONTRACT", nil); err == nil ||
		!strings.Contains(err.Error(), "has no steps") {
		t.Fatalf("an empty chain: %v", err)
	}
	open := []documents.WorkflowStep{{Order: 1, Name: "Дарга"}}
	if err := documents.StepsCanRequireNamedSigners("CONTRACT", open); err == nil ||
		err.Error() != "invalid configuration: step 1 of the CONTRACT chain (Дарга) names no signer, so requiring a named signer would leave it unfillable" {
		t.Fatalf("an open step: %v", err)
	}
	repeated := []documents.WorkflowStep{
		{Order: 1, Name: "Дарга", SignerRegNumber: "УБ99010111"},
		{Order: 2, Name: "Нягтлан", SignerRegNumber: "УБ99010111"},
	}
	if err := documents.StepsCanRequireNamedSigners("CONTRACT", repeated); err == nil ||
		!strings.Contains(err.Error(), "one citizen signs a document once") {
		t.Fatalf("a repeated citizen: %v", err)
	}
	named := []documents.WorkflowStep{
		{Order: 1, Name: "Дарга", SignerRegNumber: "УБ99010111"},
		{Order: 2, Name: "Нягтлан", SignerRegNumber: "УБ99010222"},
	}
	if err := documents.StepsCanRequireNamedSigners("CONTRACT", named); err != nil {
		t.Fatalf("a chain that names everybody: %v", err)
	}
}

// An unconfigured type allows both national channels, and a method nobody
// decided to accept is refused rather than allowed.
func TestAnUnconfiguredTypeAllowsBothChannels(t *testing.T) {
	policy := documents.DefaultSignaturePolicy("CONTRACT")
	if !policy.Allows(documents.SignerEID) || !policy.Allows(documents.SignerDAN) {
		t.Fatalf("the default policy: %+v", policy)
	}
	if policy.Allows("SMS") {
		t.Fatal("a channel this platform does not have must not be allowed")
	}
	if policy.Configured {
		t.Fatal("a default policy is not a configured one, and the screen says so")
	}

	eidOnly := documents.SignaturePolicy{DocType: "CONTRACT", AllowEID: true, Configured: true}
	if eidOnly.Allows(documents.SignerDAN) {
		t.Fatal("a policy that switched ДАН off must refuse it")
	}
}

// The chain as it is stored: the type upper-cased, names trimmed, numbers
// normalised, and the order renumbered from one.
func TestValidateChainAnswersWithWhatShouldBeStored(t *testing.T) {
	docType, cleaned, err := documents.ValidateChain("contract", docTypes, []documents.WorkflowStep{
		{Name: "  Дарга  ", SignerRegNumber: " уб99010111 "},
		{Name: "Нягтлан"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if docType != "CONTRACT" {
		t.Fatalf("the type is stored upper-cased, got %q", docType)
	}
	if cleaned[0].Name != "Дарга" || cleaned[0].SignerRegNumber != "УБ99010111" || cleaned[0].Order != 1 {
		t.Fatalf("step one was not cleaned up: %+v", cleaned[0])
	}
	// An open step is allowed and keeps its place in the order.
	if cleaned[1].SignerRegNumber != "" || cleaned[1].Order != 2 {
		t.Fatalf("step two: %+v", cleaned[1])
	}
}

// A Mongolian registration number is Cyrillic — "УБ99010111" is ten characters in
// twenty bytes — and every bound on it is a bound on CHARACTERS: the column counts
// characters, the SQL that repairs stored chains counts characters, and a citizen
// reading their own number counts characters. Measuring bytes in Go made the three
// places that enforce "a named step must be fillable" disagree: "УБ9901" is eight
// bytes but six characters, so the save accepted it as a named step and the snapshot
// then opened it, leaving the workflows screen naming a citizen the document's own
// chain did not.
func TestARegistrationNumberIsBoundedInCharactersNotBytes(t *testing.T) {
	for _, tc := range []struct {
		reg       string
		plausible bool
		why       string
	}{
		{"AA90010111", true, "the transliterated form the mock uses"},
		{"УБ99010111", true, "the real Cyrillic form, 10 characters in 20 bytes"},
		{"уб99010111", true, "the same, normalised on the way in"},
		{"УБ9901", false, "6 characters — 8 bytes, which once passed"},
		{"AA1", false, "3 characters"},
		{"", false, "an open step names nobody"},
		{strings.Repeat("У", documents.RegNumberMax), true, "64 characters is what the column holds"},
		{strings.Repeat("У", documents.RegNumberMax+1), false, "65 would not survive being stored"},
	} {
		if got := documents.PlausibleRegNumber(documents.NormaliseRegNumber(tc.reg)); got != tc.plausible {
			t.Errorf("documents.PlausibleRegNumber(%q) = %v, want %v — %s", tc.reg, got, tc.plausible, tc.why)
		}
	}

	// And the shortest Cyrillic number that passes must be exactly the limit in
	// characters, not in bytes.
	if !documents.PlausibleRegNumber(strings.Repeat("Ү", documents.RegNumberLimit)) {
		t.Error("a number of exactly the limit in characters must be accepted")
	}
	if documents.PlausibleRegNumber(strings.Repeat("Ү", documents.RegNumberLimit-1)) {
		t.Error("a number one character short must be refused")
	}
}

// The same rule, applied where a document's chain is decided: what cannot be saved
// must not be copied, and what can be saved must be copied intact.

// The same rule, applied where a document's chain is decided: what cannot be saved
// must not be copied, and what can be saved must be copied intact.
func TestFillableChainOpensOnlyWhatNobodyCouldFill(t *testing.T) {
	got := documents.FillableChain([]documents.WorkflowStep{
		{Order: 1, Name: "Ня-бо", SignerRegNumber: "  уб99010111 "}, // normalised, kept
		{Order: 2, Name: "Дахилт", SignerRegNumber: "УБ99010111"},   // the same citizen
		{Order: 3, Name: "Алдаа", SignerRegNumber: "УБ9901"},        // 6 characters
		{Order: 4, Name: "Хэн ч", SignerRegNumber: ""},              // open already
		{Order: 5, Name: "Захирал", SignerRegNumber: "cc90010111"},  // fine, normalised
	})
	want := []string{"УБ99010111", "", "", "", "CC90010111"}
	if len(got) != len(want) {
		t.Fatalf("got %d steps, want %d — the tenant asked for that many approvals", len(got), len(want))
	}
	for i, step := range got {
		if step.SignerRegNumber != want[i] {
			t.Errorf("step %d = %q, want %q", i+1, step.SignerRegNumber, want[i])
		}
		if step.Order != i+1 {
			t.Errorf("step %d carries order %d", i+1, step.Order)
		}
	}
}
