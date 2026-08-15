-- «Өртөө» хоёр шугам болов.
--
-- Эхний хэрэгжилт нь нэг л урсгалыг мэддэг байсан: дээрээс доошоо ирсэн
-- АЛБАН ДААЛГАВАР, биелэлт нь дээшээ буцна. Тэр урсгал хэвээрээ. Гэхдээ
-- эх шаардлага нь өөр байсан бөгөөд одоо хоёулаа зэрэгцэн явна:
--
--   ҮЙЛЧИЛГЭЭ ('service')
--     Иргэн эсвэл байгууллага үндсэн платформ дээр төрийн үйлчилгээ авах
--     ХҮСЭЛТ гаргана. Хүсэлт нь биелүүлэх ёстой суулгац руу доошоо явж,
--     ХАРИУ нь заавал буцаж ирнэ. Хүсэгч тал платформын гадна байгаа тул
--     хариуг нь хэн нэгэн буцааж хүргэх ёстой — хариугүй хаагдсан хүсэлт
--     гэдэг нь иргэний асуултыг зүгээр л алга болгосон хэрэг.
--
--   ДААЛГАВАР ('assignment')
--     Дээд байгууллага доод байгууллагадаа ажил өгнө. Хүсэгч гэж байхгүй —
--     эхлүүлсэн байгууллага өөрөө үр дүнг нь хүлээж авна.
--
-- Хоёр шугамын ялгаа нь дэлгэцийн шүүлтүүр биш, ӨӨР АМЛАЛТ:
--
--   1. Үйлчилгээнд ХҮСЭГЧ бий. Хэн, ямар регистртэй, хаана хариу авах вэ —
--      үүнгүйгээр биелүүлэх талд ажил хийх мэдээлэл байхгүй;
--   2. Үйлчилгээ ХАРИУГҮЙГЭЭР БИЕЛСЭН гэж тэмдэглэгдэж болохгүй. Энэ нь
--      Go-гийн шалгалт биш, схемийн шалгалт болж суусан: шалгалт нь нэг
--      өдөр мартагдаж болно, CHECK мартагдахгүй.
--
-- Шугам нь кодоос ирнэ. ring.dgov.mn-ээс импортлогдсон код бол мөн чанараар
-- нь төрийн үйлчилгээ; байгууллагын дотоод код бол даалгавар. Тиймээс
-- урьдчилан бүртгэгдсэн код өөрөө аль шугамынхаа болохыг хэлнэ, даалгавар
-- үүсгэгч хүн биш — эс бөгөөс нэг код хоёр өөр амлалттай хоёр газар
-- хэрэглэгдэх байв.

-- +goose Up

-- Кодын шугам. Байгаа бүх код 'assignment' болно: энэ миграцаас өмнө
-- үүссэн бүх зүйл яг тэр урсгалынх байсан.
ALTER TABLE urtuu_request_codes
    ADD COLUMN IF NOT EXISTS line TEXT NOT NULL DEFAULT 'assignment';

ALTER TABLE urtuu_request_codes
    DROP CONSTRAINT IF EXISTS urtuu_request_codes_line_check;
ALTER TABLE urtuu_request_codes
    ADD CONSTRAINT urtuu_request_codes_line_check CHECK (line IN ('service', 'assignment'));

ALTER TABLE urtuu_tasks
    ADD COLUMN IF NOT EXISTS line TEXT NOT NULL DEFAULT 'assignment',
    -- Хүсэгч: нэр, регистрийн дугаар, холбоо барих зам, иргэн эсвэл
    -- байгууллага эсэх. Даалгаврын шугамд хоосон — тэнд хүсэгч гэж байхгүй.
    --
    -- Энэ нь §2.4-ийн "өгөгдөл нүүдэггүй" дүрмийг зөрчихгүй: тэр дүрэм нь
    -- БАЙГУУЛЛАГЫН дотоод өгөгдөл дээшээ урсахгүй гэсэн үг. Хүсэгчийн
    -- мэдээлэл нь ажлын чиглэлийн дагуу ДООШОО явна, тэр ч бүү хэл
    -- хүсэгч өөрөө яг үүний тулд өгсөн байдаг: түүнгүйгээр биелүүлэх тал
    -- юу хийхээ мэдэхгүй.
    ADD COLUMN IF NOT EXISTS applicant JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- Хариу. Үйлчилгээний шугамд заавал, дуусгах агшинд.
    ADD COLUMN IF NOT EXISTS answer TEXT NOT NULL DEFAULT '';

ALTER TABLE urtuu_tasks DROP CONSTRAINT IF EXISTS urtuu_tasks_line_check;
ALTER TABLE urtuu_tasks
    ADD CONSTRAINT urtuu_tasks_line_check CHECK (line IN ('service', 'assignment'));

-- Хүсэлт хэнийх нь мэдэгдэхгүйгээр үйлчилгээ гэж байхгүй.
ALTER TABLE urtuu_tasks DROP CONSTRAINT IF EXISTS urtuu_tasks_service_has_applicant;
ALTER TABLE urtuu_tasks
    ADD CONSTRAINT urtuu_tasks_service_has_applicant
        CHECK (line <> 'service' OR applicant <> '{}'::jsonb);

-- ХАРИУГҮЙГЭЭР БИЕЛСЭН ГЭЖ БАЙХГҮЙ.
--
-- Зөвхөн ЭНД хийгдэж буй ажилд хамаарна (target_peer_id IS NULL). Толь мөр
-- нь нөгөө талын төлөвийн хуулбар — тэдний илгээсэн хариуг нь хадгалдаг
-- бөгөөд тэдний өмнөөс хариу зохиох ёсгүй. Хэрэв доод тал хариугүй
-- "биелсэн" гэж мэдэгдвэл тэр нь тэдний зөрчил бөгөөд тэдний схем дээр
-- зогсоно.
ALTER TABLE urtuu_tasks DROP CONSTRAINT IF EXISTS urtuu_tasks_service_has_answer;
ALTER TABLE urtuu_tasks
    ADD CONSTRAINT urtuu_tasks_service_has_answer
        CHECK (line <> 'service'
               OR target_peer_id IS NOT NULL
               OR status NOT IN ('COMPLETED', 'CLOSED')
               OR answer <> '');

-- Хоёр шугамыг тусад нь харах нь дэлгэц бүрийн үндсэн query.
CREATE INDEX IF NOT EXISTS idx_urtuu_tasks_line
    ON urtuu_tasks (tenant_id, line, status, created_at DESC);

-- Хүсэгчийн регистрээр хайх: "энэ иргэний хүсэлт хаана явж байна вэ".
CREATE INDEX IF NOT EXISTS idx_urtuu_tasks_applicant
    ON urtuu_tasks ((applicant->>'registry_number'))
 WHERE line = 'service' AND applicant->>'registry_number' IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_urtuu_tasks_applicant;
DROP INDEX IF EXISTS idx_urtuu_tasks_line;
ALTER TABLE urtuu_tasks
    DROP CONSTRAINT IF EXISTS urtuu_tasks_service_has_answer,
    DROP CONSTRAINT IF EXISTS urtuu_tasks_service_has_applicant,
    DROP CONSTRAINT IF EXISTS urtuu_tasks_line_check,
    DROP COLUMN IF EXISTS answer,
    DROP COLUMN IF EXISTS applicant,
    DROP COLUMN IF EXISTS line;
ALTER TABLE urtuu_request_codes
    DROP CONSTRAINT IF EXISTS urtuu_request_codes_line_check,
    DROP COLUMN IF EXISTS line;
