-- Хаана буцаж очихыг нь бүртгэсэн клиент бүрийн "гарсны дараах" хаягууд.
--
-- Дискавери баримт нь end_session_endpoint-ийг эхнээсээ зарлаж байсан ч тэр
-- эндпойнт байгаагүй. Одоо байна (ssoprovider/logout.go), тэгэхээр RP-initiated
-- logout-ийн ганц аюултай параметр болох post_logout_redirect_uri-д хаана
-- байрлах вэ гэдэг асуулт гарч ирнэ: шалгалтгүй бол энэ нь провайдерийг
-- нээлттэй чиглүүлэгч болгоно. redirect_uris-тэй яг адилхан — тухайн клиентэд
-- бүртгэлтэй жагсаалттай яг таарч байж зөвшөөрөгдөнө.
--
-- Тусдаа багана, redirect_uris-ийг дахин ашиглаагүй нь санаатай: нэвтрэх
-- callback бол код хүлээж авдаг машин уншдаг зам, гарсны дараах хаяг бол хүн
-- харах хуудас. Хоёул нэг жагсаалтад байвал аль нэгийг нь нэмэхэд нөгөө нь
-- санамсаргүй өргөжинө.

-- +goose Up
ALTER TABLE oauth2_clients
    ADD COLUMN IF NOT EXISTS post_logout_redirect_uris TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE oauth2_clients
    DROP COLUMN IF EXISTS post_logout_redirect_uris;
