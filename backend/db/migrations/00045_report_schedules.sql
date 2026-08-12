-- Товлосон тайлан: "сар бүрийн 1-нд энэ тайланг эдгээр хаяг руу илгээ".
--
-- Хуваарь нь cron-ийн 5 талбарт бичлэг (минут цаг өдөр сар гараг). Түүнийг
-- сонгосон шалтгаан: оператор аль хэдийн мэддэг, "сарын эхний өдөр"-ийг ямар
-- ч тусгай талбаргүйгээр илэрхийлдэг, мөн энэ платформ дээр илэрхийлэл нь
-- өөрөө уншигдахуйц зүйл болж үлдэнэ. Шинэ процесс НЭМЭГДЭХГҮЙ — минут тутам
-- ажилладаг backend доторх goroutine хуваарийг шалгана.
--
-- last_run_at нь давхар илгээлтээс хамгаалах цорын ганц зүйл. Хэд хэдэн
-- replica ажиллаж байгаа үед хоёулаа нэг мөчид болзсон тайланг олж, хоёулаа
-- нэг хүнд илгээж болно. Тиймээс шинэчлэлт нь advisory lock доор явна
-- (scheduler.go-г үз): нэг replica түгжээг барьж, last_run_at-ыг бичиж,
-- дараа нь илгээнэ.
--
-- recipients нь text[] — и-мэйл хаягуудын жагсаалт. Тусдаа хүснэгт болгох
-- нь ямар ч давуу тал өгөхгүй: хаягууд нь зөвхөн энэ мөрийн хүрээнд
-- утгатай, тусад нь хайгддаггүй, тусад нь өөрчлөгддөггүй.

-- +goose Up
CREATE TABLE IF NOT EXISTS report_schedules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_key  TEXT        NOT NULL,
    -- Хэрэглэгчийн өгсөн нэр. Тайлангийн нэр биш: нэг тайланг өөр
    -- параметртэйгээр хоёр хуваариар илгээж болно ("Сарын орлого",
    -- "Улирлын орлого") бөгөөд жагсаалтад тэднийг ялгах хэрэгтэй.
    name        TEXT        NOT NULL DEFAULT '',
    params      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- Cron-ийн 5 талбар. Хэлбэрийг backend шалгана — буруу бичлэгтэй мөрийг
    -- хадгалахаас татгалзана, эс бөгөөс хэзээ ч ажиллахгүй хуваарь чимээгүй
    -- сууна.
    cron        TEXT        NOT NULL,
    format      TEXT        NOT NULL DEFAULT 'xlsx',
    recipients  TEXT[]      NOT NULL DEFAULT '{}',
    active      BOOLEAN     NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    -- Сүүлийн гүйлт хэрхэн болсон. Амжилтгүй хуваарь бол хэн ч анзаардаггүй
    -- зүйл: тайлан ирэхгүй байгааг хүлээж авагч л мэднэ, тэр ч гэсэн
    -- хэдэн долоо хоногийн дараа.
    last_status TEXT        NOT NULL DEFAULT '',
    last_error  TEXT        NOT NULL DEFAULT '',
    created_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT report_schedules_format_check CHECK (format IN ('xlsx', 'csv'))
);

CREATE INDEX IF NOT EXISTS idx_report_schedules_tenant
    ON report_schedules (tenant_id, report_key);

-- Scheduler-ийн гол query: идэвхтэй хуваариудыг тенант дамнаж уншина
-- (housekeeping зам, login role, RLS-ээс гадна).
CREATE INDEX IF NOT EXISTS idx_report_schedules_active
    ON report_schedules (active, last_run_at) WHERE active;

GRANT SELECT, INSERT, UPDATE, DELETE ON report_schedules TO gerege_nexus_app;

ALTER TABLE report_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_schedules FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON report_schedules;
CREATE POLICY tenant_isolation ON report_schedules TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- +goose Down
DROP TABLE IF EXISTS report_schedules;
