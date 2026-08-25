-- Иргэний geID — Gerege экосистемийн тогтвортой дугаар.
--
-- eID-гийн session хариу COMPLETE+OK үед `person.geID` буцаадаг болсон
-- (core.gerege.mn-ий `users.id`). Түүнийг хадгалахын өмнө энэ платформ eID-ээр
-- анх нэвтэрсэн иргэнд `eid+<32 hex>@identity.invalid` гэсэн зохиомол хаяг
-- үүсгэдэг байв: нэвтрэлтэд ажилладаг ч хүн харахад утгагүй, доод урсгалын RP-д
-- дамжихдаа бүр ч утгагүй.
--
-- Одоо geID мэдэгдэж байвал хаяг нь `<geID>@gemail.com` болно. Хуучин
-- зохиомол хаягтай бүртгэлүүд дараагийн нэвтрэлтдээ шинэчлэгдэнэ — тэр хаягаар
-- хэн ч нэвтэрч байгаагүй тул (нууц үг нь санамсаргүй) алдах зүйл алга.
--
-- Давхардлыг хүснэгт өөрөө барина: нэг geID = нэг хүн. Хоёр бүртгэл нэг дугаар
-- зарлавал тэр хүний гарын үсгийн түүх чимээгүй хоёр хуваагдана — user_eid_
-- identities дээрх person_etsi-ийн unique нь яг үүнээс сэргийлдэг.
--
-- NULL зөвшөөрнө: geID нь eID-гийн заавал байх талбар БИШ (Core тохируулаагүй
-- эсвэл backfill хийгээгүй иргэн), мөн энэ платформын нууц үгтэй бүртгэлүүдэд
-- огт хамаагүй.
--
-- +goose Up

ALTER TABLE platform.users ADD COLUMN IF NOT EXISTS ge_id BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_ge_id
    ON platform.users (ge_id) WHERE ge_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS platform.idx_users_ge_id;
ALTER TABLE platform.users DROP COLUMN IF EXISTS ge_id;
