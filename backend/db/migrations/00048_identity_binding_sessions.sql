-- Google-ээр анх удаа ирсэн хүнийг eID-ээр баталгаажуулах хүртэлх завсрын төлөв.
--
-- Өмнө нь энэ хүнийг татгалздаг байсан: Google түүнийг баталгаажуулсан ч энд
-- бүртгэл олдохгүй бол мухардал. Одоо Google-ийн хэлсэн бүхнийг түр хадгалж,
-- ямар мэдээлэл хаанаас хаашаа дамжихыг үзүүлж зөвшөөрөл авч, дараа нь eID-ээр
-- хэн болохыг нь батална. Хоёулаа болсны дараа л бүртгэл үүснэ.
--
-- Cookie биш хүснэгт — энэ нэг л удаагийн урсгалын бусад cookie-гаас ялгаатай.
-- Дотор нь баталгаажсан claim-ууд байх ба тэдгээр нь хөтчид очих ёсгүй: хүнд
-- харуулах нь өөр, түүнд эзэмшүүлэх нь өөр. Мөн зөвшөөрөл өгсөн эсэхийг сервер
-- талд шийдэх ёстой — cookie доторх "зөвшөөрсөн" гэсэн тэмдэглэгээ бол
-- зөвшөөрөл биш.
--
-- token_hash — SHA-256. Мөр алдагдсан ч холбоосыг нь дахин тоглуулах боломжгүй,
-- session-ийн токенуудтай ижил зарчим.
--
-- tenant_id байхгүй тул RLS ч байхгүй: энэ хүн ямар байгууллагад орохыг мэдэх
-- үедээ энэ мөр аль хэдийн устсан байна.

-- +goose Up
CREATE TABLE IF NOT EXISTS identity_binding_sessions (
    token_hash   CHAR(64)    PRIMARY KEY,
    issuer       TEXT        NOT NULL,
    subject      TEXT        NOT NULL,
    email        TEXT,
    name         TEXT,
    claims       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- Зөвшөөрөл өгсөн мөч. NULL бол eID рүү шилжих зам хаалттай.
    consented_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Хугацаа нь дууссаныг цэвэрлэхэд.
CREATE INDEX IF NOT EXISTS idx_identity_binding_expiry
    ON identity_binding_sessions (expires_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON identity_binding_sessions TO gerege_nexus_app;

-- +goose Down
DROP TABLE IF EXISTS identity_binding_sessions;
