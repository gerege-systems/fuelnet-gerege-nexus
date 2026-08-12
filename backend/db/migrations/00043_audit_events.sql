-- Audit-ийн мөрийг stdout-оос гадна хадгалдаг болгоно.
--
-- Өнөөдрийг хүртэл audit.Record нь зөвхөн slog.Info("AUDIT_EVENT", ...) бичдэг
-- байсан. Тэр мөр нь контейнерийн лог дотор амьдардаг: контейнер дахин үүсэхэд
-- алга болно, тенант эсвэл хэрэглэгчээр хайх боломжгүй, "миний өгөгдлийг хэн
-- харав" гэсэн асуултад хариулах эх сурвалж болохгүй. Тэр асуулт нь Үе шат 4б
-- дэх тенант дамнасан тайлангийн үндсэн шаардлага тул мөрийг өгөгдлийн санд
-- үлдээх нь заавал байх зүйл болов.
--
-- stdout-ын мөр хэвээр үлдэнэ. Хоёулаа байх шалтгаан нь: DB-д бичих нь
-- best-effort (үндсэн үйлдлийг унагахгүй) тул өгөгдлийн сан унасан үед
-- лог нь цорын ганц ул мөр болно.
--
-- tenant_id нь NULL байж болно: платформын түвшний үйлдэл (нэвтрэх оролдлого,
-- тенант сонгох, каталог сихрончлол) нь ямар ч байгууллагад харьяалагдахгүй.
-- RLS бодлого нь 00029-ийн загвартай ижил боловч USING нь `tenant_id IS NULL
-- OR ...` биш: платформын мөрийг тенантын хэрэглэгчид харах ёсгүй. Тэдгээрийг
-- зөвхөн login role (dbguard-ын платформ зам) уншина.

-- +goose Up
CREATE TABLE IF NOT EXISTS audit_events (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    TEXT,
    action     TEXT        NOT NULL,
    resource   TEXT        NOT NULL,
    details    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- user_id нь UUID биш TEXT бөгөөд гадаад түлхүүргүй. Хоёр шалтгаан:
-- audit нь устгагдсан хэрэглэгчийн үйлдлийг ч санаж байх ёстой (CASCADE нь
-- мөрийг өөрийг нь устгана, SET NULL нь "хэн" гэдгийг устгана — хоёулаа
-- зорилготой зөрчилдөнө); мөн дуудагчид үргэлж хэрэглэгчийн UUID байдаггүй —
-- native_operations_handlers.go нь "device:<id>" гэж бичдэг, тэр нь хүн биш
-- төхөөрөмжийн хийсэн үйлдэл гэдгийг илэрхийлнэ.

-- "Энэ байгууллагад юу болов" — хамгийн түгээмэл асуулт, цаг хугацааны
-- дарааллаар.
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_time
    ON audit_events (tenant_id, created_at DESC);

-- "Энэ хүн юу хийв" ба "энэ төрлийн үйлдэл хэдэн удаа болов".
CREATE INDEX IF NOT EXISTS idx_audit_events_user_time
    ON audit_events (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_action_time
    ON audit_events (action, created_at DESC);

GRANT SELECT, INSERT ON audit_events TO gerege_nexus_app;

-- UPDATE, DELETE зориудаар олгогдоогүй: audit гэдэг нь бичээд өөрчлөхгүй
-- бүртгэл. Хадгалалтын хугацааны цэвэрлэгээ хийх болбол тэр нь login role-оор
-- (housekeeping зам) явна.

ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON audit_events;
CREATE POLICY tenant_isolation ON audit_events TO gerege_nexus_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- +goose Down
DROP TABLE IF EXISTS audit_events;
