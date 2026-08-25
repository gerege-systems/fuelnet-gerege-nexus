-- Автоцистернээр ШТС рүү явж буй ачилт.
--
-- Нэг мөр = нэг рейс: аль баазаас, аль ШТС рүү, ямар түлш, хэдэн литр,
-- лацны дугаар, хаана явж байгаа.
--
-- # Байрлалыг хоёр эх сурвалжаас авна
--
-- `current_lat`/`current_lon` нь төхөөрөмжийн СҮҮЛД мэдээлсэн байрлал. NULL
-- байвал систем хуваариас нь тооцоолно: `origin` → `to_station` хоорондын
-- шулуун дээр `departed_at` → `eta_at` хугацааны хувиар. Тиймээс GPS-гүй
-- машин ч зураг дээр гарна, зөвхөн бодит биш гэдгээ `position_source`-оор
-- хэлнэ.
--
-- Ингэж хийсэн шалтгаан: тооцоолсон байрлалыг тусад нь хүснэгтэд бичвэл хоёр
-- үнэн үүсээд, аль нь зөв болохыг дараагийн уншигч таамаглана. Уншихдаа
-- гаргаснаар үнэн нэг л газар — сүүлийн телеметр эсвэл хуваарь — үлдэнэ.

-- +goose Up

CREATE TABLE fuel_dispatch_trips (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    trip_code            TEXT NOT NULL,
    tanker_plate         TEXT NOT NULL,

    -- Хүн. Нийтийн замаар ХЭЗЭЭ Ч гарахгүй — жолоочийн нэр, утас нь тухайн
    -- хүний хувийн мэдээлэл бөгөөд «шатахуун хэзээ ирэх вэ» гэсэн асуултад
    -- хариулахад шаардлагагүй.
    driver_name          TEXT NOT NULL DEFAULT '',
    driver_phone         TEXT NOT NULL DEFAULT '',

    from_depot           TEXT NOT NULL DEFAULT '',
    origin_lat           DOUBLE PRECISION,
    origin_lon           DOUBLE PRECISION,

    to_station_id        UUID REFERENCES fuel_stations (id) ON DELETE SET NULL,

    fuel_type            TEXT NOT NULL,
    fuel_label           TEXT NOT NULL DEFAULT '',
    volume_liters        NUMERIC(12, 3) NOT NULL DEFAULT 0,

    -- Цахим лац. Нийтийн замд гарахгүй: лацны дугаар бол хяналтын хэрэгсэл,
    -- түүнийг мэдэх нь түүнийг хуурах эхний алхам.
    seal_no              TEXT NOT NULL DEFAULT '',
    seal_status          TEXT NOT NULL DEFAULT 'sealed_intact',

    status               TEXT NOT NULL DEFAULT 'in_transit',

    current_lat          DOUBLE PRECISION,
    current_lon          DOUBLE PRECISION,
    speed_kmh            NUMERIC(6, 2),
    heading              NUMERIC(6, 2),
    reported_at          TIMESTAMPTZ,

    departed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    eta_at               TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,

    -- Демо өгөгдлийг бодитоос ялгах. Дахин үүсгэхэд давхардуулахгүй.
    source               TEXT,
    source_ref           TEXT,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_fuel_trips_source
    ON fuel_dispatch_trips (tenant_id, source, source_ref)
    WHERE source IS NOT NULL AND source_ref IS NOT NULL;

CREATE INDEX idx_fuel_trips_tenant ON fuel_dispatch_trips (tenant_id);
CREATE INDEX idx_fuel_trips_station ON fuel_dispatch_trips (to_station_id);
-- Идэвхтэй рейсийг олох нь хамгийн олон давтагдах query. Дууссан рейс нь
-- индексээс гарна: тэднийг тоолдог нь тайлан, зураг биш.
CREATE INDEX idx_fuel_trips_active ON fuel_dispatch_trips (status)
    WHERE completed_at IS NULL;

ALTER TABLE fuel_dispatch_trips ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_dispatch_trips FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON fuel_dispatch_trips TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON fuel_dispatch_trips TO gerege_nexus_tenant;

-- +goose Down

DROP TABLE IF EXISTS fuel_dispatch_trips;
