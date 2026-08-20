-- Баримт нь гарын үсэг зурагдах зүйлээ өөртөө авч явдаг болов.
--
-- `document_records` нь өнөөг хүртэл агуулгагүй байсан — гарчиг, төрөл,
-- төлөв. Тиймээс энэ аппын «гарын үсэг» нь агуулгад холбогдоогүй
-- зөвшөөрөл байв (ADR 0002). Энэ хүснэгт нь тэр дутууг нөхнө.
--
-- Нэг баримт нэг файл: гарын үсэг зурагдах зүйл нэг байх ёстой, эс бөгөөс
-- «юуг гарын үсэг зурсан бэ» гэдэг асуулт хариулт олонтой болно.
--
-- Байтууд Postgres-д, `esign_documents.original_pdf`-ийн сонголттой ижил.
-- Объект хадгалалт руу шилжих нь тусдаа шийдвэр бөгөөд хоёуланг нь нэг
-- дор шилжүүлэх ёстой — ADR 0003-ыг үз.

-- +goose Up
CREATE TABLE IF NOT EXISTS document_files (
    -- Нэг баримт нэг файл: PRIMARY KEY нь document_id өөрөө.
    document_id  UUID        PRIMARY KEY REFERENCES document_records(id) ON DELETE CASCADE,
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    file_name    VARCHAR(255) NOT NULL,
    -- Байтуудаас тогтоогдсон төрөл, зарлагдсанаас нь биш. `.pdf` дагавартай
    -- боловч `%PDF-`-ээр эхлэхгүй файл нь PDF биш.
    content_type VARCHAR(128) NOT NULL,
    size_bytes   BIGINT      NOT NULL,
    -- SHA-256, hex, жижиг үсгээр. Detached гарын үсэг үүнийг хамарна, мөн
    -- хадгалагдсан файл нь зурагдсан файл мөн эсэхийг энэ баталгаажуулна.
    sha256       CHAR(64)    NOT NULL,
    content      BYTEA       NOT NULL,
    uploaded_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT document_files_size_check CHECK (size_bytes > 0 AND size_bytes <= 26214400)
);

CREATE INDEX IF NOT EXISTS idx_document_files_tenant ON document_files (tenant_id);

-- 00029-ийн үүсгэсэнтэй яг ижил тусгаарлалт. Тэр миграц нь `tenant_id`-тай
-- бүх хүснэгтэд бодлого тавьсан бөгөөд түүнээс хойш үүссэн хүснэгт өөрөө
-- тавих ёстой — тавихгүй бол өгөгдөл нь мөрийн түвшинд хамгаалалтгүй
-- үлдэнэ, харин бусад хүснэгттэй ижил харагдана.
GRANT SELECT, INSERT, UPDATE, DELETE ON document_files TO gerege_nexus_app;

ALTER TABLE document_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_files FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON document_files;
CREATE POLICY tenant_isolation ON document_files TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

COMMENT ON TABLE document_files IS
    'Баримтын хавсралт — гарын үсэг хамаарах зүйл. Нэг баримт нэг файл; зурагдсаны дараа солигдохгүй (ADR 0003).';

-- +goose Down
DROP TABLE IF EXISTS document_files;
