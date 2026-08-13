-- CP-3 — динамик тохиргоо, feature flag, засварын горим, зарлал.
--
-- Гурван хүснэгтийн бүлэг, нэг зорилготой: платформын зан төлөвийн нэг хэсгийг
-- deploy-оос салгах. Өнөөдрийг хүртэл "session хэдэн минут идэвхгүй байвал
-- дуусах вэ" гэдэг асуултын хариу нь env файлд байсан бөгөөд түүнийг өөрчлөх
-- нь контейнер дахин ачаалахыг шаарддаг байв — өөрөөр хэлбэл жижиг тохиргооны
-- өөрчлөлт бүр deploy-ын эрсдэлтэй хамт явна.
--
-- Гурван зүйлийг ЗОРИУДААР хийхгүй:
--
--   1. Нууц утга энд хэзээ ч орохгүй. Registry-д `secret` гэсэн төрөл БАЙХГҮЙ
--      (internal/platform/settings/registry.go) тул нууц утгыг бүртгэх Go-гийн
--      түвшинд боломжгүй. Нууц нь GitHub secrets ба env-д үлдэнэ.
--   2. Утга бүр Go код дотор бүртгэгдсэн байх ёстой. DB-д байгаа ч registry-д
--      байхгүй түлхүүр нь уншигдахгүй — өөрөөр хэлбэл энэ хүснэгтэд мөр нэмэх
--      нь платформын зан төлөвийг өөрчлөх зам биш.
--   3. Түүхгүй өөрчлөлт байхгүй. Утга солигдох бүрд `platform_settings_history`
--      мөр бичигдэнэ; буцаах товч нь тэр мөрийг уншина.

-- +goose Up

-- ==================================================== Динамик тохиргоо

-- Утга нь TEXT: registry нь Go талд төрөл, шалгалт, анхдагчийг мэддэг тул
-- өгөгдлийн санд хадгалах ёстой зүйл нь "оператор юу бичсэн" гэдэг мөр л юм.
-- jsonb байсан бол энгийн `45m` гэсэн утга ч хашилтад орж, гараар засахад
-- төөрөгдөл үүсгэнэ.
--
-- Платформын түвшний хүснэгт — tenant RLS хамаарахгүй.
CREATE TABLE IF NOT EXISTS platform_settings (
    key        TEXT        PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Өөрчлөлт бүрийн түүх. Буцаах товч нь энэ хүснэгтийн мөрийг сонгоод
-- `previous_value`-г дахин бичнэ — өөрөөр хэлбэл буцаалт нь бас нэг өөрчлөлт
-- бөгөөд өөрөө ч түүхэнд үлдэнэ. Түүхийг "цэвэрлэдэг" буцаалт нь юу болсныг
-- нуух хамгийн хялбар арга байх байсан.
CREATE TABLE IF NOT EXISTS platform_settings_history (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key            TEXT        NOT NULL,
    previous_value TEXT,
    new_value      TEXT        NOT NULL,
    reason         TEXT        NOT NULL DEFAULT '',
    changed_by     UUID,
    changed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_settings_history_key
    ON platform_settings_history (key, changed_at DESC);

-- ========================================================== Feature flag

-- Unleash-ийн туршлагаар: flag бүр нэр, зориулалт, эзэмшигчтэй бөгөөд
-- ХУГАЦААТАЙ. `expires_at` нь техникийн хязгаарлалт биш, сануулга: хугацаа
-- нь өнгөрсөн flag консолын нүүрэнд гарч ирнэ. Flag debt гэдэг нь кодод
-- үлдсэн, хэн ч санахгүй болсон салаа замуудын нэр юм.
CREATE TABLE IF NOT EXISTS feature_flags (
    key         TEXT        PRIMARY KEY,
    description TEXT        NOT NULL DEFAULT '',
    -- Хэн энэ flag-ийг арилгах үүрэгтэй вэ. Багийн нэр, хүний нэр аль нь ч
    -- байж болно — чухал нь эзэнгүй flag байхгүй байх.
    owner       TEXT        NOT NULL DEFAULT '',
    -- release: шинэ боломжийг аажим нээх. kill_switch: ажиллаж байгаа зүйлийг
    -- яаралтай унтраах. experiment: хэсэг тенантад турших.
    kind        TEXT        NOT NULL DEFAULT 'release'
                CHECK (kind IN ('release', 'kill_switch', 'experiment')),
    enabled     BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Хувиар нээх. Тенантын id-гийн тогтвортой hash-аар шийдэгддэг тул нэг
    -- тенант нэг өдөр нээлттэй, маргааш хаалттай болохгүй.
    rollout     INTEGER     NOT NULL DEFAULT 100 CHECK (rollout BETWEEN 0 AND 100),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Тенант тус бүрийн онцгой тохиргоо. Байвал хувь ба ерөнхий төлвөөс давуу.
CREATE TABLE IF NOT EXISTS feature_flag_overrides (
    flag_key   TEXT        NOT NULL REFERENCES feature_flags(key) ON DELETE CASCADE,
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    enabled    BOOLEAN     NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (flag_key, tenant_id)
);

-- `tenant_id`-тай хүснэгт тул 00029-ийн журмаар RLS.
--
-- Тенантын аппын role-д ямар ч GRANT өгөөгүй бөгөөд өгөх ч шаардлагагүй:
-- flag-ийн үнэлгээ нь платформын замаар ачаалагдсан кэшээс хийгддэг
-- (internal/platform/flags). Бодлогыг нь ингэж байхад нь бичиж байгаа шалтгаан
-- нь — хожим хэн нэгэн GRANT нэмэх өдөр нь тусгаарлалт нь аль хэдийн байрандаа
-- байх ёстой, тэр өдөр санаж бичихийг найдах ёсгүй.
ALTER TABLE feature_flag_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flag_overrides FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON feature_flag_overrides;
CREATE POLICY tenant_isolation ON feature_flag_overrides FOR SELECT TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- ============================================== Засварын горим ба зарлал

-- Тенант тус бүрийн засварын горим. Платформ даяарх нь `platform_settings`-ийн
-- `platform.maintenance` түлхүүрээр — тэр нь нэг утга бөгөөд хүснэгт
-- шаардахгүй.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS maintenance_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS maintenance_message TEXT NOT NULL DEFAULT '';

-- Зарлал. `tenant_id` нь NULL бол бүх тенантад.
--
-- Хугацаатай: эхлэх ба дуусах цагтай тул "маргааш 22:00 цагт засвар хийнэ"
-- гэсэн мэдэгдлийг өнөөдөр бичээд, дараа нь мартаж болно. Хугацаа дуусмагц
-- өөрөө алга болно — гараар устгах шаардлагагүй нь эдгээрийг бичих саадыг
-- бууруулна.
CREATE TABLE IF NOT EXISTS announcements (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        REFERENCES tenants(id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL DEFAULT 'info' CHECK (kind IN ('info', 'warning', 'maintenance')),
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL DEFAULT '',
    starts_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at    TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Хэсэгчилсэн индекс дээр `NOW()` ашиглах боломжгүй (PostgreSQL нь index
-- predicate-д IMMUTABLE функц шаардана — тэгэхгүй бол өнөөдөр индексэд байсан
-- мөр маргааш байхгүй болж, индекс өөрөө худал болно). Тиймээс энгийн индекс:
-- зарлал цөөхөн байх тул хугацааны шүүлт нь query-д үлдэнэ.
CREATE INDEX IF NOT EXISTS idx_announcements_live
    ON announcements (starts_at DESC);

-- Тенант нь өөрийнхөө болон бүх нийтийнхийг уншина. Бичих эрх байхгүй:
-- зарлалыг оператор бичдэг.
ALTER TABLE announcements ENABLE ROW LEVEL SECURITY;
ALTER TABLE announcements FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON announcements;
CREATE POLICY tenant_isolation ON announcements FOR SELECT TO gerege_nexus_app
    USING (tenant_id IS NULL
           OR tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
GRANT SELECT ON announcements TO gerege_nexus_app;

-- ================================ CP-2-ийн үлдэгдэл: тенант үүсгэх trigger

-- 00008-ийн `tenant_access_role_seed` trigger нь тенант үүсэх бүрд гурван
-- үүрэг ба тэдгээрийн эрхийг өөрөө бичдэг. Trigger нь дуудагчийн role-оор
-- ажилладаг (SECURITY INVOKER) тул консолын тенант үүсгэх зам түүн дээр
-- унаж байв: "permission denied for table role_permissions".
--
-- Энэ нь CP-2-т илрэх ёстой байсан бөгөөд CP-3-ын integration тест барьж авав
-- — өөрөөр хэлбэл өгөгдлийн сангийн эрхийг нарийн заасны үнэ: мартагдсан
-- GRANT нь чимээгүй өнгөрдөггүй, харин тестээр унадаг.
GRANT SELECT ON permissions TO gerege_nexus_operator;
GRANT INSERT ON role_permissions TO gerege_nexus_operator;

-- Мөн адил: шинэ байгууллагын эхний админыг үүсгэхэд `users`-д мөр нэмэх
-- шаардлагатай. 00050 нь тэр хүснэгтэд ЗӨВХӨН хоёр баганын UPDATE өгсөн
-- бөгөөд INSERT-ийг мартсан байв.
--
-- INSERT нь UPDATE-ээс аюулгүй: ингэж үүссэн бүртгэл нь хэрэглэх боломжгүй
-- нууц үгтэй (санамсаргүй 32 байт, хаана ч харагдахгүй) бөгөөд урилгын
-- холбоосоор л нээгдэнэ. Өөрөөр хэлбэл консол бүртгэл ҮҮСГЭЖ чадна, гэхдээ
-- түүгээр нэвтэрч чадахгүй — тэр ялгаа нь support.go-ийн бүхэл зорилго.
GRANT INSERT ON users TO gerege_nexus_operator;

-- ==================================================== Операторын эрхүүд

GRANT SELECT, INSERT, UPDATE ON platform_settings TO gerege_nexus_operator;
GRANT SELECT, INSERT ON platform_settings_history TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE, DELETE ON feature_flags TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE, DELETE ON feature_flag_overrides TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE, DELETE ON announcements TO gerege_nexus_operator;

-- Flag ба зарлал нь консолын өөрийн өгөгдөл тул устгах эрх нь энд байгаа нь
-- зөв: хугацаа нь өнгөрсөн flag-ийг арилгах чадваргүй бол flag debt-тэй
-- тэмцэх боломжгүй. Тенантын өгөгдөлд DELETE эрх ХЭВЭЭР байхгүй.

-- `feature_flag_overrides` ба `announcements` нь `tenant_id`-тай тул RLS-тэй
-- бөгөөд дээрх бодлогууд нь зөвхөн app role-д хамаарна. Операторын role-д
-- тусад нь бичнэ.
DROP POLICY IF EXISTS operator_write ON feature_flag_overrides;
CREATE POLICY operator_write ON feature_flag_overrides FOR ALL TO gerege_nexus_operator
    USING (true) WITH CHECK (true);
DROP POLICY IF EXISTS operator_write ON announcements;
CREATE POLICY operator_write ON announcements FOR ALL TO gerege_nexus_operator
    USING (true) WITH CHECK (true);

-- +goose Down

DROP POLICY IF EXISTS operator_write ON announcements;
DROP POLICY IF EXISTS operator_write ON feature_flag_overrides;

REVOKE ALL PRIVILEGES ON announcements FROM gerege_nexus_operator, gerege_nexus_app;
REVOKE ALL PRIVILEGES ON feature_flag_overrides FROM gerege_nexus_operator;
REVOKE ALL PRIVILEGES ON feature_flags FROM gerege_nexus_operator;
REVOKE ALL PRIVILEGES ON platform_settings_history FROM gerege_nexus_operator;
REVOKE ALL PRIVILEGES ON platform_settings FROM gerege_nexus_operator;
REVOKE INSERT ON role_permissions FROM gerege_nexus_operator;
REVOKE SELECT ON permissions FROM gerege_nexus_operator;
REVOKE INSERT ON users FROM gerege_nexus_operator;

DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS feature_flag_overrides;
DROP TABLE IF EXISTS feature_flags;
DROP TABLE IF EXISTS platform_settings_history;
DROP TABLE IF EXISTS platform_settings;

ALTER TABLE tenants DROP COLUMN IF EXISTS maintenance_message;
ALTER TABLE tenants DROP COLUMN IF EXISTS maintenance_at;
