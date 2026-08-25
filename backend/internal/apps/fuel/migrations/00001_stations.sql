-- ШТС-ын бүртгэл ба түлшний нөөц.
--
-- Хүснэгтүүд угтваргүй үүсэж, `search_path`-ийн эхний `tenant` тэднийг зөв
-- plane-д буулгана — миграц 00080-ийн модульд зориулсан гэрээ. Эзэмшлийн
-- дүрэм (docs/TWO_PLANES_PROPOSAL.md §2.1): мөр нь тенант бүрд өөр өөрөө
-- оршдог тул тенантынх.
--
-- ЭНЭ БОЛ ХУУЛБАР БИШ. benzin-gerege-mn/backend/pkg/benzin/store.go-гийн
-- AutoMigrate-аас гарал үүсэлтэй ч дөрвөн зүйл зориуд өөр:
--
--   1. `tenant_id` — компани бүр өөрийн ШТС-аа эзэмшинэ. Эх хувилбар нь нэг
--      эзэнтэй байсан тул ийм багана хэрэггүй байв.
--   2. Мөнгө ба литр NUMERIC, DOUBLE PRECISION биш. Иргэний өдрийн эрх
--      төгрөгөөр тоологдож, нөөц нэмэгдэж хасагдана — хөвөгч таслалын алдаа
--      хуримтлагддаг газар мөнгө байх ёсгүй.
--   3. `id` нь UUID, эх сурвалжийн мөр биш. Импортын идемпотент байдлыг
--      (tenant_id, source, source_ref) хангана. Эх хувилбар Google-ийн
--      place_id-г шууд PK болгосон нь тухайн нэг эх сурвалжид уяж орхисон.
--   4. `rating` / `rating_count` аваагүй. Тэдгээр нь Google Places-ийн агуулга
--      бөгөөд Google бус зураг дээр харуулахыг ToS хориглодог. Зургийн шийдвэр
--      гарахаас өмнө татаж авахгүй нь дээр — багана нэмэх нь хасахаас хялбар.
--
-- `current_queue_count` нь энд байгаа ч БОДИТ ЭХ СУРВАЛЖГҮЙ. Ү5 хүртэл 0
-- хэвээр. Тэр багана одоо байгаа нь дараалал харуулах гэсэн амлалт биш,
-- харин тоо ирэхэд хаана суухыг тэмдэглэсэн хэрэг.

-- +goose Up

CREATE TABLE fuel_stations (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    name                     TEXT NOT NULL,
    brand                    TEXT NOT NULL,
    brand_label              TEXT NOT NULL DEFAULT '',

    lat                      DOUBLE PRECISION NOT NULL,
    lon                      DOUBLE PRECISION NOT NULL,
    aimag                    TEXT NOT NULL DEFAULT '',
    district                 TEXT NOT NULL DEFAULT '',
    address                  TEXT NOT NULL DEFAULT '',
    phone                    TEXT NOT NULL DEFAULT '',
    opening_hours            TEXT NOT NULL DEFAULT '24/7',

    total_pumps              INT NOT NULL DEFAULT 0,
    active_pumps             INT NOT NULL DEFAULT 0,
    avg_service_time_sec     INT NOT NULL DEFAULT 180,
    current_queue_count      INT NOT NULL DEFAULT 0,

    status                   TEXT NOT NULL DEFAULT 'available',
    is_voucher_enabled       BOOLEAN NOT NULL DEFAULT TRUE,

    -- Хаанаас ирсэн, тэнд юу гэж нэрлэгддэг вэ. Дахин импортлоход мөр
    -- давхардуулахгүйн тулд; NULL нь гараар үүсгэсэн гэсэн үг.
    source                   TEXT,
    source_ref               TEXT,

    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Импорт давтагдахад шинэчилнэ, хоёр дахь мөр үүсгэхгүй. Тенантаар
-- хязгаарласан нь санаатай: хоёр компани ижил эх сурвалжаас өөрийн салбарыг
-- татаж болно.
CREATE UNIQUE INDEX idx_fuel_stations_source
    ON fuel_stations (tenant_id, source, source_ref)
    WHERE source IS NOT NULL AND source_ref IS NOT NULL;

CREATE INDEX idx_fuel_stations_tenant ON fuel_stations (tenant_id);
CREATE INDEX idx_fuel_stations_coords ON fuel_stations (lat, lon);
CREATE INDEX idx_fuel_stations_brand ON fuel_stations (brand);

CREATE TABLE fuel_station_inventory (
    station_id               UUID NOT NULL REFERENCES fuel_stations (id) ON DELETE CASCADE,
    tenant_id                UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    fuel_type                TEXT NOT NULL,
    fuel_label               TEXT NOT NULL DEFAULT '',

    -- Төгрөг. Иргэний өдрийн эрхээс хасагдах тоо энэ үнээр үржигдэнэ, тиймээс
    -- энэ баганын нарийвчлал нь мөнгөнийх.
    price_mnt                NUMERIC(10, 2) NOT NULL,

    tank_capacity_liters     NUMERIC(12, 3) NOT NULL DEFAULT 0,
    current_stock_liters     NUMERIC(12, 3) NOT NULL DEFAULT 0,

    status                   TEXT NOT NULL DEFAULT 'available',
    last_reported_at         TIMESTAMPTZ,

    PRIMARY KEY (station_id, fuel_type)
);

CREATE INDEX idx_fuel_inventory_tenant ON fuel_station_inventory (tenant_id);
CREATE INDEX idx_fuel_inventory_type ON fuel_station_inventory (fuel_type, status);

-- Мөрийн түвшний тусгаарлалт — 00037/00073-ийн ӨРГӨН хэлбэр.
--
-- Уншихдаа session-ий зөвшөөрөгдсөн БҮХ байгууллага, бичихдээ зөвхөн идэвхтэй
-- нэг нь. Нарийн хэлбэр (`= app.current_tenant`) нь хаагдах тал руугаа алдана —
-- мэдээлэл задрахгүй ч олон компанид харьяалагдах хүн харах ёстой мөрөө
-- хардаггүй болно, тэр нь хоосон дэлгэц шиг харагддаг алдаа.
--
-- FORCE нь хүснэгтийн эзэнд ч бодлогыг үйлчлүүлнэ.
--
-- Иргэний нийтийн зам эдгээр бодлогод ОГТ хамаарахгүй: тэнд tenant байхгүй
-- тул dbguard `SET ROLE NONE` хийж, login role-оор бүгдийг харна. Тэр замын
-- шүүлт нь гараар бичигдэнэ — Ү2б.

ALTER TABLE fuel_stations ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_stations FORCE ROW LEVEL SECURITY;
ALTER TABLE fuel_station_inventory ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_station_inventory FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON fuel_stations TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolation ON fuel_station_inventory TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON fuel_stations TO gerege_nexus_tenant;
GRANT SELECT, INSERT, UPDATE, DELETE ON fuel_station_inventory TO gerege_nexus_tenant;

-- +goose Down

DROP TABLE IF EXISTS fuel_station_inventory;
DROP TABLE IF EXISTS fuel_stations;
