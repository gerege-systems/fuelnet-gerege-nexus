# Changelog

All notable changes to **open-gerege-nexus** (Gerege Nexus) will be documented in
this file.

Entries below the rebrand keep the names that were true when they shipped — the
`open-gerege-mn-erp` repository, the ERP framing, and the `openerp.gerege.mn`
deployment, which has since moved to `nexus.gerege.mn`. A changelog edited to
match the present tense stops being a record.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added — Control Plane Catalog Management & Migration Deprecation (CP-46)

- **Control Plane Catalog Endpoints** (`/cp/api/catalog/sync`, `/cp/api/catalog/status`, `/cp/api/catalog/overview`): Operators can now monitor app catalog status and trigger catalog sync on demand with step-up & audit logging.
- **Store Admin Deprecation**: Added `Deprecation: true` and `Link` headers to tenant-level store admin endpoints (`/admin/store/sync`, `/admin/store/status`) to initiate graceful client migration.
- **Native macOS App Icon**: Updated macOS native AppKit app to load `logo.jpg` dynamically at runtime.
- **Asset Optimization**: Compressed `login_landing_bg.jpg` (~65% reduction, 1.0MB -> 360KB).
- **Code & Doc Audit**: Full codebase verification, removing obsolete files and unused imports.

### Added — The break-glass account, and the storage limit that refuses

The last two things the design document asked for and the phases had not
delivered.

- **Break-glass** (§2.4 of the plan): one emergency operator account whose
  password lives in a safe. It grants nothing extra — the same password and the
  same authenticator as everybody — and the whole of it is what happens when it
  is used: an ERROR log line naming who and from where, a metric of its own, a
  `severity: page` alert on that metric, and its own action in the operator
  audit. A locked door with an alarm on it rather than a hidden one. The
  database permits exactly one such account, because the second one becomes
  somebody's ordinary login and the alarm starts crying wolf.
- **The storage limit now refuses.** CP-2 recorded it and CP-5 measured it;
  this is the check, at the one place on the platform where a file of any size
  is kept. It compares against the last nightly measurement rather than summing
  every blob on each upload — a storage quota is a commercial boundary, and the
  disk alert is what protects the platform.
- The metering query for storage now sums `esign_documents.byte_size`, a column
  the upload already writes, rather than measuring the blobs themselves.

### Added — Counting what each organisation used, from the database rather than from the metrics

CP-5, the last phase of the control plane. `usage_events` holds one row per
organisation per metric per day, written by a job that runs nightly and again
during the day, so the console is never showing yesterday's picture to somebody
looking at it after lunch.

- **Not from Prometheus, deliberately.** The first phase decided that no metric
  would ever carry a tenant label, because a label whose values are customers
  is a series count that only grows. The bill for that decision comes due here
  and is paid in SQL — and arguably at a profit, because what gets counted is
  *acts recorded in the audit trail* rather than HTTP requests, which is what a
  usage line should be based on in the first place.
- **Not an event per request.** A row per API call is a table nobody can query
  by the second month. What is stored is the day's total, and re-running a
  day's collection rewrites it rather than doubling it.
- **Two of the five metrics are not sums, and nothing sums them.** Active people
  is a peak — adding daily actives over a month counts the same person thirty
  times — and storage is a reading, where a sum would be a number that never
  goes down. The screen labels them accordingly.
- **The console reads usage and cannot write it.** `usage_events` is granted to
  the operator role for SELECT alone, so there is no console request that can
  change a number a bill might rest on. That is the first question anybody asks
  of a metering system in a dispute, and the answer should be a grant rather
  than a promise.
- The monthly AI limit CP-2 could only record is now enforced against these
  numbers, in middleware rather than in each of the six AI handlers, answering
  429 — the allowance is spent, not the request forbidden. Storage remains
  measured and shown as not-yet-enforced, because refusing an upload means a
  check on every upload path and the screen should not imply one that is not
  there.

### Added — A front page that answers "is the platform well", and the platform's first backup

CP-4. The console's home screen is now the deployment's health — requests,
errors, latency, the government systems' lights, what is alerting, the disk,
which background jobs have quietly stopped, what version is running — and the
organisation list moved to its own page. Every panel links into Grafana, and
none of it tries to replace Grafana.

- **This platform had no backups.** Not a gap in the console: a gap in the
  platform. `deploy/scripts/backup.sh` takes a `pg_dump`, prunes what is older
  than a fortnight, and — the part that matters — writes the outcome into
  `platform_backups` whether it worked or not, because a cron job that fails
  silently is discovered on the morning somebody needs it. The console shows
  the last backup, its size, and the date of the last *restore test*, which is
  recorded by hand: an untested backup is not a backup, and the only way to
  know one has been tested is that somebody says so.
- **The deploy button asks GitHub, and can do nothing else.** It dispatches the
  workflow this repository already has, with a fine-grained token scoped to
  that workflow, behind a superadmin capability and a second factor — and it
  hands back the link to the run rather than polling somebody else's API for
  ten minutes. No shell, no `docker exec`, no environment editing: the plan
  lists all three among the things this console deliberately does not have.
- **Per-organisation error rates come from the audit trail, not from
  Prometheus**, because the very first phase decided that no metric would ever
  carry a tenant label. That decision has a price and this is it — paid
  deliberately, and arguably at a profit, since what an operator wants to know
  is whose *work* is failing.
- Every panel degrades on its own. A Prometheus that is down leaves the alerts,
  the jobs, the backups and the versions intact, and a deployment with no
  monitoring stack at all gets a page that says so rather than a page of zeroes
  that reads as "everything is fine". NaN — what PromQL returns when it divides
  by nothing — is read as zero rather than reaching a screen.

### Added — A platform that can be told to behave differently, without a deploy

CP-3, and the setting it exists for: **this platform is private by default**.
Until now, whether a stranger who could authenticate somewhere else became a
user here was decided by which environment variables happened to be set —
`EID_JIT_TENANT_SLUG` in one file, `SSO_CLIENT_TENANT` in another, read by two
packages, with no single place that answered "can somebody get in".

- **`platform.access_mode`.** One check, at the one thing every provisioning
  path does — create an account — and it is closed unless somebody says
  otherwise. Signing in as an existing user is untouched; so is an invitation,
  because being invited *is* the registration. The sign-in screen reads the
  mode and explains itself rather than letting people discover it by failing,
  and switching to public takes effect on the next request with no restart.
- **Settings that cannot hold a secret.** Every key is declared in Go with a
  kind, a default, a validation and the environment variable it still falls
  back to. There is no `secret` kind — not "should not be used", does not
  exist — and `Register` panics on a key that reads like one, so a credential
  cannot reach a table an operator can edit. A row whose key is not declared is
  ignored, so writing to the table is not a way to introduce behaviour.
- **Every change has a reason, a history and one button back.** The rollback is
  itself a change: it writes a new history row rather than removing the one it
  undoes, because a history that can be rewound is a history somebody can edit.
  Values reach the running platform through a thirty-second refresh *and* the
  invalidation bus, so a change is felt everywhere at once where Redis is
  present and within half a minute where it is not.
- **Feature flags with an expiry that is a reminder, not a switch.** Kinds
  (release, kill switch, experiment), per-organisation overrides, and a
  percentage rollout keyed on a stable hash — an organisation inside 10% is
  still inside 50%, and two flags at the same percentage select different
  organisations. Flags that have outlived their date keep working and appear as
  a warning on the configuration screen, which is the only thing that ever
  clears flag debt. A module's kill switch is a naming convention rather than
  new machinery: `module.<app id>.disabled`.
- **Maintenance and announcements.** Read-only for the platform or for one
  organisation, refusing writes with 503 while leaving every read and the way
  out working. Announcements carry their own expiry and arrive with `/me`, so
  the shell shows a banner without a second thing to poll.
- The first four settings — the access mode, the session idle timeout, the
  catalogue's sync interval and the AI model — are read where they are used
  rather than captured at startup, so changing them means changing them.

### Fixed

- Creating an organisation from the console could not have worked on a
  deployment with the row-level grants applied: the trigger that seeds a
  tenant's roles writes `role_permissions`, and the first administrator's
  account needs `INSERT` on `users`, neither of which the console's role held.
  Both were found by CP-3's integration tests rather than by a customer, which
  is the argument for narrow grants: a forgotten one fails loudly.

### Added — Operating an organisation from the console, without being able to take it

CP-2 gives the console the buttons CP-1 deliberately withheld: creating an
organisation, closing one, deleting one, helping somebody back into their
account, and — with a reason and a banner — looking at the platform as they see
it. Everything in it is shaped by one rule: an operator should be able to do
their job without being able to do quiet damage.

- **Nothing is sudden.** Suspension is reversible and ends every live session in
  the same transaction. Deletion is not a button: one superadmin asks, a
  *different* one agrees — the database refuses a self-approval with a CHECK
  constraint, not just the Go code — and only then does a thirty-day countdown
  start, cancellable throughout. The console holds no DELETE privilege on any
  table; a sweep on the platform path removes the rows when the time comes.
- **The console cannot change a password.** It sends a link. Migration `00050`
  grants it UPDATE on exactly two columns of `users` — the lockout counter and
  its expiry — so a support handler that tried to write a password hash is
  refused by PostgreSQL. The platform had no password-reset flow at all before
  this; invitations and resets now share one single-use, 24-hour token, sent
  through the mail rail the platform already had.
- **Impersonation, made impossible to do quietly.** A typed reason, the second
  factor again, thirty minutes, an amber banner the session itself drives, and
  two audit trails — ours and the organisation's own, which their
  administrators can read. A single-use sixty-second handover carries it across
  the hostname boundary, because a cookie cannot. A suspended organisation
  cannot be entered this way.
- **Limits.** `tenant_quotas` records people, storage and AI calls per
  organisation, soft (warns) or hard (refuses). Only the user count is enforced
  today, because it is the only one this platform can count; the screen says so
  rather than implying otherwise, and CP-5's metering is what switches the
  other two on.
- Creating an organisation installs its apps through the platform's own
  installer — the same path the store uses, with its dependency resolution —
  and reports which ones did not land instead of throwing the organisation
  away. Its first administrator gets an invitation, never a password an
  operator chose.
- The four operator roles are not a ladder: `operator` can open an organisation
  and cannot look inside one; `support` is the other way round. The whole
  authorization model is one map in `operator.go`.

### Added — A console for operating the platform, kept away from the platform

Somebody has to be able to see which organisations exist, which apps they run
and what has been done to them — and until now that somebody used `psql`. This
is the first phase of the operator console described in
[`docs/CONTROL_PLANE_PLAN.md`](docs/CONTROL_PLANE_PLAN.md): the foundation, on
which suspension, support and configuration are built next. Guide in
[`docs/CONTROL_PLANE.md`](docs/CONTROL_PLANE.md).

- **One binary, nothing else shared.** The console is `/cp/api` on the same Go
  process and `/cp` on the same Next.js build, because a second service would
  double the deployment and the monitoring for a console two people use. It
  shares nothing else: its own hostname, accounts, sessions, cookie, database
  role and audit table. A tenant administrator's account being taken reaches
  none of it.
- **Three layers to reach it, none trusting another.** nginx serves
  `cp.nexus.gerege.mn` behind an address allowlist that ships denying everyone;
  the API and the frontend both answer **404** — not 403, which would confirm
  something is there — to a request on any other hostname; and sign-in needs a
  password *and* an authenticator code. `CONTROL_PLANE_HOST` unset in
  production means the console does not exist, which is the safe reading of a
  variable nobody set.
- **The console cannot write.** Migration `00049` gives it a database role of
  its own with SELECT on ten named tables and read-only policies to match — so
  "the operator sees every organisation" is a list of permissions rather than a
  switch that turns row-level security off. A table nobody granted is a table
  the console cannot read, and a new one does not become visible by existing.
- **A write that was not recorded did not happen.** Every console write goes
  through one function that puts the change and its `operator_audit` row in the
  *same transaction*, and the middleware above it withholds the response —
  headers, cookies and all — from any write that answered successfully without
  an audit row. The table refuses UPDATE and DELETE at the database, by trigger,
  including from the role that owns it.
- **A code works once.** The time step of every accepted TOTP is stored and must
  strictly increase, so a code read over somebody's shoulder within its thirty
  seconds is already spent. Verified against RFC 6238's own test vectors.
- **No sign-up screen, ever.** The first operator is created by
  `operator-bootstrap` on the host, by somebody who already holds the database
  credentials — no default password, no first-run page, no environment variable
  left behind. The account cannot sign in until a code from its authenticator is
  confirmed, so an interrupted setup leaves a locked door rather than a
  password-only one.
- Sessions last 8 hours and end after 30 minutes idle, against the platform's
  12 and 90; step-up re-confirms the second factor for five minutes before a
  dangerous action, and is mounted on nothing yet because nothing dangerous
  exists yet.
- `cp_login_attempts_total{result}` counts sign-in attempts by outcome, apart
  from `logins_total`: platform sign-ins fail all day because people mistype
  passwords, while a dozen failures against the console in an hour is somebody
  trying.

### Fixed

- The CSRF middleware defended the tenant session cookie and would not have
  defended the console's, which was introduced in the same change. Both are
  named in one place now, so the next cookie-authenticated surface is a line
  there rather than a hole.

### Added — One organisation seeing another's report, with their permission

A coal mine contracts a hundred transport companies. Each keeps its trips in its
own tenant; the mine wants one consolidated "Transport" report. That request
runs against everything this platform is built to prevent, so the answer is not
to weaken the isolation but to add a separate, permissioned path beside it.
§3.5 of the design; guide in
[`docs/REPORT_SHARING.md`](docs/REPORT_SHARING.md).

- **Nothing crosses a tenant boundary.** A consolidated run calls the *ordinary*
  report once per grantor, **inside that grantor's own tenant context**. No
  policy is relaxed, no clause is rewritten, no query reads across
  organisations, and the report cannot tell it is being consolidated except by
  the counterparty reference it is handed.
- **Default deny, in three places.** No grant, no rows. A report must
  implement `Shareable` to be nameable in a grant at all — a report written
  assuming one organisation may aggregate in ways its rows do not. A request is
  not a permission: `accepted_at` is null until the owning organisation's
  administrator answers, and the query that reads grants ignores anything
  unaccepted, revoked or expired.
- **The scope is the agreement.** `counterparty` shows the mine the work done
  *for the mine*; `full` is the hierarchical case, a parent consolidating a
  subsidiary. A report that cannot filter by counterparty cannot be granted that
  scope — refused, rather than quietly widening to everything.
- **A two-sided row-level policy**, which is why `report_grants` deliberately
  has no `tenant_id` column: the general policy from 00029 would have attached
  itself and hidden each party's own agreements from the other, leaving the
  receiving side unable to see what it had been given. Accepting is still the
  owner's alone — the `grantor_tenant_id = $2` clause on that one statement is
  what stops a grantee accepting their own request.
- **Both sides are audited, every run.** The reader records that it read; the
  owner records that it was read, and can open "who has read our data" and see
  the organisation, the report, the moment and the row count. A transport
  company will only agree to this if it can check afterwards.
- **Revoking is immediate and grants are never deleted.** `DELETE` is not
  granted on the table: "who could see our data, and when" is a question asked
  after the fact.
- **One grantor failing does not fail the run.** A hundred organisations is a
  hundred chances of one slow query, and the ninety-seventh timing out must not
  produce nothing. That company is named in the result's notes instead — a total
  quietly missing a company is worse than one that says so.
- **Neither shipped billing report offers counterparty scope**, and that is the
  schema rather than a decision: `billing_invoices` records a contact *name*,
  not a registration number, and matching organisations by typed-in name is the
  mistake §3.5 avoids by keying grants on a registration number. Both offer
  `full`. The mechanism is complete and proven against a live database by the
  four tests §3.5 asks for, plus three more.

### Added — Reports, as a platform layer rather than a screen

`io.gerege.nexus.reports`. Every app that keeps data now has reports; the module
serving them knows about none of them. Guide in
[`docs/REPORTS.md`](docs/REPORTS.md).

- **A report is a declaration.** A module implements a Go interface saying what
  it is called in seven languages, what parameters it accepts, what columns it
  produces and how to produce them. The listing, the parameter form, the table,
  the chart, the Excel and CSV export, the schedule and the audit entry are
  written once and apply to every report anybody ever adds. Adding one is a file
  in a module and a line in its constructor — no handler, no frontend change.
- **Eight reports to start with**: revenue by month and invoice status
  (billing); stock on hand and movement summary (inventory); signatures by rail
  and signer activity (e-signature); user activity and headcount by unit (core).
  The e-signature one separates the rails and marks which is qualified, because
  only the eID rail produces a qualified signature in Mongolian law and a report
  that counted both together would answer the wrong question.
- **The tenant boundary is the database's, not the query author's.** A report
  runs inside the caller's tenant binding, in a **read-only transaction**, under
  a thirty-second `statement_timeout` — the SQL one, not a context deadline,
  because a cancelled context stops this process waiting and does not stop
  PostgreSQL working. A report that forgets its `WHERE tenant_id` returns
  nothing rather than everyone's rows, and there is an integration test against
  a real database that proves it.
- **Three gates, not one.** A report belonging to an app the organisation has
  not installed is absent from the listing, refused by key with a 404, and
  refused again when a schedule names it. Filtering the list is not enough: the
  API is a separate door.
- **Every run and every export is audited**, separately. An export is a copy of
  the organisation's numbers leaving the platform, which is a different act from
  reading them on screen — and §3.5 of the design requires both before one
  tenant may see anything of another's.
- **Scheduled reports, with no second process.** A cron expression, a
  minute-ticker goroutine in the API, and a PostgreSQL advisory lock. Due rows
  are claimed *before* the report is produced, not after it is sent: a replica
  restarting between "sent" and "recorded" would send the report twice, and a
  second copy is indistinguishable from the first while its numbers may differ.
- **Delivery is SMTP, and the design document said otherwise.** It called for
  the hosted verification service; that service sends one thing, a verification
  link, and has no endpoint for a subject, a body or an attachment. With
  `REPORT_SMTP_URL` unset a due schedule is still produced and still recorded,
  with "delivery not configured" as its outcome and a warning on the screen —
  which is a different and more useful state than not running.
- **Exports that can be used.** xlsx via `excelize` with real numeric cells, a
  totals row and a frozen header; CSV with a UTF-8 BOM, without which Excel on
  Windows renders every Mongolian heading as mojibake — the whole content of the
  file.

### Added — Traces, and errors that group themselves

The third pillar and the tool beside it, both env-gated and both off by default.
Guides in [`docs/MONITORING.md`](docs/MONITORING.md) §11 and §12.

- **`SetupTracing` now sets up tracing.** It was a stub that logged
  "opentelemetry tracing initialized" and initialized nothing — worse than no
  tracing, because an operator reading the startup log had every reason to
  believe traces existed somewhere. It is now the OTLP exporter, a batch
  processor, and a `ParentBased(TraceIDRatioBased)` sampler at 10%.
- **Off means off.** With `OTEL_EXPORTER_OTLP_ENDPOINT` unset there is no
  exporter, no batch processor, no background goroutine and no sampling
  decision: every span the code starts is a no-op. That is the condition for
  putting tracing in the default path rather than behind a build tag.
- **`otelhttp` on the router and `otelpgx` on the pool**, so a slow request
  resolves into the queries it waited on. Spans are named by chi's route
  pattern, never the URL — a span per document id is unbounded, the same
  cardinality argument the metrics middleware already makes. `/health`,
  `/ready` and `/metrics` are excluded: Docker and Prometheus call them every
  few seconds, and at any sampling rate they would be most of what is stored.
- **Query *parameters* are never recorded**, which is otelpgx's default and is
  now a comment saying it must stay that way. The arguments are the row a query
  is about — an address, a national identifier, a password hash on the way in —
  and a span is readable by anyone who can open Grafana. Verified against a
  live Tempo: `db.query.text` comes through as placeholders.
- **Logs join traces.** Every `slog` line written inside a span carries
  `trace_id` and `span_id`, and Grafana's Loki datasource turns the first into
  a link. Deliberately *not* the `otelslog` bridge: that ships logs over OTLP,
  a second delivery path for something Alloy already carries to Loki. Tempo's
  datasource links back the other way, to the container's logs and to the RED
  metrics for the service.
- **Tempo** in the monitoring stack — 72h retention, filesystem storage, no
  index over span contents. A trace is found by id from a log line, or through
  the span metrics and service graph Tempo generates into Prometheus.
- **GlitchTip** as `deploy/docker-compose.glitchtip.yml`, its own stack with its
  own Postgres. Separate rather than a compose profile because compose resolves
  `${VAR:?}` for every service in a file whether or not its profile is active,
  so a required secret there would have stopped the metrics stack from starting
  on a deployment that never wanted error tracking.
- **Panics are reported, with a scrubber that fails closed.** chi's `Recoverer`
  printed a stack trace to stdout and nothing else; the replacement logs it with
  the request id, the route and the tenant, and sends an event when `SENTRY_DSN`
  is set. What never leaves: the query string (where single-use references
  live), cookies, the `Authorization` header, the request body, and the person —
  e-mail, name and IP are dropped, the tenant id stays so "how many
  organisations does this affect" still has an answer. Headers are an
  allow-list, so one added by a future proxy is dropped rather than forwarded.
  The frontend half is `@sentry/nextjs` with the same rules and **no Session
  Replay**: it records the DOM of what the person was looking at, which here is
  a registration number or a document awaiting signature.

### Added — A monitoring stack that reads the platform

`deploy/docker-compose.monitoring.yml`: Prometheus, Alertmanager, Loki, Alloy,
Grafana, node_exporter, cAdvisor, postgres_exporter, redis_exporter. Guides in
[`docs/MONITORING.md`](docs/MONITORING.md) and, for every alert,
[`docs/RUNBOOKS.md`](docs/RUNBOOKS.md).

- **A separate compose file, brought up as a separate project.** Nothing in
  `docker-compose.prod.yml` depends on anything in it, no service in it is in a
  request path, and taking it down is a safe thing to do at any hour. It reaches
  the platform by joining the platform's own Docker network as an external one,
  so Prometheus scrapes `gerege_nexus_backend:8080` directly rather than through
  a published port.
- **Alerting on the error budget, not on a threshold.** The Google SRE Workbook's
  multi-window multi-burn-rate pattern against a 99.9% objective: 14.4× over
  1h+5m and 6× over 6h+30m page, 1× over 3d+6h opens a ticket. Both windows must
  be over the rate, which is what stops a spike that has already ended from
  waking anybody and what lets the alert clear itself. Every external-system rule
  is additionally guarded by a traffic condition — without one, a system called
  twice at 04:00 with one failure is a 50% error rate.
- **Runbooks are part of the alert.** Each rule carries a `runbook` annotation
  pointing at its section, and each section says what happened, what to check in
  the first five minutes, how to fix it and when to escalate. An alert nobody
  knows what to do about is an alert that gets silenced.
- **Dashboards as code**, with `allowUiUpdates: false`. Four of them: API
  overview (RED plus remaining error budget), external systems, infrastructure,
  and resilience/volume. A panel fixed at 02:00 during an incident is worth
  keeping, and the way to keep it is a commit rather than a row in a volume.
- **`monitoring` database role** (migration 00044) holding `pg_monitor` and
  nothing else — no table, no tenant row. Created without a password, because a
  migration is a file in this repository; the operator sets one once, and
  `docs/MONITORING.md` §2.2 is the command.
- **No secret in the repository.** Grafana's password is required with no
  default. Alertmanager's receivers are rendered at container start from the
  environment, so leaving SMTP or Telegram empty genuinely disables that channel
  instead of producing a config Alertmanager refuses to load — and a stack that
  will not start because nobody has a mail server is a stack nobody installs.
- **Logs, without turning Loki into Elasticsearch.** Alloy reads the Docker
  socket read-only and attaches `container`, `service`, `level` and `deployment`.
  `request_id` and `tenant_id` are deliberately *not* labels: each distinct value
  would be its own stream. They stay in the line, where `| json | request_id=…`
  finds them at query time.
- **Uptime Kuma is documented and deliberately not deployed here.** A monitor on
  the server it monitors goes down with it. `docs/MONITORING.md` §9 has the
  instructions for running it somewhere else.

### Added — The platform can now be measured

`/metrics` carried two series: a request count and a request duration. That is
the R and the D of RED and nothing else — no saturation, no business volume, no
sign that a call to ХУР or eID had gone slow, and no way to tell a breach of the
in-flight ceiling from any other 503. Everything a dashboard would need was
missing before the dashboards were, which is why this lands before the stack
that reads it (design: [`docs/MONITORING_AND_REPORTING_PROPOSAL.md`](docs/MONITORING_AND_REPORTING_PROPOSAL.md)).

- **Saturation.** The Go runtime and process collectors are asserted rather than
  assumed — client_golang registers both, and a test now fails if that ever
  stops being true. The pgx pool is exported as a collector read at scrape time:
  connections acquired, idle and total, the ceiling they are measured against,
  and the counters for acquisitions that had to wait or were abandoned.
- **Outbound calls.** One histogram,
  `external_request_duration_seconds{system,operation,status}`, across ХУР, eID,
  ДАН, the eSign HSM, Gemini and the address-verification service. It wraps the
  call rather than the transport, because three of those six are reached through
  clients whose `http.Client` is private to `open-gerege-core`. `system` is a
  closed list and an unrecognised name folds into `other`, so a call site added
  without a constant cannot widen the label set.
- **Business volume.** `logins_total{method,result}`, `invoices_created_total`,
  `documents_signed_total{rail,result}` and `ai_requests_total{kind}`, each
  incremented at the one place every path through it converges — `failGoogle`
  for one, `store.markSigned` for both e-signature rails. No tenant appears in
  any label: that breakdown is a reporting question, answered against rows that
  can be deleted rather than series that cannot.
- **The load shedder is visible.** `resilience_load_shed_total` and
  `resilience_in_flight_requests`. There is deliberately no breaker gauge: the
  adaptive breaker the design document assumed was removed from
  `platform/resilience` before this work began, and a gauge pinned at zero would
  render a panel claiming every breaker is closed on a platform that has none.
- **Logs carry the request.** Every `slog` line written while serving a request
  now carries `request_id` and `tenant_id`, read from the context by a handler
  wrapper rather than passed by hand through several hundred call sites. chi's
  colour access logger is gone with it — it wrote an unparseable second format
  into the middle of a JSON stream, named no request, and printed the raw path,
  which for `/api/v1/verify/{ref}` meant logging a single-use credential.
- **Audit events are kept.** New `audit_events` table (migration 00043) with the
  00029 tenant policy, written alongside the existing log line by
  `audit.Record` — same signature, so none of its sixty-eight call sites moved.
  The write is best effort and bounded at one second: an audit row failing must
  never fail the act it is recording, and the log line has already been written
  by then. `user_id` is text and unconstrained, because the trail has to outlive
  a deleted user and because the device handlers record `device:<id>` for an act
  nobody signed in for.

### Added — Signing in with Google

A "Google-ээр нэвтрэх" button beside eID on the platform's own sign-in screen,
off unless `GOOGLE_LOGIN_CLIENT_ID` is set. It is an *addition*, not the
federation added a moment ago: `SSO_CLIENT_ISSUER` closes this deployment's own
sign-in paths and hands the question of who somebody is to a provider, while
this is one more of its own answers and closes nothing. On a deployment that
does federate, the button is withdrawn along with the rest — a front door
nobody manages is exactly what federating was meant to remove.

Google is an ordinary OpenID Connect provider, so there is no second
implementation: the same discovery, PKCE, code exchange and RS256 `id_token`
verification serve both, and both land on the same `(issuer, subject)` account
resolution. What is written separately is only what differs — which cookie the
flow parks in, and who is allowed through.

- **The credentials are deliberately not the connectors'.** Drive and Meet
  already use `GOOGLE_OAUTH_CLIENT_ID`; the same Google project usually issues
  both and they may hold the same value, but inheriting a sign-in path from a
  document connector would mean enabling the connector quietly opened a new
  front door.
- **Three filters, in order.** An unverified address is refused, because the
  address is what an existing local account is matched on and an unverified one
  would let anybody who can type into a Google profile claim somebody else's
  account. Then `GOOGLE_LOGIN_ALLOWED_DOMAINS`, when set. Then the account
  itself: with no `GOOGLE_LOGIN_TENANT` nobody is provisioned, so a Google
  identity only ever reaches an account that already exists here.
- **No `id_token` is kept.** That cookie exists so signing out can end the
  session at a provider this deployment federates to; ending somebody's Google
  session because they signed out of this platform is not this platform's
  business.

### Added — A deployment can now be an SSO client, not only a provider

The platform has always been an OpenID Connect provider: it could hand
identities out and never take one in, so a group running several deployments had
one sign-in per deployment and no way to make one of them the source of truth.
This is the other half. Setting `SSO_CLIENT_ISSUER` makes a deployment a relying
party of the provider named there — including of another Gerege Nexus — and the
two halves are independent: an instance can be a provider, a client, or both,
which is what a regional deployment federating upward while still issuing
identities to its own installed apps needs. Full guide in
[`docs/SSO_FEDERATION.md`](docs/SSO_FEDERATION.md).

- **`ssoclient`, the relying-party protocol.** Discovery with the issuer check
  that makes every advertised endpoint trustworthy, a JWKS cache that refetches
  on an unknown `kid`, authorization with mandatory PKCE, the code exchange, and
  `id_token` verification that is deliberately narrow: RS256 only — `none` and
  the HMAC family are what alg confusion is made of — with `iss`, `aud`, `azp`,
  `exp`, `iat` and `nonce` all checked before a claim is believed. The pending
  sign-in lives in a short-lived HttpOnly cookie rather than a table, because a
  row would be written for every click of a sign-in button including every
  crawler's.
- **Client mode closes the local front door.** With a provider named, this
  deployment's password, eID and DAN sign-in endpoints stop answering and say
  where sign-in actually happens; the login screen becomes a hand-off. A
  deployment that federates its identity and also keeps its own password login
  has not federated anything — it has two front doors and one of them is
  unmanaged. `SSO_CLIENT_LOCAL_LOGIN=true` keeps them, as the documented way
  back in when the provider is the thing that is broken.
- **Signing out signs you out at the provider.** `/auth/logout` now answers with
  an `end_session_url` on a federated deployment, and the browser follows it: the
  provider ends its own session and returns the person to this deployment's
  registered post-logout address. Without that step, "sign out" followed by
  "sign in" walks straight back into the still-live session upstream.
- **Accounts are keyed on `(issuer, subject)`, never on the email address.** An
  address is a label a provider can change, and treating one as an identity means
  whoever is given a departed colleague's address inherits their account. A local
  account with a matching *verified* address is adopted on first federated
  sign-in, which is what makes federating a running deployment possible;
  `SSO_CLIENT_TENANT` decides whether a stranger the provider vouches for is
  provisioned at all, and unset means refused.

### Added — RP-initiated logout at the provider (`/oauth2/logout`)

The discovery document has advertised `end_session_endpoint` since it was
written, and nothing served it: a relying party that ended its own session and
sent the person here — which is what a conformant client does — landed on a 404
while staying signed in. That is worse than not advertising it, because the next
click on "sign in" looks like the logout was ignored.

- `post_logout_redirect_uris` is a new column on `oauth2_clients` (migration
  00041), editable from the developer portal and matched exactly. It is not
  `redirect_uris` reused: a sign-in callback is a machine-read path that receives
  a code, a post-logout address is a page a person looks at, and one list would
  widen both whenever either was extended. An unregistered return address is
  refused rather than followed — a logout URL is one a client hands out freely,
  so following one unchecked would make the provider an open redirector.
- The client is resolved from `client_id` or from a verified `id_token_hint`. An
  unverifiable hint is ignored rather than refused: by the time it is read the
  person is already signed out, and failing there would strand them.

### Fixed — Two defects the new tests turned up

- **A client registered with no post-logout addresses failed to insert.** A nil
  Go slice is sent as SQL `NULL`, and every array column on `oauth2_clients` is
  `NOT NULL` with an empty-array default — a default that only applies when the
  column is left out of the statement, and these are listed explicitly. Every
  array is now normalised at the store boundary.
- **HTTP Basic client credentials were not URL-decoded.** RFC 6749 §2.3.1 has a
  client form-urlencode both halves before base64ing them, so a conformant
  client's secret arrived escaped and was compared, still escaped, against what
  was registered. It never bit in practice because the secrets this provider
  mints are hex, but it would have bitten the first integrator who chose their
  own. A value that does not decode is used as it stands, so a client that
  skipped the encoding is not refused over a disagreement about transport.

### Changed — The session cookie is `SameSite=Lax`

Without this, single sign-on is not single. A relying party signing somebody in
sends the browser to `/oauth2/auth`, which is a top-level navigation arriving
from another site, and a `Strict` cookie is not sent on one — so the
authorization endpoint saw no session and showed a login screen to somebody who
had signed in a minute earlier. It costs nothing in CSRF terms, because the
cookie was never the defence: `Lax` adds exactly one thing over `Strict`, a
cross-site top-level *GET*, and every state-changing request goes through
`security.CSRFMiddleware`, which demands positive evidence that a page of ours
made it.

### Fixed — A lockout that never let go, and three silent truncations

- **A lapsed login lockout re-locked the account on the next single failure.**
  Five bad passwords lock an account for fifteen minutes, but the counter that
  decides that was only ever reset by a successful sign-in. Once it had reached
  five it stayed there, so after the window passed the next mistyped password
  met the threshold on its own and locked the account for another full fifteen
  minutes — indefinitely. Two consequences: nobody who had been locked out once
  could afford to typo again, and anybody who knew an address could hold it shut
  with one request every quarter of an hour. The count now restarts when the
  lock it produced has expired, and the lapsed lock is cleared by the same
  statement rather than left asserting a lockout that is over. Reaching five
  again still locks, so this is a restart and not a way out. The statement moved
  to a named constant behind `recordLoginFailure`; all of the behaviour is in
  the SQL, so the three tests that come with it need a real schema
  (`AUTH_TEST_DATABASE_URL`), and CI fails if they skip.
- **A tenant's menu could lose apps it had installed.**
  `GetEnabledAppIDsForTenant` discarded the per-row scan error and never checked
  the stream error, so a read that broke partway reached the caller as a short
  list with a nil error — and a broken stream leaves `rows.Next()` returning
  false exactly as a clean end does. That list is what the menu is built from,
  so the apps that fell off it read as ones the organisation had never
  installed. Its neighbour `GetInstallationsForTenant` already did this
  correctly.
- **The AI copilot could state a truncated search as fact.** Its product and
  knowledge tools dropped the same two errors, and there the truncation becomes
  a sentence: the model presents whatever it is handed, so half a result set is
  "you do not stock that" rather than an error the person can retry.

### Removed — Code that had stopped being reachable

- `oauthError.Error` made the type satisfy `error`, but it is a carrier — the
  code and description are rendered into an RFC 6749 §5.2 body and it is never
  wrapped or unwrapped — so the method was unreachable.
- `issueTokenSet` kept the token `SaveToken` hands back only to discard it with
  `_ = stored`; `SaveToken` returns the same pointer it was given.
- The integrations screen still rendered an error paragraph from a state nothing
  had set since the page moved to the banner, and the warehouses screen imported
  `useMemo` without using it.

### Changed — The platform's apps stop calling themselves examples

`io.example.*` was placeholder vocabulary from the first week — the reverse
domain of nobody, borrowed the way `example.com` is borrowed — and it had been
the primary key of every app in the store ever since. These are Gerege Nexus's
own apps and they now say so: **`io.gerege.nexus.*`**.

- A rename of a primary key is a data migration, not a search and replace.
  `00035` moves `apps`, `app_installations`, `app_versions` and
  `app_dependencies`, and rewrites the id inside each stored manifest — the copy
  an upgrade compares against to decide whether a new version asks for more than
  the installed one. Both foreign keys are `ON UPDATE NO ACTION`, so they come
  off and go back on around the update.
- The registry carries the matching migration and is deployed first. Between the
  two deployments an instance can sync a catalogue that already carries the new
  ids and file them as apps it has never seen; `00035` folds those back into the
  rows that hold the tenant's history — including an installation somebody made
  in that window — rather than colliding with them and failing the deployment.
- The migrations before `00035` are left as they were. They already ran
  everywhere, and a fresh database is expected to seed the old ids and then
  arrive here, which is what makes the migration equally true of a database
  created yesterday and one created next year. The entries above this one keep
  the ids they shipped with, for the same reason this file keeps the old
  repository name.
- `mn.example.hrms` in the test fixtures stays as it is: it stands in for
  somebody else's app, and there `example` is the point.

### Added — An organisation to be about

The module Odoo calls `base`, as a core app: the organisation itself, the
people in it, and how it is arranged. The platform had tenants, users and
memberships carrying only what signing somebody in needs — a slug, a name, an
email. A document that has to print a registration number, an approval that has
to name a department, a deadline counted in some timezone: none of those had
anywhere to come from, so each app either invented its own or went without.

- **Three screens** — `/organisation` (legal identity, address, contact, and the
  defaults everything else inherits: timezone, locale, currency),
  `/organisation/people` (the directory, with job title, department and roles),
  `/organisation/departments` (the structure as a tree, with a manager per unit).
- **The split follows Odoo's**, because the distinctions it draws are real:
  `res.company` → tenants + tenant_profiles, `res.users` → users, `hr.employee`
  → memberships, `hr.department` → departments. A language preference belongs to
  a person and follows them between organisations; a job title does not — the
  same person can be a director in one tenant and a clerk in another.
- **What the schema refuses rather than checks.** A department whose parent or
  manager belongs to another organisation is unrepresentable, not merely
  rejected: the foreign keys are composite over `(id, tenant_id)`. A tenant
  without a profile is impossible — a trigger creates one with the tenant, so no
  reader needs the null check. Both new tables carry the same forced RLS policy
  as everything else; migration 00029 wrote those once, over the tables that
  existed then, and a table added later has to say so itself.
- **What the handlers refuse**: deactivating yourself, and deactivating the last
  administrator. Both are support tickets otherwise. Nobody is deleted — a
  membership is referenced by everything the person did here, so people and
  departments are deactivated and archived instead.
- **Editing is partial by design.** The form sends the fields it touched and the
  server merges field by field, so correcting a phone number cannot blank a
  registration number.
- **Being core means two things the store now honours**: every tenant has it
  whether or not anybody installed it, and nobody can disable it. Settings →
  Apps says so where the Disable button would be, rather than offering one whose
  only outcome is a refusal.
- **A module with no blueprint no longer goes unlisted.** The sidebar was built
  only for apps named in `menu/blueprints.go` — the list of screens still to be
  built — so an app that had built everything it meant to build contributed
  nothing, including the menus it registers itself. Core walked into exactly
  that: three working screens and nothing pointing at them.
- The registry imports the bundled catalogue on every boot rather than only when
  it is empty. Otherwise a platform app added later reaches nobody: the registry
  is long past empty, the import is skipped, and every instance polls a
  catalogue without it.

### Added — The App Store moved to appstore.gerege.mn

The catalogue now comes from a registry of its own, and the apps in it can
be published by people who do not work here.

- **A registry service** (`backend/cmd/appstore`) serving a signed catalogue
  every instance pulls: Ed25519 over the raw bytes of the apps array, an ETag
  so an unchanged catalogue costs a 304, and the document built once per
  revision and stored as the bytes that were signed — rebuilding per request
  would hold only for as long as Go's encoder is byte-stable, and the failure
  when it is not is silent everywhere at once. It shares the platform's
  `appcatalog` types with the client that reads it, and a test signs a
  catalogue the way the endpoint does and feeds it to that client.
- **A storefront** (`appstore.gerege.mn`) that needs no account: server-rendered,
  seven languages as path segments, real 404s, and no install button — installing
  happens inside an organisation's own Nexus, so every page says that instead.
- **A publishing console** (`developer.gerege.mn`) where a publisher registers,
  submits a manifest and watches it through review. The authorization code is
  exchanged server-side and the identity token lives in an httpOnly cookie, so
  no token reaches page JavaScript and the platform needs no new CORS origin.
- **Installations follow the catalogue on their own**, unless the new version
  asks for more than the installed one — a widened permission, a widened OAuth
  scope, or a launch URL that has moved to another host. Those are held at the
  version they are on, with what they added recorded, and offered to the
  tenant's administrator as a decision rather than a button.
- `catalog-sign` generates the signing pair and signs a catalogue offline, for
  an air-gapped operator or for testing a client with no registry running.
- The OIDC endpoints at the root of nexus.gerege.mn are routed to the API. Only
  `/oauth2/token` ever was, which was enough for the platform's own screens and
  for nothing outside it.

### Added — Preparing the App Store to live at appstore.gerege.mn

The catalogue is on its way out of this repository and into a registry of its
own (`docs/APPSTORE_SEPARATION_PLAN.md`). Everything here works today in file
mode, which stays the default and the whole story for a self-hosted deployment;
the registry is opt-in and this platform never depends on it.

- **An installation's version now moves.** `InstallApp`'s reinstall branch
  updated status and enabled and left `installed_version` alone, so a tenant sat
  on 1.0.0 for ever while the catalogue carried the app forward — nothing could
  tell a current installation from a stale one. A version that actually changes
  is recorded as `'upgraded'` with the version it came from, and `SyncCatalog`
  finally writes `app_versions`, the table migration 00002 created and nobody
  ever filled.
- **Three records of a version are held to each other**: the compiled module,
  the catalogue entry and the manifest. They had drifted — esign shipped 2.0.0
  as a module and 1.0.0 in the catalogue, and the developer portal did the same.
  Both are corrected and the drift is now a startup error. `PlatformVersion`
  became a var a release build can stamp with `-ldflags`, and `/health` reports
  it.
- **A tenant can update an app it has already installed.**
  `POST /api/v1/store/apps/{slug}/upgrade` (admin) re-resolves dependencies,
  moves the version, records the event and refuses with 409 when there is
  nothing to move to. The store answers with `installed_version`,
  `latest_version` and `update_available`, compared as semver rather than as
  text, and the card carries an Update button beside enable/disable. Migration
  00033 adds `auto_update` and `pinned_version`.
- **The catalogue can come from a registry** (`APP_CATALOG_URL`): fetched with
  an ETag, verified against `APPSTORE_PUBLIC_KEY` before a single field of it is
  read, cached to disk, and behind all of it the bundled file. Boot never fails
  because of the registry — an unreachable or lying one costs an instance its
  updates and a line in the log. `CATALOG_SYNC_INTERVAL` drives a background
  refresh; `POST /api/v1/admin/store/sync` runs one on demand.
- **An app can be a platform that runs somewhere else** (`"type": "external"`).
  No Go module is required or looked for, permissions come from the manifest,
  and its menu entry opens in a new tab rather than pretending to be a route
  here. Its OAuth2 client is gated by installation: a user whose tenant has not
  installed the app is refused at `/oauth2/auth` with `access_denied`, and
  tokens carry `tenant_slug` beside `tenant_id` so the third party knows which
  organisation it has been handed.

### Added — Switching between the organisations you belong to

- **The membership table always allowed several; the runtime allowed one.**
  Which tenant a session acted for was decided at login by whichever membership
  was oldest (`internal/platform/auth_handlers.go`), and nothing could change it
  afterwards — signing out and back in landed the same person in the same
  tenant, deliberately, so somebody working for two organisations could reach
  only the first. `GET /api/v1/auth/tenants` lists the ones they may act for and
  `POST /api/v1/auth/switch-tenant` moves the session to one of them.
- **The token is rotated, not the row updated.** A session token is the
  authority to act inside one tenant, and the tenant is what changes; the new
  session inherits the old one's expiry, so moving between two organisations
  cannot be used to keep a session alive without signing in again. The
  membership check lives in the store, where no route can reach the insert
  without it, and a tenant the caller is not in answers 403 rather than 404 —
  whether it exists is not their business.
- **Both queries deliberately leave the tenant behind** (`tenant.Without`):
  `memberships` carries a `tenant_id` and is under the row-level policy, so a
  request bound to the current tenant would answer "which tenants may you act
  for" with the one the caller is already in.
- **The brand mark is the control**, and the account menu carries the same list
  — the mobile shell hides the header brand below 900px, and a phone is exactly
  where somebody moves between two organisations. Both render one component
  over one cached answer, so the two cannot drift and opening the second does
  not re-ask the server. The mark used to link to `/apps`, which the Platform
  tile beneath it in the rail still does. Choosing another organisation reloads
  the shell rather than patching state: the menus, the permissions and every
  list on screen belonged to the tenant just left.
- **The demo seed now has two organisations** (`cmd/api/seed.go`): Demo
  Corporation, with contacts, products, inventory and documents, and Demo Trade
  LLC, with contacts and billing. One tenant exercises nothing — the switcher,
  the row-level isolation and the per-tenant permission set all behave
  identically on a single-tenant deployment and identically wrongly if they are
  broken. The seeder runs after the platform server is built rather than before
  it, because an installation row references the `apps` table that the catalogue
  sync fills, and it installs through the installer so a demo tenant cannot
  claim an app whose Go module is not in the binary.

### Removed — The Swift macOS client (`desktop-mac/`)

- The bundle was a WKWebView pointed at the web client, plus a menu bar, Touch
  ID and a preferences window for the two server URLs. It was built by a shell
  script outside CI, so nothing compiled it on a pull request and nothing
  caught a Swift file that no longer built until somebody ran `make build-mac`
  by hand. `make build` ran it, which meant a build of this repository failed
  on any machine without Xcode.
- What it offered over the browser, the browser now offers: the web client is
  installable as a PWA and gets its own dock icon and window from
  `/manifest.webmanifest`, with no download and no store. The README section
  that documented `make build-mac` / `make run-mac` says that instead.
- The API keeps the path a native client would use — bearer tokens, no ambient
  cookie — so this is a client leaving, not the platform closing a door. Anyone
  wanting a native macOS app can build one against the same API in its own
  repository, where it can have a real build and a real signing identity.

### Removed — the Swift macOS shell, in favour of one shell for all platforms

- **`desktop-mac/` is gone.** It was the reference implementation of the bridge
  contract and it did its job: the contract exists because that shell was written
  first and the second one had to meet it. But once the Tauri shell shipped, macOS
  had two applications doing the same work, and the Tauri one had outgrown it —
  native sign-in, session restore, and a menu built from the tenant's own menu
  rather than a hand-written list. Keeping both meant implementing every contract
  change twice and running two CI workflows to prove the same thing.
- **What is actually lost is the `NSToolbar`**, which Tauri cannot draw. Its
  contents survive elsewhere: the app shortcuts are in the native menu, search is
  ⌘/Ctrl+F, reload and preferences are menu items, and the server status moved to
  the tray icon's tooltip when the Tauri shell was written.
- `make build-mac` / `make run-mac` are now `make build-desktop` /
  `make run-desktop`, and `.github/workflows/desktop-mac.yml` is removed — the
  three-platform Tauri workflow already covers what it checked.
- The entries below that describe `desktop-mac` are left as written. They record
  what shipped at the time, which is what a changelog is for.

### Fixed — the Tauri shell's bridge was dead on arrival

Found by running the app and signing in — none of it was visible to `cargo build`,
`clippy -D warnings`, `cargo test`, or the three-platform CI, all of which stayed
green throughout.

- **The work area could not reach the shell at all.** Tauri's ACL grants app
  commands to local pages but nothing to a remote origin, and the capability that
  was supposed to grant them was never listed in `tauri.conf.json`, so it was
  silently ignored. Every bridge call was rejected, the rejection was swallowed by
  a `catch`, and the request sat until its 40-second timeout. Both halves are now
  explicit, and the work area is granted only the three bridge commands — sign-in
  and preferences stay with the shell's own windows.
- **Defining any permission closed the door on the local windows too**: once an
  ACL exists, every app command is subject to it. The shell's own commands are now
  listed as well.
- **Every app appeared in the menu bar as "Модуль"**: the submenu was named after
  the first row the server returned, and the server returns a pathless group
  header first. A menu bar wants the application's name.
- **A request made before the work area finished loading hung until it timed out**,
  because the injected script it evaluates into did not exist yet. The script now
  announces itself, and the bridge waits for that.
- **The shell asked for a password on every launch** although the session cookie
  outlives the process in the webview's store; it now checks `/api/v1/auth/me`
  first — and, having restored a session, navigates to the work area instead of
  leaving the person on the sign-in landing page.
- **The health check was really a port check**: it polled `/healthz`, which this
  API does not serve, and counted the 404 as healthy. It now polls `/health` and
  requires a 2xx.

### Added — Tauri v2 desktop shell ([`desktop-tauri/`](desktop-tauri))

- **A second implementation of one contract, not a second product.** The bridge
  contract ([`docs/SHELL_CONTRACT.md`](docs/SHELL_CONTRACT.md)) is the
  specification; `desktop-mac/` is its Swift reference and this is
  the cross-platform one. Both inject the same `window.GeregeShell`, so the web
  app cannot tell them apart — it hides its own chrome and renders as a work area
  either way, and in a browser neither exists and nothing changes.
- **`platform` comes from the build target** (`macos`, `windows`, `linux`) and
  reaches the styling as `<html data-shell>`. Declared capabilities are
  `notify`, `badge`, `external.open`, `print.system`, `fs.save`, `menu.native`.
- **Native sign-in window** with email/password and both eID flows. The polling
  loop is Rust ([`auth.rs`](desktop-tauri/src-tauri/src/auth.rs)) and carries the
  same reasoning as [`EIDLogin.tsx`](frontend/components/EIDLogin.tsx): one check
  in flight at a time, a 400 ms gap between them because the server already holds
  each request for 25 s, three tolerated failures because a dropped long-poll is
  ordinary on a mobile network, and a 15-minute backstop that is a stop condition
  rather than a deadline. The QR is rendered to SVG in Rust so the window depends
  on no JavaScript library.
- **The session cookie forced the transport.** `session_token` is `HttpOnly` and
  belongs to the API origin, the web app authenticates with `credentials:
  "include"` and no bearer header, and neither Tauri nor wry can write a cookie
  into a webview from outside. The only way it lands in the right jar is for that
  webview to receive the `Set-Cookie` itself, so the sign-in requests are issued
  there ([`bridge.rs`](desktop-tauri/src-tauri/src/bridge.rs)) while the flow
  logic stays in Rust. The work-area window is created hidden and stays hidden
  until sign-in completes.
- **The native menu is the tenant's menu.** `GET /api/v1/menus` with the
  `Accept-Language` the person chose, grouped per app; `menu.changed` rebuilds
  it; choosing an item emits `shell:navigate` so the work area routes without a
  full reload. macOS maps a small set of icon names to native symbols and leaves
  the rest bare, which is steadier than half-matching them.
- **Server health lives on the tray icon**, checked every 5 s the way
  `ServerManager.swift` does it. Tauri has
  no native status bar and drawing an HTML one under the work area would put the
  shell inside the page it is supposed to stay out of. Being offline opens a
  native window that says what has to be running, not an alert that vanishes when
  dismissed.
- **Security**: main-frame navigation is confined to the Web URL's origin and
  everything else opens in the system browser; the bridge is main-frame only and
  the remote origin allowed to reach IPC is pinned in
  [`capabilities/`](desktop-tauri/src-tauri/capabilities); every native→web value
  is JSON-encoded rather than concatenated into JavaScript; `external.open`
  accepts only `http`, `https`, `mailto`, `tel`; `fs.saveAs` writes only where the
  person pointed. In a release build the API and Web URLs are compile-time
  constants — an installed shell cannot be aimed at a server it was not built for.
- **`gerege://` deep links** resolve to `shell:navigate`.
- **Auto-update is present and deliberately inert.** The plugin is left
  uninitialised with `TODO`s in three places; an updater carrying no signing key
  is a mechanism for installing unsigned code, so it stays off until a key exists.
- **Two capabilities are withheld, and why is recorded.** `secure-store` has no
  method in contract v1, so advertising it would be a claim nothing can act on —
  using it needs `secure.get`/`set`/`delete` added to the contract and a minor
  version bump. `biometric.authenticate` exists in the contract but Tauri's
  biometric plugin is mobile-only, so the capability is not declared and the call
  is rejected, which is what lets the web app fall back.
- **Not included**: installers and code signing. The shell builds; shipping it
  needs a Developer ID identity plus notarisation on macOS and an Authenticode
  certificate on Windows, both listed as TODO in
  [`desktop-tauri/README.md`](desktop-tauri/README.md).

### Added — CI for both desktop shells

- **[`desktop-tauri.yml`](.github/workflows/desktop-tauri.yml) builds on Linux,
  Windows and macOS.** Much of the shell sits behind `#[cfg(target_os = ...)]`, so
  a green build on one machine says nothing about the other two. It runs
  `cargo clippy --all-targets -- -D warnings`, `cargo build --locked` and
  `cargo test --locked`, with `fail-fast: false` because one platform failing is
  the signal the job exists to produce.
- **It found a real break on its first run**: `tauri-build` needs
  `icons/icon.ico` to generate the Windows resource, and the repository had only
  PNGs. Added `icon.ico` — sizes below 256 packed as classic DIB entries, since
  some resource compilers reject an all-PNG `.ico` — and `icon.icns` for macOS
  bundling.
- **`desktop-mac.yml` compiles the Swift
  shell** and then checks the two things a successful `swiftc` cannot: that
  `build.sh` still names every file under `src/` (a source missing from that fixed
  list is not a compile error — it is code that silently never ships), and that
  the produced bundle is one macOS would launch (`Info.plist`, an executable
  Mach-O, `codesign --verify --strict`).
- **The bridge fixes are guarded, not just documented.** The job fails on a
  `WKUserScript` injected into subframes or on JavaScript built by string
  interpolation — the exact two shapes that were removed. Both guards were checked
  against the pre-fix sources to confirm they actually catch them rather than
  passing vacuously.
- **Neither workflow produces a distributable artifact**, and both are filtered by
  path. A path-filtered workflow reports no status on runs that miss its filter,
  so making either a required check needs a merge queue or a companion job.

### Added — Native Shell + Web Work Area

- **The web app now knows whether it is a whole product or part of one.** Inside a
  native shell, sign-in, the header, the menus and device access belong to the
  shell; the web app hides its own chrome and renders as a **work area**. In a
  browser there is no shell, and everything below evaluates to nothing — the
  browser rendering is unchanged to the pixel, which is the constraint the whole
  design is built around rather than an afterthought.
- **One contract, written down** ([`docs/SHELL_CONTRACT.md`](docs/SHELL_CONTRACT.md)):
  injection rules, every method's parameters, result and failure, every event's
  payload, the capability names, the versioning rule (adding is minor, changing is
  major, and the shell announces its own version), and the security requirements a
  shell must meet. Two shells written by different people meet here or not at all.
- **`window.GeregeShell` in TypeScript** ([`frontend/lib/shell.ts`](frontend/lib/shell.ts)):
  `getShell()` returns `null` during SSR and in a browser, `hasCapability()`,
  a `useShell()` hook, and `invokeShell()` — an invoke that neither throws nor
  hangs, because callers mostly need to know whether the shell took the request,
  and "not supported", "failed" and "never answered" all mean the same thing: run
  the web fallback. Method, event and capability names are constants, so renaming
  one is a compiler error rather than a silent no-op.
- **Chromeless rendering** ([`Layout.tsx`](frontend/components/Layout.tsx)): in a
  shell the top bar, sidebar, mobile tabs and drawer are not rendered at all, but
  the menu and user fetches still run — RBAC and access checks depend on them, and
  only the drawing is removed. The AI assistant stays; it is part of the work area.
- **Session expiry asks the shell first.** There is no web `/login` page inside a
  shell, so a 401 calls `auth.reLogin` and falls back to `router.push("/login")`
  only if the shell will not, cannot, or does not answer — attempted once per
  session, so a re-login that leaves the session invalid cannot loop.
- **The two halves talk over the contract, not over URLs.** A menu change tells the
  shell with `menu.changed` so it can rebuild its native menu; the shell moves the
  work area with `shell:navigate` (internal paths only — a protocol-relative
  `//host` is not one) and opens its search with `shell:search`.
- **Native-leaning styling, scoped by attribute**
  ([`theme.tsx`](frontend/lib/theme.tsx), [`globals.css`](frontend/app/globals.css)):
  the shell's platform lands on `<html data-shell>`, which switches the app to the
  host's system font stack and chrome-free spacing, with a few per-platform
  touches. The attribute is absent in a browser, so no rule can reach it. The block
  sits above the density rules on purpose — a person who chose "compact" must not
  have it overruled by being in a shell.

### Fixed — Security: the macOS shell's JavaScript bridge

- **Native results were concatenated into JavaScript.** The biometric callback was
  assembled as `onBiometricResult('\(cb)', \(success), '\(err)')`, so a single
  quote anywhere in a system error message ran as code in the work area. The
  toolbar search field went the same way, which made anything the user typed a
  script. Every native→web value is now JSON-encoded and returned through one
  entry point (`WebViewController.swift`).
- **The bridge was injected into every frame.** `WKUserScript` is now main-frame
  only, and each message is checked twice — `isMainFrame`, and that the frame's
  origin matches the platform's web origin. An embedded third-party page has no
  business reaching biometrics, files or notifications.
- **The main frame could navigate anywhere.** It is now confined to an explicit
  allowlist — the web and API origins plus named identity origins — and every other
  address opens in the system browser rather than beside our session and our
  bridge. Deployments whose integration consent screens must stay in-app can name
  those origins in `gerege_nav_allowlist`; unlisted ones continue in the browser
  rather than breaking.
- **No functional regression**: tray, toolbar, printing, downloads and
  `gerege://` deep links continue to work, and deep links and menu items now move
  the work area through the router instead of reloading it, which no longer
  discards a half-filled form.
### Added — Email verification as a platform capability

- **One flow instead of one per app**
  ([`internal/platform/emailverify`](backend/internal/platform/emailverify)):
  proving that somebody controls an address is not one module's business.
  Contacts wants it before it trusts an address, Documents before a signing link
  leaves for an outsider, Gov Services before it answers a citizen at one. Each
  is the same act, so it lives in the platform: an app module takes the service
  in its constructor the way `gov_services` takes the integration manager and
  calls `emailverify.Service.Send` with its own app id as the source.
- **The mail is sent by a hosted service, deliberately.** Delivering mail that
  arrives is not a matter of holding an SMTP password: it is SPF, DKIM, DMARC,
  reverse DNS and a sending reputation, maintained continuously. enigma.mn runs
  that, so this platform holds no mailbox credential, composes no message and
  owns no sender address. What stays here is what only this platform can know —
  which module asked, for whom, why, and whether the person came back.
- **The return is good exactly once** (migrations `00026`, `00027`): the request
  carries a single-use reference in the return address, stored as a SHA-256, and
  claimed by one conditional `UPDATE`. A browser reloading the landing page
  races itself, and a reference that travelled through a mailbox and a browser's
  history must not be replayable. A spent, expired or invented reference is
  `410` alike.
- **The platform is not an open redirector**: the onward destination is
  validated when the request is made — HTTPS only (HTTP tolerated for localhost
  outside production), no embedded credentials — not when somebody arrives, by
  which time the mail has gone.
- **Mail bombing has a cost**: a per-tenant hourly allowance in front of the
  shared key and a one-minute pause per recipient, answered `429` with a
  `Retry-After` somebody can obey — a limit we can avoid provoking upstream is
  one we do not have to explain. A request the service refuses withdraws its own
  row, so the Overview screen never shows a verification nobody was asked for.
- **Errors say who has to act**: a bad address is `400`, a missing key or an
  HTTP `PUBLIC_ORIGIN` or a rejected key is `503` (this deployment, not the
  request), and a failure at the service is `502` and retryable. An answer
  nobody documented is never read as success.
- **Settings → Email verification**: whether the service is reachable, what has
  been asked for and by whom, the verified rate, and a test send. No key
  management — keys belong to the sending service and are administered there,
  and this platform's copy is a server-side environment variable that never
  reaches a browser.
- **The page shown after a click exists in all seven platform languages.** It is
  read outside the product, by somebody who may never have seen it.
- **Known limitation, stated on the screen**: the service has no webhook yet, so
  a verification is recorded only when the person returns here. Somebody who
  confirms on another device and never comes back stays `PENDING`. That is the
  honest reading — this platform did not see it happen — and it is what the
  Overview screen says rather than something the code knows and the operator
  does not.

### Added — PDF E-Sign v2: eID Mongolia qualified remote signing

- **eID Mongolia signature client ([`internal/platform/eidsign`](backend/internal/platform/eidsign))**:
  a real relying-party client for the v3 signature API. The citizen's own device
  holds the private key and approves with PIN2, so nothing here ever touches a
  signing key: we hash the PDF, eID pushes that digest to the phone, and the
  signed document is assembled by eID's own doc-signer
  (`POST /v3/signature/stamp/{sessionId}`), which embeds the PKCS#7 together with
  OCSP and CRL data. Certificate level defaults to `QUALIFIED` — accepting
  `ADVANCED` would silently downgrade every document the ERP produces.
- **Asynchronous signing ceremony** (`/api/v1/esign/sign/init`, `/sign/{id}`,
  `/sign/{id}/download`, `/sign/{id}/cancel`): upload → verification code →
  PIN2 on the phone → long-poll → PAdES-signed PDF. Sessions carry the exact
  bytes whose digest was approved, so a document edited mid-ceremony cannot
  produce a signature that fails to verify.
- **Signing on behalf of an organisation**: representation rights are read live
  from the national registry rather than from a certificate, because a director
  who resigned yesterday still holds yesterday's certificate.
- **eID identity linkage** (`user_eid_identities`, migration `00010`): sign-in
  now records who a user is to eID. Without it every signature would make the
  citizen retype the registration number they had just authenticated with, and a
  typo would push the PIN2 prompt at somebody else's phone.
- **The five module screens are now real** — signature log (filters, pagination,
  CSV export), batch signing, stamp placement (with an A4 preview), HSM
  connection (read-only, with a connection probe) and signing policy — replacing
  the `/module/esign/*` coming-soon placeholders.
- **Signing policy**: a tenant can require qualified eID signatures and disable
  the HSM rail outright, including for callers hitting the API directly.

### Fixed — PDF E-Sign

- **The app's permissions were declared but never enforced.** `io.example.esign`
  is absent from the platform's blanket app gate
  ([`server.go`](backend/internal/platform/server.go)) and its handlers only
  checked the tenant, so anyone in a tenant could sign. Every route now asserts
  `esign.read`, `esign.sign` or `esign.manage` explicitly. Migration `00010`
  backfills the grants existing roles should already have had, so no current
  user loses access.
- **`esign.sign` was ungrantable by the installer**: the default-role rules key
  off a `.read`/`.manage` suffix, so only administrators would ever have
  received it.
- **The signature log recorded only successes**, so a refused or expired
  ceremony left no trace — exactly the event an auditor looks for. Failures,
  refusals, expiries and downloads are now recorded with an outcome.
- **Non-ASCII download filenames were mangled**: signed PDFs are now served with
  an RFC 5987 `filename*`, so a Cyrillic document keeps its name.
- **Truncated PDFs were accepted**: a valid `%PDF-` header on a truncated body
  was passed to the signing service, which returned something that would not
  open. Uploads are now checked for a trailer as well as a header.
- **Sidebar sub-menus rendered as identical grey boxes**: the icons named by the
  server's menu blueprints were never mapped in the frontend
  ([`Layout.tsx`](frontend/components/Layout.tsx)).

### Added

- **PDF E-Sign App Module ([`io.example.esign`](backend/internal/apps/esign))**:
  - PDF document upload with tenant-scoped storage (migration `00009`), page-count detection, and original/signed download endpoints (`/api/v1/esign`).
  - Digital signature (тоон гарын үсэг) certificate validation and PKCS#7 PDF signing via the Gerege eSign HSM platform client ([`internal/platform/gerege/esign.go`](backend/internal/platform/gerege/esign.go)) — the private signing key never leaves the HSM.
  - Visible signature stamp placement with last-page auto-targeting and signature audit log (`esign_signature_logs`).
  - Frontend signing flow (`/esign`): certificate check → canvas signature pad → HSM signing → signed PDF download.
  - Mock mode by default (`ESIGN_MOCK_MODE`); configure `ESIGN_LOGIN_URL`, `ESIGN_SIGN_URL`, and `ESIGN_TOKEN` for live signing.

### Fixed — CI/CD pipeline

- **Go toolchain mismatch broke every job**: `backend/go.mod` requires `go 1.25.7`
  while the workflows pinned `go-version: "1.24"` and both Dockerfiles used
  `golang:1.24-alpine` (which sets `GOTOOLCHAIN=local`, so the build hard-fails).
  All jobs now resolve the version from `backend/go.mod` and the image builder
  sets `GOTOOLCHAIN=auto`.
- **Security workflow could never pass**: `govulncheck ./...` and `gosec ./...`
  ran at the repository root, which contains no `go.mod`. They now run against
  `backend/`. The workflow also only triggered on PRs to `master` while the
  default branch is `main`.
- **GHCR push lacked `packages: write`**, so deployment failed with `denied:
  installation not allowed to Write the repository`.
- **Removed the `swag-drift` job**: the sources carry no swagger annotations and
  `backend/docs/` is untracked, so it could only fail or pass vacuously.
- `deploy.yml` no longer duplicates lint/test from `ci.yml`; it runs migrations
  before swapping the API over, uses `docker compose`, lower-cases the GHCR
  image path, and skips cleanly when deployment secrets are absent.
- Added a **frontend CI job** (`npm ci` + `tsc --noEmit` + `next build`) — the
  Next.js app was never built by CI.
- Added `.dockerignore`, a pinned `backend/.golangci.yml` (v2), deleted the
  duplicate `backend/Dockerfile`, untracked committed `.DS_Store` files, and
  `gofmt`-ed the 11 files that had drifted.
- `docker-compose.yml`: added a one-shot **migration service** (the API used to
  start against an empty schema), health checks, and build-time
  `NEXT_PUBLIC_API_URL`; `frontend/Dockerfile` now uses `npm ci` with the lock
  file. Database credentials are consistent across compose, `.env.example` and
  the Makefile.

### Fixed — Security

- **Session tokens were the user's UUID**, the same value returned by
  `/api/v1/auth/me`. Replaced with opaque 256-bit `crypto/rand` tokens stored as
  SHA-256 digests in a new `sessions` table, with expiry and real revocation on
  logout (logout previously only dropped the cookie).
- **Mock national-identity mode was on by default** (`os.Getenv(...) != "false"`),
  so in production `/auth/eid/login` and `/auth/dan/login` accepted any
  registration number and logged the caller in as the first user in the table
  with `is_admin: true`. Mock mode is now refused in production unless requested
  explicitly, and identities are matched against a real ERP user.
- **OAuth2 token endpoint accepted any known `client_id` with no secret**
  (`clientSecret != "" && ...` skipped the check entirely). Client
  authentication is now mandatory and constant-time, supports HTTP Basic,
  validates the grant type, and is also enforced on `/oauth2/introspect` and
  `/oauth2/revoke`.
- Removed the **hard-coded client secret** `secret_gerege_dev_2026`;
  `ListClients` no longer discloses client secrets.
- App install/enable/disable and integration registration now require a **tenant
  administrator** — any authenticated user could previously reconfigure the
  tenant.
- Login rate limiting no longer trusts `X-Forwarded-For` unless
  `TRUST_PROXY_HEADERS=true`.
- Halved-entropy `generateRandomString` (hex output truncated back to `n`) fixed.
- `/metrics` no longer labels unmatched routes with the raw request path —
  unbounded Prometheus cardinality driven by unauthenticated requests.

### Fixed — App store & modules

- **Billing, Documents and the Developer Portal could not be installed**: their
  modules were never registered in `appregistry`, their rows were missing from
  the `apps` table (foreign-key violation), and `developer_portal` was rejected
  by the slug validator, which forbade underscores.
- The `apps` table is now **synchronised from `catalog/apps.json` on boot**
  instead of a hand-maintained INSERT that listed three of six apps.
- **Three manifests were malformed** (`"dependencies": {}` instead of an array,
  permissions as plain strings, `depends`/`sequence`/`action` keys). They parsed
  into a silent stub with no dependencies, permissions or menus. Manifests are
  fixed, and a manifest that fails to load is now a startup error.
- Billing and Documents no longer create their tables at boot with the error
  discarded; the schema moved into migration `00004`. Both stop answering failed
  writes with **fabricated demo records** (`inv_demo_100`, `doc_demo_200`).
- The app store reported disabled apps as "not installed" because `installed`
  and `enabled` were both derived from the enabled-only query.
- Fixed a nil-interface panic in `InstallApp` when a module was missing from the
  registry, and a nil-pointer dereference in the E-ID/DAN login handlers that
  called `err.Error()` on a nil error.

### Fixed — Reliability

- `AsyncOTPMailer.Shutdown` closed the queue while workers and retries could
  still send on it (**panic: send on closed channel**) and dropped already-queued
  mail; it now drains, is idempotent, and refuses post-shutdown enqueues.
- AI Copilot intent classification was case-sensitive against a lowercase
  keyword table, so "Stock" never matched.
- Restored demo-data seeding (dropped from `cmd/api`), now idempotent and
  disabled in production unless `SEED_DEMO_DATA` is set.

---

## [0.1.0] - 2026-08-05

### Added
- **Modular Monolith Core Architecture**:
  - Pure Go compile-time `Module` interface and global module registry (`appregistry`).
  - Tenant-level app installation, enablement, and menu visibility engine (`appinstaller`).
  - **ORY Hydra-Grade OAuth2 & OpenID Connect (OIDC) SSO Provider ([`internal/platform/ssoprovider`](backend/internal/platform/ssoprovider))**:
  - OpenID Connect Discovery (`/.well-known/openid-configuration`), JWKS URI (`/.well-known/jwks.json`), and OAuth2 Authorization Server (`/oauth2/token`, `/oauth2/introspect`, `/oauth2/revoke`).
  - Supports `authorization_code`, `client_credentials`, and `refresh_token` grant flows.
- **Developer Portal App Module ([`io.example.developer_portal`](backend/internal/apps/developer_portal))**:
  - Developer portal interface (`/developer/apps`) to register third-party OAuth2 client applications, issue Client IDs and Client Secrets, and manage redirect URIs.
- **Automated Production Deployment & CI/CD Pipeline ([`openerp.gerege.mn`](.github/workflows/deploy.yml))**:
  - Continuous Integration & Automated Deployment pipeline building GHCR Docker images and deploying to `openerp.gerege.mn`.
  - Production Multi-Stage Dockerfile ([`deploy/Dockerfile`](deploy/Dockerfile)) and Nginx SSL Reverse Proxy config (then `deploy/nginx/openerp.gerege.mn.conf`, since renamed to [`deploy/nginx/nexus.gerege.mn.conf`](deploy/nginx/nexus.gerege.mn.conf)).
  - Recursive dependency resolution algorithm with cycle detection and semver validation.
- **Shared-Schema Multi-Tenancy**:
  - Context-scoped `tenant_id` isolation across all business entities and repositories.
  - Tenant app gating middleware returning `403 Forbidden` for disabled modules.
- **Business Modules (Vertical Slices)**:
  - **Contacts (`io.example.contacts`)**: Business contacts directory with full CRUD.
  - **Products (`io.example.products`)**: Product catalog management with unique tenant-scoped SKUs.
  - **Inventory (`io.example.inventory`)**: Warehouse management, live stock levels, append-only stock movement log, and transactional stock adjustments with negative stock protection.
- **Next.js App Router Admin Shell**:
  - Top navigation bar with tenant badge (`Demo Corporation`), user profile menu, and logout.
  - Dynamic sidebar navigation driven by `/api/v1/menus`.
  - App Store (`/apps`) with search, categories, dependency badges, and Install/Enable/Disable controls.
  - Installed Apps Settings (`/settings/apps`).
  - Dedicated business UIs for `/contacts`, `/products`, and `/inventory`.
- **High-Performance Resilience Engine (go-zero Inspired)**:
  - **Adaptive Circuit Breaker (`resilience/breaker.go`)**: Google SRE style sliding window adaptive circuit breaker.
  - **Adaptive Load Shedding (`resilience/loadshedder.go`)**: In-flight HTTP request concurrency shedder returning `503 Service Unavailable` under heavy traffic spikes.
  - **Singleflight Coalescing (`resilience/singleflight.go`)**: Duplicate query suppressor preventing thundering herd cache stampedes.
  - **Exponential Backoff Retry (`resilience/retry.go`)**: `DoWithRetry` execution helper for resilient DB/network operations.
- **Observability & Async Messaging**:
  - Prometheus metrics endpoint (`/metrics`) recording HTTP request rates and latency histograms (`github.com/prometheus/client_golang`).
  - OpenTelemetry tracing initialization (`SetupTracing`).
  - Async OTP Mailer queue with worker pool, retry logic, and graceful shutdown (`internal/platform/mailer`).
- **Public Billing & e-Barimt Module ([`io.example.billing`](backend/internal/apps/billing))**:
  - Public service fee invoices, 10% VAT calculation for Mongolia e-Barimt, and status tracking (`/billing`).
- **Gerege DAN SSO Gateway System ([`dan.gerege.mn`](backend/internal/platform/dan))**:
  - Official Gerege Systems DAN SSO Gateway integration service (`POST /api/v1/auth/dan/login`).
  - Citizen identity verification and session token validation against `https://dan.gerege.mn/api/v1`.
- **E-ID Digital Identity & DAN SSO Authentication ([`internal/platform/eid`](backend/internal/platform/eid))**:
  - Aligned 100% with official **[eidmongolia.mn](https://eidmongolia.mn)** & **[developer.gerege.mn](https://developer.gerege.mn)** OAuth2 and OpenID Connect (OIDC) specifications.
  - Supports 4 official Mongolian authentication channels: PKI Digital Signature (Тоон гарын үсэг), Mobile OTP, Bank SSO, and Biometric Face Verification.
- **External System Integrations & Webhook Engine ([`internal/platform/integration`](backend/internal/platform/integration))**:
  - Event Dispatcher & Connector Manager supporting HMAC-SHA256 signature signing, asynchronous webhooks, and third-party REST connectors.
  - Dedicated Integration Settings Manager UI (`/settings/integrations`) with real-time status & health tracking.
- **XYP State Data Exchange System ([`xyp.gerege.mn`](backend/internal/platform/gerege/xyp.go))**:
  - Official Mongolian State Data Exchange (ХУР Төрийн мэдээлэл солилцооны систем) integration service.
  - Citizen civil registration (`POST /api/v1/xyp/citizen`) & company legal entity verification (`POST /api/v1/xyp/company`).
  - Interactive "ХУР / XYP Auto-fill" button integration on Contacts page.
- **Database & Migrations**:
  - Goose SQL migrations (`00001_platform_core.sql`, `00002_app_store.sql`, `00003_business_apps.sql`).
  - Automated initial demo data seeder (`admin@example.com` / `Password123!`).

---

### Inspirations & Acknowledgements
- **[snykk/go-rest-boilerplate](https://github.com/snykk/go-rest-boilerplate)** by [@snykk](https://github.com/snykk): Initial Go REST API structure.
- **[Odoo](https://github.com/odoo/odoo)**: Modular app ecosystem, App Store dependency resolver, and dynamic menu architecture.
- **[go-zero](https://github.com/zeromicro/go-zero)**: High-performance cloud-native resilience engine (Adaptive Circuit Breaker, Load Shedder, Singleflight, Exponential Retry).

### Authors & Contributors
- **Gerege Systems Development Team**
- **Gemini AI**
- **Claude AI**
