-- CP-5 — хэрэглээний хэмжилт (metering).
--
-- Нэг хүснэгт, өдрөөр нэгтгэсэн. Хэмжилтийн систем барихдаа хамгийн түгээмэл
-- алдаа нь үйл явдал бүрийг бичих явдал: хүсэлт бүрд нэг мөр гэдэг нь өдөрт
-- хэдэн сая мөр, хэдэн сарын дараа хэн ч query хийж чадахгүй хүснэгт болно.
-- Энд бичигдэх зүйл нь өдрийн ЭЦЭСТ тоологдсон дүн — тенант тус бүрд,
-- хэмжигдэхүүн тус бүрд, өдөрт нэг мөр.
--
-- Хаанаас тоолох вэ гэдэг нь илүү сонирхолтой асуулт. Prometheus-оос БИШ:
-- Үе шат 1-д хэмжүүрт тенантын label хэзээ ч оруулахгүй гэж шийдсэн бөгөөд
-- тэр шийдвэр хэвээр. Тиймээс тоолол нь өгөгдлийн сангаас — session,
-- audit_events, файлын хэмжээ — явагдана. Энэ нь илүү удаан ч, илүү үнэн:
-- хэмжүүр нь хүсэлтийг тоолдог, эдгээр нь ҮЙЛДЛИЙГ тоолдог бөгөөд
-- төлбөрийн суурь болох ёстой зүйл нь сүүлийнх юм.
--
-- `metric` нь чөлөөт текст биш: Go талын хаалттай жагсаалт
-- (internal/platform/metering). Бүртгэгдээгүй нэртэй мөр бичигдэхгүй.

-- +goose Up

CREATE TABLE IF NOT EXISTS usage_events (
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Тухайн өдрийн огноо (UTC биш, суулгацын цагийн бүсээр — тайлан уншиж
    -- буй хүн "өчигдөр" гэж юу гэсэн үг болохыг мэддэг байх ёстой).
    day         DATE        NOT NULL,
    metric      TEXT        NOT NULL,
    value       BIGINT      NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, day, metric)
);

-- "Энэ байгууллагын энэ сарын хэрэглээ" — квотын шалгалт ба график хоёулаа
-- ижил query.
CREATE INDEX IF NOT EXISTS idx_usage_events_metric_day
    ON usage_events (metric, day DESC);

-- `tenant_id`-тай хүснэгт тул 00029-ийн журмаар RLS.
--
-- Тенант нь ӨӨРИЙНХӨӨ хэрэглээг харна: "би хязгаартаа хэр ойрхон байна вэ"
-- гэдэг нь тэдний асуулт бөгөөд хариулт нь тэдэнд байх ёстой. Бичих эрх
-- байхгүй — тоолол нь платформын ажил.
ALTER TABLE usage_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_events FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON usage_events;
CREATE POLICY tenant_isolation ON usage_events FOR SELECT TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
GRANT SELECT ON usage_events TO gerege_nexus_app;

-- Консол уншина. Бичих нь платформын зам (тоолох ажил) тул INSERT энд
-- байхгүй — консол хэрэглээг ЗАСАЖ чадах ёсгүй, тэр нь төлбөрийн маргаанд
-- хамгийн түрүүнд асуугдах зүйл.
GRANT SELECT ON usage_events TO gerege_nexus_operator;
DROP POLICY IF EXISTS operator_read ON usage_events;
CREATE POLICY operator_read ON usage_events FOR SELECT TO gerege_nexus_operator
    USING (true);

-- +goose Down

DROP POLICY IF EXISTS operator_read ON usage_events;
REVOKE ALL PRIVILEGES ON usage_events FROM gerege_nexus_operator, gerege_nexus_app;
DROP TABLE IF EXISTS usage_events;
