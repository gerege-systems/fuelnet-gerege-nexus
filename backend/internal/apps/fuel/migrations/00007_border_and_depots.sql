-- Хил ба нефть бааз — гинжний дээд хоёр шат.
--
-- Х1-д парти ба ШТС-ын буулгалт орсон: «энэ литр тэр партиас». Энэ миграц
-- партийг хаанаас ирснийг нь хэлдэг болгоно:
--
--     хил → бааз → цистерн → ШТС → ваучер
--     ↑      ↑
--     энэ миграц
--
-- # Тонн ба литр — хоёрын аль нь үнэн вэ
--
-- Гаалийн мэдүүлэг тонноор бичигддэг, сав литрээр хэмжигддэг, хоёрын хооронд
-- нягт байх бөгөөд нягт нь температураас хамаарч хөвдөг. benzin-gerege-mn
-- хоёуланг нь хольж хадгалдаг байсан — тэр нь алдааны эх үүсвэр.
--
-- Энд **литр нь үнэн**. `declared_tons` нь цаасан дээр юу бичигдсэнийг
-- хадгална, тулгагдахгүй. Х1-ийн дагалдах бичиг ба хэмжсэн литрийн ялгаа
-- ижил зарчмаар: систем зөрүүг тайлбарлахгүй, зөрүү байгааг хэлнэ.
--
-- # Сав яагаад тусдаа хүснэгт вэ
--
-- Нэг баазад олон сав, сав бүр өөр түлш, өөр багтаамж, өөр температуртай.
-- Баазын нийт үлдэгдэл бол савнуудын нийлбэр — тусад нь хадгалбал хоёр үнэн
-- үүсэж, аль нь зөв болохыг дараагийн уншигч таамаглана.

-- +goose Up

-- ─────────────────────────────────────────────────────────────────────────
-- Хил
-- ─────────────────────────────────────────────────────────────────────────

CREATE TABLE fuel_customs_shipments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    -- Гаалийн мэдүүлгийн дугаар. Улсын баримт, тиймээс байгууллага дотор
    -- давхардахгүй байх нь хамгийн бага шаардлага.
    declaration_no    TEXT NOT NULL,

    border_port       TEXT NOT NULL DEFAULT '',
    origin_country    TEXT NOT NULL DEFAULT '',
    exporter          TEXT NOT NULL DEFAULT '',

    fuel_type         TEXT NOT NULL,
    fuel_label        TEXT NOT NULL DEFAULT '',

    -- Литр нь үнэн; тонн нь мэдүүлэгт бичигдсэн зүйл.
    declared_liters   NUMERIC(14, 3) NOT NULL DEFAULT 0,
    declared_tons     NUMERIC(12, 3),

    -- Төмөр замаар ирдэг тул цистерн вагоны тоо, цувааны дугаар.
    wagons            INT NOT NULL DEFAULT 0,
    convoy_code       TEXT NOT NULL DEFAULT '',

    -- border_arrived · inspecting · cleared · in_transit · at_depot
    status            TEXT NOT NULL DEFAULT 'border_arrived',

    -- Лабораторийн шинжилгээ. Партид хуулагдаж, хошуу хүртэл дагана.
    lab_status        TEXT NOT NULL DEFAULT 'pending',
    quality_cert_no   TEXT NOT NULL DEFAULT '',
    octane_tested     NUMERIC(5, 2),
    sulfur_ppm        NUMERIC(8, 2),

    entered_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cleared_at        TIMESTAMPTZ,
    expected_at       TIMESTAMPTZ,
    note              TEXT NOT NULL DEFAULT '',

    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, declaration_no)
);

CREATE INDEX idx_customs_tenant ON fuel_customs_shipments (tenant_id);
CREATE INDEX idx_customs_status ON fuel_customs_shipments (status)
    WHERE status <> 'at_depot';

-- Парти нь мэдүүлгээс төрнө. NULL нь энэ системд бүртгэгдээгүй мэдүүлэг —
-- одоо байгаа демо өгөгдөл тийм, мөн шилжилтийн үед бодит парти ч тийм байна.
ALTER TABLE fuel_batches
    ADD COLUMN customs_shipment_id UUID REFERENCES fuel_customs_shipments (id) ON DELETE SET NULL;

CREATE INDEX idx_batches_shipment ON fuel_batches (customs_shipment_id);

-- ─────────────────────────────────────────────────────────────────────────
-- Бааз
-- ─────────────────────────────────────────────────────────────────────────

CREATE TABLE fuel_depots (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    name               TEXT NOT NULL,
    brand              TEXT NOT NULL DEFAULT '',
    aimag              TEXT NOT NULL DEFAULT '',
    district           TEXT NOT NULL DEFAULT '',
    address            TEXT NOT NULL DEFAULT '',
    lat                DOUBLE PRECISION,
    lon                DOUBLE PRECISION,

    -- Төмөр замын татах зам байгаа эсэх нь ямар баазад вагон ирж чадахыг
    -- шийднэ — хилээс ирэх урсгалын гол хязгаарлалт.
    has_rail_siding    BOOLEAN NOT NULL DEFAULT FALSE,
    rail_station_code  TEXT NOT NULL DEFAULT '',

    -- operational · high_load · maintenance
    status             TEXT NOT NULL DEFAULT 'operational',

    source             TEXT,
    source_ref         TEXT,

    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_depots_tenant ON fuel_depots (tenant_id);
CREATE UNIQUE INDEX idx_depots_source
    ON fuel_depots (tenant_id, source, source_ref)
    WHERE source IS NOT NULL AND source_ref IS NOT NULL;

CREATE TABLE fuel_depot_tanks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    depot_id            UUID NOT NULL REFERENCES fuel_depots (id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    tank_no             TEXT NOT NULL,
    -- vertical_steel · horizontal_steel · underground
    tank_type           TEXT NOT NULL DEFAULT 'vertical_steel',
    fuel_type           TEXT NOT NULL,
    fuel_label          TEXT NOT NULL DEFAULT '',

    capacity_liters     NUMERIC(14, 3) NOT NULL CHECK (capacity_liters > 0),
    current_liters      NUMERIC(14, 3) NOT NULL DEFAULT 0,

    -- Хэмжил зүй. Нягт нь температураас хамаардаг тул хоёулаа хэрэгтэй:
    -- тонн ↔ литр хөрвүүлэх бол эдгээрээр л зөв болно.
    temperature_c       NUMERIC(5, 2),
    density_kg_m3       NUMERIC(7, 2),

    -- optimal · warning_temp · high_fill · low_level · maintenance
    safety_status       TEXT NOT NULL DEFAULT 'optimal',
    last_inspection_at  DATE,
    next_inspection_at  DATE,

    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Сав хэзээ ч багтаамжаасаа хэтрэхгүй, сөрөг ч болохгүй. Гол дүрэм
    -- өгөгдлийн санд бичигдсэн байх ёстой — код дээр л үлдээвэл дараагийн
    -- бичигдэх зам түүнийг давтахаа мартах эрхтэй болно.
    CONSTRAINT tank_within_capacity
        CHECK (current_liters >= 0 AND current_liters <= capacity_liters),
    UNIQUE (depot_id, tank_no)
);

CREATE INDEX idx_tanks_depot ON fuel_depot_tanks (depot_id, fuel_type);

-- Хилээс бааз руу орсон ачаа. Х1-ийн fuel_station_receipts-тэй ижил үүрэг:
-- баазын савны үлдэгдэл нэмэгддэг цорын ганц газар.
CREATE TABLE fuel_depot_receipts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    depot_id         UUID NOT NULL REFERENCES fuel_depots (id) ON DELETE CASCADE,
    tank_id          UUID NOT NULL REFERENCES fuel_depot_tanks (id) ON DELETE CASCADE,
    shipment_id      UUID REFERENCES fuel_customs_shipments (id) ON DELETE SET NULL,
    batch_id         UUID REFERENCES fuel_batches (id) ON DELETE SET NULL,

    liters           NUMERIC(14, 3) NOT NULL CHECK (liters > 0),
    manifest_liters  NUMERIC(14, 3),

    received_by      UUID,
    received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note             TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_depot_receipts_depot ON fuel_depot_receipts (depot_id, received_at DESC);
CREATE INDEX idx_depot_receipts_batch ON fuel_depot_receipts (batch_id);
-- Нэг мэдүүлэг нэг л удаа хүлээж авагдана — Х1-ийн рейстэй ижил шалтгаан.
CREATE UNIQUE INDEX idx_depot_receipts_one_per_shipment
    ON fuel_depot_receipts (shipment_id) WHERE shipment_id IS NOT NULL;

-- Цистерн аль баазаас, аль савнаас ачигдсан бэ.
--
-- `from_depot` гэсэн чөлөөт текст багана хэвээр үлдэнэ: тэр нь хүн уншихад
-- зориулсан нэр бөгөөд бүртгэгдээгүй баазаас гарсан рейст ганцхан хариулт.
ALTER TABLE fuel_dispatch_trips
    ADD COLUMN from_depot_id UUID REFERENCES fuel_depots (id) ON DELETE SET NULL,
    ADD COLUMN from_tank_id  UUID REFERENCES fuel_depot_tanks (id) ON DELETE SET NULL;

CREATE INDEX idx_trips_depot ON fuel_dispatch_trips (from_depot_id);

-- ─────────────────────────────────────────────────────────────────────────

ALTER TABLE fuel_customs_shipments ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_customs_shipments FORCE ROW LEVEL SECURITY;
ALTER TABLE fuel_depots ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_depots FORCE ROW LEVEL SECURITY;
ALTER TABLE fuel_depot_tanks ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_depot_tanks FORCE ROW LEVEL SECURITY;
ALTER TABLE fuel_depot_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_depot_receipts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON fuel_customs_shipments TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolation ON fuel_depots TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolation ON fuel_depot_tanks TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolation ON fuel_depot_receipts TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE ON fuel_customs_shipments TO gerege_nexus_tenant;
GRANT SELECT, INSERT, UPDATE ON fuel_depots TO gerege_nexus_tenant;
GRANT SELECT, INSERT, UPDATE ON fuel_depot_tanks TO gerege_nexus_tenant;
GRANT SELECT, INSERT ON fuel_depot_receipts TO gerege_nexus_tenant;

-- +goose Down

ALTER TABLE fuel_dispatch_trips
    DROP COLUMN IF EXISTS from_tank_id,
    DROP COLUMN IF EXISTS from_depot_id;
DROP TABLE IF EXISTS fuel_depot_receipts;
DROP TABLE IF EXISTS fuel_depot_tanks;
DROP TABLE IF EXISTS fuel_depots;
ALTER TABLE fuel_batches DROP COLUMN IF EXISTS customs_shipment_id;
DROP TABLE IF EXISTS fuel_customs_shipments;
