-- Control Plane — операторын консолын суурь.
--
-- Энэ файл гурван хүснэгт ба нэг DB role үүсгэнэ. Бүгд платформын түвшний
-- зүйл: аль ч тенантад харьяалагдахгүй, тиймээс 00029-ийн tenant RLS бодлого
-- ЭДГЭЭРТ ХАМААРАХГҮЙ. Шалтгаан нь энгийн — оператор гэдэг нь ямар нэг
-- байгууллагын хэрэглэгч биш, платформыг удирддаг хүн; `tenant_id` багана нь
-- утгагүй байх тул түүнд суурилсан бодлого ч утгагүй. Оронд нь эдгээр
-- хүснэгтийг тенантын аппын role (`gerege_nexus_app`) огт хүрэхгүй байхаар
-- эрхийг нь хааж, зөвхөн доор үүсэх операторын role уншина.
--
-- Энэ REVOKE нь чимээгүй боловч чухал: 00029 нь
-- `ALTER DEFAULT PRIVILEGES ... GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES
-- TO gerege_nexus_app` гэж бичсэн байдаг тул ЭНЭ ФАЙЛААР ҮҮССЭН ХҮСНЭГТҮҮД
-- автоматаар тэр эрхийг өвлөнө. Хэрэв доорх REVOKE байхгүй бол дурын тенантын
-- хүсэлт операторын нууц үгийн hash, TOTP-ийн нууц, session-ийн hash уншиж
-- чадах байсан — өөрөөр хэлбэл tenant-ын нэг эмзэг цэг нь control plane-ыг
-- бүхэлд нь өгнө. dbguard-ын зорилго яг үүний эсрэг тул энд давхар хаана.
--
-- Операторын role (`gerege_nexus_operator`) нь RLS-ийг тойрдоггүй
-- (NOBYPASSRLS). Тенантын өгөгдлийг харах эрхийг нь хүснэгт тус бүрээр
-- ЗӨВХӨН SELECT-ээр, зориудаар нэрлэсэн жагсаалтаар өгнө. Тиймээс "оператор
-- бүхнийг хардаг" гэдэг нь "операторын холболт бүх query-д нээлттэй" гэсэн үг
-- биш: жагсаалтад байхгүй хүснэгт нь операторт огт байхгүйтэй адил, бичих эрх
-- нь хаана ч байхгүй. CP-2 дээр тенант үүсгэх/түдгэлзүүлэх ажил гарах үед
-- тэр хүснэгтүүдэд шаардлагатай эрхийг тухайн миграц нь өөрөө, тодорхойлж
-- нэмнэ.

-- +goose Up

CREATE TABLE IF NOT EXISTS operator_accounts (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 TEXT        NOT NULL,
    name                  TEXT        NOT NULL,
    -- Дөрвөн үүрэг, CONTROL_PLANE_PLAN.md §2.2-ын дагуу. CHECK нь Go талын
    -- жагсаалтыг давхарлаж байгаа юм биш — жагсаалтыг мэдэхгүй ямар нэг
    -- скрипт мөр оруулах үед л энэ шалгалт ажиллана.
    role                  TEXT        NOT NULL CHECK (role IN ('superadmin', 'operator', 'support', 'auditor')),
    password_hash         TEXT        NOT NULL,
    -- TOTP-ийн нууц нь base32, `otpauth://` URI-д ордог хэлбэрээрээ. Хоосон
    -- байх нь "бүртгэл үүссэн ч authenticator холбоогүй" гэсэн үе шат бөгөөд
    -- нэвтрэлт тэр үед зогсоно (login.go-г үз): 2FA-гүй оператор бүртгэл
    -- ажиллаж эхлэх боломжгүй байх ёстой.
    totp_secret           TEXT        NOT NULL DEFAULT '',
    totp_confirmed_at     TIMESTAMPTZ,
    -- Хамгийн сүүлд хүлээн зөвшөөрөгдсөн TOTP-ийн цагийн алхам. Код нь 30
    -- секунд хүчинтэй байдаг тул нэг кодыг хоёр удаа хэрэглэж болно —
    -- мөрөө хартал, эсвэл хажуугаас нь хартал. Алхам нь заавал өсөж байх
    -- ёстой гэсэн дүрэм үүнийг хаана: нэг код нэг л удаа ажиллана.
    totp_last_step        BIGINT      NOT NULL DEFAULT 0,
    -- 00028-ийн хэрэглэгчийн lockout-тай ижил механик, гэхдээ тусдаа багана:
    -- операторын нэвтрэлт нь тенантын нэвтрэлтээс өөр босготой, өөр хүснэгтэд
    -- амьдардаг, нэгнийх нь тоолуур нөгөөгийнхөө хаалтад нөлөөлөх ёсгүй.
    failed_login_attempts INTEGER     NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    locked_until          TIMESTAMPTZ,
    -- Устгал биш идэвхгүйжүүлэлт. Операторын үйлдлүүд `operator_audit`-д
    -- түүний id-гаар үлддэг тул мөрийг устгах нь түүхийг нэргүй болгоно.
    disabled_at           TIMESTAMPTZ,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- И-мэйл нь жижиг үсгээр давхардахгүй — 00030 нь `users`-т яг үүнийг хийсэн.
-- Тэнд шалтгаан нь Postgres-ийн UNIQUE нь том/жижиг үсгийг ялгадаг явдал
-- байсан: `Bat@` ба `bat@` хоёр өөр мөр болж, нэвтрэлт аль нь болохыг таахаа
-- больдог.
CREATE UNIQUE INDEX IF NOT EXISTS idx_operator_accounts_email
    ON operator_accounts (lower(email));

-- WebAuthn-ы credential энд багана болж ороогүй. Төлөвлөгөө нь "боломжтой бол
-- WebAuthn, үгүй бол TOTP-оор эхэл" гэсэн бөгөөд TOTP-оор эхэлж байна.
-- Хэрэглэгдэхгүй jsonb багана нь схемийн амлалт: хожим бодит хэрэгжүүлэлт нь
-- credential бүрийг тусдаа мөр (counter, transports, тэмдэглэсэн огноо) болгож
-- шаардах магадлалтай тул тэр үед `operator_webauthn_credentials` хүснэгтийг
-- өөрийнх нь миграц үүсгэнэ.

CREATE TABLE IF NOT EXISTS operator_sessions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 00004-ийн `sessions`-тай ижил загвар: зөвхөн SHA-256 hash хадгална.
    token_hash    CHAR(64)    UNIQUE NOT NULL,
    operator_id   UUID        NOT NULL REFERENCES operator_accounts(id) ON DELETE CASCADE,
    user_agent    TEXT        NOT NULL DEFAULT '',
    ip_address    VARCHAR(64) NOT NULL DEFAULT '',
    -- Аюултай үйлдлийн өмнөх дахин баталгаажуулалт хэзээ болсон бэ. NULL нь
    -- "энэ session хэзээ ч step-up хийгээгүй". Хугацаа нь Go талд
    -- (controlplane.StepUpWindow) шийдэгдэнэ: DB нь баримтыг хадгална, дүрмийг
    -- биш.
    stepped_up_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_operator_sessions_operator
    ON operator_sessions (operator_id);
CREATE INDEX IF NOT EXISTS idx_operator_sessions_expires
    ON operator_sessions (expires_at);

CREATE TABLE IF NOT EXISTS operator_audit (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Гадаад түлхүүргүй, 00043-ийн `audit_events.user_id`-тай ижил
    -- шалтгаанаар: audit нь бүртгэл нь устсан операторын үйлдлийг ч санаж
    -- байх ёстой. И-мэйл нь давхардуулж бичигдэнэ — "энэ id хэн байсан бэ"
    -- гэдгийг хожим join-оор олохгүй байх магадлалтай.
    operator_id    UUID        NOT NULL,
    operator_email TEXT        NOT NULL DEFAULT '',
    action         TEXT        NOT NULL,
    target_type    TEXT        NOT NULL DEFAULT '',
    target_id      TEXT        NOT NULL DEFAULT '',
    -- Шалтгаан нь заавал бичигддэг талбар (§2.5). Хоосон мөр бичихийг DB
    -- түвшинд хориглоогүй: унших үйлдэл шалтгаан шаарддаггүй, харин бичих
    -- үйлдлийн шалтгааныг Go тал (audit.go) шаардана.
    reason         TEXT        NOT NULL DEFAULT '',
    before         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    after          JSONB       NOT NULL DEFAULT '{}'::jsonb,
    ip             VARCHAR(64) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operator_audit_time
    ON operator_audit (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_operator_audit_operator_time
    ON operator_audit (operator_id, created_at DESC);
-- "Энэ тенантад оператор юу хийсэн бэ" — CP-ийн тенантын дэлгэрэнгүй хуудас
-- ба тенантын админд өгөх хариултын аль аль нь ижил query.
CREATE INDEX IF NOT EXISTS idx_operator_audit_target
    ON operator_audit (target_type, target_id, created_at DESC);

-- Append-only гэдгийг хоёр давхаргаар барина.
--
-- Эхнийх нь эрх: доор operator role-д зөвхөн SELECT, INSERT олгоно.
-- Хоёр дахь нь trigger, учир нь эрх дангаараа хангалтгүй — миграц ба
-- dbguard-ын платформ зам нь хүснэгтийн эзэн болох login role-оор явдаг ба
-- эзэн нь GRANT/REVOKE-д захирагддаггүй. Тиймээс "хэн ч, хэзээ ч" гэдгийг
-- баталгаажуулах цорын ганц зүйл нь чимээгүй үгүйсгэдэг RULE биш, чанга
-- унадаг trigger юм: буруу код нь өөрчилж чадахгүйгээс гадна өөрчлөхийг
-- оролдсоноо мэдэх болно.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION operator_audit_is_append_only() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'operator_audit is append-only: % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS operator_audit_append_only ON operator_audit;
CREATE TRIGGER operator_audit_append_only
    BEFORE UPDATE OR DELETE ON operator_audit
    FOR EACH ROW EXECUTE FUNCTION operator_audit_is_append_only();

-- Тенантын аппын role эдгээрт хүрэхгүй. Дээрх толгойн тайлбарыг үз: 00029-ийн
-- default privileges нь эрхийг нь аль хэдийн олгосон байгаа тул энэ нь
-- болгоомжлол биш, засвар юм.
REVOKE ALL PRIVILEGES ON operator_accounts, operator_sessions, operator_audit
    FROM gerege_nexus_app;

-- +goose StatementBegin
DO $role$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_operator') THEN
        CREATE ROLE gerege_nexus_operator NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
    END IF;
END
$role$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO gerege_nexus_operator;

-- Операторын өөрийн хүснэгтүүд.
--
-- `operator_accounts`-д UPDATE хэрэгтэй: амжилтгүй оролдлогын тоолуур,
-- lockout, TOTP баталгаажуулалт, сүүлийн нэвтрэлтийн цаг. INSERT нь ЗӨВХӨН
-- CP-2-т (оператор нэмэх) хэрэгтэй болох тул одоохондоо олгогдоогүй — анхны
-- superadmin нь `cmd/operator-bootstrap`-аар, login role-оор үүснэ.
GRANT SELECT, UPDATE ON operator_accounts TO gerege_nexus_operator;
GRANT SELECT, INSERT, UPDATE ON operator_sessions TO gerege_nexus_operator;
GRANT SELECT, INSERT ON operator_audit TO gerege_nexus_operator;

-- Тенантын өгөгдөл: зөвхөн унших, зөвхөн нэрлэсэн хүснэгтүүд.
GRANT SELECT ON tenants, tenant_profiles, users, memberships, roles,
                membership_roles, apps, app_installations, sessions, audit_events
    TO gerege_nexus_operator;

-- `tenant_id` багана бүхий хүснэгтүүд 00029-ээр RLS-тэй болсон бөгөөд тэдгээрийн
-- цорын ганц бодлого нь `TO gerege_nexus_app` гэж бичигдсэн. RLS-ийн дүрмээр
-- өөр role-д хамаарах бодлого байхгүй бол мөр НЭГ Ч харагдахгүй — өөрөөр
-- хэлбэл GRANT SELECT дангаараа операторт юу ч өгөхгүй. Тиймээс хүснэгт тус
-- бүрд нь зөвхөн уншдаг бодлогыг тодорхой бичнэ. Жагсаалт нь давталтаар
-- олдсон биш, гараар бичигдсэн: цаашид `tenant_id`-тай шинэ хүснэгт нэмэгдэхэд
-- оператор түүнийг АВТОМАТААР харахгүй, харин түүнийг харах шаардлага гарсан
-- өдөр нь тухайн миграц нь өөрөө шийднэ.
-- +goose StatementBegin
DO $operator_read$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY[
        'tenant_profiles', 'memberships', 'roles',
        'app_installations', 'sessions', 'audit_events'
    ] LOOP
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
    FOREACH target IN ARRAY ARRAY[
        'tenant_profiles', 'memberships', 'roles',
        'app_installations', 'sessions', 'audit_events'
    ] LOOP
        EXECUTE format('DROP POLICY IF EXISTS operator_read ON public.%I', target);
    END LOOP;
END
$operator_read$;
-- +goose StatementEnd

REVOKE ALL PRIVILEGES ON tenants, tenant_profiles, users, memberships, roles,
                         membership_roles, apps, app_installations, sessions, audit_events
    FROM gerege_nexus_operator;
REVOKE USAGE ON SCHEMA public FROM gerege_nexus_operator;

DROP TRIGGER IF EXISTS operator_audit_append_only ON operator_audit;
DROP FUNCTION IF EXISTS operator_audit_is_append_only();

DROP TABLE IF EXISTS operator_audit;
DROP TABLE IF EXISTS operator_sessions;
DROP TABLE IF EXISTS operator_accounts;

-- 00029, 00044-тэй ижил шалтгаанаар: role нь кластерын өмч, хөрш өгөгдлийн сан
-- түүнд эрх олгосон хэвээр бол DROP унаж, буцаалтыг бүхэлд нь дагуулна.
-- +goose StatementBegin
DO $drop$
BEGIN
    DROP ROLE IF EXISTS gerege_nexus_operator;
EXCEPTION WHEN dependent_objects_still_exist OR insufficient_privilege THEN
    RAISE NOTICE 'left role gerege_nexus_operator in place: %', SQLERRM;
END
$drop$;
-- +goose StatementEnd
