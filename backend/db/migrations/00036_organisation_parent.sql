-- An organisation that has organisations under it.
--
-- The word covers two different things, and the platform answers them
-- differently on purpose:
--
--   a branch, an office, a unit — one legal entity, arranged internally.
--     That is a department. Departments are already a tree with a manager and
--     people, and "Улаанбаатар салбар" is a node in it. Nothing new is needed
--     and nothing here applies.
--
--   a subsidiary — its own legal entity, its own registration number, its own
--     invoices, its own seal. That is its own tenant, and it has to be: every
--     boundary this platform has — the row-level policies, the permissions,
--     the documents, the audit trail — is drawn per tenant, and a subsidiary
--     whose data silently merged with its parent's would be a reporting error
--     that no screen could undo.
--
-- What was missing is only the relationship. Two tenants could be parent and
-- subsidiary in the world and nothing recorded it, so a document could not
-- print "a subsidiary of X" and no future consolidated view had anywhere to
-- start.
--
-- This changes no isolation whatsoever. A parent tenant gains no access to a
-- subsidiary's rows: the policies from 00029 are unchanged, and a person who
-- works in both organisations still reaches each one through their own
-- membership, as they do today. This column is a statement about the world,
-- not a grant.

-- +goose Up

ALTER TABLE tenant_profiles
    ADD COLUMN IF NOT EXISTS parent_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;

-- The one cycle the schema can see on its own. Longer ones — A under B under A
-- — are refused by the handler, which can walk the chain; a CHECK cannot.
ALTER TABLE tenant_profiles
    ADD CONSTRAINT tenant_profiles_not_own_parent CHECK (parent_tenant_id IS NULL OR parent_tenant_id <> tenant_id);

CREATE INDEX IF NOT EXISTS idx_tenant_profiles_parent ON tenant_profiles(parent_tenant_id)
    WHERE parent_tenant_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_tenant_profiles_parent;
ALTER TABLE tenant_profiles DROP CONSTRAINT IF EXISTS tenant_profiles_not_own_parent;
ALTER TABLE tenant_profiles DROP COLUMN IF EXISTS parent_tenant_id;
