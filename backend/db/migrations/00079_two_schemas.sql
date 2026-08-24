-- Хоёр урсгалыг өгөгдлийн сан өөрөө барина.
--
-- Энэ хүртэл ялгаа нь кодод байсан: тенантын урсгал `gerege_nexus_tenant`-аар,
-- платформынх `gerege_nexus_operator`-оор ажилладаг (internal/kernel/dbguard).
-- Гэвч 66 хүснэгт бүгд нэг `public` schema-д байсан тул тенантын handler
-- `operator_audit`-аас уншихыг оролдвол зогсоох зүйл нь review хийсэн хүн л
-- байлаа.
--
-- Schema бол нэр + эрх. `platform` доторх хилийн таван хүснэгтийг уншихад
-- schema-ийн USAGE заавал хэрэгтэй ч тэр нь хүснэгт өөрийг нь нээдэггүй.
-- Тиймээс хил нь хоёр давхар: schema-ийн USAGE нэрийг олно, хүснэгт тус бүрийн
-- grant зөвхөн зөвшөөрсөн мөрийг нээнэ. `platform.operator_audit`-д table
-- privilege байхгүй тул тенантын handler түүнийг уншихыг DB өөрөө зогсооно.
--
-- Хуваарилалт нь db/migrations/ownership_test.go-ийн `plane` талбар — 26
-- платформынх, 40 тенантынх. Дүрэм нь docs/TWO_PLANES_PROPOSAL.md §2.1: мөр
-- deployment-д ганц удаа оршдог бол платформынх, тенант бүрд өөр өөрөө оршдог
-- бол тенантынх.
--
-- БИЗНЕС QUERY-Н SQL НЭГ Ч МӨР ӨӨРЧЛӨГДӨӨГҮЙ. `search_path` үүнийг хийнэ:
-- `SELECT … FROM sessions` хэвээр ажиллана. Зөвхөн schema-г өөрийг нь шалгадаг
-- metadata/export query шинэ байршлыг нэрлэнэ. Бусад query-г бүрэн нэрлэх нь
-- Үе E.
--
--
-- `search_path`-ийг ROLE дээр биш DATABASE дээр тавьсан шалтгаан.
--
-- §2.4 нь `ALTER ROLE gerege_nexus_tenant SET search_path = …` гэж бичсэн.
-- Тэр нь энд ажиллахгүй: PostgreSQL role-ын тохиргоог **session эхлэхэд**,
-- login хийсэн role-ынхыг л хэрэглэдэг. dbguard бол login хийдэггүй — тэр нь
-- аль хэдийн нээгдсэн холболт дээр `set_config('role', …)` дууддаг, өөрөөр
-- хэлбэл SET ROLE. SET ROLE нь зорилтот role-ын тохиргоог ХЭРЭГЛЭХГҮЙ. Тиймээс
-- ALTER ROLE-оор тавьсан search_path чимээгүй нөлөөгүй үлдэж, тэр өдөр бүх
-- query «relation does not exist» гэж унана.
--
-- ALTER DATABASE нь холбогдсон бүх session-д, ямар role байхаас үл хамааран
-- үйлчилнэ — тэр дундаа login role (dbguard-ийн «платформын зам», SET ROLE
-- NONE) -д ч мөн. Хоёр role дээр нь мөн тавьсан нь тэдгээр хэзээ нэгэн цагт
-- login хийвэл зөв байхын тулд, өнөөдөр тэр нь идэвхгүй.
--
-- Дараалал нь `tenant, platform, public` — нэг л дараалал. §2.4 операторт
-- эсрэг дарааллыг санал болгосон ч 66 нэрийн хооронд давхардал байхгүй тул
-- дараалал ямар ч ялгаа гаргахгүй; нэг session дотор role солигддог тул
-- role бүрд өөр дараалал өгөх боломж ч байхгүй.
--
--
-- goose-ийн бүртгэлийн хүснэгтүүд `public`-д үлдэнэ.
--
-- `goose_db_version` бол миграц ажиллуулагчийн дэвтэр, аль ч урсгалын өгөгдөл
-- биш. Модулийн `goose_db_version_<slug>` мөн адил — internal/tenant/appinstall
-- түүнийг одооноос `public.` угтвартай нэрлэнэ, тэгэхгүй бол шинэ модулийн
-- дэвтэр `tenant`-д, хуучных нь `public`-д унаж хоёр тийш салах байлаа.
-- Модулийн өөрийн ХҮСНЭГТҮҮД харин `tenant`-д унана — `search_path`-ийн эхний
-- элемент тэр — бөгөөд яг тэр нь зөв: модуль тенантын өмнөөс ажилладаг.

-- +goose Up

CREATE SCHEMA IF NOT EXISTS tenant;
CREATE SCHEMA IF NOT EXISTS platform;

-- Хүснэгтүүд. `SET SCHEMA` нь index, constraint, эзэмшдэг sequence, RLS
-- бодлогуудыг бүгдийг дагуулж авч явна — тиймээс 00029-ийн `tenant_isolation`
-- ба 00049-ийн `operator_read` бодлогууд эвдрэхгүй.
-- +goose StatementBegin
DO $move$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY[
        'access_change_events', 'ai_knowledge', 'ai_prompts', 'app_installations',
        'audit_events', 'device_enrollment_codes', 'device_telemetry', 'devices',
        'email_verifications', 'esign_batch_items', 'esign_batches', 'esign_documents',
        'esign_settings', 'esign_sign_sessions', 'esign_signature_logs', 'installation_events',
        'integration_deliveries', 'integration_oauth_states', 'integrations', 'membership_roles',
        'memberships', 'oauth2_access_tokens', 'oauth2_authorization_codes', 'oauth2_clients',
        'oauth2_consents', 'oauth2_tokens', 'push_tokens', 'report_grants',
        'report_schedules', 'role_permissions', 'roles', 'sessions',
        'staff_pin_credentials', 'tenant_profiles', 'urtuu_deliveries', 'urtuu_inbox',
        'urtuu_outbox', 'urtuu_peer_codes', 'urtuu_peers', 'urtuu_request_codes'
    ] LOOP
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = target) THEN
            EXECUTE format('ALTER TABLE public.%I SET SCHEMA tenant', target);
        END IF;
    END LOOP;

    FOREACH target IN ARRAY ARRAY[
        'announcements', 'app_dependencies', 'app_versions', 'apps',
        'credential_grants', 'eid_sign_state', 'feature_flag_overrides', 'feature_flags',
        'identity_binding_sessions', 'oauth2_signing_keys', 'operator_accounts', 'operator_audit',
        'operator_impersonations', 'operator_sessions', 'pending_approvals', 'permissions',
        'platform_backups', 'platform_settings', 'platform_settings_history', 'store_app_versions',
        'tenant_quotas', 'tenants', 'usage_events', 'user_eid_identities',
        'user_sso_identities', 'users'
    ] LOOP
        IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = target) THEN
            EXECUTE format('ALTER TABLE public.%I SET SCHEMA platform', target);
        END IF;
    END LOOP;
END
$move$;
-- +goose StatementEnd

-- Модулиудын хүснэгт мөн тенантынх: тэдгээр нь тенантын өмнөөс ажилладаг
-- модулийн мөрүүд. Дээрх 66-г нүүлгэсний дараа public-д үлдсэн base table бүр
-- модулийнх эсвэл goose-ийн дэвтэр байна. Нэрээр нь урьдчилан мэдэхгүй, мөн
-- parent хүснэгтээрээ tenant-д холбогддог child table заавал tenant_id-тай
-- байдаггүй тул баганаар шүүхгүй. Зөвхөн `goose_db_version%` энд үлдэнэ: тэр
-- нь өгөгдөл биш deployment-ийн бүртгэл.
-- +goose StatementBegin
DO $modules$
DECLARE target RECORD;
BEGIN
    FOR target IN
        SELECT tablename AS table_name
          FROM pg_tables
         WHERE schemaname = 'public'
           AND tablename NOT LIKE 'goose_db_version%'
    LOOP
        EXECUTE format('ALTER TABLE public.%I SET SCHEMA tenant', target.table_name);
    END LOOP;
END
$modules$;
-- +goose StatementEnd

-- SECURITY DEFINER функц өөрийн тогтоосон search_path-тай бол database-ийн
-- шинэ default-ыг авахгүй. Эдгээр гурав `public`-д өөрсдөө үлдэнэ, харин
-- дотроо уншиж/бичдэг хүснэгтүүд нь `tenant` руу нүүсэн тул замыг нь хамт
-- шинэчилнэ. Үүнийг орхивол шинэ tenant үүсгэх trigger хамгийн эхний INSERT
-- дээр `relation tenant_profiles does not exist` гэж унана.
ALTER FUNCTION public.create_tenant_profile() SET search_path = tenant, platform, public;
ALTER FUNCTION public.resolve_device_enrollment(TEXT) SET search_path = tenant, platform, public;
ALTER FUNCTION public.authenticate_device(TEXT) SET search_path = tenant, platform, public;

-- Role-ын нэр. «app» гурван зүйл заана — RLS-ийн role, суулгадаг модуль, ба
-- бинарь (docs/TWO_PLANES_PROPOSAL.md §1.9). «tenant» нэгийг заана, мөн
-- `gerege_nexus_operator`-той хосолж уншигдана.
--
-- Бодлого дотор бичигдсэн role-ын нэр дагаж шинэчлэгдэнэ: pg_policy нь role-ыг
-- OID-ээр хадгалдаг тул RENAME нь бодлогуудад ил харагдахгүй. Үүнийг
-- db/migrations/schema_split_test.go шалгана — бодлогуудыг дахин үүсгэх
-- шаардлагагүй гэдгийг батлах нь тэдгээрийг дахин бичихээс хямд.
-- +goose StatementBegin
DO $rename$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_app')
       AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_tenant') THEN
        ALTER ROLE gerege_nexus_app RENAME TO gerege_nexus_tenant;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_tenant') THEN
        CREATE ROLE gerege_nexus_tenant NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
    END IF;
END
$rename$;
-- +goose StatementEnd

-- Хилийг барих мөрүүд.
GRANT USAGE ON SCHEMA tenant TO gerege_nexus_tenant;
REVOKE USAGE ON SCHEMA platform FROM gerege_nexus_tenant;
GRANT USAGE ON SCHEMA platform TO gerege_nexus_operator;
-- 00049-ийн жагсаалт хэвээр: оператор тенантын нэрлэсэн хүснэгтүүдийг уншина.
GRANT USAGE ON SCHEMA tenant TO gerege_nexus_operator;

-- Хүснэгтүүдийн эрх нь `public`-д өгөгдсөн байсан бөгөөд хүснэгттэйгээ хамт
-- нүүсэн — GRANT нь хүснэгт дээр байдаг, schema дээр биш. Schema дээрх USAGE
-- дээрх хоёр мөрөөр тавигдав. Sequence-үүд эзэн хүснэгтээ дагаж нүүсэн.
--
-- Шинээр үүсэх хүснэгтүүд: 00029 нь `ALTER DEFAULT PRIVILEGES IN SCHEMA public`
-- гэж тавьсан бөгөөд шинэ хүснэгтүүд `tenant`-д үүсэх тул тэр анхдагч эрх
-- хүрэхээ болино. Модуль өөрийн хүснэгтээ үүсгээд тэнд нь хүрч чадахгүй болох
-- нь энэ мөрүүдгүйгээр гарах алдаа.
ALTER DEFAULT PRIVILEGES IN SCHEMA tenant
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gerege_nexus_tenant;
ALTER DEFAULT PRIVILEGES IN SCHEMA tenant
    GRANT USAGE, SELECT ON SEQUENCES TO gerege_nexus_tenant;

-- Хилийн таван ширээ: платформ бичнэ, тенант уншина (§2.5). Энэ migration
-- platform schema дээр шинээр зөвхөн эдгээр таван table grant нэмнэ. Өмнөх
-- migration-уудын grant хүснэгттэйгээ хамт нүүсэн: tenant урсгалын identity,
-- tenant registry, каталогийн одоогийн query-нууд тэдгээрийг хэрэглэсээр байна.
GRANT SELECT ON platform.announcements, platform.feature_flag_overrides,
                platform.operator_impersonations, platform.tenant_quotas,
                platform.usage_events
    TO gerege_nexus_tenant;

-- Тэдгээрийг уншихын тулд schema-д USAGE хэрэгтэй. USAGE нь «энэ schema дотор
-- нэр хайж болно» гэсэн үг л, өөрөө юу ч нээхгүй: доторх хүснэгт бүр өөрийн
-- grant-аар хамгаалагдсан хэвээр. Ялангуяа 00049-өөр эрхийг нь бүрэн цуцалсан
-- operator_audit-ыг tenant role нэрлэсэн ч уншиж чадахгүй.
GRANT USAGE ON SCHEMA platform TO gerege_nexus_tenant;

-- +goose StatementBegin
DO $search_path$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET search_path = tenant, platform, public',
                   current_database());
END
$search_path$;
-- +goose StatementEnd

ALTER ROLE gerege_nexus_tenant SET search_path = tenant, platform, public;
ALTER ROLE gerege_nexus_operator SET search_path = platform, tenant, public;

-- +goose Down

ALTER FUNCTION public.create_tenant_profile() SET search_path = public;
ALTER FUNCTION public.resolve_device_enrollment(TEXT) SET search_path = public;
ALTER FUNCTION public.authenticate_device(TEXT) SET search_path = public;

ALTER ROLE gerege_nexus_operator RESET search_path;
ALTER ROLE gerege_nexus_tenant RESET search_path;

-- +goose StatementBegin
DO $search_path$
BEGIN
    EXECUTE format('ALTER DATABASE %I RESET search_path', current_database());
END
$search_path$;
-- +goose StatementEnd

REVOKE SELECT ON platform.announcements, platform.feature_flag_overrides,
                 platform.operator_impersonations, platform.tenant_quotas,
                 platform.usage_events
    FROM gerege_nexus_tenant;

ALTER DEFAULT PRIVILEGES IN SCHEMA tenant
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM gerege_nexus_tenant;
ALTER DEFAULT PRIVILEGES IN SCHEMA tenant
    REVOKE USAGE, SELECT ON SEQUENCES FROM gerege_nexus_tenant;

-- +goose StatementBegin
DO $back$
DECLARE target RECORD;
BEGIN
    FOR target IN
        SELECT schemaname, tablename
          FROM pg_tables
         WHERE schemaname IN ('tenant', 'platform')
    LOOP
        EXECUTE format('ALTER TABLE %I.%I SET SCHEMA public',
                       target.schemaname, target.tablename);
    END LOOP;
END
$back$;
-- +goose StatementEnd

DROP SCHEMA IF EXISTS tenant;
DROP SCHEMA IF EXISTS platform;

-- +goose StatementBegin
DO $rename$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_tenant')
       AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_app') THEN
        ALTER ROLE gerege_nexus_tenant RENAME TO gerege_nexus_app;
    END IF;
END
$rename$;
-- +goose StatementEnd
