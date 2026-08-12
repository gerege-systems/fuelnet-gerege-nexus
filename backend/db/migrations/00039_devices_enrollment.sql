-- Дугаар нь 00032/00033-аас 00039/00040 болж шилжсэн: платформын мод дээр
-- тэдгээр дугаарыг өөр migration аль хэдийн эзэлсэн байсан бөгөөд goose
-- давхардсан хувилбар олоод panic хийдэг.
--
-- Тиймээс бүх зүйл дахин ажиллахад тэсвэртэй: DS-ийн мэдээллийн санд эдгээр
-- объект хуучин дугаарын дор аль хэдийн үүссэн байгаа тул шинэ дугаараар
-- дахин ажиллахад юу ч хийхгүй өнгөрөх ёстой. Шинэ сан дээр урьдын адил үүснэ.
--
-- CREATE POLICY нь IF NOT EXISTS-ийг PostgreSQL дээр дэмждэггүй тул бодлого
-- бүрийн өмнө DROP POLICY IF EXISTS тавьсан.
-- Native kiosk/POS devices enroll once, then authenticate with an opaque
-- device token. Plain enrollment codes and device tokens are never persisted.
-- +goose Up

CREATE TABLE IF NOT EXISTS device_enrollment_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    platform TEXT NOT NULL CHECK (platform IN ('windows','android','macos','ios')),
    form_factor TEXT NOT NULL CHECK (form_factor IN ('desktop','mobile','tablet','kiosk','pos')),
    site TEXT NOT NULL DEFAULT '',
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','DISABLED','RETIRED')),
    app_version TEXT NOT NULL DEFAULT '',
    os_version TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_tenant ON devices(tenant_id, status, name);
CREATE INDEX IF NOT EXISTS idx_device_enrollment_live ON device_enrollment_codes(code_hash, expires_at) WHERE used_at IS NULL;

GRANT SELECT, INSERT, UPDATE, DELETE ON devices, device_enrollment_codes TO gerege_nexus_app;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON devices;
CREATE POLICY tenant_isolation ON devices TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
ALTER TABLE device_enrollment_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_enrollment_codes FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON device_enrollment_codes;
CREATE POLICY tenant_isolation ON device_enrollment_codes TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- Enrollment and device authentication happen before a tenant is known, so a
-- normal RLS query can never bootstrap itself. These narrowly scoped security
-- definer functions reveal only the tenant/device matching an opaque digest.
CREATE OR REPLACE FUNCTION resolve_device_enrollment(p_code_hash TEXT)
RETURNS TABLE(id UUID, tenant_id UUID)
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
    SELECT e.id, e.tenant_id FROM device_enrollment_codes e
    WHERE e.code_hash=p_code_hash AND e.used_at IS NULL AND e.expires_at>NOW()
    FOR UPDATE
$$;
CREATE OR REPLACE FUNCTION authenticate_device(p_token_hash TEXT)
RETURNS TABLE(id UUID, tenant_id UUID, name TEXT, platform TEXT, form_factor TEXT)
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
    UPDATE devices d SET last_seen_at=NOW(),updated_at=NOW()
    WHERE d.token_hash=p_token_hash AND d.status='ACTIVE'
    RETURNING d.id,d.tenant_id,d.name,d.platform,d.form_factor
$$;
REVOKE ALL ON FUNCTION resolve_device_enrollment(TEXT), authenticate_device(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION resolve_device_enrollment(TEXT), authenticate_device(TEXT) TO gerege_nexus_app;

-- +goose Down
DROP FUNCTION IF EXISTS authenticate_device(TEXT);
DROP FUNCTION IF EXISTS resolve_device_enrollment(TEXT);
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS device_enrollment_codes;
