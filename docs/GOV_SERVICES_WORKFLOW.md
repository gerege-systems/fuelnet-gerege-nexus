# Configurable government service workflow

How `io.example.gov_services` turns one codebase into a service-delivery
capability that every tenant, and every service inside a tenant, configures for
itself — locally fulfilled, delegated with verification, or routed per request.

<p>
  <img src="assets/icons/flag-en.png" width="18" height="18" alt=""> <b>English</b>
</p>

[Back to the documentation hub](README.md)

---

## 1. The model

| Concept | Table | Purpose |
| --- | --- | --- |
| Organisational unit | `gov_org_units` | Adjacency-list hierarchy per tenant |
| Unit membership | `gov_unit_members` | Which unit a user acts for, and whether they supervise it |
| Workflow | `gov_workflows` | A named process |
| Workflow version | `gov_workflow_versions` | An immutable published definition |
| Step | `gov_workflow_steps` | Order, executor rule, SLA, verification requirement |
| Transition | `gov_workflow_transitions` | Optional narrowing of the state machine |
| Routing rule | `gov_routing_rules` | Tenant- or service-scoped strategy selection |
| Service | `gov_services` | Points at a published version and a fulfilment mode |
| Request | `gov_applications` | The top-level service request |
| Task | `gov_tasks` | Per-level fulfilment work |
| Timeline | `gov_application_events` | Append-only audit of every transition |
| Outbox | `gov_delivery_outbox` | Status updates awaiting upstream delivery |

### Fulfilment modes

- **`LOCAL`** — the receiving unit fulfils the request itself.
- **`DELEGATE`** — the request is forwarded to a lower unit while the upper unit
  monitors and verifies the result.
- **`HYBRID`** — a routing rule decides per request which of the two applies.

A service selects its mode and its published workflow version; no code changes
are involved.

---

## 2. State machine

Task statuses:

```
RECEIVED → ASSIGNED → IN_PROGRESS ─┬─→ COMPLETED → CLOSED
                                   ├─→ AWAITING_VERIFICATION → COMPLETED
                                   │                        └→ RETURNED → …
                                   ├─→ FORWARDED (delegation; child task opens)
                                   ├─→ INFO_REQUESTED → IN_PROGRESS
                                   ├─→ REJECTED
                                   └─→ CANCELLED
```

**Code decides what is possible; configuration decides what is offered.** The
canonical transition table lives in `workflow.go` and encodes the invariants —
you cannot verify work that is not awaiting verification, you cannot act on a
terminal task. A published version may *narrow* that set through
`gov_workflow_transitions`; it can never widen it. This keeps the invariants
unit-testable without a database and means a misconfigured tenant cannot reach
an impossible state.

Rules that hold everywhere:

- The resulting status is computed server-side. A client sends an **action**,
  never a status.
- `reject`, `return` and `request_info` are refused without a comment.
- Completion lands in `AWAITING_VERIFICATION` when the step sets
  `requires_verification`, so **a lower unit finishing work never closes the
  overall request**. The parent stays `FORWARDED` until the child is verified.
- **Overdue is derived** (`due_at < now()` on a non-terminal status), never
  written over the business status.
- Every transition is one transaction: task update, timeline event, cascade to
  the parent, derived request status and outbox row all commit together.

### Concurrency

`gov_tasks.row_version` is required on every action and guards the update. Two
officers acting on the same task from the same read produce one winner and one
`CONFLICT_VERSION` — never a lost update.

---

## 3. Tenant and organisational isolation

Isolation is enforced in the schema, not only in queries. Every table that
references a unit, a workflow version or a request carries a **composite
foreign key including `tenant_id`**, so a row cannot point across tenants even
if application code has a bug:

```sql
CONSTRAINT gov_tasks_unit_fk FOREIGN KEY (unit_id, tenant_id)
    REFERENCES gov_org_units (id, tenant_id)
```

Within a tenant:

- A user's scope is the units in `gov_unit_members`; a `SUPERVISOR` also covers
  every descendant of its unit.
- Queue and dashboard queries apply that scope in SQL, so paging cannot walk
  past the boundary.
- A user with no unit assignment sees an empty queue, not the tenant.
- Tenant administrators are scoped to the whole tenant.
- `verify` and `return` are checked against the **parent** task's unit, because
  they are exercised by the level above.

---

## 4. Routing

| Strategy | Chooses |
| --- | --- |
| `SELF` | The current unit |
| `PARENT` | The unit above (fails at the root) |
| `CHILD` | A child unit; ambiguous when several exist, so the caller names one |
| `SPECIFIC_UNIT` | A configured unit |
| `REGION_MATCH` | A unit whose `region_code` matches a request field |

Every strategy is an explicit branch. There is no expression evaluator, so
tenant configuration can never make the platform evaluate arbitrary input.
Rules are ordered by priority, and a service-scoped rule outranks a tenant-wide
one at equal priority.

---

## 5. Upstream ingestion

```http
POST /api/v1/gov/requests/ingest
{
  "source_system": "E-MONGOLIA",
  "external_request_id": "EM-2026-000123",
  "service_code": "PASSPORT",
  "applicant_name": "Бат Болд",
  "fields": { "region": "UB-NORTH" }
}
```

**Idempotency contract**

- Identity is `(tenant_id, source_system, external_request_id)`, enforced by a
  unique index.
- An identical retry returns the existing request with `"created": false` and
  creates no second task and no second timeline entry.
- A replay carrying the same identity but a materially different payload is
  refused with `409 IDEMPOTENCY_CONFLICT`. "Different" means a different
  SHA-256 over the canonical field set, stored as `payload_fingerprint`.

**Authentication.** The endpoint runs behind the platform's session/bearer
middleware, the app gate and the `gov.apply` permission. Binding it to OAuth2
client credentials (so machine clients are distinguishable from staff) is the
one deliberately deferred step — see §8.

---

## 6. Status delivery boundary

A remote endpoint must never be able to roll back or stall a workflow
transition, so delivery is decoupled:

1. The state change writes a row into `gov_delivery_outbox` **inside the same
   transaction**.
2. `gov_upstream_connectors` holds the target URL per source system and a
   `secret_ref` — the *name* of the connector holding the signing secret. The
   secret itself is never stored in the outbox, never returned by the API and
   never logged.
3. `GET /api/v1/gov/outbox` exposes delivery state (without payloads) for
   operators.

Dispatching is intentionally left to a caller — a cron entry or the platform's
scheduler — rather than adding a worker framework.

---

## 7. Example configurations

### Local fulfilment

```jsonc
// 1. Create and publish the template
POST /api/v1/gov/workflows        { "template": "LOCAL_FULFILMENT" }
POST /api/v1/gov/workflow-versions/{id}/publish

// 2. Point the service at it
PUT  /api/v1/gov/services/{serviceId}/configuration
{ "fulfillment_mode": "LOCAL", "workflow_version_id": "…", "owner_unit_id": "…" }
```

A request then opens one task at the owning unit: `RECEIVED → IN_PROGRESS →
COMPLETED → CLOSED`.

### Delegated fulfilment with verification

```jsonc
POST /api/v1/gov/workflows        { "template": "DELEGATE_ONE_LEVEL" }
POST /api/v1/gov/workflow-versions/{id}/publish
PUT  /api/v1/gov/services/{serviceId}/configuration
{ "fulfillment_mode": "DELEGATE", "workflow_version_id": "…", "owner_unit_id": "<HQ>" }
```

Flow: HQ receives → `delegate` (child task opens at the district, HQ becomes
`FORWARDED`) → district completes → child becomes `AWAITING_VERIFICATION` →
HQ `verify` → child `COMPLETED` → HQ `COMPLETED` → `close`.

Returning instead of verifying puts the child in `RETURNED` for rework.

---

## 8. Migration and compatibility

Migration `00007_gov_workflow.sql` is additive. Existing services, requests,
timelines and appointments keep their rows.

Backfill for every tenant already using the module:

1. A root unit `HQ` is created.
2. A `LEGACY_LOCAL` workflow is created, given a single `FULFILL` step and
   published.
3. Services with no configuration are pointed at it in `LOCAL` mode.
4. Each existing request gets exactly one task carrying its state:

   | `gov_applications.status` | `gov_tasks.status` |
   | --- | --- |
   | `SUBMITTED` | `RECEIVED` |
   | `IN_REVIEW` | `IN_PROGRESS` |
   | `INFO_REQUESTED` | `INFO_REQUESTED` |
   | `APPROVED` | `COMPLETED` |
   | `REJECTED` | `REJECTED` |
   | `COMPLETED` | `CLOSED` |
   | `CANCELLED` | `CANCELLED` |

**One source of truth.** `gov_tasks` holds the state. `gov_applications.status`
remains as the compatibility field the original clients read, derived from the
root task inside the same transaction by `syncApplication` — the two cannot
drift.

The original endpoints (`/services`, `/applications`, `/applications/{id}/…`,
`/appointments`, `/officer/queue`) keep working and now drive the same state
machine underneath: a legacy submission opens a task, and a legacy decision is
translated into a workflow action.

**Deferred:** binding `/requests/ingest` to OAuth2 client credentials. The
platform issues opaque tokens and exposes `/oauth2/introspect`, so the remaining
work is a middleware that introspects a bearer token, resolves the client to a
tenant and injects a machine actor. Until then the endpoint requires an
authenticated principal with `gov.apply`.

---

## 9. Permissions

| Permission | Grants |
| --- | --- |
| `gov.read` | Registry, requests, queues, unit tree |
| `gov.apply` | Submit, ingest and cancel requests |
| `gov.process` | Assign, start, complete, reject, close |
| `gov.delegate` | Forward to a lower unit |
| `gov.verify` | Verify or return a lower unit's work |
| `gov.configure` | Units, workflows, routing, service configuration |
| `gov.report` | Dashboards across the authorised scope |

They are enforced server-side through `rbac.SQLPermissionStore`, which walks
`memberships → membership_roles → role_permissions → permissions`. The Go module
and `catalog/manifests/gov-services.json` declare the same set. Hiding a button
in the UI is a convenience, never the control.

---

## 10. API summary

| Method | Path | Permission |
| --- | --- | --- |
| `GET` | `/api/v1/gov/units`, `/units/tree` | `gov.read` |
| `POST` | `/api/v1/gov/units`, `/units/members` | `gov.configure` |
| `GET` | `/api/v1/gov/workflow-templates`, `/workflows`, `/workflow-versions/{id}` | `gov.read` |
| `POST` | `/api/v1/gov/workflows`, `/workflow-versions/{id}/publish` | `gov.configure` |
| `GET`/`POST` | `/api/v1/gov/routing-rules` | `gov.read` / `gov.configure` |
| `PUT` | `/api/v1/gov/services/{id}/configuration` | `gov.configure` |
| `POST` | `/api/v1/gov/requests/ingest` | `gov.apply` |
| `GET` | `/api/v1/gov/requests/{id}` | `gov.read` |
| `GET` | `/api/v1/gov/tasks` | `gov.read` |
| `POST` | `/api/v1/gov/tasks/{id}/actions` | per action |
| `GET` | `/api/v1/gov/dashboard` | `gov.report` |
| `GET` | `/api/v1/gov/outbox` | `gov.configure` |

`/tasks` supports `status`, `unit_id`, `service_id`, `overdue`, `from`, `to`,
`page` and `page_size`, and returns `{items, total, page, page_size}`. Errors
carry a stable `code` beside the human-readable `error`.

---

## 11. Testing

Twelve unit tests cover the state machine, routing and templates with no
database. Eleven integration tests cover what only SQL can enforce: composite
tenant foreign keys, the published-version immutability trigger, optimistic
concurrency on `row_version`, idempotent ingestion and derived overdue.

```bash
# State machine and routing (no database required)
cd backend && go test ./internal/apps/gov_services/...

# Full workflow against a migrated throwaway database
docker run -d --name gov-test-db -e POSTGRES_PASSWORD=test -e POSTGRES_DB=govtest \
  -p 55433:5432 postgres:16-alpine
DATABASE_URL='postgres://postgres:test@localhost:55433/govtest?sslmode=disable' \
  go run ./cmd/migrate up
GOV_TEST_DATABASE_URL='postgres://postgres:test@localhost:55433/govtest?sslmode=disable' \
  go test -race ./internal/apps/gov_services/...
```

The integration tests skip when `GOV_TEST_DATABASE_URL` is unset, so `go test
./...` stays green on a machine without a database.

**CI runs them for real.** The `test` job starts a `postgres:16-alpine`
service, migrates it with `cmd/migrate` and exports `GOV_TEST_DATABASE_URL`.
Because a skip and a pass look identical in the run summary, the job ends with
an explicit check that greps for `--- SKIP` and fails if the database wiring
ever stops reaching the tests.
