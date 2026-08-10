-- POS staff switching, shifts, mobile push registration and fleet telemetry.
-- +goose Up

CREATE TABLE staff_pin_credentials (
    membership_id UUID PRIMARY KEY REFERENCES memberships(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pin_hash TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE pos_shifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices(id),
    membership_id UUID NOT NULL REFERENCES memberships(id),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    opening_float NUMERIC(18,2) NOT NULL DEFAULT 0,
    closing_total NUMERIC(18,2),
    notes TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX pos_one_open_shift_per_device ON pos_shifts(device_id) WHERE closed_at IS NULL;
CREATE TABLE push_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('APNS','FCM')),
    token_hash TEXT NOT NULL UNIQUE,
    token_ciphertext TEXT NOT NULL,
    app_id TEXT NOT NULL DEFAULT 'mn.gerege.nexus',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (user_id IS NOT NULL OR device_id IS NOT NULL)
);
CREATE TABLE device_telemetry (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    level TEXT NOT NULL CHECK (level IN ('INFO','WARN','ERROR')),
    event TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX device_telemetry_fleet ON device_telemetry(tenant_id,device_id,received_at DESC);

GRANT SELECT,INSERT,UPDATE,DELETE ON staff_pin_credentials,pos_shifts,push_tokens,device_telemetry TO gerege_nexus_app;
GRANT USAGE,SELECT ON SEQUENCE device_telemetry_id_seq TO gerege_nexus_app;
ALTER TABLE staff_pin_credentials ENABLE ROW LEVEL SECURITY; ALTER TABLE staff_pin_credentials FORCE ROW LEVEL SECURITY;
ALTER TABLE pos_shifts ENABLE ROW LEVEL SECURITY; ALTER TABLE pos_shifts FORCE ROW LEVEL SECURITY;
ALTER TABLE push_tokens ENABLE ROW LEVEL SECURITY; ALTER TABLE push_tokens FORCE ROW LEVEL SECURITY;
ALTER TABLE device_telemetry ENABLE ROW LEVEL SECURITY; ALTER TABLE device_telemetry FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON staff_pin_credentials TO gerege_nexus_app USING (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid) WITH CHECK (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid);
CREATE POLICY tenant_isolation ON pos_shifts TO gerege_nexus_app USING (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid) WITH CHECK (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid);
CREATE POLICY tenant_isolation ON push_tokens TO gerege_nexus_app USING (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid) WITH CHECK (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid);
CREATE POLICY tenant_isolation ON device_telemetry TO gerege_nexus_app USING (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid) WITH CHECK (tenant_id=NULLIF(current_setting('app.current_tenant',true),'')::uuid);

-- +goose Down
DROP TABLE IF EXISTS device_telemetry;
DROP TABLE IF EXISTS push_tokens;
DROP TABLE IF EXISTS pos_shifts;
DROP TABLE IF EXISTS staff_pin_credentials;
