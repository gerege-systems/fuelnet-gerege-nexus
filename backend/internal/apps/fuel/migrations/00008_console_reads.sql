-- Консол эдгээр хүснэгтийг уншина.
--
-- 00001-ээс 00007 хүртэл бүх grant `gerege_nexus_tenant`-д өгөгдсөн: компани
-- өөрийн мөрөө уншиж бичнэ, тэр нь зөв. Гэхдээ суулгацыг ажиллуулж буй хүн —
-- `/cp` консол, `gerege_nexus_operator` — платформын үндсэн сэдэв болох
-- шатахууны талаар юу ч харж чадахгүй байв: хэдэн байгууллага байгааг мэдэх ч
-- тэдгээрийн нэг нь ч нөөцөө мэдээлж байгаа эсэхийг мэдэх аргагүй.
--
-- Хэлбэр нь платформын өөрийн хүснэгтүүд 00049-өөс хойш барьж ирсэнтэй ижил:
-- `USING (TRUE)` бүхий SELECT бодлого, SELECT grant, өөр юу ч биш. Оператор
-- бичих эрхгүй — консолын дэлгэц нь нэгтгэсэн тоо харуулдаг, компанийн
-- бүртгэлийг засдаг газар биш.
--
-- Хуулбар үүсгэхгүй байх нь энэ migration-ий гол утга. Улсын нөөцийн тоог
-- шөнийн job-оор өөр хүснэгт рүү хуулж болох байсан — тэр үед компанийн
-- харж буй тоо, консолын харж буй тоо хоёр зөрөх боломж үүснэ. Нэг эх сурвалж
-- дээр хоёр эрхээр унших нь тэр боломжийг үлдээхгүй.

-- +goose Up

-- +goose StatementBegin
DO $console_reads$
DECLARE
    t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['fuel_stations', 'fuel_station_inventory',
                             'fuel_station_receipts', 'fuel_dispatch_trips',
                             'fuel_batches', 'fuel_customs_shipments',
                             'fuel_depots', 'fuel_depot_tanks', 'fuel_depot_receipts']
    LOOP
        -- Idempotent: бодлого нэрээр давхардвал migration унана, харин энэ
        -- migration нь суулгац бүр дээр яг нэг удаа ажиллана гэдэг баталгаа
        -- goose-д байгаа тул шалгалт нь зөвхөн гар аргаар дахин ажиллуулсан
        -- тохиолдолд хамаатай.
        IF NOT EXISTS (SELECT 1 FROM pg_policies
                        WHERE schemaname = 'tenant' AND tablename = t
                          AND policyname = 'operator_read') THEN
            EXECUTE format(
                'CREATE POLICY operator_read ON %I FOR SELECT TO gerege_nexus_operator USING (TRUE)', t);
        END IF;
        EXECUTE format('GRANT SELECT ON %I TO gerege_nexus_operator', t);
    END LOOP;
END
$console_reads$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $console_reads_down$
DECLARE
    t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['fuel_stations', 'fuel_station_inventory',
                             'fuel_station_receipts', 'fuel_dispatch_trips',
                             'fuel_batches', 'fuel_customs_shipments',
                             'fuel_depots', 'fuel_depot_tanks', 'fuel_depot_receipts']
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_read ON %I', t);
        EXECUTE format('REVOKE SELECT ON %I FROM gerege_nexus_operator', t);
    END LOOP;
END
$console_reads_down$;
-- +goose StatementEnd
