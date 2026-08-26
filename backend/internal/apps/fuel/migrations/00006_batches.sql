-- Партия — гинжин хэлхээг холбодог дугаар.
--
-- # Юуг засаж байна
--
-- Өнөөг хүртэл цистерн ирээд «дууссан» гэж тэмдэглэгдэнэ, ШТС-ын нөөц ХЭВЭЭР
-- үлдэнэ. 12,000 литр хаашаа орсон нь хаана ч бүртгэгдэхгүй. Гинж тасарсан,
-- нөөцийн тоо нь импортоор оруулсан зохиомол утга хэвээр.
--
-- # Яагаад дугаар хэрэгтэй вэ
--
-- Гааль, бааз, цистерн, ШТС гэсэн дөрвөн бүртгэл байж болно — тэдгээр нь
-- дугаараар холбогдоогүй бол цуглуулга л болно, гинж биш. Зохицуулагчийн
-- асуулт нь «хэдэн тонн орж ирсэн бэ» биш:
--
--     энэ хошуунаас гарсан литр ХААНААС ирсэн бэ
--
-- Октаны тоог худал зарласан партийг гарал үүслээр нь олох (FUELNET_PLAN §7.2
-- К4) гэдэг нь энэ гинж бүтэн байхыг шаардана. Чанарын гэрчилгээ нь партид
-- наалдаж, партитай хамт хошуу хүртэл явна.
--
-- # `received_liters` яагаад партид байна вэ
--
-- Хэдэн литр орж ирснийг мэдэхэд хангалтгүй — хаана хүрснийг мэдэх хэрэгтэй.
-- Хоёрын зөрүү нь алдагдал, хулгай, эсвэл хэмжлийн алдаа: аль нь болохыг энэ
-- систем хэлэхгүй ч ЗӨРҮҮ БАЙГААГ хэлнэ, тэр нь хэлж чадах ба хэлэх ёстой зүйл.

-- +goose Up

CREATE TABLE fuel_batches (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    -- Хүн уншиж, утсаар давтаж чадах дугаар. Мэдүүлэг, дагалдах бичиг,
    -- лабораторийн хариу дээр энэ дугаар бичигдэнэ.
    batch_code       TEXT NOT NULL,

    fuel_type        TEXT NOT NULL,
    fuel_label       TEXT NOT NULL DEFAULT '',

    origin_country   TEXT NOT NULL DEFAULT '',
    refinery         TEXT NOT NULL DEFAULT '',
    -- Гаалийн мэдүүлгийн дугаар. Х2 хүртэл хоосон — тэр үе шатанд
    -- fuel_customs_shipments үүсэхэд FK болно.
    customs_decl_no  TEXT NOT NULL DEFAULT '',

    -- Хилээр орсон хэмжээ, ба ШТС-уудад бодитоор хүрсэн хэмжээ.
    imported_liters  NUMERIC(14, 3) NOT NULL DEFAULT 0,
    received_liters  NUMERIC(14, 3) NOT NULL DEFAULT 0,

    -- Чанар. Гинжээр дамжих гол ачаа — К4 үүн дээр тогтоно.
    quality_cert_no  TEXT NOT NULL DEFAULT '',
    octane_tested    NUMERIC(5, 2),
    sulfur_ppm       NUMERIC(8, 2),
    lab_status       TEXT NOT NULL DEFAULT 'pending',

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Нэг байгууллага дотор дугаар давхардахгүй. Улс даяар давхардаж болно:
    -- партийн дугаар нь импортлогчийнх, гаалийн мэдүүлгийн дугаар нь улсынх.
    UNIQUE (tenant_id, batch_code)
);

CREATE INDEX idx_batches_tenant ON fuel_batches (tenant_id);
CREATE INDEX idx_batches_fuel ON fuel_batches (fuel_type);

-- Цистерн ямар партийн түлш зөөж байгаа. NULL нь партийн дугааргүй хуучин
-- рейс — гинжинд орохгүй, гэхдээ түүхээс арилахгүй.
ALTER TABLE fuel_dispatch_trips
    ADD COLUMN batch_id UUID REFERENCES fuel_batches (id) ON DELETE SET NULL;

CREATE INDEX idx_trips_batch ON fuel_dispatch_trips (batch_id);

-- ШТС дээр буулгасан ачаа.
--
-- Энэ бол гинжний сүүлийн холбоос ба нөөц бодитоор нэмэгддэг цорын ганц газар.
-- Мөр бүр нь хэн, хэзээ, аль партиас, хэдэн литр гэдгийг хэлнэ.
CREATE TABLE fuel_station_receipts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES platform.tenants (id) ON DELETE CASCADE,

    station_id        UUID NOT NULL REFERENCES fuel_stations (id) ON DELETE CASCADE,
    -- Рейсгүй ч ачаа ирж болно (гараар бүртгэсэн, эсвэл мөрдөгдөөгүй тээвэр).
    trip_id           UUID REFERENCES fuel_dispatch_trips (id) ON DELETE SET NULL,
    batch_id          UUID REFERENCES fuel_batches (id) ON DELETE SET NULL,

    fuel_type         TEXT NOT NULL,
    liters            NUMERIC(12, 3) NOT NULL CHECK (liters > 0),

    -- Лац бүтэн ирсэн үү. Энэ бол хяналтын гол баримт: эвдэрсэн лац нь
    -- замд ачаа хөндөгдсөнийг хэлнэ.
    seal_status       TEXT NOT NULL DEFAULT 'sealed_intact',
    -- Дагалдах бичигт бичигдсэн хэмжээ. Бодитоор хүлээж авсантай зөрвөл
    -- зөрүү нь энд харагдана.
    manifest_liters   NUMERIC(12, 3),

    received_by       UUID,
    received_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note              TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_receipts_station ON fuel_station_receipts (station_id, received_at DESC);
CREATE INDEX idx_receipts_batch ON fuel_station_receipts (batch_id);
-- Нэг рейс нэг л удаа хүлээж авагдана. Хоёр дахь удаа дарахад нөөц хоёр
-- дахин нэмэгдэх нь тоолуурыг чимээгүй эвдэх хамгийн хялбар зам.
CREATE UNIQUE INDEX idx_receipts_one_per_trip
    ON fuel_station_receipts (trip_id) WHERE trip_id IS NOT NULL;

ALTER TABLE fuel_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_batches FORCE ROW LEVEL SECURITY;
ALTER TABLE fuel_station_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE fuel_station_receipts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON fuel_batches TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

CREATE POLICY tenant_isolation ON fuel_station_receipts TO gerege_nexus_tenant
    USING (tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE ON fuel_batches TO gerege_nexus_tenant;
GRANT SELECT, INSERT ON fuel_station_receipts TO gerege_nexus_tenant;

-- +goose Down

DROP TABLE IF EXISTS fuel_station_receipts;
ALTER TABLE fuel_dispatch_trips DROP COLUMN IF EXISTS batch_id;
DROP TABLE IF EXISTS fuel_batches;
