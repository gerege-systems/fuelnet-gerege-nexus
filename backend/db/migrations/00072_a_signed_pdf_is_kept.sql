-- Гарын үсэгтэй PDF нь хадгалагдана.
--
-- ADR 0003: PDF нь PAdES замаар зурагдвал үр дүн нь **шинэ файл** —
-- гарын үсгээ дотроо агуулсан, энэ платформгүйгээр шалгагддаг баримт. Тэр
-- нь эх хувийг орлохгүй: эх хувь нь юуг гарын үсэг зурсныг хэлдэг тул
-- хөлдсөн хэвээр, зурагдсан хувь нь гарын үсэг нэмэгдэх бүрд ургана
-- (PAdES нь гарын үсгийг сольдоггүй, нэмдэг).
--
-- Тиймээс хоёр багана нэг мөрөнд: эх хувь ба хамгийн сүүлийн зурагдсан
-- хувь. Хүн татахад зурагдсаныг нь авна — тэр бол хүссэн зүйл нь.

-- +goose Up
ALTER TABLE document_files
    ADD COLUMN IF NOT EXISTS signed_content BYTEA,
    ADD COLUMN IF NOT EXISTS signed_at      TIMESTAMPTZ;

COMMENT ON COLUMN document_files.signed_content IS
    'PAdES-ээр зурагдсан хамгийн сүүлийн хувь. NULL бол энэ баримт detached эсвэл approval-аар зурагдсан — ADR 0003.';

-- +goose Down
ALTER TABLE document_files
    DROP COLUMN IF EXISTS signed_content,
    DROP COLUMN IF EXISTS signed_at;
