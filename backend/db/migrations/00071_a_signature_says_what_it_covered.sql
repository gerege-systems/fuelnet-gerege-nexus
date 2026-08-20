-- Гарын үсэг юуг хамарснаа хэлдэг болов.
--
-- ADR 0003-ын дагуу нэг апп гурван хэлбэрийг бүртгэдэг болно: PDF дээрх
-- PAdES, өөр төрлийн файлын digest дээрх detached, хавсралтгүй баримтын
-- approval. Тэдгээр нь өөр өөр зүйлийг нотолдог тул мөр өөрөө аль нь
-- болохыг хэлэх ёстой — уншигч хоёрыг ялгаж чадахгүй бол ялгаа нь байхгүйтэй
-- адил.
--
-- Хоёулаа NULL байж болно: өмнө бичигдсэн мөрүүд юу болохыг ADR 0002
-- тайлбарласан бөгөөд тэдгээрийг дахин тайлбарлахгүй. NULL format нь
-- «энэ мөр асуулт асуухаас өмнө бичигдсэн» гэсэн үг, түүнийг API нь
-- `approval` гэж уншина — тэр нь тэдгээрийн үнэн утга.

-- +goose Up
ALTER TABLE document_signatures
    ADD COLUMN IF NOT EXISTS format         VARCHAR(16),
    -- Юуг хамарсан бэ: файлын SHA-256 (hex, жижиг үсгээр). Хэлбэр нь
    -- approval бол NULL — хамарсан зүйл байхгүй.
    ADD COLUMN IF NOT EXISTS covered_digest CHAR(64);

COMMENT ON COLUMN document_signatures.format IS
    'pades | detached | approval — ADR 0003. NULL нь асуулт асуухаас өмнөх мөр, approval гэж уншигдана.';
COMMENT ON COLUMN document_signatures.covered_digest IS
    'Гарын үсэг хамарсан файлын SHA-256. approval дээр NULL: хамарсан зүйл байхгүй.';

-- Ёслол эхлэхэд юуг гарын үсэг зуруулахаар илгээснийг хадгална. Poll
-- буцаж ирэхэд үйлчилгээний баталгаажуулсан digest үүнтэй таарах ёстой —
-- таарахгүй бол өөр зүйл зурагдсан гэсэн үг.
ALTER TABLE document_eid_sign_sessions
    ADD COLUMN IF NOT EXISTS requested_digest CHAR(64),
    ADD COLUMN IF NOT EXISTS format           VARCHAR(16);

-- +goose Down
ALTER TABLE document_signatures
    DROP COLUMN IF EXISTS format,
    DROP COLUMN IF EXISTS covered_digest;
ALTER TABLE document_eid_sign_sessions
    DROP COLUMN IF EXISTS requested_digest,
    DROP COLUMN IF EXISTS format;
