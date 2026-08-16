-- Даалгавар, хүсэлт бүрд хүн уншихуйц бүртгэлийн дугаар.
--
-- Одоо болтол мөр бүр зөвхөн UUID-тай байсан. UUID нь машинд төгс, хүнд
-- хэрэггүй: сумын ажилтан утсаар "манай 14 дэх даалгавар" гэж хэлдэг,
-- "f6215cd1-4d5f-4c1b-b287-720eccd16884" гэж хэлдэггүй.
--
-- ФОРМАТ
--
--   Д2026-00412   албан даалгавар
--   Ү2026-01875   үйлчилгээний хүсэлт
--
-- Эхний тэмдэгт нь ШУГАМ: Ү = үйлчилгээ, Д = даалгавар. Дугаарыг харангуут
-- ямар амлалттайг мэдэх нь хамгийн их хэрэглэгддэг мэдээлэл — хариу заавал
-- буцах ёстой юу, үгүй юү гэдэг.
--
-- Кирилл үсэг санамсаргүй биш: энэ бүтээгдэхүүн монголоор ярьдаг, шугамын
-- нэр нь Үйлчилгээ/Даалгавар, дугаар нь албан бичиг дээр хэвлэгдэнэ. Дугаар
-- нь API-ийн танигч БИШ (тэнд UUID хэвээр) тул URL-д ороод асуудал үүсгэхгүй.
--
-- Дараалал нь ТУХАЙН СУУЛГАЦ, ТУХАЙН ШУГАМ, ТУХАЙН ОН тус бүрд 1-ээс эхэлнэ —
-- албан хэрэг хөтлөлтийн жишиг.
--
-- ЯАГААД ПЛАТФОРМЫН УГТВАР БАЙХГҮЙ ВЭ
--
-- "BSHUYA/Д2026-00412" гэсэн хувилбар авч үзэгдсэн бөгөөд гээгдсэн: холбоос
-- бүрийн мөр аль хэдийн `name` талбартай ("Ховд аймаг") бөгөөд түүнийг админ
-- handshake хийхдээ өөрөө өгдөг. Суулгац өөр богино код зарлаж эхэлбэл нэг
-- зүйлийн ХОЁР ДАХЬ хүний нэр үүсэх байв — хэн ч баталгаажуулдаггүй, хоорондоо
-- зөрж болох нэр. Тиймээс дугаарыг нөгөө талын нэртэй нь ХАМТ харуулна:
--
--   Ховд аймаг · Д2026-0087
--
-- Цаасан дээр ч яг ийм: бичгийн дугаарыг гаргасан байгууллагынх нь нэртэй
-- хамт уншдаг, дугаар дангаараа хэзээ ч утгагүй.
--
-- ШАТ БҮР ӨӨРИЙН ДУГААР ҮҮСГЭЖ, ЭХИЙНХИЙГ ИШ ТАТНА
--
--   БШУЯ      Д2026-00412
--     └ Ховд    Д2026-0087   ← БШУЯ, Д2026-00412
--        └ Буянт  Д2026-0014 ← Ховд, Д2026-0087
--
-- Ирсэн бичгийг өөрийн бүртгэлд дугаарлаж, илгээгчийн дугаарыг иш татдаг —
-- цаасан бичиг яг ингэж ажилладаг. Энэ нь `origin_chain`-тэй давхардахгүй:
-- origin_chain нь машинд (мөчлөгийн шалгалт), дугаар нь хүнд.

-- +goose Up

ALTER TABLE urtuu_tasks
    -- Энэ суулгац дээрх бүртгэлийн дугаар.
    ADD COLUMN IF NOT EXISTS number        VARCHAR(32) NOT NULL DEFAULT '',
    -- Илгээгч талын дугаар, иш татахад. Хоосон бол энд үүссэн ажил.
    ADD COLUMN IF NOT EXISTS origin_number VARCHAR(32) NOT NULL DEFAULT '';

-- Хэсэгчилсэн индекс: энэ миграцаас өмнөх мөрүүд дугааргүй ('') байх бөгөөд
-- тэднийг дугаарлах гэж оролдохгүй. Дугаар нь бүртгэлийн баримт — хойшоо
-- зохиовол тухайн үед хэвлэгдсэн зүйлтэй зөрнө.
CREATE UNIQUE INDEX IF NOT EXISTS idx_urtuu_tasks_number
    ON urtuu_tasks (tenant_id, number) WHERE number <> '';

-- Дугаарлагч. Тенант, шугам, он тус бүрд нэг мөр.
--
-- Postgres-ийн SEQUENCE биш: sequence нь тенантад харьяалагдаж чадахгүй,
-- RLS-д хамрагдахгүй, он бүрд тэглэгдэхгүй, мөн тенант устахад хамт устахгүй.
-- Нэг мөр дээрх УPDATE нь мөрийн түгжээгээр цуваачилдаг тул advisory lock ч
-- хэрэггүй.
CREATE TABLE IF NOT EXISTS urtuu_numbers (
    tenant_id UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    line      TEXT    NOT NULL,
    year      INTEGER NOT NULL,
    -- Хамгийн сүүлд ОЛГОСОН дугаар. Дараагийнх нь next + 1.
    next      INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (tenant_id, line, year),
    CONSTRAINT urtuu_numbers_line_check CHECK (line IN ('service', 'assignment'))
);

GRANT SELECT, INSERT, UPDATE ON urtuu_numbers TO gerege_nexus_app;
-- DELETE олгогдоогүй: дугаарлагчийг тэглэх нь аль хэдийн олгогдсон дугаарыг
-- дахин олгох гэсэн үг.

-- +goose StatementBegin
DO $rls$
BEGIN
    EXECUTE 'ALTER TABLE public.urtuu_numbers ENABLE ROW LEVEL SECURITY';
    EXECUTE 'ALTER TABLE public.urtuu_numbers FORCE ROW LEVEL SECURITY';
    EXECUTE 'DROP POLICY IF EXISTS tenant_isolation ON public.urtuu_numbers';
    EXECUTE 'CREATE POLICY tenant_isolation ON public.urtuu_numbers TO gerege_nexus_app '
            'USING (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid) '
            'WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)';
END
$rls$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS urtuu_numbers;
DROP INDEX IF EXISTS idx_urtuu_tasks_number;
ALTER TABLE urtuu_tasks
    DROP COLUMN IF EXISTS origin_number,
    DROP COLUMN IF EXISTS number;
