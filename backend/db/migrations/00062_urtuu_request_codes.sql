-- «Өртөө» — хүсэлтийн кодын бүртгэл (docs/URTUU_PROPOSAL.md §2.5, §4).
--
-- Даалгавар чөлөөт текстээр үүсэхгүй. Үүсгэхийн тулд урьдчилан бүртгэгдсэн
-- КОД сонгоно, тэр код нь юуг бөглөхийг (schema) болон хэдэн хоногт хийхийг
-- (default_sla) өөрөө хэлнэ. Шалтгаан нь энгийн: "тооллого явуулна уу" гэсэн
-- чөлөөт мөрийг тоолж, хугацаа тавьж, тайлагнаж болохгүй.
--
-- Үгсийн сангийн эзэн нь энэ платформ БИШ. Төрийн үйлчилгээний процессууд
-- ring.dgov.mn дээр байдаг тул кодууд тэндээс импортлогдоно (source='ring').
-- Дээд платформ өөрийн багцаасаа холбоос бүрд аль кодыг нээхээ шийдэж
-- зарлана; доод тал түүнийг синклэж авна (source='link'). Ring-д байхгүй
-- дотоод хэрэгцээнд зөвхөн 'local.' угтвартай код зөвшөөрөгдөнө — угтваргүй
-- бол өнөөдөр зохиосон дотоод код маргааш ring-ээс ирэх кодтой мөргөлдөж,
-- мөргөлдөөн нь нэр солигдсон мэт харагдана.
--
-- Кодууд ЛОКАЛ хадгалагдана. Энэ бол кэш биш, эх сурвалж: ring унасан ч,
-- дээд тал долоо хоног унтарсан ч даалгавар үүсгэх, ирсэн даалгаврыг унших
-- ажиллагаа зогсохгүй (каталог sync-ийн fallback зарчим).

-- +goose Up

CREATE TABLE IF NOT EXISTS urtuu_request_codes (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code             TEXT        NOT NULL,
    -- 7 хэл: {"mn": "...", "en": "...", ...}. Сервер эзэмшдэг агуулга тул
    -- орчуулга нь кодтойгоо хамт аялна — доод тал кодыг синклэхдээ нэрийг нь
    -- ч авна, өөрийн толь бичигтээ оруулах шаардлагагүй.
    names            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- Даалгаврын биеийн JSON Schema. Форм үүнээс үүснэ, бөглөсөн зүйл нь
    -- үүгээр шалгагдана — хоёулаа нэг эх сурвалжтай тул зөрөх боломжгүй.
    schema           JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- Хугацааны норм. NULL нь "энэ код норм заагаагүй" гэсэн үг бөгөөд
    -- хугацааг гараар тавина; INTERVAL 0 гэдэг нь "хугацаагүй" гэсэн өөр
    -- утга болох тул хоёрыг ялгав.
    default_sla      INTERVAL,
    source           TEXT        NOT NULL,
    -- source='link' үед аль холбоосоос ирснийг заана. Холбоос цуцлагдвал
    -- түүний зарласан кодууд хамт алга болно: тэдгээр нь тэр холбоосын
    -- үгсийн сан байсан бөгөөд өөр дээрээ утгагүй.
    source_peer_id   UUID        REFERENCES urtuu_peers(id) ON DELETE CASCADE,
    -- ring.dgov.mn дэх процессын id. Дахин импортлоход давхардуулахгүй
    -- шинэчлэхийн тулд, мөн "энэ норм хаанаас гарав" гэдэгт хариулах газар.
    ring_process_ref TEXT        NOT NULL DEFAULT '',
    -- Эх сурвалж дээрх хувилбар. Ирсэн зарлал энэ тооноос хойш явбал л
    -- дарж бичнэ — хоцорсон дугтуй шинэ тодорхойлолтыг буцаахгүй.
    version          INTEGER     NOT NULL DEFAULT 1,
    active           BOOLEAN     NOT NULL DEFAULT TRUE,
    created_by       UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT urtuu_request_codes_source_check
        CHECK (source IN ('ring', 'link', 'local')),
    -- Нэрийн орон зайн дүрэм схемд суусан: ЭНД зохиогдсон код заавал 'local.'
    -- угтвартай. Код бүрийг шалгадаг Go функц нэг өдөр мартагдана; CHECK
    -- мартагдахгүй.
    --
    -- Урвуу нь ҮНЭН БИШ бөгөөд энэ нь санамсаргүй биш: дээд тал өөрийн локал
    -- кодоо доод руугаа нээж болно (яамны дотоод код агентлагтаа), тэр код
    -- доод тал дээр source='link' боловч 'local.' угтвартайгаа хэвээр ирнэ.
    -- Код нь хоёр суулгац дээр ИЖИЛ нэртэй байх ёстой — өөрчилбөл ирсэн
    -- даалгаврыг кодтой нь тааруулах аргагүй болно.
    --
    -- Нэр давхцвал (доод тал өөрөө local.x зохиосон, дээд тал бас local.x
    -- зарласан) ӨӨРИЙН тодорхойлолт ялна — upsertCode-ийн UPDATE нь
    -- source='local' мөрийг хөнддөггүй. Хэн нэгний зарлал таны өөрийн
    -- тодорхойлолтыг чимээгүй солих нь илүү муу төгсгөл.
    CONSTRAINT urtuu_request_codes_local_namespace
        CHECK (source <> 'local' OR code LIKE 'local.%'),
    CONSTRAINT urtuu_request_codes_link_has_peer
        CHECK ((source = 'link') = (source_peer_id IS NOT NULL)),
    CONSTRAINT urtuu_request_codes_unique UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_urtuu_request_codes_active
    ON urtuu_request_codes (tenant_id, source) WHERE active;

-- Холбоос бүрд нээгдсэн кодууд. Дээд тал ring-ээс авсан бүх багцаа доод бүрд
-- нээдэггүй: аймагт хамаарах код сумд хамаарахгүй, нэг доод байгууллагад
-- нээсэн код нөгөөд нээгдсэн гэсэн үг биш.
--
-- Тусдаа хүснэгт болсон шалтгаан нь код дээрх массив биш: холбоос устахад
-- нээлт нь хамт устах ёстой бөгөөд үүнийг гадаад түлхүүр үнэгүй хийж өгнө.
CREATE TABLE IF NOT EXISTS urtuu_peer_codes (
    tenant_id UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    peer_id   UUID        NOT NULL REFERENCES urtuu_peers(id) ON DELETE CASCADE,
    code      TEXT        NOT NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    opened_by UUID        REFERENCES users(id) ON DELETE SET NULL,

    PRIMARY KEY (peer_id, code),
    -- Зөвхөн энэ тенантад бүртгэлтэй кодыг нээж болно.
    CONSTRAINT urtuu_peer_codes_known
        FOREIGN KEY (tenant_id, code) REFERENCES urtuu_request_codes (tenant_id, code)
        ON DELETE CASCADE
);

GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_request_codes TO gerege_nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_peer_codes    TO gerege_nexus_app;

-- 00029-ийн ерөнхий бодлого нь тэр миграци ажилласан агшны хүснэгтүүдэд л
-- хамаарсан тул шинэ хүснэгт бүр өөрийн ижил бодлогоо тунхаглана.
-- +goose StatementBegin
DO $rls$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['urtuu_request_codes', 'urtuu_peer_codes']
    LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', target);
        EXECUTE format('ALTER TABLE public.%I FORCE ROW LEVEL SECURITY', target);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', target);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON public.%I TO gerege_nexus_app '
            'USING (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid) '
            'WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)',
            target);
    END LOOP;
END
$rls$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS urtuu_peer_codes;
DROP TABLE IF EXISTS urtuu_request_codes;
