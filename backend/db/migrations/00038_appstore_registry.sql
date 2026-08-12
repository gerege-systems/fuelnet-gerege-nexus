-- +goose Up

-- The App Store registry, as tables this platform owns.
--
-- Until now the registry was a service of its own with a schema of its own,
-- because the App Store was split out of this repository. It is coming back as
-- three modules — see docs/APPSTORE_PHASE2_PLAN.md §3 — and this is the schema
-- they run on. The instance that mounts them is an ordinary Nexus instance
-- whose catalogue happens to list those three apps; every other instance
-- carries these tables empty and never looks at them, exactly as it carries the
-- gov_services tables when nobody has installed gov_services.
--
-- The one modelling change is the point of the whole exercise:
--
--   a publisher is a tenant.
--
-- The separate service had `owner_sub`, `owner_email`, `owner_tenant_id` and
-- `owner_tenant_slug` on the publisher row, and one person owned one publisher.
-- A team belongs to an organisation, and the code that shipped that arrangement
-- said so in a comment. Here the organisation already exists: it is a tenant,
-- with memberships, roles, an audit trail, seven languages and E-ID
-- verification of the legal entity — all of which the registry was going to
-- have to reinvent, worse. So `store_publishers` is a tenant's publishing
-- profile, and who may submit on its behalf is a role its own administrator
-- grants.

-- A tenant's publishing identity. One per tenant, hence the unique constraint.
CREATE TABLE IF NOT EXISTS store_publishers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    slug          VARCHAR(64) UNIQUE NOT NULL,
    name          VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL DEFAULT '',
    -- Verified means somebody checked this publisher is who they say they are.
    -- On this platform that check has an answer already: a tenant whose legal
    -- entity has been confirmed against the national registry. Nothing is
    -- published under an unverified publisher without review either way.
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- An app somebody publishes. Not this platform's own `apps` table, which is
-- what a tenant has *installed*: this is what the registry *offers*.
CREATE TABLE IF NOT EXISTS store_apps (
    id           VARCHAR(128) PRIMARY KEY,
    publisher_id UUID NOT NULL REFERENCES store_publishers(id) ON DELETE RESTRICT,
    slug         VARCHAR(64) UNIQUE NOT NULL,
    type         VARCHAR(16)  NOT NULL DEFAULT 'module',
    name         VARCHAR(255) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    icon_url     TEXT NOT NULL DEFAULT '',
    category     VARCHAR(64) NOT NULL DEFAULT 'General',
    visibility   VARCHAR(32) NOT NULL DEFAULT 'public',
    -- Manifest v2.1 provenance, hoisted out of the manifest so the storefront
    -- can render a credit line without decoding one.
    authors     JSONB NOT NULL DEFAULT '[]',
    maintainers JSONB NOT NULL DEFAULT '[]',
    repository  TEXT NOT NULL DEFAULT '',
    homepage    TEXT NOT NULL DEFAULT '',
    license     VARCHAR(128) NOT NULL DEFAULT '',
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_store_apps_publisher ON store_apps(publisher_id);

CREATE TABLE IF NOT EXISTS store_app_versions (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id  VARCHAR(128) NOT NULL REFERENCES store_apps(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    channel VARCHAR(16) NOT NULL DEFAULT 'stable',
    -- The platform constraint out of the manifest, hoisted into a column so the
    -- catalogue endpoint can filter by it in SQL rather than by decoding every
    -- manifest on every request.
    min_platform VARCHAR(128) NOT NULL DEFAULT '>=0.1.0',
    -- The manifest exactly as it will be served. An instance validates it again
    -- on arrival and discards the whole document if it fails, so it is
    -- validated here before it is ever published.
    manifest       JSONB NOT NULL,
    release_notes  JSONB,
    authors        JSONB NOT NULL DEFAULT '[]',
    package_url    TEXT,
    package_sha256 TEXT,
    -- draft → in_review → published, or rejected; yanked once withdrawn.
    status       VARCHAR(16) NOT NULL DEFAULT 'in_review',
    submitted_by VARCHAR(128) NOT NULL DEFAULT '',
    review_note  TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (app_id, version)
);

CREATE INDEX IF NOT EXISTS idx_store_app_versions_app ON store_app_versions(app_id);
CREATE INDEX IF NOT EXISTS idx_store_app_versions_status ON store_app_versions(status);
CREATE INDEX IF NOT EXISTS idx_store_app_versions_chronicle
    ON store_app_versions(app_id, published_at DESC) WHERE status = 'published';

-- Catalogue copy in the seven platform languages.
CREATE TABLE IF NOT EXISTS store_app_texts (
    app_id      VARCHAR(128) NOT NULL REFERENCES store_apps(id) ON DELETE CASCADE,
    locale      VARCHAR(8) NOT NULL,
    name        VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category    VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (app_id, locale)
);

-- An external app's registration, in queryable form.
CREATE TABLE IF NOT EXISTS store_external_registrations (
    app_id        VARCHAR(128) PRIMARY KEY REFERENCES store_apps(id) ON DELETE CASCADE,
    launch_url    TEXT NOT NULL,
    sso_client_id VARCHAR(128) NOT NULL DEFAULT '',
    scopes        TEXT[] NOT NULL DEFAULT '{}',
    embed         VARCHAR(16) NOT NULL DEFAULT 'new_tab',
    health_url    TEXT NOT NULL DEFAULT '',
    webhook_url   TEXT NOT NULL DEFAULT ''
);

-- What happened to a version, and who decided it.
CREATE TABLE IF NOT EXISTS store_review_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES store_app_versions(id) ON DELETE CASCADE,
    actor_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    actor      VARCHAR(255) NOT NULL,
    action     VARCHAR(32) NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_store_review_events_version ON store_review_events(version_id);

-- The signed catalogue, stored as the bytes that were signed.
--
-- The signature covers the raw bytes of the apps array, so those bytes must
-- reach the client unchanged. Rebuilding the document per request would mean
-- trusting Go's encoder to be identical every time — and a single byte of drift
-- makes every instance in the field reject the catalogue and silently stop
-- taking updates. So it is built once per revision, kept here verbatim, and
-- served as-is. The ETag falls out of the same bytes, which is what makes a 304
-- cheap and honest.
CREATE TABLE IF NOT EXISTS store_catalog_snapshots (
    channel      VARCHAR(16) NOT NULL,
    platform     VARCHAR(32) NOT NULL,
    revision     BIGINT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    etag         VARCHAR(80) NOT NULL,
    document     BYTEA NOT NULL,
    built_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel, platform)
);

-- One row, holding the number that says "the catalogue changed". Anything that
-- publishes, yanks or edits catalogue content bumps it; a snapshot built under
-- an older revision is rebuilt rather than served.
CREATE TABLE IF NOT EXISTS store_registry_state (
    id       BOOLEAN PRIMARY KEY DEFAULT TRUE,
    revision BIGINT NOT NULL DEFAULT 1,
    CONSTRAINT store_registry_state_single_row CHECK (id)
);

INSERT INTO store_registry_state (id, revision) VALUES (TRUE, 1) ON CONFLICT (id) DO NOTHING;

-- Row-level isolation for the one table here that carries a tenant.
--
-- Migration 00029 applied the policy by looping over every table with a
-- tenant_id, which cannot reach a table created afterwards. It is applied here
-- explicitly, in the same shape, so the invariant test in internal/platform/
-- tenant keeps passing and a publishing profile is readable only by the
-- organisation it belongs to.
--
-- The other tables carry no tenant_id and get no policy, deliberately: the
-- catalogue is public by nature, and the queries that build it run on the
-- platform path with no tenant bound. A publisher's own view of its apps is
-- filtered by publisher_id, resolved from the caller's tenant.
-- +goose StatementBegin
DO $rls$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_app') THEN
        GRANT USAGE ON SCHEMA public TO gerege_nexus_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            store_publishers, store_apps, store_app_versions, store_app_texts,
            store_external_registrations, store_review_events,
            store_catalog_snapshots, store_registry_state
            TO gerege_nexus_app;

        ALTER TABLE store_publishers ENABLE ROW LEVEL SECURITY;
        ALTER TABLE store_publishers FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON store_publishers;
        CREATE POLICY tenant_isolation ON store_publishers TO gerege_nexus_app
            USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);
    END IF;
END
$rls$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS store_registry_state;
DROP TABLE IF EXISTS store_catalog_snapshots;
DROP TABLE IF EXISTS store_review_events;
DROP TABLE IF EXISTS store_external_registrations;
DROP TABLE IF EXISTS store_app_texts;
DROP TABLE IF EXISTS store_app_versions;
DROP TABLE IF EXISTS store_apps;
DROP TABLE IF EXISTS store_publishers;
