package documents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
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
	if !strings.Contains(session.DisplayText, "гарын үсэг") {
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

	doc, err := f.m.CreateDocument(ctx, f.tenantID, "Дотоод батламж", "APPROVAL")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := f.m.SignWithDAN(ctx, f.tenantID, doc.ID, "AA90010111", "123456"); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("DAN against an E-ID-only policy: got %v, want ErrSignatureRejected", err)
	}

	// Requiring a named signer cannot be saved while the chain names nobody: it
	// would leave the type unsignable by anyone.
	if _, err := f.m.SaveSignaturePolicy(ctx, f.tenantID, SignaturePolicy{
		DocType: "APPROVAL", AllowEID: true, RequireNamedSigner: true,
	}); err == nil {
		t.Error("expected the policy to be refused while the chain names nobody")
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

	// The refusal lands at start: there is no reason to push a request to somebody
	// whose approval could never be recorded.
	if _, err := f.m.StartEIDSignature(ctx, f.tenantID, doc.ID, "AA90010111"); !errors.Is(err, ErrSignatureRejected) {
		t.Errorf("signer the chain does not name: got %v, want ErrSignatureRejected", err)
	}
	if _, err := signWithEID(t, f, doc.ID, "CC90010111"); err != nil {
		t.Errorf("the named signer must be accepted: %v", err)
	}

	// And the chain cannot be emptied of named signers while the policy demands one.
	if _, err := f.m.ReplaceWorkflow(ctx, f.tenantID, "APPROVAL", []WorkflowStep{{Name: "Хэн ч"}}); err == nil {
		t.Error("expected the chain to be refused while the policy requires a named signer")
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
