package gov_services

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise the workflow against a real PostgreSQL schema, because
// the invariants they protect live partly in SQL (composite tenant foreign
// keys, the immutability trigger, optimistic concurrency).
//
// They are skipped unless a migrated throwaway database is provided:
//
//	GOV_TEST_DATABASE_URL=postgres://... go test ./internal/apps/gov_services/...
//
// CI runs without one, so `go test ./...` stays green on a machine with no
// database.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("GOV_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set GOV_TEST_DATABASE_URL to a migrated test database to run workflow integration tests")
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
	m         *Module
	tenantID  string
	adminID   string
	officerID string
	hqID      string
	childID   string
	serviceID string
	versionID string
	admin     Actor
}

// newFixture builds an isolated tenant with a two-level hierarchy and a
// published delegation workflow.
func newFixture(t *testing.T, template string) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := testPool(t)
	m := &Module{db: pool, store: &store{db: pool}}

	var tenantID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('wf-' || substr(gen_random_uuid()::text, 1, 8), 'Workflow test')
		 RETURNING id::text`).Scan(&tenantID); err != nil {
		t.Fatalf("tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenantID)
	})

	newUser := func(label string) string {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO users (email, password_hash, name)
			 VALUES ($1 || '-' || substr(gen_random_uuid()::text, 1, 8) || '@test.local', 'x', $1)
			 RETURNING id::text`, label).Scan(&id); err != nil {
			t.Fatalf("user: %v", err)
		}
		return id
	}
	adminID, officerID := newUser("admin"), newUser("officer")

	admin := Actor{UserID: adminID, Email: "admin@test.local", IsAdmin: true}

	hq, err := m.store.createUnit(ctx, tenantID, OrgUnit{Code: "HQ", Name: "Төв", UnitType: "ROOT"})
	if err != nil {
		t.Fatalf("hq: %v", err)
	}
	child, err := m.store.createUnit(ctx, tenantID, OrgUnit{
		Code: "DIST", Name: "Дүүрэг", UnitType: "DISTRICT", ParentID: &hq.ID, RegionCode: "UB",
	})
	if err != nil {
		t.Fatalf("child unit: %v", err)
	}

	tpl, ok := TemplateByCode(template)
	if !ok {
		t.Fatalf("unknown template %s", template)
	}
	version, err := m.store.createWorkflowFromTemplate(ctx, tenantID, admin.Email, tpl, "", "")
	if err != nil {
		t.Fatalf("workflow: %v", err)
	}
	if err := m.store.publishVersion(ctx, tenantID, version.ID, admin.Email); err != nil {
		t.Fatalf("publish: %v", err)
	}

	var serviceID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO gov_services (tenant_id, code, name, category, fulfillment_mode,
		                           workflow_version_id, owner_unit_id)
		 VALUES ($1, 'PASSPORT', 'Гадаад паспорт', 'IDENTITY', $2, $3, $4)
		 RETURNING id::text`, tenantID, tpl.Mode, version.ID, hq.ID).Scan(&serviceID); err != nil {
		t.Fatalf("service: %v", err)
	}

	return &fixture{
		m: m, tenantID: tenantID, adminID: adminID, officerID: officerID,
		hqID: hq.ID, childID: child.ID, serviceID: serviceID, versionID: version.ID, admin: admin,
	}
}

func (f *fixture) ingest(t *testing.T, externalID string, fields map[string]string) *IngestResult {
	t.Helper()
	res, err := f.m.Ingest(context.Background(), f.tenantID, f.admin, IngestInput{
		SourceSystem: "E-MONGOLIA", ExternalRequestID: externalID, ServiceCode: "PASSPORT",
		ApplicantName: "Бат Болд", ApplicantReg: "АА90010111", Fields: fields,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	return res
}

func (f *fixture) openTask(t *testing.T, applicationID string) *Task {
	t.Helper()
	var id string
	if err := f.m.db.QueryRow(context.Background(),
		`SELECT id::text FROM gov_tasks
		  WHERE application_id = $1 AND status = ANY($2) ORDER BY created_at DESC LIMIT 1`,
		applicationID, activeTaskStatuses).Scan(&id); err != nil {
		t.Fatalf("open task: %v", err)
	}
	task, err := f.m.store.lockTask(context.Background(), f.m.db, f.tenantID, id)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	return task
}

func (f *fixture) applicationStatus(t *testing.T, applicationID string) string {
	t.Helper()
	var status string
	if err := f.m.db.QueryRow(context.Background(),
		`SELECT status FROM gov_applications WHERE id = $1`, applicationID).Scan(&status); err != nil {
		t.Fatalf("application status: %v", err)
	}
	return status
}

func codeOf(err error) string {
	var domain *WorkflowError
	if errors.As(err, &domain) {
		return domain.Code
	}
	return ""
}

// ─── Happy paths ─────────────────────────────────────────────────────────────

func TestLocalFulfilmentHappyPath(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")
	ctx := context.Background()

	res := f.ingest(t, "ext-local-1", nil)
	appID := res.Application["id"].(string)

	task := f.openTask(t, appID)
	if task.Status != TaskReceived || task.UnitID != f.hqID {
		t.Fatalf("the first task must sit at HQ in RECEIVED, got %s at %s", task.Status, task.UnitID)
	}

	task, err := f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{Action: ActionStart, RowVersion: task.RowVersion})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	task, err = f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{
		Action: ActionComplete, RowVersion: task.RowVersion, ResultCode: "ISSUED",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	// A local step needs no verification, so the work is done immediately.
	if task.Status != TaskCompleted {
		t.Fatalf("expected %s, got %s", TaskCompleted, task.Status)
	}
	if got := f.applicationStatus(t, appID); got != "APPROVED" {
		t.Fatalf("request status should follow the task, got %s", got)
	}

	task, err = f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{Action: ActionClose, RowVersion: task.RowVersion})
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if task.Status != TaskClosed {
		t.Fatalf("expected %s, got %s", TaskClosed, task.Status)
	}
	if got := f.applicationStatus(t, appID); got != "COMPLETED" {
		t.Fatalf("a closed task must close the request, got %s", got)
	}
}

func TestDelegationCreatesChildTaskAndWaitsForVerification(t *testing.T) {
	f := newFixture(t, "DELEGATE_ONE_LEVEL")
	ctx := context.Background()

	appID := f.ingest(t, "ext-delegate-1", nil).Application["id"].(string)
	parent := f.openTask(t, appID)

	parent, err := f.m.Act(ctx, f.tenantID, f.admin, parent.ID, ActionInput{
		Action: ActionDelegate, RowVersion: parent.RowVersion, TargetUnitID: f.childID,
		Comment: "Дүүрэгт шилжүүлэв",
	})
	if err != nil {
		t.Fatalf("delegate: %v", err)
	}
	if parent.Status != TaskForwarded {
		t.Fatalf("the delegating task must wait in %s, got %s", TaskForwarded, parent.Status)
	}

	child := f.openTask(t, appID)
	if child.ParentTaskID == nil || *child.ParentTaskID != parent.ID {
		t.Fatal("delegation must create a child task linked to its parent")
	}
	if child.UnitID != f.childID {
		t.Fatalf("the child task must sit at the district, got %s", child.UnitID)
	}

	child, err = f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{Action: ActionStart, RowVersion: child.RowVersion})
	if err != nil {
		t.Fatalf("start child: %v", err)
	}
	child, err = f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{Action: ActionComplete, RowVersion: child.RowVersion})
	if err != nil {
		t.Fatalf("complete child: %v", err)
	}

	// The lower unit finishing must NOT close the request on its own.
	if child.Status != TaskAwaitingVerification {
		t.Fatalf("expected %s, got %s", TaskAwaitingVerification, child.Status)
	}
	if got := f.applicationStatus(t, appID); got == "COMPLETED" {
		t.Fatal("the request must not be closed before upper-level verification")
	}

	verified, err := f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{
		Action: ActionVerify, RowVersion: child.RowVersion, Comment: "Хангалттай",
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verified.Status != TaskCompleted {
		t.Fatalf("verification must complete the child task, got %s", verified.Status)
	}

	// The parent picks up once its only child is done.
	parentAfter, err := f.m.store.lockTask(ctx, f.m.db, f.tenantID, parent.ID)
	if err != nil {
		t.Fatalf("reload parent: %v", err)
	}
	if parentAfter.Status != TaskCompleted {
		t.Fatalf("the parent must complete once the child is verified, got %s", parentAfter.Status)
	}
}

func TestReturnSendsWorkBackForRework(t *testing.T) {
	f := newFixture(t, "DELEGATE_ONE_LEVEL")
	ctx := context.Background()

	appID := f.ingest(t, "ext-return-1", nil).Application["id"].(string)
	parent := f.openTask(t, appID)
	if _, err := f.m.Act(ctx, f.tenantID, f.admin, parent.ID, ActionInput{
		Action: ActionDelegate, RowVersion: parent.RowVersion, TargetUnitID: f.childID,
	}); err != nil {
		t.Fatalf("delegate: %v", err)
	}

	child := f.openTask(t, appID)
	child, _ = f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{Action: ActionStart, RowVersion: child.RowVersion})
	child, err := f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{Action: ActionComplete, RowVersion: child.RowVersion})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Returning without a reason is refused.
	if _, err := f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{
		Action: ActionReturn, RowVersion: child.RowVersion,
	}); codeOf(err) != "COMMENT_REQUIRED" {
		t.Fatalf("expected COMMENT_REQUIRED, got %v", err)
	}

	returned, err := f.m.Act(ctx, f.tenantID, f.admin, child.ID, ActionInput{
		Action: ActionReturn, RowVersion: child.RowVersion, Comment: "Бичиг дутуу",
	})
	if err != nil {
		t.Fatalf("return: %v", err)
	}
	if returned.Status != TaskReturned {
		t.Fatalf("expected %s, got %s", TaskReturned, returned.Status)
	}

	// Rework and complete again.
	reworked, err := f.m.Act(ctx, f.tenantID, f.admin, returned.ID, ActionInput{
		Action: ActionComplete, RowVersion: returned.RowVersion,
	})
	if err != nil {
		t.Fatalf("rework: %v", err)
	}
	if reworked.Status != TaskAwaitingVerification {
		t.Fatalf("reworked completion must await verification again, got %s", reworked.Status)
	}
}

// ─── Guards ──────────────────────────────────────────────────────────────────

func TestOptimisticConcurrencyRejectsAStaleUpdate(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")
	ctx := context.Background()

	appID := f.ingest(t, "ext-concurrency-1", nil).Application["id"].(string)
	task := f.openTask(t, appID)
	stale := task.RowVersion

	if _, err := f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{Action: ActionStart, RowVersion: stale}); err != nil {
		t.Fatalf("first transition: %v", err)
	}
	// A second caller working from the version it read earlier must lose.
	_, err := f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{Action: ActionComplete, RowVersion: stale})
	if codeOf(err) != "CONFLICT_VERSION" {
		t.Fatalf("expected CONFLICT_VERSION, got %v", err)
	}

	// Omitting the version entirely is refused too.
	if _, err := f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{Action: ActionComplete}); codeOf(err) != "ROW_VERSION_REQUIRED" {
		t.Fatalf("expected ROW_VERSION_REQUIRED, got %v", err)
	}
}

func TestIngestionIsIdempotentAndDetectsConflictingReplay(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")
	ctx := context.Background()

	first := f.ingest(t, "ext-idem-1", map[string]string{"region": "UB"})
	if !first.Created {
		t.Fatal("the first delivery must create the request")
	}

	// An identical retry returns the same request and creates nothing.
	second := f.ingest(t, "ext-idem-1", map[string]string{"region": "UB"})
	if second.Created {
		t.Fatal("a retry must not create a second request")
	}
	if second.Application["id"] != first.Application["id"] {
		t.Fatal("a retry must return the original request")
	}

	var tasks, events int
	appID := first.Application["id"].(string)
	_ = f.m.db.QueryRow(ctx, `SELECT COUNT(*) FROM gov_tasks WHERE application_id = $1`, appID).Scan(&tasks)
	_ = f.m.db.QueryRow(ctx, `SELECT COUNT(*) FROM gov_application_events WHERE application_id = $1`, appID).Scan(&events)
	if tasks != 1 || events != 1 {
		t.Fatalf("a retry must not duplicate work: %d tasks, %d events", tasks, events)
	}

	// The same identity with a different payload is a conflict, not a retry.
	_, err := f.m.Ingest(ctx, f.tenantID, f.admin, IngestInput{
		SourceSystem: "E-MONGOLIA", ExternalRequestID: "ext-idem-1", ServiceCode: "PASSPORT",
		ApplicantName: "Өөр Хүн", Fields: map[string]string{"region": "GOBI"},
	})
	if codeOf(err) != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("expected IDEMPOTENCY_CONFLICT, got %v", err)
	}
}

func TestTenantIsolation(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")
	other := newFixture(t, "LOCAL_FULFILMENT")
	ctx := context.Background()

	appID := f.ingest(t, "ext-isolation-1", nil).Application["id"].(string)
	task := f.openTask(t, appID)

	// The other tenant's admin cannot even load the task.
	if _, err := other.m.Act(ctx, other.tenantID, other.admin, task.ID, ActionInput{
		Action: ActionStart, RowVersion: task.RowVersion,
	}); codeOf(err) != "NOT_FOUND" {
		t.Fatalf("cross-tenant action must not find the task, got %v", err)
	}

	// Nor see it in a queue or the dashboard.
	page, err := other.m.ListTasks(ctx, other.tenantID, other.admin, TaskFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, item := range page.Items {
		if item.ID == task.ID {
			t.Fatal("a task leaked across tenants")
		}
	}
	if _, err := other.m.RequestDetail(ctx, other.tenantID, other.admin, appID); codeOf(err) != "NOT_FOUND" {
		t.Fatalf("cross-tenant detail must not be found, got %v", err)
	}
}

func TestOrganisationalScopeAuthorisation(t *testing.T) {
	f := newFixture(t, "DELEGATE_ONE_LEVEL")
	ctx := context.Background()

	appID := f.ingest(t, "ext-scope-1", nil).Application["id"].(string)
	parent := f.openTask(t, appID)

	// An officer of the district cannot touch the HQ task.
	district := Actor{
		UserID: f.officerID, Email: "officer@test.local",
		Perms:   map[string]bool{PermProcess: true, PermDelegate: true, PermVerify: true, PermRead: true},
		UnitIDs: []string{f.childID},
	}
	if _, err := f.m.Act(ctx, f.tenantID, district, parent.ID, ActionInput{
		Action: ActionStart, RowVersion: parent.RowVersion,
	}); codeOf(err) != "OUT_OF_SCOPE" {
		t.Fatalf("expected OUT_OF_SCOPE, got %v", err)
	}

	// After delegation the district owns the child task.
	if _, err := f.m.Act(ctx, f.tenantID, f.admin, parent.ID, ActionInput{
		Action: ActionDelegate, RowVersion: parent.RowVersion, TargetUnitID: f.childID,
	}); err != nil {
		t.Fatalf("delegate: %v", err)
	}
	child := f.openTask(t, appID)
	if _, err := f.m.Act(ctx, f.tenantID, district, child.ID, ActionInput{
		Action: ActionStart, RowVersion: child.RowVersion,
	}); err != nil {
		t.Fatalf("the owning unit must be able to work its task: %v", err)
	}

	// A user with no unit assignment sees an empty queue rather than the tenant.
	stranger := Actor{UserID: f.officerID, Perms: map[string]bool{PermRead: true}}
	page, err := f.m.ListTasks(ctx, f.tenantID, stranger, TaskFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("a user without units must see nothing, got %d rows", len(page.Items))
	}

	// Missing the permission is refused even inside the right unit.
	unprivileged := Actor{UserID: f.officerID, Perms: map[string]bool{PermRead: true}, UnitIDs: []string{f.childID}}
	child = f.openTask(t, appID)
	if _, err := f.m.Act(ctx, f.tenantID, unprivileged, child.ID, ActionInput{
		Action: ActionComplete, RowVersion: child.RowVersion,
	}); codeOf(err) != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %v", err)
	}
}

func TestPublishedWorkflowVersionIsImmutable(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")
	ctx := context.Background()

	// An in-flight request pins the version it started on.
	f.ingest(t, "ext-immutable-1", nil)

	_, err := f.m.db.Exec(ctx,
		`UPDATE gov_workflow_steps SET sla_hours = 1 WHERE version_id = $1`, f.versionID)
	if err == nil {
		t.Fatal("editing a published version must be refused by the database")
	}

	_, err = f.m.db.Exec(ctx,
		`INSERT INTO gov_workflow_steps (tenant_id, version_id, code, name, step_order, executor_rule)
		 VALUES ($1, $2, 'EXTRA', 'Нэмэлт', 99, 'SELF')`, f.tenantID, f.versionID)
	if err == nil {
		t.Fatal("adding a step to a published version must be refused")
	}

	// The way to change behaviour is a new version.
	tpl, _ := TemplateByCode("LOCAL_FULFILMENT")
	next, err := f.m.store.createWorkflowFromTemplate(ctx, f.tenantID, f.admin.Email, tpl, "", "")
	if err != nil {
		t.Fatalf("new version: %v", err)
	}
	if next.Version < 2 {
		t.Fatalf("expected a new version number, got %d", next.Version)
	}
}

func TestDashboardCountsAndOverdueDerivation(t *testing.T) {
	f := newFixture(t, "DELEGATE_ONE_LEVEL")
	ctx := context.Background()

	// One request left in RECEIVED, one delegated, one overdue.
	f.ingest(t, "ext-dash-1", nil)

	appTwo := f.ingest(t, "ext-dash-2", nil).Application["id"].(string)
	parent := f.openTask(t, appTwo)
	if _, err := f.m.Act(ctx, f.tenantID, f.admin, parent.ID, ActionInput{
		Action: ActionDelegate, RowVersion: parent.RowVersion, TargetUnitID: f.childID,
	}); err != nil {
		t.Fatalf("delegate: %v", err)
	}

	appThree := f.ingest(t, "ext-dash-3", nil).Application["id"].(string)
	overdue := f.openTask(t, appThree)
	if _, err := f.m.db.Exec(ctx,
		`UPDATE gov_tasks SET due_at = NOW() - INTERVAL '2 days' WHERE id = $1`, overdue.ID); err != nil {
		t.Fatalf("age the task: %v", err)
	}

	summary, err := f.m.Dashboard(ctx, f.tenantID, f.admin)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if summary.Received < 2 {
		t.Fatalf("expected at least two received tasks, got %d", summary.Received)
	}
	if summary.Delegated != 1 {
		t.Fatalf("expected one delegated task, got %d", summary.Delegated)
	}
	if summary.Overdue != 1 {
		t.Fatalf("expected exactly one overdue task, got %d", summary.Overdue)
	}
	if summary.TotalRequests != 3 {
		t.Fatalf("expected three requests in scope, got %d", summary.TotalRequests)
	}

	// Overdue is derived, so the business status is untouched.
	reloaded, err := f.m.store.lockTask(ctx, f.m.db, f.tenantID, overdue.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Status != TaskReceived {
		t.Fatalf("an overdue task must keep its status, got %s", reloaded.Status)
	}
	if !isOverdue(reloaded) {
		t.Fatal("the task should be reported as overdue")
	}
}

func TestLegacyApplicationsRemainReadableAfterMigration(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")
	ctx := context.Background()

	appID := f.ingest(t, "ext-legacy-1", nil).Application["id"].(string)
	task := f.openTask(t, appID)
	if _, err := f.m.Act(ctx, f.tenantID, f.admin, task.ID, ActionInput{
		Action: ActionStart, RowVersion: task.RowVersion,
	}); err != nil {
		t.Fatalf("start: %v", err)
	}

	// The compatibility status the original clients read follows the workflow.
	if got := f.applicationStatus(t, appID); got != "IN_REVIEW" {
		t.Fatalf("expected IN_REVIEW, got %s", got)
	}

	timeline, err := f.m.Timeline(ctx, f.tenantID, appID)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(timeline) < 2 {
		t.Fatalf("the timeline must record ingestion and the transition, got %d entries", len(timeline))
	}
	detail, err := f.m.RequestDetail(ctx, f.tenantID, f.admin, appID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(detail.Tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(detail.Tasks))
	}
}

func TestSLADeadlineIsSetFromTheStep(t *testing.T) {
	f := newFixture(t, "LOCAL_FULFILMENT")

	appID := f.ingest(t, "ext-sla-1", nil).Application["id"].(string)
	task := f.openTask(t, appID)
	if task.DueAt == nil {
		t.Fatal("a task must carry the deadline derived from its step SLA")
	}
	// LOCAL_FULFILMENT is configured with 72 hours.
	want := time.Now().Add(72 * time.Hour)
	if task.DueAt.After(want.Add(time.Hour)) || task.DueAt.Before(want.Add(-time.Hour)) {
		t.Fatalf("deadline %s is not ~72h out", task.DueAt)
	}
}
