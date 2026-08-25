-- Рейсийн бодит зам.
--
-- Үүнээс өмнө цистерний байрлалыг бааз↔ШТС хоёрын ШУЛУУН дээр тооцоолж
-- байсан. Тэр нь зурган дээр машинуудыг гэр хороолол, барилга дундуур
-- явуулж, ямар ч жолооч хийхгүй зүйлийг харуулж байв.
--
-- Одоо зам нь илгээх үед НЭГ УДАА тооцоологдож энд хадгалагдана. Уншихдаа
-- тооцоолохгүй шалтгаан: замын хөдөлгүүр гадаад үйлчилгээ, түүнийг хүсэлт
-- бүрд дуудвал зураг нь тэр үйлчилгээний ажиллах хугацаанаас хамаарна.
-- Хадгалсан зам нь мөн «энэ ачаа хаагуур явах вэ» гэсэн асуултын бичигдсэн
-- хариулт болно — маргааш замаа өөрчилвөл тэр нь шинэ баримт.
--
-- Формат нь GeoJSON-ий байрлалын жагсаалт: [[lon,lat], ...]. LineString
-- дугтуй биш, зөвхөн цэгүүд — geometry төрөл нь уншигчийнх, энд хэрэггүй.

-- +goose Up

ALTER TABLE fuel_dispatch_trips
    ADD COLUMN route_geom       JSONB,
    ADD COLUMN route_distance_m NUMERIC(12, 1),
    ADD COLUMN route_duration_s INTEGER;

-- +goose Down

ALTER TABLE fuel_dispatch_trips
    DROP COLUMN IF EXISTS route_geom,
    DROP COLUMN IF EXISTS route_distance_m,
    DROP COLUMN IF EXISTS route_duration_s;
