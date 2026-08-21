-- Өртөө-гийн мөрүүд идэвхтэй байгууллагын хэмжээнд биш, session-ий
-- зөвшөөрөгдсөн байгууллагуудын хэмжээнд харагдана.
--
-- 00037 нь `tenant_id = current_setting('app.current_tenant')` гэсэн
-- шалгуурыг `= ANY(COALESCE(app.allowed_tenants, ARRAY[app.current_tenant]))`
-- болгож өргөтгөсөн: олон байгууллагатай хүн нэг session дотор бүгдийг нь
-- уншина, гэхдээ бичихдээ зөвхөн идэвхтэй байгууллагадаа бичнэ. Тэр өдрөөс
-- хойш үүссэн хүснэгтүүдийн нэлээд нь тэр өргөтгөлийг аваагүй — 00061-ээс
-- 00066 хүртэлх Өртөө-гийн есөн хүснэгт багтана.
--
-- Аюулгүй биш гэхээсээ **зөрүүтэй**: хаагдах тал руугаа алдсан тул мэдээлэл
-- задраагүй, зүгээр л олон байгууллагатай хүн харах ёстой Өртөөний
-- даалгавраа хардаггүй. Гэхдээ нэг платформ дээр хоёр өөр тусгаарлалтын
-- дүрэм ажиллаж байгаа нь өөрөө асуудал: аль нь зөв болохыг дараагийн хүн
-- хуулж бичсэн policy-гоос таамаглана.
--
-- Зөвхөн Өртөө-гийнхийг зассан. Үлдсэн 16 нарийн policy-ийн заримынх нь
-- нарийн байх нь зориудынх (`FOR SELECT` бүхий консолын хүснэгтүүд,
-- төхөөрөмжийн хүрээний хүснэгтүүд), тиймээс бөөнөөр нь өргөтгөх нь
-- шийдвэр биш осол болно. Аль нь аль хэлбэртэйг
-- `TestTenantPoliciesHaveTheShapeOnRecord` бичиж авдаг.

-- +goose Up

-- +goose StatementBegin
DO $rls$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY[
        'urtuu_peers', 'urtuu_peer_codes', 'urtuu_request_codes', 'urtuu_numbers',
        'urtuu_tasks', 'urtuu_task_events', 'urtuu_inbox', 'urtuu_outbox',
        'urtuu_deliveries'
    ]
    LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', target);
        EXECUTE format('ALTER TABLE public.%I FORCE ROW LEVEL SECURITY', target);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', target);
        -- 00037-ийн хэлбэр, үг үсгээрээ: уншихдаа зөвшөөрөгдсөн
        -- байгууллагууд, бичихдээ зөвхөн идэвхтэй нэг нь.
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON public.%I TO gerege_nexus_app '
            'USING (tenant_id IS NULL OR tenant_id = ANY (COALESCE('
            '    NULLIF(current_setting(''app.allowed_tenants'', true), '''')::uuid[], '
            '    ARRAY[NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid]))) '
            'WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)',
            target);
    END LOOP;
END
$rls$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $rls$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY[
        'urtuu_peers', 'urtuu_peer_codes', 'urtuu_request_codes', 'urtuu_numbers',
        'urtuu_tasks', 'urtuu_task_events', 'urtuu_inbox', 'urtuu_outbox',
        'urtuu_deliveries'
    ]
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', target);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON public.%I TO gerege_nexus_app '
            'USING (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid) '
            'WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)',
            target);
    END LOOP;
END
$rls$;
-- +goose StatementEnd
