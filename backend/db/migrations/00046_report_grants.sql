-- Тенант дамнасан тайлангийн зөвшөөрөл — саналын §3.5.
--
-- Бодит хэрэгцээ: нүүрсний уурхай 100 тээврийн компанитай гэрээтэй. Компани
-- бүр өөрийн тенант дотроо рейс, түлш, хүргэлтээ бүртгэдэг; уурхай "Тээвэр"
-- гэсэн ганц нэгдсэн тайлан харахыг хүснэ. Энэ бол тенант тусгаарлалтын эсрэг
-- урсгал тул далд, автоматаар хэзээ ч үүсэхгүй: RLS хэвээрээ, харин энэ хүснэгт
-- нь ТУСГАЙ, ЗӨВШӨӨРӨЛД СУУРИЛСАН зам болно.
--
-- Таван зарчим (§3.5):
--   1. Default deny. Grant байхгүй бол юу ч харагдахгүй.
--   2. Тодорхой, буцаагдах. Эзэмшигч тал яг аль тайланг, ямар хүрээгээр, хэдий
--      хугацаанд харуулахаа өөрөө зөвшөөрнө; revoked_at бичигдмэгц тэр дороо
--      хаагдана.
--   3. Counterparty scope. Уурхай тээврийн компанийн БҮХ үйл ажиллагааг биш,
--      зөвхөн өөртэй нь холбоотой мөрүүдийг харна.
--   4. Түүхий мөр биш, тайлангийн үр дүн. Хүлээн авагч эзэмшигчийн хүснэгтээс
--      шууд уншихгүй — хөдөлгүүр эзэмшигчийн контекст дотор query ажиллуулна.
--   5. Бүрэн audit. Хэн, хэзээ, хэний өгөгдлөөс юу татсан нь ХОЁР ТАЛДАА
--      бүртгэгдэнэ.
--
-- RLS-ийн загвар нь 00029-ээс ялгаатай бөгөөд энэ нь энэ хүснэгтийн гол
-- онцлог: grantor БА grantee хоёулаа өөрт хамаарах мөрөө харна. Ердийн
-- tenant_id = current_setting(...) бодлого нь grantee-д өөрт нь өгөгдсөн
-- grant-ыг харуулахгүй байх байсан — тэр мөр эзэмшигчийнх учраас — өөрөөр
-- хэлбэл хүлээн авагч тал ямар эрхтэйгээ мэдэхгүй байх байв.
--
-- tenant_id гэсэн багана ЗОРИУДААР байхгүй. Байсан бол 00029-ийн ерөнхий
-- бодлого автоматаар хамаарч, хоёр талын харагдацыг хаах байсан. Оронд нь
-- grantor_tenant_id / grantee_tenant_id хоёрын аль нэг нь таарахыг шалгана.

-- +goose Up
CREATE TABLE IF NOT EXISTS report_grants (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Өгөгдөл эзэмшигч (тээврийн компани). Зөвшөөрөл өгөх, цуцлах эрх нь
    -- эцсийн эцэст энэ талынх.
    grantor_tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Хүлээн авагч (уурхай).
    grantee_tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_key         TEXT        NOT NULL,
    -- 'counterparty' — зөвхөн хүлээн авагчтай холбоотой мөрүүд (гэрээт тал).
    -- 'full'         — эзэмшигчийн бүх мөр (толгой байгууллага, охин тенант).
    scope              TEXT        NOT NULL DEFAULT 'counterparty',
    -- Талуудыг холбох түлхүүр: хүлээн авагчийн улсын бүртгэлийн дугаар.
    -- Grant үүсэх үед НЭГ УДАА шийдэгдэж энд хадгалагдана — гүйлт бүрд
    -- дахин тулгах нь эзэмшигчийн профайл өөрчлөгдөхөд grant чимээгүй өөр
    -- өгөгдөл руу заахад хүргэнэ.
    counterparty_ref   TEXT        NOT NULL DEFAULT '',
    valid_from         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until        TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ,
    -- Хоёр талын баталгаажуулалт. Хүсэлтийг grantee үүсгэнэ (created_by),
    -- grantor-ын админ зөвшөөрнө (accepted_by). accepted_by NULL байх нь
    -- "хүлээгдэж буй хүсэлт" гэсэн үг бөгөөд идэвхтэй grant биш.
    created_by         UUID        REFERENCES users(id) ON DELETE SET NULL,
    accepted_by        UUID        REFERENCES users(id) ON DELETE SET NULL,
    accepted_at        TIMESTAMPTZ,
    -- Хүсэлт үүсгэсэн тал нь юу гэж хэлснийг үлдээнэ: grantor-ын админ
    -- зөвшөөрөх эсэхээ шийдэхэд "яагаад" гэдэг нь хэрэгтэй.
    note               TEXT        NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT report_grants_scope_check CHECK (scope IN ('counterparty', 'full')),
    -- Өөртөө grant өгөх нь утгагүй бөгөөд нэгдсэн тайланд өөрийн мөрийг
    -- давхардуулна.
    CONSTRAINT report_grants_not_self CHECK (grantor_tenant_id <> grantee_tenant_id)
);

-- Нэг хос тал, нэг тайлан дээр цуцлагдаагүй хоёр grant байж болохгүй: аль нь
-- үйлчлэхийг шийдэх дүрэм байхгүй ба нэгдсэн тайланд мөр давхардана.
CREATE UNIQUE INDEX IF NOT EXISTS idx_report_grants_live
    ON report_grants (grantor_tenant_id, grantee_tenant_id, report_key)
    WHERE revoked_at IS NULL;

-- Нэгдсэн тайлангийн гол query: "надад энэ тайланг хэн зөвшөөрсөн бэ".
CREATE INDEX IF NOT EXISTS idx_report_grants_grantee
    ON report_grants (grantee_tenant_id, report_key) WHERE revoked_at IS NULL;

-- "Миний өгөгдлийг хэн харж болох вэ" — эзэмшигч талын дэлгэц.
CREATE INDEX IF NOT EXISTS idx_report_grants_grantor
    ON report_grants (grantor_tenant_id) WHERE revoked_at IS NULL;

GRANT SELECT, INSERT, UPDATE ON report_grants TO gerege_nexus_app;

-- DELETE зориудаар олгогдоогүй. Grant-ыг устгах биш цуцална: "энэ өгөгдлийг
-- хэн, хэзээ, хэр хугацаанд харав" гэдэг нь устгагдвал хариулт нь алга болно.

ALTER TABLE report_grants ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_grants FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON report_grants;

-- Хоёр талын харагдац. USING нь grantor эсвэл grantee аль нэг нь одоогийн
-- тенант байхыг шаардана.
--
-- WITH CHECK нь ЗӨВХӨН grantee-г зөвшөөрнө: мөр үүсгэх нь хүсэлт гаргах явдал
-- бөгөөд түүнийг хүлээн авагч тал гаргана. Хэрэв grantor тал ч мөр үүсгэж
-- чаддаг байсан бол өөрийн өгөгдлийг хэн нэгэнд "түлхэх" зам нээгдэх байв —
-- энэ нь ажиллах ч байж болох урсгал, гэхдээ зөвшөөрлийн чиглэлийг эргүүлэх
-- тул тусад нь шийдвэрлэх ёстой. Зөвшөөрөх (accepted_by бичих) нь UPDATE
-- бөгөөд түүнийг доорх тусдаа бодлого зохицуулна.
CREATE POLICY tenant_isolation ON report_grants TO gerege_nexus_app
    USING (
        grantor_tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR grantee_tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
    )
    WITH CHECK (
        grantee_tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR grantor_tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
    );

-- +goose Down
DROP TABLE IF EXISTS report_grants;
