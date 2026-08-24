package controlplane

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/operator/optest"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/operator"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CP-2's promises, asked of a real database: an organisation cannot be deleted
// by one person, cannot be deleted today, and comes back with one button until
// the grace period runs out. Impersonation leaves a trail on both sides. A
// limit that is hard refuses.

// newTenant makes an organisation for one test and takes it away afterwards.
func newTenant(t *testing.T, pool *pgxpool.Pool) (id, slug string) {
	t.Helper()
	slug = fmt.Sprintf("cp-test-%d", time.Now().UnixNano())
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO tenants (slug, name) VALUES ($1, $1) RETURNING id::text`, slug).
		Scan(&id); err != nil {
		t.Fatalf("create a test organisation: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1::uuid`, id)
	})
	return id, slug
}

// newPerson adds somebody to an organisation.
func newPerson(t *testing.T, pool *pgxpool.Pool, tenantID string) (userID, email string) {
	t.Helper()
	email = fmt.Sprintf("person-%d@controlplane.test", time.Now().UnixNano())
	ctx := context.Background()
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, 'x', 'Test Person')
		 RETURNING id::text`, email).Scan(&userID); err != nil {
		t.Fatalf("create a test person: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1::uuid, $2::uuid)`,
		tenantID, userID); err != nil {
		t.Fatalf("add the person to the organisation: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, userID)
	})
	return userID, email
}

// sessionFor is an operator session value, as the middleware would have built
// it. The handlers and the service take it as a parameter, so a test does not
// have to sign in to exercise what they do.
func sessionFor(account operator.Operator) operator.Session {
	return operator.Session{Operator: account, SteppedUpAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
}

func TestSuspendEndsTheSessionsAndResumeRestores(t *testing.T) {
	pool := optest.Pool(t)
	service := &Service{op: operator.New(pool), db: pool}
	account, _ := optest.Account(t, pool, operator.RoleOperator)
	sess := sessionFor(account)
	tenantID, _ := newTenant(t, pool)
	userID, _ := newPerson(t, pool, tenantID)

	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, tenant_id, expires_at)
		 VALUES (repeat('a', 64), $1::uuid, $2::uuid, NOW() + INTERVAL '1 hour')`,
		userID, tenantID); err != nil {
		t.Fatalf("give the person a session: %v", err)
	}

	if err := service.Suspend(ctx, sess, tenantID, "unpaid invoices"); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	var suspended bool
	var reason string
	if err := pool.QueryRow(ctx,
		`SELECT suspended_at IS NOT NULL, suspension_reason FROM tenants WHERE id = $1::uuid`,
		tenantID).Scan(&suspended, &reason); err != nil {
		t.Fatalf("read the organisation: %v", err)
	}
	if !suspended || reason != "unpaid invoices" {
		t.Fatalf("the organisation was not suspended with its reason: %v %q", suspended, reason)
	}

	// The people already signed in are stopped now, not when their sessions
	// would have expired.
	var live int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM sessions
		  WHERE tenant_id = $1::uuid AND revoked_at IS NULL AND expires_at > NOW()`,
		tenantID).Scan(&live); err != nil {
		t.Fatalf("count the sessions: %v", err)
	}
	if live != 0 {
		t.Fatalf("%d sessions survived the suspension", live)
	}

	if err := service.Resume(ctx, sess, tenantID, "paid"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT suspended_at IS NOT NULL FROM tenants WHERE id = $1::uuid`, tenantID).
		Scan(&suspended); err != nil {
		t.Fatalf("read the organisation: %v", err)
	}
	if suspended {
		t.Fatal("the organisation is still suspended after being resumed")
	}
	// Resuming an organisation that is running is a mistake worth naming
	// rather than a silent no-op.
	if err := service.Resume(ctx, sess, tenantID, "again"); !errors.Is(err, ErrNotSuspended) {
		t.Fatalf("resuming a running organisation answered %v", err)
	}
}

// The whole of the deletion rule in one test: one person cannot do it, a
// second person can only schedule it, and it is reversible until the day it is
// not.
func TestDeletionNeedsTwoPeopleAndThirtyDays(t *testing.T) {
	pool := optest.Pool(t)
	service := &Service{op: operator.New(pool), db: pool}
	asker, _ := optest.Account(t, pool, operator.RoleSuperadmin)
	approver, _ := optest.Account(t, pool, operator.RoleSuperadmin)
	tenantID, _ := newTenant(t, pool)
	ctx := context.Background()

	approvalID, err := service.RequestDeletion(ctx, sessionFor(asker), tenantID, "customer asked us to")
	if err != nil {
		t.Fatalf("request the deletion: %v", err)
	}

	// Nothing has happened to the organisation yet.
	var scheduled *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT deletion_scheduled_at FROM tenants WHERE id = $1::uuid`, tenantID).
		Scan(&scheduled); err != nil {
		t.Fatalf("read the organisation: %v", err)
	}
	if scheduled != nil {
		t.Fatal("asking for a deletion scheduled it")
	}

	// The person who asked cannot be the person who agrees.
	if err := service.Approve(ctx, sessionFor(asker), approvalID, "me again"); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("a self-approval answered %v", err)
	}

	if err := service.Approve(ctx, sessionFor(approver), approvalID, "confirmed with the customer"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	var suspended bool
	if err := pool.QueryRow(ctx,
		`SELECT deletion_scheduled_at, suspended_at IS NOT NULL FROM tenants WHERE id = $1::uuid`,
		tenantID).Scan(&scheduled, &suspended); err != nil {
		t.Fatalf("read the organisation: %v", err)
	}
	if scheduled == nil {
		t.Fatal("approving did not schedule the deletion")
	}
	if !suspended {
		t.Fatal("an organisation on its way out is still open for business")
	}
	// Thirty days, not now.
	if until := time.Until(*scheduled); until < DeletionGrace-time.Hour {
		t.Fatalf("the grace period is %v, want about %v", until, DeletionGrace)
	}

	// The sweep leaves it alone until the day comes.
	service.sweepDeletions(ctx)
	var alive bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tenants WHERE id = $1::uuid)`, tenantID).
		Scan(&alive); err != nil {
		t.Fatalf("look for the organisation: %v", err)
	}
	if !alive {
		t.Fatal("the sweep deleted an organisation whose grace period had not ended")
	}

	// One button puts it back.
	if err := service.CancelDeletion(ctx, sessionFor(approver), tenantID, "customer changed their mind"); err != nil {
		t.Fatalf("cancel the deletion: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT deletion_scheduled_at FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&scheduled); err != nil {
		t.Fatalf("read the organisation: %v", err)
	}
	if scheduled != nil {
		t.Fatal("cancelling did not take the organisation off the list")
	}
}

// The other half of the sweep: when the day does come, it goes.
func TestTheSweepDeletesWhenTheGracePeriodHasEnded(t *testing.T) {
	pool := optest.Pool(t)
	service := &Service{op: operator.New(pool), db: pool}
	tenantID, _ := newTenant(t, pool)
	ctx := context.Background()

	if _, err := pool.Exec(ctx,
		`UPDATE tenants SET deletion_scheduled_at = NOW() - INTERVAL '1 minute' WHERE id = $1::uuid`,
		tenantID); err != nil {
		t.Fatalf("backdate the deletion: %v", err)
	}

	service.sweepDeletions(ctx)

	var alive bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tenants WHERE id = $1::uuid)`, tenantID).
		Scan(&alive); err != nil {
		t.Fatalf("look for the organisation: %v", err)
	}
	if alive {
		t.Fatal("an organisation whose grace period ended is still there")
	}
}

// Impersonation writes to both trails before anybody has gone anywhere, and
// refuses an organisation that is closed.
func TestImpersonationRecordsBothSides(t *testing.T) {
	pool := optest.Pool(t)
	service := &Service{op: operator.New(pool), db: pool}
	account, _ := optest.Account(t, pool, operator.RoleSupport)
	sess := sessionFor(account)
	tenantID, _ := newTenant(t, pool)
	userID, _ := newPerson(t, pool, tenantID)
	ctx := context.Background()

	if _, err := service.op.BeginImpersonation(ctx, sess, tenantID, userID, ""); !errors.Is(err, operator.ErrReasonRequired) {
		t.Fatalf("impersonating without a reason answered %v", err)
	}

	link, err := service.op.BeginImpersonation(ctx, sess, tenantID, userID, "customer reported a missing invoice")
	if err != nil {
		t.Fatalf("begin the impersonation: %v", err)
	}
	if link == "" {
		t.Fatal("no handover link was produced")
	}

	// The operator's trail.
	if got := optest.AuditCount(t, pool, account.ID, "user.impersonate"); got != 1 {
		t.Fatalf("the operator audit has %d rows, want 1", got)
	}
	// And the organisation's own — which is the half that makes this
	// answerable by them rather than by us.
	var theirs int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_events
		  WHERE tenant_id = $1::uuid AND action = 'security.impersonation.requested'`,
		tenantID).Scan(&theirs); err != nil {
		t.Fatalf("read the organisation's trail: %v", err)
	}
	if theirs != 1 {
		t.Fatalf("the organisation's trail has %d rows, want 1", theirs)
	}

	// A person who does not work there cannot be borrowed.
	otherTenant, _ := newTenant(t, pool)
	if _, err := service.op.BeginImpersonation(ctx, sess, otherTenant, userID, "fishing"); !errors.Is(err, operator.ErrNotAMember) {
		t.Fatalf("impersonating a non-member answered %v", err)
	}

	// And a suspended organisation is not a way in.
	if err := service.Suspend(ctx, sessionFor(account), tenantID, "suspended"); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	if _, err := service.op.BeginImpersonation(ctx, sess, tenantID, userID, "still fishing"); !errors.Is(err, operator.ErrTenantSuspended) {
		t.Fatalf("impersonating into a suspended organisation answered %v", err)
	}
}

func TestQuotaIsStoredAndCounted(t *testing.T) {
	pool := optest.Pool(t)
	service := &Service{op: operator.New(pool), db: pool}
	account, _ := optest.Account(t, pool, operator.RoleOperator)
	tenantID, _ := newTenant(t, pool)
	newPerson(t, pool, tenantID)
	ctx := context.Background()

	limit := 1
	if err := service.SetQuota(ctx, sessionFor(account), tenantID, Quota{
		MaxUsers: &limit, Enforcement: EnforcementHard,
	}, "trial account"); err != nil {
		t.Fatalf("set the limits: %v", err)
	}

	quota, err := service.GetQuota(ctx, tenantID)
	if err != nil {
		t.Fatalf("read the limits: %v", err)
	}
	if quota.MaxUsers == nil || *quota.MaxUsers != 1 {
		t.Fatalf("the user limit came back as %v", quota.MaxUsers)
	}
	if quota.Users != 1 {
		t.Fatalf("the organisation has %d people, want 1", quota.Users)
	}
	// A limit nobody set stays nil rather than becoming zero — the difference
	// between "no limit" and "nobody may join".
	if quota.MaxStorageMB != nil {
		t.Fatalf("an unset storage limit came back as %v", *quota.MaxStorageMB)
	}
	if quota.Enforcement != EnforcementHard {
		t.Fatalf("enforcement is %q", quota.Enforcement)
	}
	if err := service.SetQuota(ctx, sessionFor(account), tenantID,
		Quota{Enforcement: "whenever"}, "typo"); !errors.Is(err, ErrUnknownEnforcement) {
		t.Fatalf("an unknown enforcement mode answered %v", err)
	}
}

// Unlocking is the one thing the console may write to somebody's account, and
// the database is what says so: the same handler cannot touch a password.
func TestUnlockIsAllTheConsoleMayWriteToAnAccount(t *testing.T) {
	pool := optest.Pool(t)
	service := &Service{op: operator.New(pool), db: pool}
	account, _ := optest.Account(t, pool, operator.RoleSupport)
	tenantID, _ := newTenant(t, pool)
	userID, _ := newPerson(t, pool, tenantID)
	ctx := context.Background()

	if _, err := pool.Exec(ctx,
		`UPDATE users SET failed_login_attempts = 5, locked_until = NOW() + INTERVAL '15 minutes'
		  WHERE id = $1::uuid`, userID); err != nil {
		t.Fatalf("lock the account: %v", err)
	}

	if err := service.Unlock(ctx, sessionFor(account), userID, "they telephoned"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	var locked bool
	if err := pool.QueryRow(ctx,
		`SELECT locked_until IS NOT NULL FROM users WHERE id = $1::uuid`, userID).Scan(&locked); err != nil {
		t.Fatalf("read the account: %v", err)
	}
	if locked {
		t.Fatal("the account is still locked")
	}

	// The column grant, exercised directly. If this ever succeeds, the console
	// can set somebody's password, and every claim made in support.go about
	// what it cannot do is false.
	if _, err := pool.Exec(operator.Scoped(ctx),
		`UPDATE users SET password_hash = 'x' WHERE id = $1::uuid`, userID); err == nil {
		t.Fatal("the operator role changed a password")
	}
	if _, err := pool.Exec(operator.Scoped(ctx),
		`UPDATE users SET email = 'taken@example.test' WHERE id = $1::uuid`, userID); err == nil {
		t.Fatal("the operator role changed an address")
	}
	// And it cannot delete an organisation, which is what makes the grace
	// period a guarantee rather than a habit.
	if _, err := pool.Exec(operator.Scoped(ctx), `DELETE FROM tenants WHERE id = $1::uuid`, tenantID); err == nil {
		t.Fatal("the operator role deleted an organisation")
	}
}
