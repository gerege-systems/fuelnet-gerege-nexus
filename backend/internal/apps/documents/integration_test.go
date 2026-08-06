package documents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/dan"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
)

// These tests exercise signing against a real PostgreSQL schema, because what
// they protect lives partly in SQL: the status guard that keeps a decided
// document from being signed twice, the unique constraint that keeps one citizen
// from counting as two approvals, and the two counts the list derives from the
// ledger and the approval chain.
//
// They are skipped unless a migrated throwaway database is provided:
//
//	DOCUMENTS_TEST_DATABASE_URL=postgres://... go test ./internal/apps/documents/...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DOCUMENTS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCUMENTS_TEST_DATABASE_URL to a migrated test database to run documents integration tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type fixture struct {
	m        *DocumentsModule
	tenantID string
}

// newFixture builds a module wired to its own tenant, so one test's documents
// are never another's. The identity clients are pinned to mock mode: these tests
// are about what the module records, not about reaching sso.gov.mn.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("EID_MOCK_MODE", "true")
	t.Setenv("DAN_MOCK_MODE", "true")

	pool := testPool(t)
	m := &DocumentsModule{db: pool, eidSvc: eid.NewEIDService(), danSvc: dan.NewDANService()}

	var tenantID string
	slug := fmt.Sprintf("docs-test-%s", randomSuffix(t))
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id`,
		slug, "Documents integration test").Scan(&tenantID); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	t.Cleanup(func() {
		// The schema cascades from tenants, so one delete clears the documents,
		// signatures, chains, policies, templates and rules this test made.
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	return &fixture{m: m, tenantID: tenantID}
}

// signWithEID runs the whole approval ceremony: push the request to the citizen,
// then poll until they approve. In mock mode the citizen approves after a moment,
// which is exactly the shape a live approval has.
func signWithEID(t *testing.T, f *fixture, docID, regNumber string) (*Document, error) {
	t.Helper()
	ctx := context.Background()

	session, err := f.m.StartEIDSignature(ctx, f.tenantID, docID, regNumber)
	if err != nil {
		return nil, err
	}
	if session.VerificationCode == "" {
		t.Error("the citizen has no code to check the request against")
	}
	if !strings.Contains(session.DisplayText, "Гарын үсэг") {
		t.Errorf("display text %q does not say what is being approved", session.DisplayText)
	}
	return pollUntilApproved(t, f, docID, session.SessionID)
}

// pollUntilApproved waits on one already-started session, the way a screen does.
func pollUntilApproved(t *testing.T, f *fixture, docID, sessionID string) (*Document, error) {
	t.Helper()
	ctx := context.Background()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		progress, err := f.m.PollEIDSignature(ctx, f.tenantID, docID, sessionID)
		if err != nil {
			return nil, err
		}
		if progress.State == ApprovalComplete {
			return progress.Document, nil
		}
		if progress.State != ApprovalRunning {
			return nil, fmt.Errorf("approval ended as %s", progress.State)
		}
		time.Sleep(150 * time.Millisecond)
	}
	return nil, errors.New("the approval never completed")
}

// randomSuffix borrows the database's uuid generator rather than math/rand, so a
// test never has to seed anything to get an unused tenant slug.
func randomSuffix(t *testing.T) string {
	t.Helper()
	var suffix string
	if err := testPool(t).QueryRow(context.Background(),
		`SELECT left(gen_random_uuid()::text, 8)`).Scan(&suffix); err != nil {
		t.Fatalf("generate suffix: %v", err)
	}
	return suffix
}

func TestSigningApprovesWhenNoChainIsConfigured(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Хамтран ажиллах гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if doc.Status != StatusPending {
		t.Fatalf("new document status = %q, want %q", doc.Status, StatusPending)
	}
	if doc.RequiredSignatures != 1 || doc.SignatureCount != 0 {
		t.Errorf("new document progress = %d/%d, want 0/1", doc.SignatureCount, doc.RequiredSignatures)
	}

	signed, err := signWithEID(t, f, doc.ID, "AA90010111")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if signed.Status != StatusApproved {
		t.Errorf("status = %q, want %q", signed.Status, StatusApproved)
	}
	if signed.SignatureCount != 1 || signed.RequiredSignatures != 1 {
		t.Errorf("progress = %d/%d, want 1/1", signed.SignatureCount, signed.RequiredSignatures)
	}
	if signed.SignerRegNumber != "AA90010111" {
		t.Errorf("signer = %q, want AA90010111", signed.SignerRegNumber)
	}
	if signed.SignedAt == nil {
		t.Error("signed_at was not recorded")
	}

	ledger, err := f.m.ListSignatures(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("signatures: %v", err)
	}
	if len(ledger) != 1 {
		t.Fatalf("ledger has %d rows, want 1", len(ledger))
	}
	if ledger[0].SignerMethod != SignerEID || ledger[0].SignatureHash == "" {
		t.Errorf("ledger row = %+v, want an E-ID row referencing the approval", ledger[0])
	}

	// A decided document is not signable again — the refusal comes before anybody
	// is asked to approve anything.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "BB90010111"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("second sign: got %v, want ErrNotSignable", err)
	}
}

// An approval is given for one document, against a display text naming it. It
// must not be redeemable against another, or the citizen's consent has been moved
// to something they never read.
func TestAnApprovalCannotBeMovedToAnotherDocument(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	signMe, err := f.m.CreateDocument(ctx, f.tenantID, "Иргэн харсан гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	other, err := f.m.CreateDocument(ctx, f.tenantID, "Иргэн хараагүй гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create other: %v", err)
	}

	session, err := f.m.StartEIDSignature(ctx, f.tenantID, signMe.ID, "AA90010111")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if _, err := f.m.PollEIDSignature(ctx, f.tenantID, other.ID, session.SessionID); !errors.Is(err, ErrSignSessionUnknown) {
		t.Errorf("polling another document with this session: got %v, want ErrSignSessionUnknown", err)
	}
	// And the document the citizen never saw stays unsigned.
	untouched, err := f.m.getDocument(ctx, f.tenantID, other.ID)
	if err != nil {
		t.Fatalf("read other: %v", err)
	}
	if untouched.SignatureCount != 0 || untouched.Status != StatusPending {
		t.Errorf("other document = %d signature(s), status %q; want it untouched", untouched.SignatureCount, untouched.Status)
	}

	// This same session, polled against the document it was started for, signs it.
	signed, err := pollUntilApproved(t, f, signMe.ID, session.SessionID)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// And it is spent: polling again reports the document it already signed rather
	// than signing a second time.
	again, err := f.m.PollEIDSignature(ctx, f.tenantID, signMe.ID, session.SessionID)
	if err != nil {
		t.Fatalf("re-poll: %v", err)
	}
	if again.State != ApprovalComplete || again.Document == nil {
		t.Fatalf("re-poll = %+v, want the signature it already produced", again)
	}
	if again.Document.SignatureCount != signed.SignatureCount {
		t.Errorf("re-poll changed the count to %d, want %d", again.Document.SignatureCount, signed.SignatureCount)
	}
}

// Another tenant cannot poll a session even holding its id.
func TestASignSessionIsScopedToItsTenant(t *testing.T) {
	owner := newFixture(t)
	intruder := newFixture(t)
	ctx := context.Background()

	doc, err := owner.m.CreateDocument(ctx, owner.tenantID, "Бусдын гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	session, err := owner.m.StartEIDSignature(ctx, owner.tenantID, doc.ID, "AA90010111")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if _, err := intruder.m.PollEIDSignature(ctx, intruder.tenantID, doc.ID, session.SessionID); !errors.Is(err, ErrSignSessionUnknown) {
		t.Errorf("cross-tenant poll: got %v, want ErrSignSessionUnknown", err)
	}
}

func TestAChainOfTwoNeedsTwoDifferentSigners(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "REQUEST", []WorkflowStep{
		{Name: "Хэлтсийн дарга"},
		{Name: "Захирал"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Албан хүсэлт", "REQUEST")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if doc.RequiredSignatures != 2 {
		t.Fatalf("required = %d, want 2", doc.RequiredSignatures)
	}

	first, err := signWithEID(t, f, doc.ID, "AA90010111")
	if err != nil {
		t.Fatalf("first sign: %v", err)
	}
	if first.Status != StatusPending {
		t.Errorf("after one of two signatures status = %q, want it to stay %q", first.Status, StatusPending)
	}
	if first.SignatureCount != 1 {
		t.Errorf("progress = %d/2, want 1/2", first.SignatureCount)
	}

	// The same citizen approving again is the same approval, not the second one.
	if _, err := signWithEID(t, f, doc.ID, "AA90010111"); !errors.Is(err, ErrAlreadySigned) {
		t.Errorf("same signer twice: got %v, want ErrAlreadySigned", err)
	}

	second, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "BB90010111", "123456")
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}
	if second.Status != StatusApproved {
		t.Errorf("status = %q, want %q once the chain is complete", second.Status, StatusApproved)
	}
	if second.SignatureCount != 2 {
		t.Errorf("progress = %d/2, want 2/2", second.SignatureCount)
	}
	// The record mirrors the newest signature, which is the DAN one.
	if second.SignerMethod != SignerDAN {
		t.Errorf("mirrored method = %q, want %q", second.SignerMethod, SignerDAN)
	}
}

func TestPolicyRefusesAChannelAndAnUnnamedSigner(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.SaveSignaturePolicy(ctx, f.tenantID, SignaturePolicy{
		DocType: "APPROVAL", AllowEID: true, AllowDAN: false,
	}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	// Requiring a named signer cannot be saved while the chain names nobody: it
	// would leave the type unsignable by anyone.
	if _, err := f.m.SaveSignaturePolicy(ctx, f.tenantID, SignaturePolicy{
		DocType: "APPROVAL", AllowEID: true, RequireNamedSigner: true,
	}); !errors.Is(err, ErrInvalidConfiguration) {
		t.Errorf("policy with no named step: got %v, want ErrInvalidConfiguration", err)
	}

	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "APPROVAL", []WorkflowStep{
		{Name: "Захирал", SignerRegNumber: "CC90010111"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}
	if _, err := f.m.SaveSignaturePolicy(ctx, f.tenantID, SignaturePolicy{
		DocType: "APPROVAL", AllowEID: true, RequireNamedSigner: true,
	}); err != nil {
		t.Fatalf("save policy once a signer is named: %v", err)
	}

	// The document takes the chain as it stands now, so it is created after it.
	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Дотоод батламж", "APPROVAL")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "CC90010111", "123456"); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("DAN against an E-ID-only policy: got %v, want ErrSignatureRejected", err)
	}

	// The refusal lands at start: there is no reason to push a request to somebody
	// whose approval could never be recorded.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "AA90010111"); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("signer the step does not name: got %v, want ErrSignatureRejected", err)
	}
	if _, err := signWithEID(t, f, doc.ID, "CC90010111"); err != nil {
		t.Errorf("the named signer must be accepted: %v", err)
	}

	// And the chain cannot be emptied of named signers while the policy demands one.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "APPROVAL", []WorkflowStep{{Name: "Хэн ч"}}); !errors.Is(err, ErrInvalidConfiguration) {
		t.Errorf("chain dropping its named signer: got %v, want ErrInvalidConfiguration", err)
	}
}

// A named step is binding on its own — the signature policy decides whether OPEN
// steps are allowed at all, not whether a named one means anything. A tenant that
// names its approvers and never opens the policy screen still gets them enforced.
func TestANamedStepIsBindingWithoutThePolicyFlag(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "REQUEST", []WorkflowStep{
		{Name: "Хэлтсийн дарга", SignerRegNumber: "AA90010111"},
		{Name: "Захирал"}, // open on purpose: anyone may take the second approval
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Албан хүсэлт", "REQUEST")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Step one names the department head, and the policy was never touched.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "ZZ99999999"); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("a stranger taking a named step: got %v, want ErrSignatureRejected", err)
	}
	if _, err := signWithEID(t, f, doc.ID, "AA90010111"); err != nil {
		t.Fatalf("the named signer must be accepted: %v", err)
	}

	// Step two names nobody, so anyone allowed to sign may finish it.
	done, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "ZZ99999999", "123456")
	if err != nil {
		t.Fatalf("open step: %v", err)
	}
	if done.Status != StatusApproved {
		t.Errorf("status = %q, want %q", done.Status, StatusApproved)
	}

	ledger, err := f.m.ListSignatures(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("signatures: %v", err)
	}
	if len(ledger) != 2 || ledger[0].StepOrder != 1 || ledger[1].StepOrder != 2 {
		t.Errorf("ledger = %+v, want the two signatures recorded against steps 1 and 2", ledger)
	}
	if ledger[0].SignerRegNumber != "AA90010111" {
		t.Errorf("step 1 was filled by %q, want the citizen it names", ledger[0].SignerRegNumber)
	}
}

// What a document needed is its own property. Editing the type's chain afterwards
// must not change what an in-flight document is waiting for, nor rewrite what a
// finished one required.
func TestAChainEditDoesNotReachDocumentsAlreadyWaiting(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "Нэг"}, {Name: "Хоёр"}, {Name: "Гурав"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	inFlight, err := f.m.CreateDocument(ctx, f.tenantID, "Гурван гарын үсэгтэй гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if inFlight.RequiredSignatures != 3 {
		t.Fatalf("required = %d, want 3", inFlight.RequiredSignatures)
	}
	if _, err := signWithEID(t, f, inFlight.ID, "AA90010111"); err != nil {
		t.Fatalf("first sign: %v", err)
	}

	// Shrinking the chain used to strand this document: its progress would read
	// 1/1 "complete" while its status stayed PENDING, and no further signature
	// could finish it.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{{Name: "Зөвхөн нэг"}}); err != nil {
		t.Fatalf("shrink chain: %v", err)
	}

	still, err := f.m.getDocument(ctx, f.tenantID, inFlight.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if still.RequiredSignatures != 3 {
		t.Errorf("required = %d, want it to stay 3 — the document was routed under a three-step chain", still.RequiredSignatures)
	}
	if still.Status != StatusPending {
		t.Errorf("status = %q, want %q", still.Status, StatusPending)
	}

	// And it can still be finished, on the terms it started with.
	if _, err := f.m.SignWithDAN(ctx, f.tenantID, inFlight.ID, "BB90010111", "123456"); err != nil {
		t.Fatalf("second sign: %v", err)
	}
	finished, err := signWithEID(t, f, inFlight.ID, "CC90010111")
	if err != nil {
		t.Fatalf("third sign: %v", err)
	}
	if finished.Status != StatusApproved {
		t.Errorf("status = %q, want %q on the third of three", finished.Status, StatusApproved)
	}

	// A document created after the edit takes the new chain.
	fresh, err := f.m.CreateDocument(ctx, f.tenantID, "Нэг гарын үсэгтэй гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create fresh: %v", err)
	}
	if fresh.RequiredSignatures != 1 {
		t.Errorf("required = %d, want 1 from the chain as it now stands", fresh.RequiredSignatures)
	}

	// And the finished document's history stays what it was, whatever happens next.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}, {Name: "E"},
	}); err != nil {
		t.Fatalf("grow chain: %v", err)
	}
	decided, err := f.m.getDocument(ctx, f.tenantID, inFlight.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if decided.RequiredSignatures != 3 || decided.SignatureCount != 3 {
		t.Errorf("decided document reads %d/%d, want 3/3 for ever",
			decided.SignatureCount, decided.RequiredSignatures)
	}
	if decided.Status != StatusApproved {
		t.Errorf("status = %q, want it to stay %q", decided.Status, StatusApproved)
	}
}

// The policy screen and the chain screen edit one setting between them: a policy
// that only accepts named signers needs a chain that names one per step. Saved
// concurrently against each other's stale state, both guards used to pass and the
// pair committed exactly what they exist to prevent — a type nobody could sign.
// One of the two must lose.
func TestThePolicyAndTheChainCannotLockATypeOutTogether(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// A chain that names its only signer, and a policy that does not yet insist.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "REQUEST", []WorkflowStep{
		{Name: "Захирал", SignerRegNumber: "CC90010111"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	// One request opens the step; the other demands that no step be open.
	start := make(chan struct{})
	var chainErr, policyErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, chainErr = f.m.ReplaceWorkflow(ctx, f.tenantID, "REQUEST", []WorkflowStep{{Name: "Хэн ч"}})
	}()
	go func() {
		defer wg.Done()
		<-start
		_, policyErr = f.m.SaveSignaturePolicy(ctx, f.tenantID, SignaturePolicy{
			DocType: "REQUEST", AllowEID: true, AllowDAN: true, RequireNamedSigner: true,
		})
	}()
	close(start)
	wg.Wait()

	if chainErr == nil && policyErr == nil {
		t.Error("both saves succeeded, so the type is now unsignable by anybody")
	}

	// Whatever won, the stored pair has to be one a citizen could satisfy.
	policy, err := f.m.SignaturePolicyFor(ctx, f.tenantID, "REQUEST")
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	chains, err := f.m.ListWorkflows(ctx, f.tenantID)
	if err != nil {
		t.Fatalf("read chains: %v", err)
	}
	var steps []WorkflowStep
	for _, chain := range chains {
		if chain.DocType == "REQUEST" {
			steps = chain.Steps
		}
	}
	if policy.RequireNamedSigner {
		if err := stepsCanRequireNamedSigners("REQUEST", steps); err != nil {
			t.Errorf("stored state cannot be satisfied by anyone: %v (chain %+v)", err, steps)
		}
	}

	// And the type is still signable in practice.
	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Албан хүсэлт", "REQUEST")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	signer := "CC90010111"
	if !policy.RequireNamedSigner && len(steps) > 0 && steps[0].SignerRegNumber == "" {
		signer = "AA90010111" // the open step lets anyone in
	}
	if _, err := signWithEID(t, f, doc.ID, signer); err != nil {
		t.Errorf("nobody can sign a %s any more: %v", "REQUEST", err)
	}
}

// A chain naming one citizen at two steps could never be completed — they sign a
// document once — so it has to be refused when it is saved, whatever the signature
// policy says. The old guard only ran when the policy demanded named signers.
func TestAChainCannotNameOneCitizenTwice(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	_, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "Хэлтсийн дарга", SignerRegNumber: "AA90010111"},
		{Name: "Захирал", SignerRegNumber: "AA90010111"},
	})
	if !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("got %v, want ErrInvalidConfiguration", err)
	}
	if !strings.Contains(err.Error(), "AA90010111") {
		t.Errorf("got %q, want it to name the repeated citizen", err)
	}

	// Case matters as little here as anywhere else: the check runs on the
	// normalised value.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "Нэг", SignerRegNumber: "aa90010111"},
		{Name: "Хоёр", SignerRegNumber: "AA90010111 "},
	}); !errors.Is(err, ErrInvalidConfiguration) {
		t.Errorf("same citizen written two ways: got %v, want ErrInvalidConfiguration", err)
	}
}

// An open step is open to anyone — except somebody the chain still needs further
// along. Letting them spend their one signature early would leave their own step
// unfillable and the document unapprovable, with a legal chain and no policy flag
// involved.
func TestAnOpenStepHoldsBackALaterStepsSigner(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// The natural ordering: a reviewer anyone may be, then the director.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "Хянагч"}, // open
		{Name: "Захирал", SignerRegNumber: "AA90010111"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Хоёр шаттай гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// The director cannot take the reviewer's step, even though it names nobody.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "AA90010111"); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("the director taking the open step: got %v, want ErrSignatureRejected", err)
	}

	// Anyone else may.
	if _, err := signWithEID(t, f, doc.ID, "ZZ99999999"); err != nil {
		t.Fatalf("open step: %v", err)
	}
	// And then the director finishes their own.
	done, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "AA90010111", "123456")
	if err != nil {
		t.Fatalf("director: %v", err)
	}
	if done.Status != StatusApproved {
		t.Errorf("status = %q, want %q", done.Status, StatusApproved)
	}
}

// Asking a citizen to approve on their own device and then throwing the approval
// away because they had already signed is worse than not asking.
func TestARepeatSignerIsRefusedBeforeTheCitizenIsAsked(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "REQUEST", []WorkflowStep{
		{Name: "Нэг"}, {Name: "Хоёр"}, // both open
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Хоёр нээлттэй шат", "REQUEST")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := signWithEID(t, f, doc.ID, "AA90010111"); err != nil {
		t.Fatalf("first signature: %v", err)
	}

	// The same citizen for the second, still-open step: refused at start, before a
	// request reaches their phone.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "AA90010111"); !errors.Is(err, ErrAlreadySigned) {
		t.Errorf("eid start: got %v, want ErrAlreadySigned", err)
	}
	if _, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "AA90010111", "123456"); !errors.Is(err, ErrAlreadySigned) {
		t.Errorf("dan: got %v, want ErrAlreadySigned", err)
	}

	// Somebody else still finishes it.
	done, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "BB90010111", "123456")
	if err != nil {
		t.Fatalf("second signer: %v", err)
	}
	if done.Status != StatusApproved {
		t.Errorf("status = %q, want %q", done.Status, StatusApproved)
	}
}

// The citizen has already approved by the time the signature is written, so an
// operator closing the dialog — which aborts the poll and cancels its context —
// must not destroy it.
func TestASignatureSurvivesTheCallerHangingUp(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Цуцлагдсан хүсэлт", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	session, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "AA90010111")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	// Wait for the mock citizen to approve, then poll on a context that is
	// cancelled the instant the write begins — the shape of a dialog being closed.
	time.Sleep(1800 * time.Millisecond)
	cancelled, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	_, _ = f.m.PollEIDSignature(cancelled, f.tenantID, doc.ID, session.SessionID)

	// Whatever the caller was told, the citizen's approval is either fully applied
	// or not applied at all — never half.
	after, err := f.m.getDocument(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	ledger, err := f.m.ListSignatures(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("signatures: %v", err)
	}
	if len(ledger) != after.SignatureCount {
		t.Errorf("ledger has %d rows but the document reports %d", len(ledger), after.SignatureCount)
	}
	if after.SignatureCount == 1 && after.Status != StatusApproved {
		t.Errorf("a single-signature document with its signature must be %q, got %q", StatusApproved, after.Status)
	}
	// And the session cannot be spent twice: a later poll reports the same outcome.
	if after.SignatureCount == 1 {
		again, err := f.m.PollEIDSignature(ctx, f.tenantID, doc.ID, session.SessionID)
		if err != nil {
			t.Fatalf("re-poll: %v", err)
		}
		if again.State != ApprovalComplete || again.Document == nil || again.Document.SignatureCount != 1 {
			t.Errorf("re-poll = %+v, want the one signature it already produced", again)
		}
	}
}

// A document may hold a signature on a LATER step while an earlier one is still
// open: migration 00014 credits a pre-existing signature to the step that names its
// signer, not to its place in time, because order did not matter before. The next
// approval is therefore the lowest unfilled step — not "one more than the count",
// which would ask a citizen who has already signed and stick for ever.
func TestASignatureOnALaterStepLeavesTheEarlierOneOpen(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "Захирал", SignerRegNumber: "AA90010111"},
		{Name: "Ня-бо", SignerRegNumber: "BB90010111"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Дараалал зөрсөн гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Stand in for what the migration leaves behind: the accountant signed first,
	// credited to the step that names them.
	if _, err := f.m.db.Exec(ctx,
		`INSERT INTO document_signatures
		        (tenant_id, document_id, signer_name, signer_reg_number, signer_method,
		         signature_hash, signed_at, step_order)
		 VALUES ($1, $2, 'Ня-бо', 'BB90010111', 'EID', 'legacy', NOW() - INTERVAL '2 days', 2)`,
		f.tenantID, doc.ID); err != nil {
		t.Fatalf("insert legacy signature: %v", err)
	}

	// One signature, but step 1 is what is missing — and it belongs to the director.
	// Anyone else is turned away by whose step it is, which is the more useful thing
	// to be told: the accountant is not asked to sign twice, they are told it is the
	// director's turn.
	for _, who := range []string{"BB90010111", "ZZ99999999"} {
		err := func() error {
			_, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, who)
			return err
		}()
		if !errors.Is(err, ErrSignatureRejected) {
			t.Errorf("%s taking the director's step: got %v, want ErrSignatureRejected", who, err)
		}
		if err != nil && !strings.Contains(err.Error(), "AA90010111") {
			t.Errorf("%s: got %q, want it to say whose step it is", who, err)
		}
	}

	done, err := signWithEID(t, f, doc.ID, "AA90010111")
	if err != nil {
		t.Fatalf("the director must be able to finish it: %v", err)
	}
	if done.Status != StatusApproved {
		t.Errorf("status = %q, want %q once both steps are filled", done.Status, StatusApproved)
	}
	if done.SignatureCount != 2 || done.RequiredSignatures != 2 {
		t.Errorf("progress = %d/%d, want 2/2", done.SignatureCount, done.RequiredSignatures)
	}

	ledger, err := f.m.ListSignatures(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("signatures: %v", err)
	}
	if len(ledger) != 2 || ledger[0].StepOrder != 1 || ledger[1].StepOrder != 2 {
		t.Errorf("ledger = %+v, want step 1 then step 2", ledger)
	}
	if ledger[0].SignerRegNumber != "AA90010111" || ledger[1].SignerRegNumber != "BB90010111" {
		t.Errorf("each step must be filled by the citizen it names, got %q then %q",
			ledger[0].SignerRegNumber, ledger[1].SignerRegNumber)
	}
}

// A chain naming one citizen twice cannot be saved any more, but chains stored
// before that rule existed still can be. The path that decides who may sign must
// never copy one verbatim: the second step would be owed to somebody who has
// already signed, and no document of that type could ever be approved.
func TestASnapshotOpensARepeatedSigner(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Straight into the table, the way a chain stored under the old rules looks.
	for order, name := range map[int]string{1: "Ня-бо", 2: "Захирал"} {
		if _, err := f.m.db.Exec(ctx,
			`INSERT INTO document_workflow_steps (tenant_id, doc_type, step_order, name, signer_reg_number)
			      VALUES ($1, 'CONTRACT', $2, $3, 'AA90010111')`,
			f.tenantID, order, name); err != nil {
			t.Fatalf("insert legacy step %d: %v", order, err)
		}
	}

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Хуучин давхар хэлхээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if doc.RequiredSignatures != 2 {
		t.Fatalf("required = %d, want 2 — the tenant asked for two approvals", doc.RequiredSignatures)
	}

	steps, err := f.m.DocumentSteps(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("steps: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("got %d steps, want 2", len(steps))
	}
	if steps[0].SignerRegNumber != "AA90010111" {
		t.Errorf("step 1 = %q, want the citizen the chain names", steps[0].SignerRegNumber)
	}
	if steps[1].SignerRegNumber != "" {
		t.Errorf("step 2 = %q, want it copied open — the same citizen cannot fill both",
			steps[1].SignerRegNumber)
	}

	// And it can actually be finished: the named citizen, then anyone else.
	if _, err := signWithEID(t, f, doc.ID, "AA90010111"); err != nil {
		t.Fatalf("the named signer: %v", err)
	}
	done, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "ZZ99999999", "123456")
	if err != nil {
		t.Fatalf("the open step: %v", err)
	}
	if done.Status != StatusApproved {
		t.Errorf("status = %q, want %q — a legacy chain must not strand its documents",
			done.Status, StatusApproved)
	}
}

func TestRejectIsFinal(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Татгалзах гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rejected, err := f.m.RejectDocument(ctx, f.tenantID, doc.ID)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Errorf("status = %q, want %q", rejected.Status, StatusRejected)
	}

	if _, err := f.m.RejectDocument(ctx, f.tenantID, doc.ID); !errors.Is(err, ErrNotSignable) {
		t.Errorf("second reject: got %v, want ErrNotSignable", err)
	}
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "AA90010111"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("signing a rejected document: got %v, want ErrNotSignable", err)
	}
}

func TestAnotherTenantsDocumentIsNotSignable(t *testing.T) {
	owner := newFixture(t)
	intruder := newFixture(t)
	ctx := context.Background()

	doc, err := owner.m.CreateDocument(ctx, owner.tenantID, "Бусдын гэрээ", "CONTRACT")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := intruder.m.StartEIDSignature(ctx, intruder.tenantID, doc.ID, "AA90010111"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("cross-tenant sign: got %v, want ErrNotSignable", err)
	}
	if _, err := intruder.m.RejectDocument(ctx, intruder.tenantID, doc.ID); !errors.Is(err, ErrNotSignable) {
		t.Errorf("cross-tenant reject: got %v, want ErrNotSignable", err)
	}

	list, err := intruder.m.ListDocuments(ctx, intruder.tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("intruder sees %d documents, want 0", len(list))
	}
}

func TestTemplatesCreateDocumentsAndKeepNamesUnique(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	tpl, err := f.m.CreateTemplate(ctx, f.tenantID, "Гэрээний загвар", "CONTRACT", "Хамтран ажиллах гэрээ {year}")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	if _, err := f.m.CreateTemplate(ctx, f.tenantID, "Гэрээний загвар", "REQUEST", "Дахин"); !errors.Is(err, ErrTemplateNameTaken) {
		t.Errorf("duplicate name: got %v, want ErrTemplateNameTaken", err)
	}

	doc, err := f.m.CreateDocumentFromTemplate(ctx, f.tenantID, tpl.ID)
	if err != nil {
		t.Fatalf("use template: %v", err)
	}
	if doc.DocType != "CONTRACT" {
		t.Errorf("doc_type = %q, want CONTRACT", doc.DocType)
	}
	if doc.Title == tpl.TitlePattern {
		t.Errorf("title %q still holds the pattern; {year} was not resolved", doc.Title)
	}
	if doc.Status != StatusPending {
		t.Errorf("a templated document must be routed like any other, got %q", doc.Status)
	}

	// Editing one changes what it will produce next, and retiring it stops it
	// producing anything without destroying the record of what it was.
	renamed, err := f.m.UpdateTemplate(ctx, f.tenantID, tpl.ID, "  Гэрээ 2027  ", "request", "Албан хүсэлт {year}", true)
	if err != nil {
		t.Fatalf("update template: %v", err)
	}
	if renamed.Name != "Гэрээ 2027" || renamed.DocType != "REQUEST" {
		t.Errorf("update = %+v, want the trimmed name and the normalised type", renamed)
	}
	if _, err := f.m.UpdateTemplate(ctx, f.tenantID, tpl.ID, "", "REQUEST", "x", true); !errors.Is(err, ErrInvalidConfiguration) {
		t.Errorf("empty name: got %v, want ErrInvalidConfiguration", err)
	}

	retired, err := f.m.UpdateTemplate(ctx, f.tenantID, tpl.ID, "Гэрээ 2027", "REQUEST", "Албан хүсэлт {year}", false)
	if err != nil {
		t.Fatalf("retire template: %v", err)
	}
	if retired.Active {
		t.Error("the template was not retired")
	}
	if _, err := f.m.CreateDocumentFromTemplate(ctx, f.tenantID, tpl.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("using a retired template: got %v, want ErrTemplateNotFound", err)
	}

	// A name another template already holds is a conflict, not a validation error.
	other, err := f.m.CreateTemplate(ctx, f.tenantID, "Өөр загвар", "CONTRACT", "Өөр {year}")
	if err != nil {
		t.Fatalf("create second template: %v", err)
	}
	if _, err := f.m.UpdateTemplate(ctx, f.tenantID, other.ID, "Гэрээ 2027", "CONTRACT", "Өөр {year}", true); !errors.Is(err, ErrTemplateNameTaken) {
		t.Errorf("duplicate name on update: got %v, want ErrTemplateNameTaken", err)
	}

	if err := f.m.DeleteTemplate(ctx, f.tenantID, tpl.ID); err != nil {
		t.Fatalf("delete template: %v", err)
	}
	if err := f.m.DeleteTemplate(ctx, f.tenantID, tpl.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("second delete: got %v, want ErrTemplateNotFound", err)
	}
	// The document the template produced outlives the template.
	if _, err := f.m.getDocument(ctx, f.tenantID, doc.ID); err != nil {
		t.Errorf("document should survive its template: %v", err)
	}
}

func TestRouteMovesADraftAndOnlyADraft(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Nothing the app creates is a draft, so the row is made the way one would
	// actually arrive: straight into the table at the column default.
	var draftID string
	if err := f.m.db.QueryRow(ctx,
		`INSERT INTO document_records (tenant_id, title, doc_type) VALUES ($1, $2, $3) RETURNING id`,
		f.tenantID, "Ноорог гэрээ", "CONTRACT").Scan(&draftID); err != nil {
		t.Fatalf("insert draft: %v", err)
	}

	draft, err := f.m.getDocument(ctx, f.tenantID, draftID)
	if err != nil {
		t.Fatalf("read draft: %v", err)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("status = %q, want the column default %q", draft.Status, StatusDraft)
	}

	// A draft is not signable until it has been routed.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, draftID, "AA90010111"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("signing a draft: got %v, want ErrNotSignable", err)
	}

	routed, err := f.m.RouteDocument(ctx, f.tenantID, draftID)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if routed.Status != StatusPending {
		t.Errorf("status = %q, want %q", routed.Status, StatusPending)
	}
	if _, err := f.m.RouteDocument(ctx, f.tenantID, draftID); !errors.Is(err, ErrNotRoutable) {
		t.Errorf("routing twice: got %v, want ErrNotRoutable", err)
	}
	if _, err := signWithEID(t, f, draftID, "AA90010111"); err != nil {
		t.Errorf("a routed draft must be signable: %v", err)
	}
}

func TestRetentionReportsWhatIsPastItsTerm(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.m.CreateDocument(ctx, f.tenantID, "Шинэ гэрээ", "CONTRACT"); err != nil {
		t.Fatalf("create: %v", err)
	}

	var oldID string
	if err := f.m.db.QueryRow(ctx,
		`INSERT INTO document_records (tenant_id, title, doc_type, status, created_at)
		      VALUES ($1, $2, 'CONTRACT', $3, NOW() - INTERVAL '6 years') RETURNING id`,
		f.tenantID, "Хуучин гэрээ", StatusApproved).Scan(&oldID); err != nil {
		t.Fatalf("insert aged document: %v", err)
	}

	if _, err := f.m.SaveRetentionRule(ctx, f.tenantID, "CONTRACT", 5, "Гэрээг 5 жил хадгална"); err != nil {
		t.Fatalf("save rule: %v", err)
	}
	if _, err := f.m.SaveRetentionRule(ctx, f.tenantID, "CONTRACT", 0, ""); err == nil {
		t.Error("expected a zero-year term to be refused")
	}

	rules, err := f.m.ListRetentionRules(ctx, f.tenantID)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != len(DocTypes) {
		t.Fatalf("got %d rows, want one per document type (%d)", len(rules), len(DocTypes))
	}

	for _, rule := range rules {
		switch rule.DocType {
		case "CONTRACT":
			if !rule.Configured || rule.RetainYears != 5 {
				t.Errorf("CONTRACT rule = %+v, want a stored 5-year term", rule)
			}
			if rule.Total != 2 {
				t.Errorf("CONTRACT total = %d, want 2", rule.Total)
			}
			if rule.Expired != 1 {
				t.Errorf("CONTRACT expired = %d, want 1 (the six-year-old document)", rule.Expired)
			}
		default:
			if rule.Configured {
				t.Errorf("%s must stay unconfigured", rule.DocType)
			}
			if rule.Expired != 0 {
				t.Errorf("%s expired = %d, want 0 without a term", rule.DocType, rule.Expired)
			}
		}
	}
}

func TestPoliciesAndChainsCoverEveryDocumentType(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	policies, err := f.m.ListSignaturePolicies(ctx, f.tenantID)
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if len(policies) != len(DocTypes) {
		t.Fatalf("got %d policies, want one per document type (%d)", len(policies), len(DocTypes))
	}
	for _, policy := range policies {
		if policy.Configured {
			t.Errorf("%s: a fresh tenant has configured nothing", policy.DocType)
		}
		if !policy.AllowEID || !policy.AllowDAN {
			t.Errorf("%s: an unconfigured type must accept both channels", policy.DocType)
		}
	}

	chains, err := f.m.ListWorkflows(ctx, f.tenantID)
	if err != nil {
		t.Fatalf("list chains: %v", err)
	}
	if len(chains) != len(DocTypes) {
		t.Fatalf("got %d chains, want one per document type (%d)", len(chains), len(DocTypes))
	}
	for _, chain := range chains {
		if len(chain.Steps) != 0 {
			t.Errorf("%s: a fresh tenant has no steps, got %d", chain.DocType, len(chain.Steps))
		}
	}

	// Replacing a chain is a swap, not an append.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{
		{Name: "Нэг"}, {Name: "Хоёр"}, {Name: "Гурав"},
	}); err != nil {
		t.Fatalf("configure chain: %v", err)
	}
	replaced, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{{Name: "Зөвхөн нэг"}})
	if err != nil {
		t.Fatalf("replace chain: %v", err)
	}
	if len(replaced.Steps) != 1 || replaced.Steps[0].Order != 1 {
		t.Errorf("chain = %+v, want a single step numbered 1", replaced.Steps)
	}

	// A step with no name is a step nobody can read.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "CONTRACT", []WorkflowStep{{Name: "  "}}); err == nil {
		t.Error("expected a nameless step to be refused")
	}
}
