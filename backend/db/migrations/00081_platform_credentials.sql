-- Консолоос тохируулдаг гадаад системийн түлхүүрүүд.
--
-- `platform_settings` нь нууц утга ЗӨВХӨН зарчмаараа биш, кодоороо ч
-- хадгалдаггүй: Kind-д "secret" гэж байхгүй, нэр нь нууц үг мэт унших түлхүүрийг
-- Register нь panic-аар татгалздаг. Тэр хил зөв бөгөөд энд ч суларсангүй —
-- харин нууц утгад үнэхээр хэрэгтэй хамгаалалттайгаар тусдаа хүснэгтэд суулаа:
--
--   * утга нь баганад хүрэхээсээ өмнө AES-256-GCM-ээр битүүмжлэгдэнэ
--     (`INTEGRATION_ENCRYPTION_KEY`). Түлхүүргүй суулгац хадгалж ЧАДАХГҮЙ —
--     цэвэр текстээр хадгалахын оронд бичилт нь унана;
--   * `hint` нь сүүлийн дөрвөн тэмдэгт. Хоёр түлхүүрийг ялгах, сэлгэлт
--     хүрснийг харахад хангалттай, ашиглахад хангалтгүй;
--   * утгыг буцаадаг API байхгүй. Энэ хүснэгтээс гарах цорын ганц зам нь
--     процессийн өөрийн уншилт.
--
-- Түүхийн хүснэгт байхгүй нь санаатай. Тохиргооны түүх нь өмнөх утгыг хадгалдаг
-- бөгөөд нууц үгийн хувьд тэр нь "хуучин түлхүүрүүдийн жагсаалт" гэсэн үг.
-- Хэн, хэзээ өөрчилснийг `operator_audit` аль хэдийн бичдэг.
--
-- +goose Up

CREATE TABLE IF NOT EXISTS platform.platform_credentials (
    name       TEXT        PRIMARY KEY,
    ciphertext BYTEA       NOT NULL,
    hint       TEXT        NOT NULL DEFAULT '',
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Консолын role. Tenant role-д ямар ч GRANT байхгүй бөгөөд шаардлагагүй:
-- түлхүүрийг процесс өөрөө эзэмшигчийн холболтоор уншина.
GRANT SELECT, INSERT, UPDATE, DELETE ON platform.platform_credentials TO gerege_nexus_operator;

-- +goose Down

DROP TABLE IF EXISTS platform.platform_credentials;
