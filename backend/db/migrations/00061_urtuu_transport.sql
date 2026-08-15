-- «Өртөө» — платформ хоорондын сувгийн тээвэр (docs/URTUU_PROPOSAL.md §2, §4).
--
-- Их Монгол Улсын өртөө шуудан: захиаг өртөөнөөс өртөөнд дамжуулж, хүрсэн
-- эсэхийг нь буцааж мэдэгддэг байсан. Энэ дөрвөн хүснэгт яг тэр — холбоос,
-- гарах дараалал, хүргэлтийн байдал, ирсэн дараалал.
--
-- Дөрвөн шийдвэр энд хатуу суусан:
--
-- 1. ДООД ТАЛ Л ХОЛБОГДОНО. Дээд тал доод руу хэзээ ч гарахгүй: доод суулгац
--    галт ханын цаана, хувийн сүлжээнд, түр унтраатай байж болно. Тиймээс
--    base_url нь ЗӨВХӨН бид доод (role='child') үед утгатай — тэр бол дээдийн
--    хаяг. Каталог sync яг ижил шалтгаанаар pull сонгосон.
--
-- 2. PAYLOAD НЬ JSONB БИШ, TEXT. Гарын үсэг байтуудыг хамардаг. JSONB нь
--    түлхүүрийн дарааллыг эмхэлж, зайг хаядаг — өөрөөр хэлбэл хадгалаад
--    буцааж уншихад БУСАД байт гарч ирж, гарын үсэг батлагдахаа болино.
--    pkg/catalog-ийн signed баримт яг үүнээс болж apps-аа түүхий JSON-оор
--    барьдаг. Ирсэн байтыг ирсэн хэвээр нь хадгална.
--
-- 3. УСТГАХ БИШ ЦУЦЛАХ. report_grants-ийн дүрэм: холбоос дээр DELETE байхгүй,
--    revoked_at л бий. "Бидэнтэй хэн холбоотой байсан бэ" гэдэг нь хожим
--    асуугддаг асуулт бөгөөд устгагдсан мөр түүнд хариулахгүй.
--
-- 4. ТЕНАНТ БҮР ӨӨРИЙН ӨРТӨӨТЭЙ. Нэг суулгац дээрх хоёр байгууллага тус
--    тусын холбоос, тус тусын дараалалтай — бүх хүснэгт tenant_id-тай, RLS-д
--    хамрагдана. Гарын үсгийн түлхүүр нь суулгацынх (URTUU_SIGNING_KEY),
--    харин "хэн ярьж байна" гэдгийг холбоос бүрийн token шийднэ: гарын үсэг
--    нь хэн бичсэн, token нь хэн ярьж байгааг хэлнэ — хоёулаа хэрэгтэй.

-- +goose Up

CREATE TABLE IF NOT EXISTS urtuu_peers (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Хүний нэрлэсэн нэр: "Ховд аймаг", "Боловсролын яам". Жагсаалт дээр
    -- UUID биш үүнийг харна.
    name               TEXT        NOT NULL DEFAULT '',
    -- ЭНЭ холбоос дээр БИД хэн бэ. Мод биш, чиглэлтэй граф: нэг суулгац
    -- олон дээдтэй, олон доодтой байж болох тул үүрэг нь холбоос бүрийнх,
    -- суулгацынх биш.
    role               TEXT        NOT NULL,
    -- Дээдийн хаяг. Зөвхөн role='child' үед бөглөгдөнө (§2.1).
    base_url           TEXT        NOT NULL DEFAULT '',
    -- Нөгөө талын Ed25519 нийтийн түлхүүр, base64. Handshake-ийн үед
    -- солилцогдоно; үүнгүй бол ирсэн дугтуйн гарын үсгийг шалгах аргагүй.
    peer_public_key    TEXT        NOT NULL DEFAULT '',
    -- Тээврийн token-ий SHA-256. Түүхий token нь үүсгэх агшинд НЭГ УДАА
    -- харагдана — sessions, devices, email_verifications-ийн ижил дүрэм.
    token_hash         CHAR(64)    NOT NULL DEFAULT '',
    status             TEXT        NOT NULL DEFAULT 'pending',
    -- Нэг удаагийн урилгын кодын SHA-256 (24 цаг). Дээд тал үүсгэж, доод тал
    -- буулгана. Хэрэглэгдмэгц NULL болно: код бол зөвхөн эхний танилцуулга
    -- бөгөөд түүнээс хойш token нь итгэмжлэл болно.
    invite_code_hash   CHAR(64),
    invite_expires_at  TIMESTAMPTZ,
    -- Холбоосын эрүүл мэнд. last_seen_at нь нөгөө тал хамгийн сүүлд хэзээ
    -- ярьсан бэ; хоосон холбоос ба унасан холбоосыг ялгах цорын ганц зүйл.
    last_seen_at       TIMESTAMPTZ,
    last_error         TEXT        NOT NULL DEFAULT '',
    -- Хоёр талын цагийн зөрүү, секундээр. SLA-г илгээгчийн created_at-аар
    -- тооцдог тул хэт зөрсөн цаг нь чимээгүй буруу хугацаа биш, харагдах
    -- анхааруулга байх ёстой (§9).
    clock_skew_seconds INTEGER     NOT NULL DEFAULT 0,
    revoked_at         TIMESTAMPTZ,
    created_by         UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT urtuu_peers_role_check   CHECK (role IN ('parent', 'child')),
    CONSTRAINT urtuu_peers_status_check CHECK (status IN ('pending', 'active', 'revoked')),
    -- Бид доод бол дээдийн хаягийг заавал мэдэх ёстой: мэдэхгүй бол
    -- холбогдох тал нь байхгүй болж, холбоос чимээгүй үхнэ.
    CONSTRAINT urtuu_peers_child_has_base_url
        CHECK (role <> 'child' OR status <> 'active' OR base_url <> '')
);

-- Тээврийн token-оор холбоос олох нь тенантаас ӨМНӨ болдог: хүсэлт нь
-- session-гүй ирдэг тул эхлээд token нь аль холбоос болохыг олж, тэндээс
-- тенантаа мэдэж авна. Тиймээс индекс нь глобал давхардалгүй байх ёстой.
CREATE UNIQUE INDEX IF NOT EXISTS idx_urtuu_peers_token
    ON urtuu_peers (token_hash) WHERE token_hash <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_urtuu_peers_invite
    ON urtuu_peers (invite_code_hash) WHERE invite_code_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_urtuu_peers_tenant
    ON urtuu_peers (tenant_id, status);

-- Гарах дугтуйнууд. Нэг дугтуй нэг л удаа гарын үсэг зурагдана; хэдэн ч
-- холбоос руу явахаас үл хамааран — задаргаа (fan-out) ба кодын зарлал хоёул
-- нэг агуулгыг олон доод руу илгээдэг тул давхардуулж хадгалах нь тэр бүрд
-- өөр гарын үсэг үүсгэх байсан.
CREATE TABLE IF NOT EXISTS urtuu_outbox (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id TEXT        NOT NULL,
    kind       TEXT        NOT NULL,
    -- Дугтуйн created_at — гарын үсгийн дотор байгаа тэр утга. Мөр үүссэн
    -- цаг биш: SLA-г үүнээс тоолно.
    created_at TIMESTAMPTZ NOT NULL,
    -- Түүхий байт. Дээрх §2-ыг үз.
    payload    TEXT        NOT NULL,
    signature  TEXT        NOT NULL,
    queued_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT urtuu_outbox_message_unique UNIQUE (tenant_id, message_id)
);

-- Хүргэлтийн байдал: дугтуй бүр холбоос бүрд нэг мөр. integration_deliveries-
-- ийн загвар дээр retry-г нэмсэн — тэнд хүргэлт нэг л удаа оролддог байсан
-- бол энд нөгөө тал долоо хоног унтраатай байж болно.
CREATE TABLE IF NOT EXISTS urtuu_deliveries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outbox_id       UUID        NOT NULL REFERENCES urtuu_outbox(id) ON DELETE CASCADE,
    peer_id         UUID        NOT NULL REFERENCES urtuu_peers(id) ON DELETE CASCADE,
    attempts        INTEGER     NOT NULL DEFAULT 0,
    -- Exponential backoff-ийн дараагийн цаг. Одооноос эхэлнэ: анхны оролдлого
    -- хойшлох ямар ч шалтгаангүй.
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT        NOT NULL DEFAULT '',
    -- Нөгөө тал БАТАЛСАН агшин, илгээсэн агшин биш. Хүлээн авсан гэдгээ
    -- нөгөө тал ack-аар хэлтэл хүргэлт дуусаагүй — хариу нь замдаа алдагдсан
    -- илгээлт дахин санал болгогдоно (хүлээн авагч талд message_id-гаар
    -- идемпотент).
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT urtuu_deliveries_once UNIQUE (outbox_id, peer_id)
);

-- Тээврийн гол query: "энэ холбоос руу одоо юу явах ёстой вэ".
CREATE INDEX IF NOT EXISTS idx_urtuu_deliveries_due
    ON urtuu_deliveries (peer_id, next_attempt_at) WHERE delivered_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_urtuu_deliveries_tenant
    ON urtuu_deliveries (tenant_id, created_at DESC);

-- Ирсэн дугтуйнууд. message_id давхардвал хоёр дахь нь чимээгүй хаягдана —
-- энэ бол алдаа биш, идемпотент хүлээн авалтын гол механизм: сүлжээ тасарч
-- нөгөө тал дахин илгээхэд даалгавар хоёр дахин үүсэхгүй.
CREATE TABLE IF NOT EXISTS urtuu_inbox (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    peer_id     UUID        NOT NULL REFERENCES urtuu_peers(id) ON DELETE CASCADE,
    message_id  TEXT        NOT NULL,
    kind        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    payload     TEXT        NOT NULL,
    signature   TEXT        NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Нөгөө талд "авсан" гэж хэлсэн агшин. Хэлтэл нь нөгөө тал ижил дугтуйг
    -- дахин санал болгосоор байна — энэ бол алдаа биш, at-least-once
    -- хүргэлтийн үнэ бөгөөд message_id-ийн давхардалгүй байдал үүнийг үнэгүй
    -- болгож байгаа юм.
    acked_at    TIMESTAMPTZ,
    -- Хэрэглэгч тал (Өртөө апп) уншиж, даалгавар болгосон агшин. NULL нь
    -- "хүлээгдэж байна" — аппгүй суулгац дугтуйг хүлээн авч, хадгалж,
    -- баталгаажуулна; апп нь суулгагдмагц тэднийг олно.
    processed_at TIMESTAMPTZ,

    CONSTRAINT urtuu_inbox_message_unique UNIQUE (tenant_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_urtuu_inbox_unprocessed
    ON urtuu_inbox (tenant_id, received_at) WHERE processed_at IS NULL;

-- Тээврийн гол query-ийн нөгөө тал: "энэ холбоост юуг нь баталгаажуулаагүй вэ".
CREATE INDEX IF NOT EXISTS idx_urtuu_inbox_unacked
    ON urtuu_inbox (peer_id, received_at) WHERE acked_at IS NULL;

GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_peers      TO gerege_nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_outbox     TO gerege_nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_deliveries TO gerege_nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_inbox      TO gerege_nexus_app;

-- 00029-ийн ерөнхий бодлого нь тэр миграци ажилласан агшны хүснэгтүүдэд л
-- хамаарсан тул шинэ хүснэгт бүр өөрийн ижил бодлогоо тунхаглана
-- (00045, 00051, 00053-ийн ижил хэв).
-- +goose StatementBegin
DO $rls$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['urtuu_peers', 'urtuu_outbox', 'urtuu_deliveries', 'urtuu_inbox']
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
DROP TABLE IF EXISTS urtuu_inbox;
DROP TABLE IF EXISTS urtuu_deliveries;
DROP TABLE IF EXISTS urtuu_outbox;
DROP TABLE IF EXISTS urtuu_peers;
