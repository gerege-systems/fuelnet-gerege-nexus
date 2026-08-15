-- «Өртөө» апп — даалгаврын амьдралын мөчлөг (docs/URTUU_PROPOSAL.md §4, §5).
--
-- Мөр бүр бол ЭНЭ суулгац дээрх нэг даалгавар. Гурван төрөл байна, гурвуулаа
-- ижил хүснэгтэд ижил төлөвийн машинтай сууна:
--
--   origin_peer_id NULL, target_peer_id NULL   — энд үүсч энд шийдэгдэх ажил
--   origin_peer_id БӨГЛӨГДСӨН                  — дээдээс ирсэн даалгавар
--   target_peer_id БӨГЛӨГДСӨН                  — доод руу илгээсэн ажлын
--                                                ТОЛЬ: төлөв нь доод талын
--                                                task_update-аар хөдөлнө
--
-- Задаргаа (fan-out) нь доод бүрд ТУСДАА мөр үүсгэнэ (§4), эх мөртэйгээ
-- parent_task_id-аар холбогдоно. Тиймээс "зорьсон peer-үүд" гэсэн массив
-- багана энд БАЙХГҮЙ: массив байсан бол нэг ажлыг хоёр газар — мөрүүдэд болон
-- массивт — бичих байсан бөгөөд хоёр нь зөрөх өдөр гарцаагүй ирнэ. Аль сум
-- хийсэн, аль нь хоцорсныг толь мөрүүд өөрсдөө хэлнэ.
--
-- origin_chain нь мөчлөгөөс хамгаална (§9): даалгавар дамжсан суулгац бүрийн
-- ID-г дараалуулан хадгална. Өөрийн ID нь гинжинд байвал даалгаврыг хүлээж
-- авахгүй — А→Б→А холбоос дээр даалгавар мөнхөд эргэлдэхээс сэргийлнэ.
--
-- Хугацаа хэтэрсэн эсэх нь БАГАНА БИШ: төлөв ба deadline хоёроос уншилт бүрд
-- тооцогдоно. Хадгалсан бол deadline засагдахад хуучин тэмдэг үлдэх байсан.

-- +goose Up

CREATE TABLE IF NOT EXISTS urtuu_tasks (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Хүсэлтийн код. urtuu_request_codes руу гадаад түлхүүр ЗОРИУДААР алга:
    -- код хожим хаагдаж, устгагдаж болох ба тэр үед хийгдсэн ажлын түүх
    -- хамт алга болох ёсгүй.
    code           TEXT        NOT NULL,
    -- Кодын нэрийг үүсэх агшинд хуулж авна. Код хаагдсан ч, орчуулга нь
    -- өөрчлөгдсөн ч, тухайн үед юу даалгасан нь уншигдсан хэвээр байх ёстой.
    title          TEXT        NOT NULL DEFAULT '',
    -- Кодын schema-гаар бөглөгдсөн бие.
    payload        JSONB       NOT NULL DEFAULT '{}'::jsonb,

    origin_peer_id UUID        REFERENCES urtuu_peers(id) ON DELETE SET NULL,
    -- Илгээгч талын мөрийн ID. Буцах task_update-ыг аль даалгавартай
    -- тааруулахыг энэ шийднэ.
    origin_task_id TEXT        NOT NULL DEFAULT '',
    target_peer_id UUID        REFERENCES urtuu_peers(id) ON DELETE SET NULL,
    parent_task_id UUID        REFERENCES urtuu_tasks(id) ON DELETE CASCADE,
    origin_chain   TEXT[]      NOT NULL DEFAULT '{}',

    status         TEXT        NOT NULL DEFAULT 'RECEIVED',
    deadline       TIMESTAMPTZ,
    -- Дотоод хариуцагч. Даалгавар хүнд оногдох нь платформ хоорондын явдал
    -- биш, дотоод явдал тул дээшээ дамждаггүй.
    assigned_user_id UUID      REFERENCES users(id) ON DELETE SET NULL,
    -- Сүүлийн тайлбар: буцаасан шалтгаан, биелэлтийн тэмдэглэл.
    note           TEXT        NOT NULL DEFAULT '',
    -- Нотолгооны лавлагаанууд (DocumentFiler ref). Баримт өөрөө хэзээ ч
    -- дамжихгүй — лавлагаа л дамжина (§2.4).
    evidence       JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_by     UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- pkg/urtuu-ийн TaskStatus-тай яг ижил багц. Go тал шилжилтийг шалгана;
    -- энэ нь ямар ч зам утгагүй төлөв бичихээс хамгаална.
    CONSTRAINT urtuu_tasks_status_check CHECK (status IN
        ('RECEIVED', 'ACCEPTED', 'IN_PROGRESS', 'DELEGATED', 'COMPLETED', 'RETURNED', 'CLOSED')),
    -- Ирсэн ба илгээсэн нь нэг мөр байж болохгүй: нэг мөр нэг талын харагдац.
    CONSTRAINT urtuu_tasks_one_direction CHECK (origin_peer_id IS NULL OR target_peer_id IS NULL),
    -- Толь мөр үргэлж эхтэйгээ холбоотой: эцэггүй толь бол хэний ч биш ажил.
    CONSTRAINT urtuu_tasks_mirror_has_parent
        CHECK (target_peer_id IS NULL OR parent_task_id IS NOT NULL)
);

-- Ирсэн болон илгээсэн дараалал — аппын хоёр үндсэн дэлгэц.
CREATE INDEX IF NOT EXISTS idx_urtuu_tasks_incoming
    ON urtuu_tasks (tenant_id, status, created_at DESC) WHERE origin_peer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_urtuu_tasks_tree
    ON urtuu_tasks (parent_task_id) WHERE parent_task_id IS NOT NULL;
-- Ирсэн task_update-ыг мөртэй нь тааруулах зам.
CREATE INDEX IF NOT EXISTS idx_urtuu_tasks_mirror
    ON urtuu_tasks (target_peer_id, id) WHERE target_peer_id IS NOT NULL;

-- Нэг холбоосын нэг даалгавар нэг л мөр. Дугтуй нь message_id-гаар аль хэдийн
-- идемпотент боловч энэ нь өөр давхарга: дугтуйг уншиж, даалгавар үүсгэсний
-- ДАРАА "уншсан" гэж тэмдэглэх бичилт унавал уншигч дахин дуудагдана.
-- Уншигч бүр давтагдахад аюулгүй байх ёстой бөгөөд энд түүнийг схем баталгаажуулж байна.
CREATE UNIQUE INDEX IF NOT EXISTS idx_urtuu_tasks_from_peer
    ON urtuu_tasks (origin_peer_id, origin_task_id) WHERE origin_peer_id IS NOT NULL;
-- Хоцорсон даалгаврын жагсаалт ба housekeeping.
CREATE INDEX IF NOT EXISTS idx_urtuu_tasks_deadline
    ON urtuu_tasks (tenant_id, deadline) WHERE deadline IS NOT NULL AND status <> 'CLOSED';

-- Шилжилт бүр. installation_events-ийн хэв: append-only, засварлагдахгүй,
-- устгагдахгүй. "Хэн, хэзээ, яагаад" гэдэг нь даалгавар хаагдсаны дараа
-- асуугддаг асуулт бөгөөд төлөвийн багана түүнд хариулж чадахгүй.
CREATE TABLE IF NOT EXISTS urtuu_task_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    task_id       UUID        NOT NULL REFERENCES urtuu_tasks(id) ON DELETE CASCADE,
    from_status   TEXT        NOT NULL DEFAULT '',
    to_status     TEXT        NOT NULL,
    -- Хэн хийв: энэ суулгац дээрх хүн, эсвэл нөгөө талын мэдэгдэл. Хоёуланг
    -- нэг "actor" багана болгосон бол "хүн үү, машин уу" гэдэг нь мөрийн
    -- утгаас таамаглагдах байв.
    actor_user_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    actor_peer_id UUID        REFERENCES urtuu_peers(id) ON DELETE SET NULL,
    note          TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_urtuu_task_events_task
    ON urtuu_task_events (task_id, created_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON urtuu_tasks       TO gerege_nexus_app;
-- UPDATE, DELETE олгогдоогүй: энэ хүснэгт зөвхөн ургана.
GRANT SELECT, INSERT                  ON urtuu_task_events TO gerege_nexus_app;

-- +goose StatementBegin
DO $rls$
DECLARE target TEXT;
BEGIN
    FOREACH target IN ARRAY ARRAY['urtuu_tasks', 'urtuu_task_events']
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
DROP TABLE IF EXISTS urtuu_task_events;
DROP TABLE IF EXISTS urtuu_tasks;
