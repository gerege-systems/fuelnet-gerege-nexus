-- Оператор консол «Өртөө»-гийн гэрлийг харна (docs/URTUU_PROPOSAL.md §6).
--
-- 00049 дээр бичигдсэн дүрэм: tenant_id-тай шинэ хүснэгт оператор руу
-- АВТОМАТААР нээгдэхгүй, харин түүнийг харах шаардлага гарсан өдөр нь тухайн
-- миграц өөрөө шийднэ. Тэр өдөр энэ.
--
-- Юуг нээж байгаа нь чухал: консолын нүүрэн дээрх «Өртөө» гэрэл гэдэг нь
-- ХОЁР ТОО — хүргэгдээгүй дугтуйн тоо, ярихаа больсон холбоосын тоо.
-- Тиймээс зөвхөн urtuu_peers ба urtuu_deliveries нээгдэнэ:
--
--   * urtuu_outbox, urtuu_inbox нээгдэхгүй. Тэнд ДУГТУЙН АГУУЛГА байдаг —
--     нэг байгууллагын нөгөөд өгсөн даалгаврын бие. Оператор "хүргэлт
--     зогссон уу" гэдгийг мэдэх ёстой, "юу бичсэн" гэдгийг биш;
--   * urtuu_tasks нээгдэхгүй. Мөн шалтгаанаар: даалгаврын агуулга нь
--     тухайн хоёр байгууллагынх.
--
-- Хэмжигдэхүүн биш, өгөгдлийн сангаас уншина. Шалтгаан нь эхний шатнаас
-- тогтсон дүрэм: ямар ч метрик дээр тенантын label байхгүй.

-- +goose Up

GRANT SELECT ON urtuu_peers, urtuu_deliveries TO gerege_nexus_operator;

-- RLS-ийн дүрмээр өөр role-д хамаарах бодлого байхгүй бол мөр НЭГ Ч
-- харагдахгүй, тиймээс GRANT дангаараа хангалтгүй.
-- +goose StatementBegin
DO $operator_read$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['urtuu_peers', 'urtuu_deliveries']
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_read ON public.%I', target);
        EXECUTE format(
            'CREATE POLICY operator_read ON public.%I FOR SELECT TO gerege_nexus_operator USING (true)',
            target);
    END LOOP;
END
$operator_read$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $operator_read$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['urtuu_peers', 'urtuu_deliveries']
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_read ON public.%I', target);
    END LOOP;
END
$operator_read$;
-- +goose StatementEnd

REVOKE SELECT ON urtuu_peers, urtuu_deliveries FROM gerege_nexus_operator;
