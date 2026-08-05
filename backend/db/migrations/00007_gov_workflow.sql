-- +goose Up

-- Configurable government service workflow.
--
-- Additive over 00006: the existing gov_services / gov_applications /
-- gov_application_events / gov_appointments tables keep their identity and
-- data. gov_applications stays the top-level service request; per-level
-- fulfilment work moves into gov_tasks.
--
-- Cross-tenant references are prevented in the schema, not only in code: every
-- table that points at an organisational unit or a workflow version carries a
-- composite foreign key that includes tenant_id.

-- ─── Organisational hierarchy ────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS gov_org_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    name_en VARCHAR(255) NOT NULL DEFAULT '',
    unit_type VARCHAR(32) NOT NULL DEFAULT 'DEPARTMENT',
    parent_id UUID,
    -- Used by the REGION_MATCH routing strategy.
    region_code VARCHAR(32) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_org_units_code_uniq UNIQUE (tenant_id, code),
    -- Referenced by every composite foreign key below.
    CONSTRAINT gov_org_units_tenant_uniq UNIQUE (id, tenant_id),
    CONSTRAINT gov_org_units_not_self CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT gov_org_units_type_valid CHECK (
        unit_type IN ('ROOT', 'MINISTRY', 'AGENCY', 'DEPARTMENT', 'DIVISION', 'BRANCH', 'DISTRICT', 'KHOROO')
    ),
    -- A parent must belong to the same tenant.
    CONSTRAINT gov_org_units_parent_fk FOREIGN KEY (parent_id, tenant_id)
        REFERENCES gov_org_units (id, tenant_id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_gov_org_units_tenant ON gov_org_units(tenant_id, active);
CREATE INDEX IF NOT EXISTS idx_gov_org_units_parent ON gov_org_units(tenant_id, parent_id);

-- Which unit a user acts for. Authorisation scope is derived from this.
CREATE TABLE IF NOT EXISTS gov_unit_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    unit_id UUID NOT NULL,
    -- SUPERVISOR additionally sees and verifies the work of descendant units.
    unit_role VARCHAR(32) NOT NULL DEFAULT 'OFFICER',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_unit_members_uniq UNIQUE (tenant_id, user_id, unit_id),
    CONSTRAINT gov_unit_members_role_valid CHECK (unit_role IN ('OFFICER', 'SUPERVISOR')),
    CONSTRAINT gov_unit_members_unit_fk FOREIGN KEY (unit_id, tenant_id)
        REFERENCES gov_org_units (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gov_unit_members_user ON gov_unit_members(tenant_id, user_id);

-- ─── Versioned workflow configuration ────────────────────────────────────────

CREATE TABLE IF NOT EXISTS gov_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    name_en VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_workflows_code_uniq UNIQUE (tenant_id, code),
    CONSTRAINT gov_workflows_tenant_uniq UNIQUE (id, tenant_id)
);

-- A published version is immutable: in-flight requests keep executing the
-- definition they started on. Changes are published as a new version.
CREATE TABLE IF NOT EXISTS gov_workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL,
    version INT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'DRAFT',
    published_at TIMESTAMPTZ,
    published_by VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_wf_versions_uniq UNIQUE (workflow_id, version),
    CONSTRAINT gov_wf_versions_tenant_uniq UNIQUE (id, tenant_id),
    CONSTRAINT gov_wf_versions_status_valid CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    CONSTRAINT gov_wf_versions_workflow_fk FOREIGN KEY (workflow_id, tenant_id)
        REFERENCES gov_workflows (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gov_wf_versions_workflow ON gov_workflow_versions(tenant_id, workflow_id, status);

CREATE TABLE IF NOT EXISTS gov_workflow_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version_id UUID NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    name_en VARCHAR(255) NOT NULL DEFAULT '',
    step_order INT NOT NULL,
    step_type VARCHAR(16) NOT NULL DEFAULT 'FULFILL',
    -- How the executing unit is chosen when a task enters this step.
    executor_rule VARCHAR(24) NOT NULL DEFAULT 'SELF',
    executor_unit_id UUID,
    -- Validated JSON only; no expression evaluation.
    executor_config JSONB NOT NULL DEFAULT '{}',
    sla_hours INT NOT NULL DEFAULT 72,
    requires_verification BOOLEAN NOT NULL DEFAULT FALSE,
    is_terminal BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT gov_wf_steps_code_uniq UNIQUE (version_id, code),
    CONSTRAINT gov_wf_steps_order_uniq UNIQUE (version_id, step_order),
    CONSTRAINT gov_wf_steps_tenant_uniq UNIQUE (id, tenant_id),
    CONSTRAINT gov_wf_steps_type_valid CHECK (step_type IN ('FULFILL', 'VERIFY', 'DECISION')),
    CONSTRAINT gov_wf_steps_rule_valid CHECK (
        executor_rule IN ('SELF', 'PARENT', 'CHILD', 'SPECIFIC_UNIT', 'REGION_MATCH')
    ),
    CONSTRAINT gov_wf_steps_sla_positive CHECK (sla_hours > 0),
    -- SPECIFIC_UNIT is the only rule that names a unit up front.
    CONSTRAINT gov_wf_steps_specific_unit CHECK (
        executor_rule <> 'SPECIFIC_UNIT' OR executor_unit_id IS NOT NULL
    ),
    CONSTRAINT gov_wf_steps_version_fk FOREIGN KEY (version_id, tenant_id)
        REFERENCES gov_workflow_versions (id, tenant_id) ON DELETE CASCADE,
    CONSTRAINT gov_wf_steps_unit_fk FOREIGN KEY (executor_unit_id, tenant_id)
        REFERENCES gov_org_units (id, tenant_id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_gov_wf_steps_version ON gov_workflow_steps(version_id, step_order);

CREATE TABLE IF NOT EXISTS gov_workflow_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version_id UUID NOT NULL,
    -- NULL from_step means "any step in this version".
    from_step_id UUID,
    from_status VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL,
    to_step_id UUID,
    to_status VARCHAR(32) NOT NULL,
    required_permission VARCHAR(64) NOT NULL DEFAULT '',
    CONSTRAINT gov_wf_transitions_uniq UNIQUE (version_id, from_step_id, from_status, action),
    CONSTRAINT gov_wf_transitions_version_fk FOREIGN KEY (version_id, tenant_id)
        REFERENCES gov_workflow_versions (id, tenant_id) ON DELETE CASCADE,
    CONSTRAINT gov_wf_transitions_from_step_fk FOREIGN KEY (from_step_id, tenant_id)
        REFERENCES gov_workflow_steps (id, tenant_id) ON DELETE CASCADE,
    CONSTRAINT gov_wf_transitions_to_step_fk FOREIGN KEY (to_step_id, tenant_id)
        REFERENCES gov_workflow_steps (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gov_wf_transitions_lookup
    ON gov_workflow_transitions(version_id, from_status, action);

-- Published versions are frozen at the database level, so no code path can
-- change the behaviour of a request that is already running.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gov_reject_published_workflow_change() RETURNS trigger AS $$
DECLARE
    v_version UUID;
    v_status TEXT;
BEGIN
    v_version := COALESCE(NEW.version_id, OLD.version_id);
    SELECT status INTO v_status FROM gov_workflow_versions WHERE id = v_version;
    IF v_status = 'PUBLISHED' THEN
        RAISE EXCEPTION 'workflow version % is published and immutable', v_version
            USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER gov_wf_steps_immutable
    BEFORE INSERT OR UPDATE OR DELETE ON gov_workflow_steps
    FOR EACH ROW EXECUTE FUNCTION gov_reject_published_workflow_change();

CREATE TRIGGER gov_wf_transitions_immutable
    BEFORE INSERT OR UPDATE OR DELETE ON gov_workflow_transitions
    FOR EACH ROW EXECUTE FUNCTION gov_reject_published_workflow_change();

-- ─── Routing rules ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS gov_routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- NULL service means the rule applies to every service of the tenant.
    service_id UUID,
    priority INT NOT NULL DEFAULT 100,
    -- Empty match_field means "always matches".
    match_field VARCHAR(64) NOT NULL DEFAULT '',
    match_value VARCHAR(128) NOT NULL DEFAULT '',
    strategy VARCHAR(24) NOT NULL,
    target_unit_id UUID,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_routing_strategy_valid CHECK (
        strategy IN ('SELF', 'PARENT', 'CHILD', 'SPECIFIC_UNIT', 'REGION_MATCH')
    ),
    CONSTRAINT gov_routing_specific_unit CHECK (
        strategy <> 'SPECIFIC_UNIT' OR target_unit_id IS NOT NULL
    ),
    CONSTRAINT gov_routing_service_fk FOREIGN KEY (service_id) REFERENCES gov_services(id) ON DELETE CASCADE,
    CONSTRAINT gov_routing_unit_fk FOREIGN KEY (target_unit_id, tenant_id)
        REFERENCES gov_org_units (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gov_routing_lookup ON gov_routing_rules(tenant_id, service_id, priority);

-- ─── Service configuration ───────────────────────────────────────────────────

ALTER TABLE gov_services ADD COLUMN IF NOT EXISTS workflow_version_id UUID;
ALTER TABLE gov_services ADD COLUMN IF NOT EXISTS fulfillment_mode VARCHAR(16) NOT NULL DEFAULT 'LOCAL';
ALTER TABLE gov_services ADD COLUMN IF NOT EXISTS owner_unit_id UUID;
ALTER TABLE gov_services ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_services_tenant_uniq') THEN
        ALTER TABLE gov_services ADD CONSTRAINT gov_services_tenant_uniq UNIQUE (id, tenant_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_services_mode_valid') THEN
        ALTER TABLE gov_services ADD CONSTRAINT gov_services_mode_valid
            CHECK (fulfillment_mode IN ('LOCAL', 'DELEGATE', 'HYBRID'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_services_workflow_fk') THEN
        ALTER TABLE gov_services ADD CONSTRAINT gov_services_workflow_fk
            FOREIGN KEY (workflow_version_id, tenant_id)
            REFERENCES gov_workflow_versions (id, tenant_id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_services_owner_unit_fk') THEN
        ALTER TABLE gov_services ADD CONSTRAINT gov_services_owner_unit_fk
            FOREIGN KEY (owner_unit_id, tenant_id)
            REFERENCES gov_org_units (id, tenant_id) ON DELETE SET NULL;
    END IF;
END
$$;
-- +goose StatementEnd

-- ─── Requests ────────────────────────────────────────────────────────────────

ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS workflow_version_id UUID;
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS fulfillment_mode VARCHAR(16) NOT NULL DEFAULT 'LOCAL';
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS source_system VARCHAR(64) NOT NULL DEFAULT 'PORTAL';
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS external_request_id VARCHAR(128);
-- SHA-256 of the canonical ingestion payload; a replay carrying the same
-- external id but a different fingerprint is a conflict, not a retry.
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS payload_fingerprint CHAR(64) NOT NULL DEFAULT '';
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS request_payload JSONB NOT NULL DEFAULT '{}';
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS origin_unit_id UUID;
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS current_unit_id UUID;
ALTER TABLE gov_applications ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_applications_tenant_uniq') THEN
        ALTER TABLE gov_applications ADD CONSTRAINT gov_applications_tenant_uniq UNIQUE (id, tenant_id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_applications_mode_valid') THEN
        ALTER TABLE gov_applications ADD CONSTRAINT gov_applications_mode_valid
            CHECK (fulfillment_mode IN ('LOCAL', 'DELEGATE', 'HYBRID'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_applications_origin_unit_fk') THEN
        ALTER TABLE gov_applications ADD CONSTRAINT gov_applications_origin_unit_fk
            FOREIGN KEY (origin_unit_id, tenant_id)
            REFERENCES gov_org_units (id, tenant_id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'gov_applications_current_unit_fk') THEN
        ALTER TABLE gov_applications ADD CONSTRAINT gov_applications_current_unit_fk
            FOREIGN KEY (current_unit_id, tenant_id)
            REFERENCES gov_org_units (id, tenant_id) ON DELETE SET NULL;
    END IF;
END
$$;
-- +goose StatementEnd

-- Idempotency identity for upstream ingestion.
CREATE UNIQUE INDEX IF NOT EXISTS idx_gov_applications_external
    ON gov_applications(tenant_id, source_system, external_request_id)
    WHERE external_request_id IS NOT NULL;

-- ─── Per-level tasks ─────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS gov_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    application_id UUID NOT NULL,
    parent_task_id UUID,
    unit_id UUID NOT NULL,
    assigned_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    workflow_version_id UUID,
    step_id UUID,
    status VARCHAR(32) NOT NULL DEFAULT 'RECEIVED',
    due_at TIMESTAMPTZ,
    result_code VARCHAR(64) NOT NULL DEFAULT '',
    result_description TEXT NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ,
    completed_by VARCHAR(255) NOT NULL DEFAULT '',
    verified_at TIMESTAMPTZ,
    verified_by VARCHAR(255) NOT NULL DEFAULT '',
    -- Optimistic concurrency: every transition bumps this and updates are
    -- guarded by the version the caller read.
    row_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_tasks_tenant_uniq UNIQUE (id, tenant_id),
    CONSTRAINT gov_tasks_status_valid CHECK (status IN (
        'RECEIVED', 'ASSIGNED', 'IN_PROGRESS', 'FORWARDED', 'INFO_REQUESTED',
        'COMPLETED', 'AWAITING_VERIFICATION', 'RETURNED', 'REJECTED', 'CLOSED', 'CANCELLED'
    )),
    CONSTRAINT gov_tasks_not_self_parent CHECK (parent_task_id IS NULL OR parent_task_id <> id),
    CONSTRAINT gov_tasks_application_fk FOREIGN KEY (application_id, tenant_id)
        REFERENCES gov_applications (id, tenant_id) ON DELETE CASCADE,
    CONSTRAINT gov_tasks_parent_fk FOREIGN KEY (parent_task_id, tenant_id)
        REFERENCES gov_tasks (id, tenant_id) ON DELETE SET NULL,
    CONSTRAINT gov_tasks_unit_fk FOREIGN KEY (unit_id, tenant_id)
        REFERENCES gov_org_units (id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT gov_tasks_version_fk FOREIGN KEY (workflow_version_id, tenant_id)
        REFERENCES gov_workflow_versions (id, tenant_id) ON DELETE SET NULL,
    CONSTRAINT gov_tasks_step_fk FOREIGN KEY (step_id, tenant_id)
        REFERENCES gov_workflow_steps (id, tenant_id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_gov_tasks_application ON gov_tasks(application_id);
CREATE INDEX IF NOT EXISTS idx_gov_tasks_queue ON gov_tasks(tenant_id, unit_id, status);
CREATE INDEX IF NOT EXISTS idx_gov_tasks_due ON gov_tasks(tenant_id, status, due_at);

-- ─── Timeline ────────────────────────────────────────────────────────────────
-- Extends the 00006 timeline table rather than introducing a second one.

ALTER TABLE gov_application_events ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE gov_application_events ADD COLUMN IF NOT EXISTS task_id UUID;
ALTER TABLE gov_application_events ADD COLUMN IF NOT EXISTS unit_id UUID;
ALTER TABLE gov_application_events ADD COLUMN IF NOT EXISTS from_status VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE gov_application_events ADD COLUMN IF NOT EXISTS to_status VARCHAR(32) NOT NULL DEFAULT '';

UPDATE gov_application_events e
   SET tenant_id = a.tenant_id
  FROM gov_applications a
 WHERE a.id = e.application_id AND e.tenant_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_gov_events_task ON gov_application_events(task_id);

-- ─── Upstream delivery outbox ────────────────────────────────────────────────
-- Status updates are written inside the same transaction as the state change
-- and delivered separately, so a remote endpoint can never roll back or stall
-- a workflow transition.

-- Where status updates for a given upstream system are delivered. The signing
-- secret is referenced by name (an integration connector id or env key); it is
-- never stored or logged here.
CREATE TABLE IF NOT EXISTS gov_upstream_connectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_system VARCHAR(64) NOT NULL,
    target_url TEXT NOT NULL,
    secret_ref VARCHAR(128) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_upstream_connectors_uniq UNIQUE (tenant_id, source_system)
);

CREATE TABLE IF NOT EXISTS gov_delivery_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    application_id UUID NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    target_url TEXT NOT NULL DEFAULT '',
    -- Name of the integration connector holding the signing secret. The secret
    -- itself is never stored here and never logged.
    signing_secret_ref VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gov_outbox_status_valid CHECK (status IN ('PENDING', 'DELIVERING', 'DELIVERED', 'FAILED')),
    CONSTRAINT gov_outbox_application_fk FOREIGN KEY (application_id, tenant_id)
        REFERENCES gov_applications (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gov_outbox_pending
    ON gov_delivery_outbox(tenant_id, status, next_attempt_at);

-- ─── Backfill ────────────────────────────────────────────────────────────────
--
-- Every tenant that already uses the module gets a root unit and a published
-- "LEGACY_LOCAL" workflow, and each existing application gets exactly one task
-- carrying its current state. Legacy status mapping:
--
--   SUBMITTED      → RECEIVED         IN_REVIEW  → IN_PROGRESS
--   INFO_REQUESTED → INFO_REQUESTED   APPROVED   → COMPLETED
--   REJECTED       → REJECTED         COMPLETED  → CLOSED
--   CANCELLED      → CANCELLED

INSERT INTO gov_org_units (tenant_id, code, name, name_en, unit_type)
SELECT DISTINCT s.tenant_id, 'HQ', 'Төв байгууллага', 'Head office', 'ROOT'
  FROM gov_services s
 WHERE NOT EXISTS (
    SELECT 1 FROM gov_org_units u WHERE u.tenant_id = s.tenant_id AND u.code = 'HQ'
 );

INSERT INTO gov_workflows (tenant_id, code, name, name_en, description)
SELECT DISTINCT s.tenant_id, 'LEGACY_LOCAL', 'Байгууллага дээрээ шийдвэрлэх (хуучин)',
       'Local fulfilment (legacy)', 'Created by migration 00007 for services that predate configurable workflows.'
  FROM gov_services s
 WHERE NOT EXISTS (
    SELECT 1 FROM gov_workflows w WHERE w.tenant_id = s.tenant_id AND w.code = 'LEGACY_LOCAL'
 );

INSERT INTO gov_workflow_versions (tenant_id, workflow_id, version, status, published_at, published_by)
SELECT w.tenant_id, w.id, 1, 'DRAFT', NULL, ''
  FROM gov_workflows w
 WHERE w.code = 'LEGACY_LOCAL'
   AND NOT EXISTS (SELECT 1 FROM gov_workflow_versions v WHERE v.workflow_id = w.id);

INSERT INTO gov_workflow_steps (tenant_id, version_id, code, name, name_en, step_order, step_type,
                                executor_rule, sla_hours, requires_verification, is_terminal)
SELECT v.tenant_id, v.id, 'FULFILL', 'Шийдвэрлэх', 'Fulfil', 1, 'FULFILL', 'SELF', 72, FALSE, TRUE
  FROM gov_workflow_versions v
  JOIN gov_workflows w ON w.id = v.workflow_id AND w.code = 'LEGACY_LOCAL'
 WHERE NOT EXISTS (SELECT 1 FROM gov_workflow_steps st WHERE st.version_id = v.id);

-- Publish only after the steps exist, otherwise the immutability trigger fires.
UPDATE gov_workflow_versions v
   SET status = 'PUBLISHED', published_at = NOW(), published_by = 'migration:00007'
  FROM gov_workflows w
 WHERE w.id = v.workflow_id AND w.code = 'LEGACY_LOCAL' AND v.status = 'DRAFT';

UPDATE gov_services s
   SET workflow_version_id = v.id,
       owner_unit_id = u.id,
       fulfillment_mode = 'LOCAL'
  FROM gov_workflow_versions v
  JOIN gov_workflows w ON w.id = v.workflow_id AND w.code = 'LEGACY_LOCAL'
  JOIN gov_org_units u ON u.tenant_id = w.tenant_id AND u.code = 'HQ'
 WHERE s.tenant_id = w.tenant_id AND s.workflow_version_id IS NULL;

UPDATE gov_applications a
   SET workflow_version_id = s.workflow_version_id,
       fulfillment_mode = s.fulfillment_mode,
       current_unit_id = s.owner_unit_id,
       origin_unit_id = s.owner_unit_id
  FROM gov_services s
 WHERE s.id = a.service_id AND a.workflow_version_id IS NULL;

INSERT INTO gov_tasks (tenant_id, application_id, unit_id, workflow_version_id, step_id, status,
                       due_at, completed_at, created_at, updated_at)
SELECT a.tenant_id,
       a.id,
       a.current_unit_id,
       a.workflow_version_id,
       st.id,
       CASE a.status
           WHEN 'SUBMITTED' THEN 'RECEIVED'
           WHEN 'IN_REVIEW' THEN 'IN_PROGRESS'
           WHEN 'INFO_REQUESTED' THEN 'INFO_REQUESTED'
           WHEN 'APPROVED' THEN 'COMPLETED'
           WHEN 'REJECTED' THEN 'REJECTED'
           WHEN 'COMPLETED' THEN 'CLOSED'
           WHEN 'CANCELLED' THEN 'CANCELLED'
           ELSE 'RECEIVED'
       END,
       a.created_at + (st.sla_hours || ' hours')::INTERVAL,
       a.decided_at,
       a.created_at,
       a.updated_at
  FROM gov_applications a
  JOIN gov_workflow_steps st ON st.version_id = a.workflow_version_id
 WHERE a.current_unit_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM gov_tasks t WHERE t.application_id = a.id);

-- +goose Down
DROP TABLE IF EXISTS gov_delivery_outbox;
DROP TABLE IF EXISTS gov_upstream_connectors;
DROP TABLE IF EXISTS gov_tasks;
DROP TRIGGER IF EXISTS gov_wf_transitions_immutable ON gov_workflow_transitions;
DROP TRIGGER IF EXISTS gov_wf_steps_immutable ON gov_workflow_steps;
DROP FUNCTION IF EXISTS gov_reject_published_workflow_change();
DROP TABLE IF EXISTS gov_routing_rules;
DROP TABLE IF EXISTS gov_workflow_transitions;
DROP TABLE IF EXISTS gov_workflow_steps;

ALTER TABLE gov_services DROP CONSTRAINT IF EXISTS gov_services_workflow_fk;
ALTER TABLE gov_services DROP CONSTRAINT IF EXISTS gov_services_owner_unit_fk;
ALTER TABLE gov_services DROP CONSTRAINT IF EXISTS gov_services_mode_valid;
ALTER TABLE gov_services DROP CONSTRAINT IF EXISTS gov_services_tenant_uniq;
ALTER TABLE gov_services DROP COLUMN IF EXISTS workflow_version_id;
ALTER TABLE gov_services DROP COLUMN IF EXISTS fulfillment_mode;
ALTER TABLE gov_services DROP COLUMN IF EXISTS owner_unit_id;
ALTER TABLE gov_services DROP COLUMN IF EXISTS updated_at;

ALTER TABLE gov_applications DROP CONSTRAINT IF EXISTS gov_applications_origin_unit_fk;
ALTER TABLE gov_applications DROP CONSTRAINT IF EXISTS gov_applications_current_unit_fk;
ALTER TABLE gov_applications DROP CONSTRAINT IF EXISTS gov_applications_mode_valid;
ALTER TABLE gov_applications DROP CONSTRAINT IF EXISTS gov_applications_tenant_uniq;
DROP INDEX IF EXISTS idx_gov_applications_external;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS workflow_version_id;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS fulfillment_mode;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS source_system;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS external_request_id;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS payload_fingerprint;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS request_payload;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS origin_unit_id;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS current_unit_id;
ALTER TABLE gov_applications DROP COLUMN IF EXISTS closed_at;

DROP INDEX IF EXISTS idx_gov_events_task;
ALTER TABLE gov_application_events DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE gov_application_events DROP COLUMN IF EXISTS task_id;
ALTER TABLE gov_application_events DROP COLUMN IF EXISTS unit_id;
ALTER TABLE gov_application_events DROP COLUMN IF EXISTS from_status;
ALTER TABLE gov_application_events DROP COLUMN IF EXISTS to_status;

DROP TABLE IF EXISTS gov_workflow_versions;
DROP TABLE IF EXISTS gov_workflows;
DROP TABLE IF EXISTS gov_unit_members;
DROP TABLE IF EXISTS gov_org_units;
