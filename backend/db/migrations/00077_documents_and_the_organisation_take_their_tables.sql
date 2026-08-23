-- Баримт бичиг ба байгууллага хүснэгтээ авч явлаа.
--
-- Арван нэгэн хүснэгт: document_* ес, departments, organisation_people.
-- Хоёул апп нь 2026-08-23-нд client-gerege-nexus руу нүүсэн бөгөөд схемээ
-- өөрсдөө үүрч явав — modules/documents/migrations/00001_documents.sql ба
-- modules/organisation/migrations/00001_organisation.sql. nexus.Migrations
-- (Үе 3) байгаагүй бол энэ боломжгүй байх байсан: хүснэгт үүсгэх цорын ганц
-- газар нь платформын миграцын хавтас байсан.
--
-- 00075 нь явсан аппуудын 28 хүснэгтийг устгасан. Тэр устгал нь аюулгүй
-- байсан шалтгаан нь маршрутыг эхэлж тоолсонд байв — цөм тэдгээр аппын нэг ч
-- endpoint үйлчилдэггүй байсан тул хүснэгтийг нь унших боломжгүй байсан. Энэ
-- нь өөр: цөм өчигдөр хүртэл эдгээр хүснэгтийг уншиж байсан. Тиймээс энэ
-- миграц нь кодоос салсны ДАРАА л зөв — internal/apps/documents ба
-- internal/apps/organisation энэ commit-д устсан, тэдний уншдаг байсан
-- бүхэн тэдэнтэй хамт явсан.
--
-- ӨГӨГДӨЛ УСТАНА. Энэ бол ил тод шийдвэр: аль ч суулгац production-д ороогүй,
-- эзэн нь үүнийг мэдэж баталсан. Байсан бол зөөлт нь өөр хэлбэртэй байх байв
-- — хүснэгтийг үлдээж, аппын миграц нь `IF NOT EXISTS`-ээр өөрийнхөө гэж
-- зарлаад, өгөгдлийг байрандаа үлдээх. Тэр замыг аппын миграцууд одоо ч
-- дэмжинэ; энд сонгосонгүй, учир нь устгахгүй бол цөм нь юу үүрч байгаагаа
-- мэдэхээ болино.
--
-- Дараалал нь гадаад түлхүүрийнх: document_files ба бусад нь
-- document_records руу заана, organisation_people нь departments руу.
-- CASCADE-гүй: хамаарал үлдсэн бол энэ миграц унах ёстой бөгөөд тэр нь
-- чимээгүй устгахаас дээр.

-- +goose Up

DROP TABLE IF EXISTS document_eid_sign_sessions;
DROP TABLE IF EXISTS document_signatures;
DROP TABLE IF EXISTS document_approval_steps;
DROP TABLE IF EXISTS document_files;
DROP TABLE IF EXISTS document_workflow_steps;
DROP TABLE IF EXISTS document_templates;
DROP TABLE IF EXISTS document_signature_policies;
DROP TABLE IF EXISTS document_retention_rules;
DROP TABLE IF EXISTS document_records;

DROP TABLE IF EXISTS organisation_people;
DROP TABLE IF EXISTS departments;

-- +goose Down

-- Буцаалт нь хүснэгтийг сэргээхгүй.
--
-- Сэргээх гэсэн оролдлого нь худал байх байсан: хоосон хүснэгт нь өгөгдлөө
-- буцаахгүй, харин goose-д буцаалт амжилттай болсон гэж бичигдэнэ. Хүснэгт
-- хэрэгтэй бол тэдгээрийг эзэмшдэг аппыг суулгах нь тэднийг үүсгэнэ — энэ бол
-- одоо тэдний оршин байх ганц зам.
SELECT 1;
