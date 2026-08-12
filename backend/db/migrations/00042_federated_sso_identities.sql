-- Өөр OpenID Connect провайдер дээрх хэн болохыг энд байгаа бүртгэлтэй холбоно.
--
-- SSO_CLIENT_ISSUER тохируулагдсан суулгац нь хүнийг өөрөө таньдаггүй — тэр
-- асуултыг провайдер хариулж, энэ хүснэгт хариултыг нь эндэх хэрэглэгчтэй
-- холбож өгнө.
--
-- Түлхүүр нь (issuer, subject). И-мэйл биш: и-мэйл бол провайдер өөрчилж
-- болох шошго бөгөөд гарсан ажилтны хаягийг өөр хүнд өгвөл тэр хүн түүний
-- бүртгэлийг өвлөнө. subject нь провайдерийн тогтвортой байлгахаа амласан,
-- дахин хуваарилахгүй гэсэн цорын ганц claim. issuer нь хамт байгаа шалтгаан
-- нь: хоёр өөр провайдер нэг subject-ийг тус тусдаа өөр хүнд өгч болно.
--
-- users хүснэгтээс тусдаа байгаа нь user_eid_identities-тэй яг адил
-- шалтгаантай: холбоос нь заавал байх ёсгүй, өөрийн гэсэн амьдралын
-- мөчлөгтэй, тусдаа байснаар гадны танигчийг санамсаргүй сонгож уншихгүй.
--
-- tenant_id байхгүй тул RLS-ийн бодлого ч байхгүй: нэг хүн хэд хэдэн
-- байгууллагад гишүүн байж болох ба энэ холбоос тэдгээрийн аль нэгэнд нь
-- харьяалагдахгүй. user_eid_identities мөн ийм.

-- +goose Up
CREATE TABLE IF NOT EXISTS user_sso_identities (
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issuer       TEXT        NOT NULL,
    subject      TEXT        NOT NULL,
    email        TEXT,
    name         TEXT,
    linked_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (issuer, subject)
);

-- "Энэ хэрэглэгэн ямар холбоостой вэ" гэсэн эсрэг чиглэлийн асуулт: бүртгэл
-- устгах, холбоосыг таслах үед хэрэгтэй.
CREATE INDEX IF NOT EXISTS idx_user_sso_identities_user
    ON user_sso_identities (user_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON user_sso_identities TO gerege_nexus_app;

-- +goose Down
DROP TABLE IF EXISTS user_sso_identities;
