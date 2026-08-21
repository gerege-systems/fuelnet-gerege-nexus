-- Явсан аппуудын 28 хүснэгт цөмөөс гарав.
--
-- db/migrations нь платформын схем бөгөөд хүснэгт үүсгэх цорын ганц газар
-- байсан. Тиймээс 108 хүснэгтийн 28 нь энэ репод байхаа больсон модулиудынх
-- байв: commerce нь business-gerege-nexus руу, төрийн үйлчилгээ gerege-gov
-- руу, касс pos-gerege-nexus руу, бүртгэл appstore-gerege-mn руу. Тэдгээр нь
-- дагаж явж чадаагүй тул суулгац бүр хэн нэгний хэзээ нэгэн цагт энд бичсэн
-- аппын схемийг үүрч байв.
--
-- Үе 3 (nexus.Migrations) нь модульд өөрийн схемээ авч явах боломж өгсөн, энэ
-- нь тэр боломжийг ашиглаж байгаа нь. business-gerege-nexus нь өөрийн
-- 00001_commerce.sql-д products, warehouses, stock_levels, stock_movements,
-- billing_invoices-ыг `IF NOT EXISTS`-ээр аль хэдийн зарладаг — тэр файлын
-- тайлбар яг энэ өдрийг тооцоолж бичигдсэн байна.
--
-- `contacts` нь тусдаа тохиолдол бөгөөд илүү энгийн: түүнийг **хэн ч уншдаггүй**.
-- Цөмд production дуудагч байхгүй (зөвхөн гурван тестийн fixture), commerce
-- өөрөө "Commerce reads no contact rows: billing keeps a contact_name string
-- rather than a foreign key" гэж бичсэн, frontend-ийн дөрвөн дэлгэц нь цөмийн
-- мөнтөддөггүй /contacts endpoint рүү хандаж байв. Эзэнгүй хүснэгт.
--
-- Хасахын өмнө маршрутыг тоолов: цөм эдгээр аппын нэг ч замыг мөнтөддөггүй
-- байсан — contacts 0, products 0, inventory 0, billing 0, pos 0, gov 0,
-- publisher 0, store-review 0, appstore 0. Ганц үлдэц нь /devices/shifts-ийн
-- гурав, тэр нь энэ өөрчлөлтөд хамт хасагдав.
--
-- CASCADE: эдгээрийн хооронд гадаад түлхүүр бий (gov_* нь бие бие рүүгээ,
-- stock_levels нь products руу). Дарааллаар нь бичихийн оронд CASCADE — юу нь
-- юунаас хамаарахыг өгөгдлийн сан аль хэдийн мэддэг, түүнийг энд гараар
-- давтах нь дараагийн өөрчлөлтөд хуучрах жагсаалт нэмнэ.

-- +goose Up

-- commerce → business-gerege-nexus (5). Тэр repo эдгээрийг өөрөө үүсгэдэг.
DROP TABLE IF EXISTS stock_movements CASCADE;
DROP TABLE IF EXISTS stock_levels CASCADE;
DROP TABLE IF EXISTS warehouses CASCADE;
DROP TABLE IF EXISTS billing_invoices CASCADE;
DROP TABLE IF EXISTS products CASCADE;

-- Эзэнгүй (1). Цөм ч, commerce ч уншдаггүй.
DROP TABLE IF EXISTS contacts CASCADE;

-- төрийн үйлчилгээ → gerege-gov (14)
DROP TABLE IF EXISTS gov_application_events CASCADE;
DROP TABLE IF EXISTS gov_applications CASCADE;
DROP TABLE IF EXISTS gov_appointments CASCADE;
DROP TABLE IF EXISTS gov_delivery_outbox CASCADE;
DROP TABLE IF EXISTS gov_tasks CASCADE;
DROP TABLE IF EXISTS gov_unit_members CASCADE;
DROP TABLE IF EXISTS gov_org_units CASCADE;
DROP TABLE IF EXISTS gov_routing_rules CASCADE;
DROP TABLE IF EXISTS gov_workflow_transitions CASCADE;
DROP TABLE IF EXISTS gov_workflow_steps CASCADE;
DROP TABLE IF EXISTS gov_workflow_versions CASCADE;
DROP TABLE IF EXISTS gov_workflows CASCADE;
DROP TABLE IF EXISTS gov_upstream_connectors CASCADE;
DROP TABLE IF EXISTS gov_services CASCADE;

-- касс → pos-gerege-nexus (1)
DROP TABLE IF EXISTS pos_shifts CASCADE;

-- бүртгэл → appstore-gerege-mn (7).
--
-- Суулгацын дэлгүүрийн дэлгэцүүд эдгээрийг уншдаггүй: тэдгээр нь `apps`,
-- `app_installations`, `store_app_versions` дээр ажилладаг бөгөөд тэр гурав
-- цөмд үлдэнэ. Энд байгаа нь бүртгэлийн өөрийнх — хэн нийтэлдэг, юу хяналтад
-- байгаа, сүүлийн каталогийн build юу гаргасан.
DROP TABLE IF EXISTS store_review_events CASCADE;
DROP TABLE IF EXISTS store_external_registrations CASCADE;
DROP TABLE IF EXISTS store_catalog_snapshots CASCADE;
DROP TABLE IF EXISTS store_app_texts CASCADE;
DROP TABLE IF EXISTS store_apps CASCADE;
DROP TABLE IF EXISTS store_publishers CASCADE;
DROP TABLE IF EXISTS store_registry_state CASCADE;

-- +goose Down

-- Буцаах зам байхгүй, бөгөөд байх ёсгүй.
--
-- Эдгээр хүснэгтийн тодорхойлолт нь өөрсдийнх нь repo-д байна: commerce-ийнх
-- business-gerege-nexus/migrations/00001_commerce.sql-д, бусад нь өөрсдийн
-- distribution-д. Тэдгээрийг энд дахин бичих нь яг устгаж байгаа зүйлээ —
-- аппын схемийг платформ дотор хадгалахыг — буцааж авчирна.
--
-- Хэрэв эдгээр хүснэгт дахин хэрэгтэй бол тэр нь аппаа суулгаж байна гэсэн үг
-- бөгөөд аппын өөрийн миграц (nexus.Migrations) түүнийг үүсгэнэ.
SELECT 1;
