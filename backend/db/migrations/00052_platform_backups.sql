-- CP-4 — нөөцлөлтийн бүртгэл.
--
-- Энэ платформ дээр өнөөдрийг хүртэл нөөцлөлтийн МЕХАНИЗМ БАЙГААГҮЙ: prod
-- compose дотор `pg_dump` гэсэн үг зөвхөн тайлбарт л байсан. Тиймээс CP-4-ийн
-- "сүүлийн нөөцлөлт хэзээ болов" гэсэн дэлгэц нь юу ч байхгүй зүйлийг харуулах
-- байсан — нөөцлөлт байхгүй гэдгийг ойлгуулах хамгийн муу арга.
--
-- Хоёр зүйл нэмэгдэв: `deploy/scripts/backup.sh` (энгийн pg_dump, cron-д
-- тавигдана) ба энэ хүснэгт. Скрипт нь ажиллаж дуусахдаа мөр бичнэ; консол
-- тэр мөрийг уншина. Файлын статус биш хүснэгт байгаа шалтгаан нь энгийн: API
-- контейнер хостын файлын системийг хардаггүй бөгөөд хардаг болгох нь
-- нөөцлөлтийн статусын төлөө хэтэрхий үнэтэй.
--
-- `restore_test` мөр нь гараар бүртгэгддэг: сэргээлтийг туршиж үзсэн огноо.
-- Туршаагүй нөөцлөлт бол нөөцлөлт биш гэдэг нь энэ хүснэгтэд хоёр төрөл байгаа
-- цорын ганц шалтгаан.
--
-- Платформын түвшний хүснэгт — tenant RLS хамаарахгүй.

-- +goose Up

CREATE TABLE IF NOT EXISTS platform_backups (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind        TEXT        NOT NULL CHECK (kind IN ('backup', 'restore_test')),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    -- Байтаар. NULL нь "мэдэгдэхгүй" — гараар бүртгэсэн сэргээлтийн туршилтад
    -- хэмжээ гэж байхгүй.
    size_bytes  BIGINT,
    ok          BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Скриптийн гаралт, эсвэл сэргээлтийг хэн, хаана туршсан тухай тэмдэглэл.
    detail      TEXT        NOT NULL DEFAULT '',
    -- Оператор гараар бүртгэсэн бол хэн бэ. Скрипт бичсэн мөрд NULL.
    recorded_by UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_backups_kind_time
    ON platform_backups (kind, started_at DESC);

-- Скрипт нь login role-оор (DATABASE_URL) бичнэ; консол уншиж, сэргээлтийн
-- туршилтыг бүртгэнэ.
GRANT SELECT, INSERT ON platform_backups TO gerege_nexus_operator;

-- +goose Down

REVOKE ALL PRIVILEGES ON platform_backups FROM gerege_nexus_operator;
DROP TABLE IF EXISTS platform_backups;
