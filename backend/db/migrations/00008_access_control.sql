-- +goose Up
-- Odoo-inspired tenant access control: additive groups (roles), model access
-- rights (permissions), membership assignments, and durable change history.
ALTER TABLE roles ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS access_change_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(96) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id UUID,
    before_state JSONB,
    after_state JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_access_events_tenant_created
    ON access_change_events(tenant_id, created_at DESC);

-- Prevent cross-tenant role assignment at the database boundary. The lookup
-- path already filters by tenant, but invalid links must not be storable.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_membership_role_tenant() RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM memberships m
        JOIN roles r ON r.id = NEW.role_id
        WHERE m.id = NEW.membership_id AND m.tenant_id = r.tenant_id
    ) THEN
        RAISE EXCEPTION 'membership and role must belong to the same tenant';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS membership_role_tenant_guard ON membership_roles;
CREATE TRIGGER membership_role_tenant_guard
    BEFORE INSERT OR UPDATE ON membership_roles
    FOR EACH ROW EXECUTE FUNCTION enforce_membership_role_tenant();

INSERT INTO roles (tenant_id, code, name, description, is_system)
SELECT t.id, seed.code, seed.name, seed.description, TRUE
FROM tenants t
CROSS JOIN (VALUES
    ('admin', 'Administrator', 'Full tenant administration and every installed application permission'),
    ('manager', 'Manager', 'Operational read and manage access across installed business applications'),
    ('user', 'User', 'Standard read access and self-service actions')
) AS seed(code, name, description)
ON CONFLICT (tenant_id, code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_system = TRUE,
    active = TRUE;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'admin'
ON CONFLICT DO NOTHING;

-- Provision the same baseline for tenants created after this migration.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION seed_tenant_access_roles() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO roles (tenant_id, code, name, description, is_system)
    VALUES
        (NEW.id, 'admin', 'Administrator', 'Full tenant administration and every installed application permission', TRUE),
        (NEW.id, 'manager', 'Manager', 'Operational read and manage access across installed business applications', TRUE),
        (NEW.id, 'user', 'User', 'Standard read access and self-service actions', TRUE)
    ON CONFLICT (tenant_id, code) DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = NEW.id AND (
        r.code = 'admin'
        OR (r.code = 'manager' AND (p.code LIKE '%.read' OR p.code LIKE '%.manage' OR p.code IN ('gov.process','gov.delegate','gov.verify','gov.report')))
        OR (r.code = 'user' AND (p.code LIKE '%.read' OR p.code = 'gov.apply'))
    ) ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS tenant_access_role_seed ON tenants;
CREATE TRIGGER tenant_access_role_seed
    AFTER INSERT ON tenants
    FOR EACH ROW EXECUTE FUNCTION seed_tenant_access_roles();

-- Existing members without an explicit group become standard users. This
-- keeps the migration deny-by-default without making normal accounts unusable.
INSERT INTO membership_roles (membership_id, role_id)
SELECT m.id, r.id
FROM memberships m
JOIN roles r ON r.tenant_id = m.tenant_id AND r.code = 'user'
WHERE NOT EXISTS (SELECT 1 FROM membership_roles mr WHERE mr.membership_id = m.id)
ON CONFLICT DO NOTHING;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION assign_default_membership_role() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO membership_roles (membership_id, role_id)
    SELECT NEW.id, r.id FROM roles r
    WHERE r.tenant_id = NEW.tenant_id AND r.code = 'user' AND r.active
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS membership_default_role ON memberships;
CREATE TRIGGER membership_default_role
    AFTER INSERT ON memberships
    FOR EACH ROW EXECUTE FUNCTION assign_default_membership_role();

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code = 'manager'
  AND (p.code LIKE '%.read' OR p.code LIKE '%.manage' OR p.code IN ('gov.process','gov.delegate','gov.verify','gov.report'))
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.code = 'user' AND (p.code LIKE '%.read' OR p.code = 'gov.apply')
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TRIGGER IF EXISTS tenant_access_role_seed ON tenants;
DROP FUNCTION IF EXISTS seed_tenant_access_roles();
DROP TRIGGER IF EXISTS membership_default_role ON memberships;
DROP FUNCTION IF EXISTS assign_default_membership_role();
DROP TRIGGER IF EXISTS membership_role_tenant_guard ON membership_roles;
DROP FUNCTION IF EXISTS enforce_membership_role_tenant();
DROP TABLE IF EXISTS access_change_events;
ALTER TABLE roles DROP COLUMN IF EXISTS is_system;
ALTER TABLE roles DROP COLUMN IF EXISTS active;
ALTER TABLE roles DROP COLUMN IF EXISTS description;
