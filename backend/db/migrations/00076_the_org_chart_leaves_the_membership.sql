-- Албан тушаал ба хэлтэс нь гишүүнчлэлээс салж, аппдаа очив.
--
-- `memberships` бол платформын хамгийн төвийн хүснэгтүүдийн нэг: хэн ямар
-- байгууллагын гишүүн бэ. Тэр хүснэгт хоёр багана үүрч байв — `job_title` ба
-- `department_id` — хоёулаа `organisation` аппынх. `department_id` нь бүр
-- аппын өөрийн `departments` хүснэгт рүү гадаад түлхүүртэй.
--
-- Өөрөөр хэлбэл платформын хүснэгт аппын хүснэгтээс хамааралтай байв. Энэ нь
-- 28 хүснэгтийн асуудлаас (цөм аппын хүснэгтийг үүрсэн) чанарын хувьд өөр,
-- бас дор: хамаарал буруу тийшээ харж, аппыг гаргах гэсэн бүх оролдлого
-- платформын биеийн загварыг чирч гарахыг шаардаж байв.
--
-- Одоо гурав нь аппынх: `organisation_people` нь гишүүнчлэлийн ид-г түлхүүр
-- болгож, албан тушаал, хэлтсийг барина. Платформ хүн ямар байгууллагын
-- гишүүн болохыг мэднэ; тэр хүн юу хийдгийг мэдэхээ болино.
--
-- `ON DELETE CASCADE` нь гишүүнчлэл дуусахад мөр нь дагаж явна — албан тушаал
-- нь гишүүнчлэлээс өөрөө утгагүй.
--
-- ЗӨӨХ ҮЕД: organisation апп энэ репогоос гарахдаа энэ хүснэгтийг өөрийн
-- миграцад `IF NOT EXISTS`-ээр дахин зарлана. business-gerege-nexus-ийн
-- 00001_commerce.sql яг тэр аргыг ашигласан бөгөөд түүний тайлбар шалтгааныг
-- бичсэн: хэрэглэгдсэн миграцыг дахин бичих боломжгүй.

-- +goose Up

CREATE TABLE IF NOT EXISTS organisation_people (
    membership_id UUID PRIMARY KEY REFERENCES memberships(id) ON DELETE CASCADE,
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_title     VARCHAR(255) NOT NULL DEFAULT '',
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organisation_people_tenant ON organisation_people(tenant_id);
CREATE INDEX IF NOT EXISTS idx_organisation_people_department ON organisation_people(department_id);

-- Тенант тусгаарлалт, 00037-ийн хэлбэрээр: уншихдаа session-ий зөвшөөрөгдсөн
-- байгууллагууд, бичихдээ зөвхөн идэвхтэй нэг нь.
ALTER TABLE organisation_people ENABLE ROW LEVEL SECURITY;
ALTER TABLE organisation_people FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON organisation_people;
CREATE POLICY tenant_isolation ON organisation_people TO gerege_nexus_app
    USING (tenant_id IS NULL OR tenant_id = ANY (COALESCE(
        NULLIF(current_setting('app.allowed_tenants', true), '')::uuid[],
        ARRAY[NULLIF(current_setting('app.current_tenant', true), '')::uuid])))
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- Байгаа өгөгдөл. Хоосон утгатай мөрийг оруулахгүй: албан тушаалгүй, хэлтэсгүй
-- гишүүнчлэлд энэ хүснэгтэд мөр байх шаардлагагүй, LEFT JOIN нь ижил хариу өгнө.
INSERT INTO organisation_people (membership_id, tenant_id, job_title, department_id)
SELECT id, tenant_id, COALESCE(job_title, ''), department_id
  FROM memberships
 WHERE COALESCE(job_title, '') <> '' OR department_id IS NOT NULL
ON CONFLICT (membership_id) DO NOTHING;

ALTER TABLE memberships DROP COLUMN IF EXISTS job_title;
ALTER TABLE memberships DROP COLUMN IF EXISTS department_id;

-- +goose Down

ALTER TABLE memberships ADD COLUMN IF NOT EXISTS job_title VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE memberships ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE SET NULL;

UPDATE memberships ms
   SET job_title = op.job_title, department_id = op.department_id
  FROM organisation_people op
 WHERE op.membership_id = ms.id;

DROP TABLE IF EXISTS organisation_people;
