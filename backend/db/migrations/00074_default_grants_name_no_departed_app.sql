-- Шинэ тенант үүсэхэд олгогдох эрхийн жагсаалтаас `gov.*` устав.
--
-- 00008 нь `seed_tenant_access_roles()` trigger функцэд дөрвөн `gov.*`
-- кодыг manager рольд, `gov.apply`-г user рольд гараар нэрлэсэн. Тэр нь
-- хэрэглэгдсэн миграцын түүх биш — trigger нь **өнөөдөр ч** тенант үүсэх
-- бүрд ажилладаг амьд код.
--
-- gov-services апп `gerege-gov` руу явсан. Энэ бинари дотор тэр таван
-- эрхийн аль нь ч байхгүй (`internal/apps/egov` нь `egov.*` зарладаг), тул
-- жагсаалт юутай ч таарахгүй бөгөөд ямар ч мөр нэмэгдэхгүй. Өөрөөр хэлбэл
-- энэ өөрчлөлт зан төлөвийг хөдөлгөхгүй.
--
-- Гэсэн ч устгах шалтгаан бий: `internal/apps/boundaries_test.go:88`-д
-- бичигдсэн түүх. Платформ дээр аппын ID-гаар түлхүүрлэсэн хоёр switch
-- байсан, App Store явахад тэдгээр нь таарахаа больж, тэр
-- бүтээгдэхүүний маршрут бүр гишүүн бүрд нээгдсэн. Хор хөнөөлгүй хувилбар
-- нь аюултай хувилбарын урьдчилсан хэлбэр: аль нэг өдөр `gov.process`
-- гэдэг код өөр утгаар эргэж ирвэл энэ trigger түүнийг чимээгүй тарааж
-- эхэлнэ.
--
-- Эрхийн анхдагч хүрээг одооноос апп өөрөө зарлана
-- (`nexus.PermissionDefinition.DefaultRoles`). Функц дэх `LIKE '%.read'`
-- дүрэм нь тэр зарлалын өмнөх нөөц бөгөөд хэвээр үлдэнэ — устгах нь
-- зарлаагүй модулиудын эрхийг тасална. Тэр нь Үе 4b-ийн ажил биш, нэршлийн
-- ёсыг гэрээ байхаа болиулах v2-ын ажил.

-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION seed_tenant_access_roles() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO roles (tenant_id, code, name, description, is_system)
    VALUES
        (NEW.id, 'admin', 'Administrator', 'Full access to this organisation', TRUE),
        (NEW.id, 'manager', 'Manager', 'Operational access to installed apps', TRUE),
        (NEW.id, 'user', 'User', 'Standard read access and self-service actions', TRUE)
    ON CONFLICT (tenant_id, code) DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = NEW.id AND (
        r.code = 'admin'
        OR (r.code = 'manager' AND (p.code LIKE '%.read' OR p.code LIKE '%.manage'))
        OR (r.code = 'user' AND p.code LIKE '%.read')
    ) ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION seed_tenant_access_roles() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO roles (tenant_id, code, name, description, is_system)
    VALUES
        (NEW.id, 'admin', 'Administrator', 'Full access to this organisation', TRUE),
        (NEW.id, 'manager', 'Manager', 'Operational access to installed apps', TRUE),
        (NEW.id, 'user', 'User', 'Standard read access and self-service actions', TRUE)
    ON CONFLICT (tenant_id, code) DO NOTHING;

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = NEW.id AND (
        r.code = 'admin'
        OR (r.code = 'manager' AND (p.code LIKE '%.read' OR p.code LIKE '%.manage' OR p.code IN ('gov.process','gov.delegate','gov.verify','gov.report')))
        OR (r.code = 'user' AND (p.code LIKE '%.read' OR p.code = 'gov.apply'))
    ) ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
