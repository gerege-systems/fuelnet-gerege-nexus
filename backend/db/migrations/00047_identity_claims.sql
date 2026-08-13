-- Провайдерийн буцаасан бүхнийг хадгалах.
--
-- Одоогийн хоёр хүснэгт нь нэвтрэлтийн зам ашигладаг талбаруудыг л барьдаг:
-- sso талд и-мэйл ба нэр, eID талд регистр, овог, нэр. Бусад нь ирээд алга
-- болдог. Хүн өөрийн профайл дээрээ "надаас юу дамжуулсан бэ" гэдгийг харах
-- боломжгүй бөгөөд асуулт гарвал хариулах эх сурвалж ч байхгүй.
--
-- JSONB — багана биш. Google өнөөдөр нэг багц claim буцаадаг, маргааш өөр
-- нэгийг нэмнэ; eID мөн адил. Түүн болгонд багана нэмдэг бол схем нь
-- провайдерийн хувилбарын түүх болж хувирна. Уншихад л хэрэгтэй өгөгдөл тул
-- индекс шаардахгүй, JSONB нь агуулгаараа биш бүтнээрээ уншигдана.
--
-- Хоосон объект анхдагч утга: багана NOT NULL байх нь "claim байхгүй" ба
-- "claim хадгалагдаагүй" хоёрыг ялгах шаардлагыг арилгана — хоёул хоосон.

-- +goose Up
ALTER TABLE user_sso_identities
    ADD COLUMN IF NOT EXISTS claims JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE user_eid_identities
    ADD COLUMN IF NOT EXISTS claims JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE user_sso_identities DROP COLUMN IF EXISTS claims;
ALTER TABLE user_eid_identities DROP COLUMN IF EXISTS claims;
