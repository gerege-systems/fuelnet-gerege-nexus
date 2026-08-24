-- Цөмийн query бүр tenant.* эсвэл platform.* schema-гаа одоо өөрөө нэрлэдэг.
-- Тиймээс public fallback нь алдааг нуухаа больж, харин буруу талын шинэ
-- query-г санамсаргүй ажиллуулах эрсдэл болж үлдсэн.
--
-- search_path бүрэн алга болоогүй. Өөр repository дахь module migration-ууд
-- хүснэгтээ угтваргүй CREATE хийдэг бөгөөд эхний `tenant` нь тэднийг зөв
-- plane-д буулгана. `platform` нь хоёр plane-д зориуд хэрэглэгддэг platform
-- хүснэгтүүдийг хуучин module query-д олгоно. Харин deployment-ийн goose
-- бүртгэлүүд public-д зориуд үлдсэн тул cmd/migrate болон appinstall тэднийг
-- `public.goose_db_version...` гэж бүрэн нэрлэдэг болсон.

-- +goose Up

-- +goose StatementBegin
DO $search_path$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET search_path = tenant, platform',
                   current_database());
END
$search_path$;
-- +goose StatementEnd

ALTER ROLE gerege_nexus_tenant SET search_path = tenant, platform;
ALTER ROLE gerege_nexus_operator SET search_path = platform, tenant;

-- Эдгээр SECURITY DEFINER функц өөрийн search_path-тай тул database default
-- өөрчлөгдөхөд дагахгүй. Доторх бүх хүснэгт tenant schema-д байгаа.
ALTER FUNCTION public.create_tenant_profile() SET search_path = tenant, platform;
ALTER FUNCTION public.resolve_device_enrollment(TEXT) SET search_path = tenant, platform;
ALTER FUNCTION public.authenticate_device(TEXT) SET search_path = tenant, platform;

-- +goose Down

ALTER FUNCTION public.create_tenant_profile() SET search_path = tenant, platform, public;
ALTER FUNCTION public.resolve_device_enrollment(TEXT) SET search_path = tenant, platform, public;
ALTER FUNCTION public.authenticate_device(TEXT) SET search_path = tenant, platform, public;

ALTER ROLE gerege_nexus_tenant SET search_path = tenant, platform, public;
ALTER ROLE gerege_nexus_operator SET search_path = platform, tenant, public;

-- +goose StatementBegin
DO $search_path$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET search_path = tenant, platform, public',
                   current_database());
END
$search_path$;
-- +goose StatementEnd
