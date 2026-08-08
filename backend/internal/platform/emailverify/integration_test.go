package emailverify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/mailer"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise the flow against a real schema, because what they
// protect lives partly in SQL: the single conditional UPDATE that makes a link
// good exactly once, the unique key hash, the cascade that takes a tenant's
// clients with it, and the counts the rate limits are computed from.
//
// They are skipped unless a migrated throwaway database is provided:
//
//	EMAILVERIFY_TEST_DATABASE_URL=postgres://... go test ./internal/platform/emailverify/...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("EMAILVERIFY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set EMAILVERIFY_TEST_DATABASE_URL to a migrated test database to run email verification integration tests")
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

// captureMailer stands in for the outbound queue. The token only ever exists in
// the mail, which is the point — so the tests read it the way a recipient does.
type captureMailer struct {
	mu       sync.Mutex
	sent     []mailer.EmailMessage
	rejectAt int // after this many messages, refuse to accept more
}

func (m *captureMailer) EnqueueMessage(msg mailer.EmailMessage) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rejectAt > 0 && len(m.sent) >= m.rejectAt {
		return false
	}
	m.sent = append(m.sent, msg)
	return true
}

func (m *captureMailer) last(t *testing.T) mailer.EmailMessage {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		t.Fatal("no mail was queued")
	}
	return m.sent[len(m.sent)-1]
}

func (m *captureMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

type fixture struct {
	svc      *Service
	mail     *captureMailer
	pool     *pgxpool.Pool
	tenantID string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("PUBLIC_ORIGIN", "https://nexus.test")

	pool := testPool(t)
	mail := &captureMailer{}

	var tenantID string
	slug := fmt.Sprintf("emailverify-test-%d-%s", time.Now().UnixNano(), strings.ToLower(t.Name()))
	if len(slug) > 60 {
		slug = slug[:60]
	}
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id::text`,
		slug, "Email verification integration test").Scan(&tenantID); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	t.Cleanup(func() {
		// The schema cascades from tenants, so one delete clears this test's
		// clients and verifications with it.
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	return &fixture{svc: NewService(pool, mail), mail: mail, pool: pool, tenantID: tenantID}
}

// tokenFromMail reads the link the way the recipient's browser would.
func tokenFromMail(t *testing.T, msg mailer.EmailMessage) string {
	t.Helper()
	const marker = "?token="
	index := strings.Index(msg.Body, marker)
	if index < 0 {
		t.Fatalf("the mail carries no token: %s", msg.Body)
	}
	token := msg.Body[index+len(marker):]
	if cut := strings.IndexAny(token, "\r\n "); cut >= 0 {
		token = token[:cut]
	}
	if token == "" {
		t.Fatal("the mail carries an empty token")
	}
	return token
}

// The whole point of the feature: a link that works once.
func TestALinkIsGoodExactlyOnce(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	sent, err := f.svc.Send(ctx, f.tenantID, Request{
		Email:       "User@Example.com",
		RedirectURL: "https://theirapp.com/verified",
		Purpose:     "signup",
		Source:      "io.example.contacts",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if sent.Status != StatusPending {
		t.Fatalf("a freshly issued link is %q, want PENDING", sent.Status)
	}
	if sent.Email != "user@example.com" {
		t.Fatalf("the address was stored as %q, want it normalised", sent.Email)
	}

	token := tokenFromMail(t, f.mail.last(t))

	confirmed, err := f.svc.Confirm(ctx, token)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != StatusVerified || confirmed.VerifiedAt == nil {
		t.Fatalf("confirm returned %+v, want a verified row with a timestamp", confirmed)
	}
	if confirmed.RedirectURL != "https://theirapp.com/verified" {
		t.Fatalf("the destination came back as %q", confirmed.RedirectURL)
	}

	// The second click — a forwarded mail, a browser prefetch, an attacker
	// replaying a link out of a mailbox — gets nothing.
	if _, err := f.svc.Confirm(ctx, token); !errors.Is(err, ErrLinkSpent) {
		t.Fatalf("a replayed link returned %v, want ErrLinkSpent", err)
	}

	// And a token nobody ever issued is refused the same way, so a caller
	// cannot tell a spent link from an invented one.
	if _, err := f.svc.Confirm(ctx, "not-a-token-anybody-issued"); !errors.Is(err, ErrLinkSpent) {
		t.Fatalf("an unissued token returned %v, want ErrLinkSpent", err)
	}
}

// Two clicks arriving together must not both win. A mail client that prefetches
// links races the recipient every time, and a read-then-write would hand both
// of them a success.
func TestConcurrentClicksProduceOneVerification(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.svc.Send(ctx, f.tenantID, Request{Email: "race@example.com"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	token := tokenFromMail(t, f.mail.last(t))

	const clicks = 8
	var wg sync.WaitGroup
	results := make([]error, clicks)
	start := make(chan struct{})
	for i := 0; i < clicks; i++ {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			<-start
			_, results[slot] = f.svc.Confirm(ctx, token)
		}(i)
	}
	close(start)
	wg.Wait()

	won := 0
	for _, err := range results {
		switch {
		case err == nil:
			won++
		case errors.Is(err, ErrLinkSpent):
		default:
			t.Fatalf("unexpected error from a concurrent click: %v", err)
		}
	}
	if won != 1 {
		t.Fatalf("%d of %d concurrent clicks succeeded, want exactly 1", won, clicks)
	}
}

func TestAnExpiredLinkIsRefused(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	sent, err := f.svc.Send(ctx, f.tenantID, Request{Email: "late@example.com"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	token := tokenFromMail(t, f.mail.last(t))

	// Age the link rather than waiting a day for it.
	if _, err := f.pool.Exec(ctx,
		`UPDATE email_verifications SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1`,
		sent.ID); err != nil {
		t.Fatalf("age the link: %v", err)
	}

	if _, err := f.svc.Confirm(ctx, token); !errors.Is(err, ErrLinkSpent) {
		t.Fatalf("an expired link returned %v, want ErrLinkSpent", err)
	}

	// The sweep is what turns it from "pending" into something the Overview
	// screen stops counting as outstanding.
	f.svc.sweep(ctx)
	var status string
	if err := f.pool.QueryRow(ctx,
		`SELECT status FROM email_verifications WHERE id = $1`, sent.ID).Scan(&status); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if status != string(StatusExpired) {
		t.Fatalf("after the sweep the row is %q, want EXPIRED", status)
	}
}

// Disabling a client has to bite on the very next request, and deleting one has
// to be permanent. Both are what an administrator reaches for when a key leaks.
func TestDisablingAndDeletingRevokeTheKeyImmediately(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	client, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{Name: "Partner platform"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if client.Secret == "" || !strings.HasPrefix(client.Secret, KeyPrefix) {
		t.Fatalf("create returned %q, want a key carrying the %q prefix", client.Secret, KeyPrefix)
	}
	key := client.Secret

	if _, err := f.svc.Authenticate(ctx, key); err != nil {
		t.Fatalf("a fresh key was refused: %v", err)
	}

	// Nothing reads the key back — the table has only its hash.
	listed, err := f.svc.ListClients(ctx, f.tenantID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].Secret != "" {
		t.Fatalf("listing returned %+v, want one client with no secret", listed)
	}
	if !strings.HasPrefix(key, listed[0].KeyPrefix) {
		t.Fatalf("the stored prefix %q does not identify the key", listed[0].KeyPrefix)
	}

	if _, err := f.svc.UpdateClient(ctx, f.tenantID, client.ID, ClientInput{
		Name: client.Name, Status: ClientDisabled, HourlyLimit: client.HourlyLimit,
	}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, err := f.svc.Authenticate(ctx, key); !errors.Is(err, ErrUnauthorizedKey) {
		t.Fatalf("a disabled key returned %v, want ErrUnauthorizedKey", err)
	}

	if _, err := f.svc.UpdateClient(ctx, f.tenantID, client.ID, ClientInput{
		Name: client.Name, Status: ClientActive, HourlyLimit: client.HourlyLimit,
	}); err != nil {
		t.Fatalf("re-enable: %v", err)
	}
	if _, err := f.svc.Authenticate(ctx, key); err != nil {
		t.Fatalf("a re-enabled key was refused: %v", err)
	}

	if err := f.svc.DeleteClient(ctx, f.tenantID, client.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := f.svc.Authenticate(ctx, key); !errors.Is(err, ErrUnauthorizedKey) {
		t.Fatalf("a deleted key returned %v, want ErrUnauthorizedKey", err)
	}
}

// A neighbouring tenant's client id must read as absent, not as forbidden:
// telling the caller it exists is already an answer about somebody else's data.
func TestClientsAreTenantScoped(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	var otherTenant string
	if err := f.pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id::text`,
		fmt.Sprintf("emailverify-neighbour-%d", time.Now().UnixNano()), "Neighbour").Scan(&otherTenant); err != nil {
		t.Fatalf("create neighbour tenant: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, otherTenant) })

	client, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{Name: "Ours"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if _, err := f.svc.UpdateClient(ctx, otherTenant, client.ID, ClientInput{
		Name: "Theirs now", Status: ClientDisabled,
	}); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("a neighbour updating our client got %v, want ErrClientNotFound", err)
	}
	if err := f.svc.DeleteClient(ctx, otherTenant, client.ID); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("a neighbour deleting our client got %v, want ErrClientNotFound", err)
	}
	if listed, err := f.svc.ListClients(ctx, otherTenant); err != nil || len(listed) != 0 {
		t.Fatalf("a neighbour listed %v (err %v), want nothing", listed, err)
	}
}

// One name per tenant: the name is what a key is revoked by during an incident,
// and two clients called the same thing is how the wrong one gets switched off.
func TestClientNamesAreUniquePerTenant(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{Name: "Mobile app"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{Name: "Mobile app"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("a duplicate name returned %v, want ErrDuplicateName", err)
	}
}

// The allowance is what stands between a leaked key and a tenant's domain being
// classified as a mail source worth blocking.
func TestAClientRunsOutOfItsHourlyAllowance(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	client, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{Name: "Throttled", HourlyLimit: 3})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := f.svc.SendForClient(ctx, client, Request{
			Email: fmt.Sprintf("user%d@example.com", i),
		}); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	_, err = f.svc.SendForClient(ctx, client, Request{Email: "user3@example.com"})
	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("the fourth send returned %v, want a rate limit", err)
	}
	if limited.RetryAfter < time.Minute {
		t.Fatalf("Retry-After is %v, want something worth waiting", limited.RetryAfter)
	}
	if f.mail.count() != 3 {
		t.Fatalf("%d mails were queued, want the refused one not to have been sent", f.mail.count())
	}
}

// The recipient never asked for any of this. Writing to the same address twice
// in a minute is the shape of a mail-bombing tool, whoever holds the key.
func TestTheSameAddressIsNotWrittenToTwiceInAMinute(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.svc.Send(ctx, f.tenantID, Request{Email: "target@example.com"}); err != nil {
		t.Fatalf("first send: %v", err)
	}
	_, err := f.svc.Send(ctx, f.tenantID, Request{Email: "TARGET@example.com"})
	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("an immediate resend returned %v, want a rate limit", err)
	}
	if limited.RetryAfter <= 0 || limited.RetryAfter > ResendInterval {
		t.Fatalf("Retry-After is %v, want it inside the resend interval", limited.RetryAfter)
	}
}

// The allowlist is per client, and it is enforced when the link is issued —
// not when it is followed, by which time the mail has already gone out.
func TestAClientCannotRedirectOutsideItsAllowlist(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	client, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{
		Name:                 "Fenced",
		AllowedRedirectHosts: []string{"https://theirapp.com/return"},
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if len(client.AllowedRedirectHosts) != 1 || client.AllowedRedirectHosts[0] != "theirapp.com" {
		t.Fatalf("the allowlist was stored as %v, want the host alone", client.AllowedRedirectHosts)
	}

	if _, err := f.svc.SendForClient(ctx, client, Request{
		Email: "inside@example.com", RedirectURL: "https://theirapp.com/verified",
	}); err != nil {
		t.Fatalf("an allowlisted destination was refused: %v", err)
	}

	_, err = f.svc.SendForClient(ctx, client, Request{
		Email: "outside@example.com", RedirectURL: "https://phish.example/verified",
	})
	var invalidErr *InvalidError
	if !errors.As(err, &invalidErr) {
		t.Fatalf("a destination outside the allowlist returned %v, want a refusal", err)
	}
	if f.mail.count() != 1 {
		t.Fatalf("%d mails were queued, want only the allowed one", f.mail.count())
	}
}

// A link nobody could be told about is worse than a refusal: the Overview
// screen would show a verification that was never asked of anyone.
func TestAnUnsendableMailLeavesNoVerificationBehind(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.mail.rejectAt = 0 // refuse everything
	f.mail.mu.Lock()
	f.mail.rejectAt = 1
	f.mail.sent = append(f.mail.sent, mailer.EmailMessage{}) // the queue is already "full"
	f.mail.mu.Unlock()

	if _, err := f.svc.Send(ctx, f.tenantID, Request{Email: "nowhere@example.com"}); !errors.Is(err, ErrMailUnavailable) {
		t.Fatalf("an unsendable mail returned %v, want ErrMailUnavailable", err)
	}

	var count int
	if err := f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_verifications WHERE tenant_id = $1 AND email = $2`,
		f.tenantID, "nowhere@example.com").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("%d rows survived a mail that was never accepted, want none", count)
	}
}

// The Overview screen is one request, and it has to answer for a client that
// has since been deleted — which is why the source label is kept on the row.
func TestOverviewCountsAndSurvivesADeletedClient(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	client, err := f.svc.CreateClient(ctx, f.tenantID, ClientInput{Name: "Reporter"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := f.svc.SendForClient(ctx, client, Request{Email: "one@example.com"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	token := tokenFromMail(t, f.mail.last(t))
	if _, err := f.svc.Confirm(ctx, token); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := f.svc.SendForClient(ctx, client, Request{Email: "two@example.com"}); err != nil {
		t.Fatalf("second send: %v", err)
	}

	overview, err := f.svc.Overview(ctx, f.tenantID, 25)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.Stats.Total != 2 || overview.Stats.Verified != 1 || overview.Stats.Pending != 1 {
		t.Fatalf("stats are %+v, want 2 total / 1 verified / 1 pending", overview.Stats)
	}
	if overview.Stats.VerifiedPct != 50 {
		t.Fatalf("verified rate is %v, want 50", overview.Stats.VerifiedPct)
	}
	if len(overview.Recent) != 2 {
		t.Fatalf("%d recent rows, want 2", len(overview.Recent))
	}
	if overview.SendURL != "https://nexus.test/api/v1/verify/send" {
		t.Fatalf("the screen was given %q as the send endpoint", overview.SendURL)
	}

	if err := f.svc.DeleteClient(ctx, f.tenantID, client.ID); err != nil {
		t.Fatalf("delete client: %v", err)
	}
	after, err := f.svc.Overview(ctx, f.tenantID, 25)
	if err != nil {
		t.Fatalf("overview after delete: %v", err)
	}
	if len(after.Recent) != 2 {
		t.Fatalf("deleting the client took %d rows of history with it", 2-len(after.Recent))
	}
	for _, row := range after.Recent {
		if row.Source != "Reporter" {
			t.Fatalf("history says it was sent by %q, want the client name kept", row.Source)
		}
		if row.ClientID != "" {
			t.Fatalf("the deleted client is still referenced as %q", row.ClientID)
		}
	}
}
