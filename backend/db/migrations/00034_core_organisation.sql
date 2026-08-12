-- The organisation and the people in it, as their own subject.
--
-- The platform has always had tenants, users and memberships, and they carried
-- the minimum needed to sign somebody in: a slug, a name, an email. Everything
-- an organisation actually is — its legal name, the registration number a
-- document has to carry, the department somebody works in, the timezone a
-- deadline is counted in — lived nowhere, or lived inside one app.
--
-- This follows Odoo's split, which is the right one and worth naming:
--
--   res.company  → tenants + tenant_profiles   what the organisation is
--   res.users    → users + user_preferences    who the person is, anywhere
--   employee     → memberships (+ job, dept)   who they are *here*
--   hr.department→ departments                 how the organisation is arranged
--
-- The middle distinction is the one that is easy to get wrong. A person's
-- language is theirs and follows them between organisations; their job title is
-- not — it belongs to the membership, and the same person can be a director in
-- one tenant and a clerk in another.

-- +goose Up

-- ─── What the organisation is ────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS tenant_profiles (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    -- The name on the seal, which is not always the name on the screen.
    legal_name VARCHAR(255) NOT NULL DEFAULT '',
    -- Улсын бүртгэлийн дугаар and татвар төлөгчийн дугаар. Kept as text: a
    -- registration number is an identifier, not a number — it has leading
    -- zeros, it has letters in some registries, and nothing ever adds one up.
    registration_number VARCHAR(64) NOT NULL DEFAULT '',
    tax_number VARCHAR(64) NOT NULL DEFAULT '',
    -- Мongolian addresses are administrative rather than street-first, so the
    -- parts are kept apart: a document that has to print аймаг/сум/баг cannot
    -- get them back out of one free-text line.
    country_code CHAR(2) NOT NULL DEFAULT 'MN',
    province VARCHAR(128) NOT NULL DEFAULT '',
    district VARCHAR(128) NOT NULL DEFAULT '',
    khoroo VARCHAR(128) NOT NULL DEFAULT '',
    address_line VARCHAR(255) NOT NULL DEFAULT '',
    postal_code VARCHAR(16) NOT NULL DEFAULT '',
    phone VARCHAR(64) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT '',
    website VARCHAR(255) NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    -- Defaults for everything the tenant does: a deadline is counted in this
    -- timezone, an invoice is written in this currency, and a notification is
    -- worded in this language unless the reader has said otherwise.
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Ulaanbaatar',
    locale VARCHAR(8) NOT NULL DEFAULT 'mn',
    currency CHAR(3) NOT NULL DEFAULT 'MNT',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Every tenant has one, from the moment it exists: a profile that has to be
-- created before it can be read is a null check in every caller.
INSERT INTO tenant_profiles (tenant_id, legal_name)
SELECT id, name FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;

-- And it stays true for tenants created later. A trigger rather than a line in
-- whoever creates the tenant: the invariant is what the readers rely on, and
-- there is no reason for it to depend on every future call site remembering.
-- SECURITY DEFINER because the insert would otherwise be judged by the RLS
-- policy below, which refuses a row for any tenant other than the bound one —
-- and creating a tenant is precisely the moment when none is bound.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION create_tenant_profile() RETURNS TRIGGER
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $fn$
BEGIN
    INSERT INTO tenant_profiles (tenant_id, legal_name)
    VALUES (NEW.id, NEW.name)
    ON CONFLICT (tenant_id) DO NOTHING;
    RETURN NEW;
END
$fn$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS tenants_create_profile ON tenants;
CREATE TRIGGER tenants_create_profile AFTER INSERT ON tenants
    FOR EACH ROW EXECUTE FUNCTION create_tenant_profile();

-- ─── How the organisation is arranged ────────────────────────────────────────

CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    parent_id UUID,
    -- Who answers for it. A membership rather than a user: the manager of a
    -- department is a person *in this organisation*, and a user who leaves it
    -- should stop being one.
    manager_membership_id UUID,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT departments_code_uniq UNIQUE (tenant_id, code),
    -- What every composite foreign key below points at, so a parent from
    -- another tenant is refused by the schema rather than by a handler that
    -- remembered to check.
    CONSTRAINT departments_tenant_uniq UNIQUE (id, tenant_id),
    CONSTRAINT departments_not_self CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT departments_parent_fk FOREIGN KEY (parent_id, tenant_id)
        REFERENCES departments (id, tenant_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_departments_tenant ON departments(tenant_id, active);

-- ─── Who somebody is here ────────────────────────────────────────────────────

ALTER TABLE memberships
    ADD COLUMN IF NOT EXISTS job_title VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS department_id UUID,
    -- Deactivated rather than deleted. A membership is referenced by everything
    -- the person did — a signature, an approval, a request they processed — and
    -- removing it would take that history with it or orphan it.
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ;

-- Same-tenant rule as above, enforced by the schema.
ALTER TABLE memberships
    ADD CONSTRAINT memberships_tenant_uniq UNIQUE (id, tenant_id);
ALTER TABLE memberships
    ADD CONSTRAINT memberships_department_fk FOREIGN KEY (department_id, tenant_id)
        REFERENCES departments (id, tenant_id) ON DELETE SET NULL;

ALTER TABLE departments
    ADD CONSTRAINT departments_manager_fk FOREIGN KEY (manager_membership_id, tenant_id)
        REFERENCES memberships (id, tenant_id) ON DELETE SET NULL;

-- ─── Who the person is, anywhere ─────────────────────────────────────────────

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone VARCHAR(64) NOT NULL DEFAULT '',
    -- Empty means "follow the organisation", which is what a new account
    -- should do: a preference nobody has expressed is not a preference.
    ADD COLUMN IF NOT EXISTS locale VARCHAR(8) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS timezone VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;

-- ─── Isolation ───────────────────────────────────────────────────────────────
--
-- 00029 put a policy on every table carrying a tenant_id, but it did it once,
-- over the tables that existed then. A table added afterwards is not covered by
-- it and has to say so itself — which is the whole point of the invariant test
-- in internal/platform/tenant: a new tenant-scoped table cannot ship without
-- this block, because the test fails until it is here.
--
-- tenant_profiles is keyed *by* tenant_id, so the policy reads oddly but means
-- exactly what it does elsewhere: under app.current_tenant, this tenant sees
-- and writes one row, its own.
ALTER TABLE tenant_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_profiles FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON tenant_profiles;
CREATE POLICY tenant_isolation ON tenant_profiles TO gerege_nexus_app
    USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
ALTER TABLE departments FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON departments;
CREATE POLICY tenant_isolation ON departments TO gerege_nexus_app
    USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS active,
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS phone;

ALTER TABLE departments DROP CONSTRAINT IF EXISTS departments_manager_fk;
ALTER TABLE memberships DROP CONSTRAINT IF EXISTS memberships_department_fk;
ALTER TABLE memberships DROP CONSTRAINT IF EXISTS memberships_tenant_uniq;
ALTER TABLE memberships
    DROP COLUMN IF EXISTS deactivated_at,
    DROP COLUMN IF EXISTS active,
    DROP COLUMN IF EXISTS department_id,
    DROP COLUMN IF EXISTS job_title;

DROP TRIGGER IF EXISTS tenants_create_profile ON tenants;
DROP FUNCTION IF EXISTS create_tenant_profile();
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS tenant_profiles;
