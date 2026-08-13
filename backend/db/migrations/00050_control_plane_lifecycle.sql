-- CP-2 — тенантын амьдралын мөчлөг, quota, дэмжлэг, impersonation.
--
-- CP-1 нь консолыг зөвхөн уншдаг болгож барьсан. Энэ миграц нь түүнд бичих
-- эрхийг өгнө — гэхдээ бөөнөөр нь биш, үйлдэл тус бүрээр нь. Доорх GRANT-ууд
-- нь "оператор одоо бичиж чадна" гэсэн үг биш, "оператор ЯГ ЭДГЭЭР баганад,
-- ЯГ ЭДГЭЭР хүснэгтэд бичиж чадна" гэсэн үг. Хамгийн тод жишээ нь `users`:
--
--     GRANT UPDATE (failed_login_attempts, locked_until) ON users
--
-- Багана нэрлэсэн GRANT нь консолыг түгжигдсэн бүртгэлийг тайлж чаддаг,
-- харин нууц үгийн hash, и-мэйл, эсвэл `is_admin`-ыг өөрчилж ЧАДАХГҮЙ болгоно.
-- Тэр хязгаарлалт нь Go кодын дүрэм биш, PostgreSQL-ийн шалгалт: дэмжлэгийн
-- handler дотор алдаа гарсан ч, эсвэл хожим хэн нэгэн тэнд өөр UPDATE бичсэн ч
-- өгөгдлийн сан татгалзана.
--
-- Устгал нь энд ч, кодод ч шууд байхгүй. Тенант "устгагдана" гэдэг нь
-- `deletion_scheduled_at` тавигдана гэсэн үг бөгөөд мөрүүд нь 30 хоногийн
-- дараа цэвэрлэгээний ажлаар (login role-оор, консолын эрхээр биш) устна.
-- Консолд DELETE эрх ХААНА Ч байхгүй.

-- +goose Up

-- ============================================================ Амьдралын мөчлөг

-- Түдгэлзүүлэлт ба устгалын хуваарь. Хоёулаа NULL байх нь "хэвийн ажиллаж
-- байна" — өөрөөр хэлбэл өнөөдрийг хүртэлх бүх тенантын төлөв.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS suspension_reason TEXT NOT NULL DEFAULT '';
-- Устгал болох цаг. Энэ хугацаа хүртэл өгөгдөл бүрэн хэвээр байх бөгөөд нэг
-- товчоор сэргээгдэнэ. §3.A: шууд hard delete гэж байхгүй.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS deletion_scheduled_at TIMESTAMPTZ;

-- "Устгал хүлээж буй" тенантуудыг цэвэрлэгээний ажил хайхад. Хэсэгчилсэн
-- индекс: мөрүүдийн 99.9% нь NULL бөгөөд тэдгээрийг индексжүүлэх нь утгагүй.
CREATE INDEX IF NOT EXISTS idx_tenants_deletion_scheduled
    ON tenants (deletion_scheduled_at) WHERE deletion_scheduled_at IS NOT NULL;

-- ================================================ Хоёр хүний зарчим (four-eyes)

-- Нэг операторын хүсэлт, өөр операторын зөвшөөрөл.
--
-- Одоохондоо ганц үйлдэлд хэрэглэгдэнэ — тенант устгах — бөгөөд `action` нь
-- чөлөөт текст биш харин Go талын хаалттай жагсаалт (approvals.go). Хүснэгт нь
-- ерөнхий байгаа нь CP-3-ын kill switch, CP-4-ийн deploy зэрэг ижил хамгаалалт
-- шаардах үйлдлүүд нэмэгдэхэд шинэ хүснэгт биш, шинэ мөр болно гэсэн үг.
--
-- Платформын түвшний хүснэгт тул tenant RLS хамаарахгүй: зөвшөөрлийг оператор
-- өгдөг, тенант биш.
CREATE TABLE IF NOT EXISTS pending_approvals (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    action           TEXT        NOT NULL,
    target_type      TEXT        NOT NULL,
    target_id        TEXT        NOT NULL,
    -- Үйлдлийг гүйцэтгэхэд хэрэгтэй нэмэлт өгөгдөл (жишээ нь grace-ийн хоног).
    payload          JSONB       NOT NULL DEFAULT '{}'::jsonb,
    requested_by     UUID        NOT NULL,
    requested_reason TEXT        NOT NULL,
    requested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Хүсэлт хэзээ хүчингүй болох. Хугацаагүй хүсэлт нь хэдэн сарын дараа
    -- хэн нэгэн санамсаргүй дарж тенант устгах товч болно.
    expires_at       TIMESTAMPTZ NOT NULL,
    approved_by      UUID,
    approved_at      TIMESTAMPTZ,
    rejected_by      UUID,
    rejected_at      TIMESTAMPTZ,
    rejected_reason  TEXT        NOT NULL DEFAULT '',
    -- Зөвшөөрөгдсөн хүсэлт нэг л удаа хэрэгжинэ.
    executed_at      TIMESTAMPTZ,
    -- Хүссэн хүн өөрөө зөвшөөрч болохгүй. Энэ нь Go талд ч шалгагдана, гэхдээ
    -- дүрмийг өгөгдлийн санд бичих нь түүнийг мартагдашгүй болгоно.
    CONSTRAINT pending_approvals_two_people CHECK (approved_by IS NULL OR approved_by <> requested_by)
);

CREATE INDEX IF NOT EXISTS idx_pending_approvals_open
    ON pending_approvals (requested_at DESC)
    WHERE approved_at IS NULL AND rejected_at IS NULL AND executed_at IS NULL;

-- =========================================================================== Quota

-- Тенант бүрийн хязгаарууд.
--
-- `enforcement` нь зөөлөн (анхааруулга бичигдэнэ, үйлдэл өнгөрнө) эсвэл хатуу
-- (үйлдэл татгалзана). Анхдагч нь зөөлөн: хязгаарыг эхлээд хэмжиж, дараа нь
-- хааж эхлэх нь дараалал бөгөөд эсрэгээр нь хийвэл хэн нэгний ажил зогсоно.
--
-- NULL нь "хязгаargүй". 0 биш: 0 нь бодит хязгаар (юу ч болохгүй) бөгөөд
-- хоёрыг ялгаж чаддаг байх нь энэ хүснэгтийн цорын ганц эмзэг цэг.
--
-- Хадгалалт ба AI-ийн хязгаар энд хадгалагдана ч CP-2 дээр ЗӨВХӨН хэрэглэгчийн
-- тоо шалгагдана — бусад хоёрыг хэмжих өгөгдөл (usage_events) CP-5-д ирнэ.
-- Хэмжигдээгүй хязгаарыг "хэрэгжиж байна" гэж харуулах нь худал тул UI нь
-- тэдгээрийг тэмдэглэж харуулна.
CREATE TABLE IF NOT EXISTS tenant_quotas (
    tenant_id            UUID        PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    max_users            INTEGER     CHECK (max_users IS NULL OR max_users >= 0),
    max_storage_mb       INTEGER     CHECK (max_storage_mb IS NULL OR max_storage_mb >= 0),
    max_ai_calls_monthly INTEGER     CHECK (max_ai_calls_monthly IS NULL OR max_ai_calls_monthly >= 0),
    enforcement          TEXT        NOT NULL DEFAULT 'soft' CHECK (enforcement IN ('soft', 'hard')),
    updated_by           UUID,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- `tenant_id`-тай хүснэгт тул 00029-ийн загвараар RLS. Гэхдээ тенантын талд
-- ЗӨВХӨН УНШИХ: өөрийнхөө хязгаарыг харах нь зөв (UI дээр "хэрэглэгчийн тоо
-- 18/20" гэж харуулна), өөрчлөх нь оператор л хийнэ. WITH CHECK бичээгүй тул
-- app role нь мөр нэмэх, өөрчлөх аргагүй.
ALTER TABLE tenant_quotas ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_quotas FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON tenant_quotas;
CREATE POLICY tenant_isolation ON tenant_quotas FOR SELECT TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
GRANT SELECT ON tenant_quotas TO gerege_nexus_app;

-- ================================================================ Impersonation

-- Оператор тенант дотор орж харах бүрийн бүртгэл.
--
-- Session өөрөө `sessions`-д үүснэ (доорх `impersonated_by`), энэ хүснэгт нь
-- түүний ЗӨВШӨӨРӨЛ ба ШАЛТГААН: хэн, ямар байгууллагад, хэний нэрээр, яагаад,
-- хэзээ хүртэл. Session дуусаад цэвэрлэгдсэн ч энэ мөр үлдэнэ.
--
-- Гар дамжуулах (handover) token нь консолын хост дээр үүсээд тенантын хост
-- дээр нэг удаа солигдоно — cookie нь домэйн хооронд тавигдахгүй тул өөр зам
-- байхгүй. Token нь SHA-256 hash хэлбэрээр хадгалагдаж, 60 секунд хүчинтэй,
-- нэг л удаа солигдоно.
CREATE TABLE IF NOT EXISTS operator_impersonations (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    operator_id        UUID        NOT NULL,
    operator_email     TEXT        NOT NULL DEFAULT '',
    tenant_id          UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id            UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason             TEXT        NOT NULL,
    handover_hash      CHAR(64)    UNIQUE NOT NULL,
    handover_expires_at TIMESTAMPTZ NOT NULL,
    redeemed_at        TIMESTAMPTZ,
    ends_at            TIMESTAMPTZ NOT NULL,
    ended_at           TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operator_impersonations_tenant
    ON operator_impersonations (tenant_id, created_at DESC);

-- Тенант нь ӨӨРИЙНХӨӨ мөрийг харна. §3.B-ийн consent + audit загварын гол
-- цэг нь энэ: "хэн миний өгөгдлийг харав" гэсэн асуулт нь операторын нууц
-- биш, тенантын эрх. Бичих эрх нь тенантад байхгүй.
ALTER TABLE operator_impersonations ENABLE ROW LEVEL SECURITY;
ALTER TABLE operator_impersonations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON operator_impersonations;
CREATE POLICY tenant_isolation ON operator_impersonations FOR SELECT TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
GRANT SELECT ON operator_impersonations TO gerege_nexus_app;

-- Session нь хэний нэрээр ажиллаж байгаа ба ХЭН түүнийг эхлүүлснийг зөөнө.
-- NULL нь энгийн session — өнөөдрийг хүртэлх бүгд.
--
-- Гадаад түлхүүргүй: операторын бүртгэл устсан ч session хэний байсныг
-- мартах ёсгүй (00043-ийн `audit_events.user_id`-тай ижил шалтгаан).
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS impersonated_by UUID;

-- =================================================== Нэвтрэх эрхийг сэргээх

-- Урилга ба нууц үг сэргээх нэг механизм.
--
-- Платформ дээр өнөөдрийг хүртэл нууц үг сэргээх зам БАЙГААГҮЙ: нууц үгээ
-- мартсан хүн админ руугаа хандаж, админ нь өгөгдлийн сан руу ордог байв.
-- Дэмжлэгийн багц (§3.B) ба шинэ тенантын эхний админ хоёулаа үүнийг
-- шаардсан тул нэг механизмаар хийв — хоёулангийнх нь утга нэг: "энэ хаягийг
-- эзэмшдэг хүн нууц үгээ тавь".
--
-- Token нь зөвхөн SHA-256 hash хэлбэрээр хадгалагдана (sessions-тэй ижил),
-- нэг л удаа хэрэглэгдэнэ, богино хугацаатай. Захидлыг өөрөө илгээхгүй —
-- одоо байгаа emailverify рүү дамжуулна.
--
-- `tenant_id` багана байхгүй тул RLS хамаарахгүй: энэ нь хүний бүртгэлийн
-- тухай, байгууллагын тухай биш — `users` хүснэгттэй яг ижил шалтгаан.
CREATE TABLE IF NOT EXISTS credential_grants (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose            TEXT        NOT NULL CHECK (purpose IN ('invite', 'reset')),
    token_hash         CHAR(64)    UNIQUE NOT NULL,
    -- Хэн илгээснийг санана. NULL нь "хүн өөрөө хүссэн" — тийм зам одоохондоо
    -- байхгүй ч энэ багана түүнийг хожим нэмэхэд хүснэгтийг өөрчлөхгүй.
    issued_by_operator UUID,
    expires_at         TIMESTAMPTZ NOT NULL,
    redeemed_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credential_grants_user
    ON credential_grants (user_id, created_at DESC);

-- ======================================================= Операторын шинэ эрхүүд

-- Тенант үүсгэх, түдгэлзүүлэх, сэргээх, устгалд тавих.
GRANT INSERT, UPDATE ON tenants TO gerege_nexus_operator;
GRANT INSERT, UPDATE ON tenant_profiles TO gerege_nexus_operator;
-- Шинэ тенантын админ: үүрэг, гишүүнчлэл, тэдгээрийн холбоос.
GRANT INSERT ON roles, memberships, membership_roles TO gerege_nexus_operator;
-- Дэмжлэг: session цуцлах, impersonation-ы session үүсгэх.
GRANT INSERT, UPDATE ON sessions TO gerege_nexus_operator;
-- Түгжээ тайлах — ЗӨВХӨН эдгээр хоёр багана. Толгойн тайлбарыг үз.
GRANT UPDATE (failed_login_attempts, locked_until) ON users TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE ON pending_approvals TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE ON tenant_quotas TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE ON operator_impersonations TO gerege_nexus_operator;
-- Урилга/сэргээлт үүсгэх. Хэрэглэх нь тенантын талд, нэвтрээгүй хүний
-- хүсэлтээр болох тул тэр нь платформын зам (login role) — операторт
-- UPDATE эрх хэрэггүй.
GRANT SELECT, INSERT ON credential_grants TO gerege_nexus_operator;
-- Тенантын өөрийнх нь audit урсгал руу бичих эрх — ЗӨВХӨН INSERT.
--
-- Оператор тенант дотор орж харах бүрд тэр байгууллагын өөрийнх нь бүртгэлд
-- мөр үлдэнэ (§3.B-ийн "мэдэгдэл"). Тэр мөр нь операторын консол дээр биш,
-- тенантын админы аль хэдийн уншдаг дэлгэц дээр гарах ёстой — эс бөгөөс
-- "хэн миний өгөгдлийг харав" гэдэг нь тэднээс биднээс асуухыг шаардсан
-- асуулт хэвээр үлдэнэ. UPDATE/DELETE байхгүй: 00043-ын шийдвэрээр audit
-- бол засагддаггүй бүртгэл.
GRANT INSERT ON audit_events TO gerege_nexus_operator;

-- `tenants` ба `users` нь `tenant_id` баганагүй тул RLS-гүй — GRANT хангалттай.
-- Бусад нь 00029-ээр RLS-тэй бөгөөд тэдгээрийн бодлого нь `TO gerege_nexus_app`
-- гэж бичигдсэн байдаг тул операторын үйлдэлд тусад нь бодлого хэрэгтэй.
-- 00049-ийн `operator_read` нь зөвхөн SELECT байсан; эдгээр нь бичилтүүд.
--
-- USING/WITH CHECK нь `true`: операторын query бүр хандах байгууллагаа өөрийн
-- WHERE-дээ нэрлэдэг бөгөөд тэр нь тухайн query-г уншихад харагддаг байх нь
-- энэ дизайны гол цэг (00049-ийн толгойг үз). RLS энд хийж чадах зүйл нь
-- "аль тенант" биш — тэр асуултад бодлого хариулах өгөгдөл байхгүй.
-- +goose StatementBegin
DO $operator_write$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['tenant_profiles', 'memberships', 'roles', 'sessions'] LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_write ON public.%I', target);
        EXECUTE format(
            'CREATE POLICY operator_write ON public.%I AS PERMISSIVE FOR ALL TO gerege_nexus_operator '
            'USING (true) WITH CHECK (true)', target);
    END LOOP;
    -- audit_events нь FOR ALL биш: операторт нэмэх эрх л хэрэгтэй бөгөөд
    -- унших эрхийг нь 00049-ийн operator_read аль хэдийн өгсөн.
    FOREACH target IN ARRAY ARRAY['audit_events'] LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_write ON public.%I', target);
        EXECUTE format(
            'CREATE POLICY operator_write ON public.%I AS PERMISSIVE FOR INSERT TO gerege_nexus_operator '
            'WITH CHECK (true)', target);
    END LOOP;
END
$operator_write$;
-- +goose StatementEnd

DROP POLICY IF EXISTS operator_write ON tenant_quotas;
CREATE POLICY operator_write ON tenant_quotas FOR ALL TO gerege_nexus_operator
    USING (true) WITH CHECK (true);

DROP POLICY IF EXISTS operator_write ON operator_impersonations;
CREATE POLICY operator_write ON operator_impersonations FOR ALL TO gerege_nexus_operator
    USING (true) WITH CHECK (true);

-- 00049-ийн `operator_read` нь SELECT-ийн бодлого байсан ба дээрх `operator_write`
-- нь FOR ALL тул хоёулаа хамт үйлчилнэ (PERMISSIVE бодлогууд нэгддэг).

-- +goose Down

DROP POLICY IF EXISTS operator_write ON operator_impersonations;
DROP POLICY IF EXISTS operator_write ON tenant_quotas;

-- +goose StatementBegin
DO $operator_write$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['tenant_profiles', 'memberships', 'roles', 'sessions', 'audit_events'] LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_write ON public.%I', target);
    END LOOP;
END
$operator_write$;
-- +goose StatementEnd

REVOKE ALL PRIVILEGES ON operator_impersonations FROM gerege_nexus_operator;
REVOKE ALL PRIVILEGES ON pending_approvals FROM gerege_nexus_operator;
REVOKE ALL PRIVILEGES ON tenant_quotas FROM gerege_nexus_operator, gerege_nexus_app;
REVOKE UPDATE (failed_login_attempts, locked_until) ON users FROM gerege_nexus_operator;
REVOKE INSERT ON audit_events FROM gerege_nexus_operator;
REVOKE INSERT, UPDATE ON sessions FROM gerege_nexus_operator;
REVOKE INSERT ON roles, memberships, membership_roles FROM gerege_nexus_operator;
REVOKE INSERT, UPDATE ON tenant_profiles FROM gerege_nexus_operator;
REVOKE INSERT, UPDATE ON tenants FROM gerege_nexus_operator;

ALTER TABLE sessions DROP COLUMN IF EXISTS impersonated_by;
DROP TABLE IF EXISTS credential_grants;
DROP TABLE IF EXISTS operator_impersonations;
DROP TABLE IF EXISTS tenant_quotas;
DROP TABLE IF EXISTS pending_approvals;

DROP INDEX IF EXISTS idx_tenants_deletion_scheduled;
ALTER TABLE tenants DROP COLUMN IF EXISTS deletion_scheduled_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS suspension_reason;
ALTER TABLE tenants DROP COLUMN IF EXISTS suspended_at;
