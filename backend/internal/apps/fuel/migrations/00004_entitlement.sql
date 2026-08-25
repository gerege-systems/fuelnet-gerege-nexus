-- Иргэний өдрийн эрх ба түүнээс гарах ваучер.
--
-- # Литр биш, ТӨГРӨГ
--
-- Хуваарилалт нь өдөрт 50,000₮. Литрээр биш байгаа шалтгаан: түлшний үнэ
-- төрлөөрөө өөр, цаг хугацаанд өөрчлөгддөг. «20 литр» гэдэг нь АИ-92-т нэг
-- утга, АИ-95-д өөр утгатай; «50,000₮» нь хоёуланд ижил утгатай.
--
-- # Машины дугаар биш, ИРГЭН
--
-- Хязгаар нь `citizen_id`-д тавигдана — платформын хэрэглэгчийн дугаар, eID-ээр
-- нэвтрэхэд үүсдэг. Гурван машинтай хүн гурав дахин авахгүй.
--
-- # ШТС-д уяагүй
--
-- Ваучер нь ДУРЫН колонк дээр хүчинтэй. Тиймээс энэ нь тухайн ШТС дээр литр
-- захиалдаггүй — очихгүй газарт нөөц түгжих нь бодит тоог гуйвуулна.
-- `intended_station_id` нь зөвхөн ЗОРИЛГЫН дохио: хаашаа явах гэж байгааг нь
-- асуусны хариу, дараалал урьдчилан таамаглахад ашиглана, үүрэг хүлээлгэхгүй.
--
-- # RLS юуг хийж, юуг ХИЙХГҮЙ вэ
--
-- `tenant_id` дээрх бодлого нь өөр байгууллагыг гаднаас нь хаана. Гэхдээ
-- иргэд БҮГД нэг тенантад (EID_JIT_TENANT_SLUG) багтдаг тул энэ бодлого
-- ИРГЭДИЙГ ХООРОНД НЬ ТУСГААРЛАХГҮЙ. Тэрхүү тусгаарлалт нь handler дахь
-- `citizen_id = <session-ээс>` шалгуур дээр л тогтоно.
--
-- Үүнийг тод бичсэн шалтгаан: RLS байгаа нь бүхнийг хамгаалсан мэт уншигдана.
-- Энд хамгаалахгүй. Эдгээр хүснэгт рүү шинэ query бичих хүн session-ий
-- эзнээр өөрөө шүүх ёстой.

-- +goose Up

CREATE TABLE fuel_entitlements (
    citizen_id   UUID NOT NULL,
    for_date     DATE NOT NULL,
    tenant_id    UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    granted_mnt  NUMERIC(12, 2) NOT NULL,
    used_mnt     NUMERIC(12, 2) NOT NULL DEFAULT 0,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (citizen_id, for_date),
    -- Хэрэглэсэн нь олгосноос хэтрэхгүй. Гол дүрэм өгөгдлийн санд бичигдсэн
    -- байх ёстой: үүнийг зөвхөн кодод үлдээвэл дараагийн бичигдэх зам түүнийг
    -- давтахаа мартах эрхтэй болно.
    CONSTRAINT entitlement_not_overdrawn CHECK (used_mnt >= 0 AND used_mnt <= granted_mnt)
);

CREATE INDEX idx_entitlements_date ON fuel_entitlements (for_date);

CREATE TABLE fuel_vouchers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    citizen_id           UUID NOT NULL,
    tenant_id            UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,
    for_date             DATE NOT NULL,

    amount_mnt           NUMERIC(12, 2) NOT NULL CHECK (amount_mnt > 0),
    fuel_type            TEXT NOT NULL,
    fuel_label           TEXT NOT NULL DEFAULT '',

    -- Хаашаа явах бодолтой байсан. Үүрэг биш — §дээрх.
    intended_station_id  UUID REFERENCES fuel_stations (id) ON DELETE SET NULL,

    -- 256 бит. Өмнөх системд ваучерын код 6 hex тэмдэгт байсан тул таамаглаж
    -- эргүүлэх боломжтой байв; энэ нь тааварлахад хэтэрхий том.
    qr_token             TEXT NOT NULL UNIQUE,

    status               TEXT NOT NULL DEFAULT 'active',
    expires_at           TIMESTAMPTZ NOT NULL,

    redeemed_at          TIMESTAMPTZ,
    redeemed_station_id  UUID REFERENCES fuel_stations (id) ON DELETE SET NULL,
    redeemed_liters      NUMERIC(10, 3),

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vouchers_citizen ON fuel_vouchers (citizen_id, created_at DESC);
CREATE INDEX idx_vouchers_active ON fuel_vouchers (status) WHERE status = 'active';

ALTER TABLE fuel_entitlements ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_entitlements FORCE ROW LEVEL SECURITY;
ALTER TABLE fuel_vouchers ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_vouchers FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON fuel_entitlements TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolation ON fuel_vouchers TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE ON fuel_entitlements TO gerege_nexus_tenant;
GRANT SELECT, INSERT, UPDATE ON fuel_vouchers TO gerege_nexus_tenant;

-- +goose Down

DROP TABLE IF EXISTS fuel_vouchers;
DROP TABLE IF EXISTS fuel_entitlements;
