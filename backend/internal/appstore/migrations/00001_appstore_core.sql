-- appstore_db — the registry behind appstore.gerege.mn.
--
-- This is a different database from the platform's. Nothing here knows about
-- tenants, sessions or installations: those belong to each Nexus instance and
-- stay there. What lives here is the published catalogue — who publishes an
-- app, which versions exist, what each version's manifest says, and which of
-- them a given platform version is allowed to see.
--
-- Identity note: an app's primary key is its reverse-DNS id (io.example.esign),
-- not a surrogate uuid. That id is what every Nexus instance already stores in
-- app_installations.app_id, so making the registry agree with it is what lets a
-- catalogue served from here be the same catalogue an instance already has.

-- +goose Up

CREATE TABLE IF NOT EXISTS publishers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL DEFAULT '',
    -- Verified means somebody checked this publisher is who they say they are.
    -- Nothing is published under an unverified publisher without review.
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    -- Who owns this publisher, as Gerege SSO knows them. `sub` is the stable
    -- user id in the id_token; the tenant is recorded because a publisher is an
    -- organisation acting through a person, and the person can change.
    owner_sub VARCHAR(128) NOT NULL,
    owner_email VARCHAR(255) NOT NULL DEFAULT '',
    owner_tenant_id VARCHAR(64) NOT NULL DEFAULT '',
    owner_tenant_slug VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_publishers_owner ON publishers(owner_sub);

CREATE TABLE IF NOT EXISTS store_apps (
    id VARCHAR(128) PRIMARY KEY,
    publisher_id UUID NOT NULL REFERENCES publishers(id) ON DELETE RESTRICT,
    slug VARCHAR(64) UNIQUE NOT NULL,
    -- 'module' is compiled into the Nexus binary; 'external' is a third party's
    -- own running service. The distinction travels in the manifest too — this
    -- column is what the storefront filters and sorts by.
    type VARCHAR(16) NOT NULL DEFAULT 'module',
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon_url TEXT NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT 'General',
    visibility VARCHAR(32) NOT NULL DEFAULT 'public',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_store_apps_publisher ON store_apps(publisher_id);

CREATE TABLE IF NOT EXISTS store_app_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id VARCHAR(128) NOT NULL REFERENCES store_apps(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    channel VARCHAR(16) NOT NULL DEFAULT 'stable',
    -- The platform constraint out of the manifest, hoisted into a column so the
    -- catalogue endpoint can filter by it in SQL rather than by decoding every
    -- manifest on every request.
    min_platform VARCHAR(128) NOT NULL DEFAULT '>=0.1.0',
    -- The manifest exactly as it will be served: appcatalog.Manifest, including
    -- the type/external fields. A Nexus instance validates it again on arrival
    -- and discards the whole document if it fails, so this is validated here
    -- before it is ever published.
    manifest JSONB NOT NULL,
    package_url TEXT,
    package_sha256 TEXT,
    -- draft → in_review → published, or rejected; yanked once withdrawn.
    status VARCHAR(16) NOT NULL DEFAULT 'in_review',
    submitted_by VARCHAR(128) NOT NULL DEFAULT '',
    review_note TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (app_id, version)
);

CREATE INDEX IF NOT EXISTS idx_store_app_versions_app ON store_app_versions(app_id);
CREATE INDEX IF NOT EXISTS idx_store_app_versions_status ON store_app_versions(status);

-- The translatable part of a catalogue entry, one row per locale. The platform
-- resolves these before answering a browser, so a tenant never sees an app
-- described in a language they did not ask for.
CREATE TABLE IF NOT EXISTS store_app_texts (
    app_id VARCHAR(128) NOT NULL REFERENCES store_apps(id) ON DELETE CASCADE,
    locale VARCHAR(8) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    PRIMARY KEY (app_id, locale)
);

-- Everything an external app needs beyond a manifest. It duplicates what the
-- manifest carries on purpose: this is the queryable form, used by the review
-- screens and by anything that has to answer "which app owns this client id".
CREATE TABLE IF NOT EXISTS external_registrations (
    app_id VARCHAR(128) PRIMARY KEY REFERENCES store_apps(id) ON DELETE CASCADE,
    launch_url TEXT NOT NULL,
    sso_client_id VARCHAR(128) NOT NULL DEFAULT '',
    scopes TEXT[] NOT NULL DEFAULT '{}',
    embed VARCHAR(16) NOT NULL DEFAULT 'new_tab',
    health_url TEXT NOT NULL DEFAULT '',
    webhook_url TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS review_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES store_app_versions(id) ON DELETE CASCADE,
    actor VARCHAR(255) NOT NULL,
    action VARCHAR(32) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_review_events_version ON review_events(version_id);

-- The signed catalogue, stored as the bytes that were signed.
--
-- The signature covers the raw bytes of the apps array, so those bytes must
-- reach the client unchanged. Rebuilding the document per request would mean
-- trusting Go's map ordering and number formatting to be identical every time —
-- and a single byte of drift makes every instance in the field reject the
-- catalogue and silently stop taking updates. So the document is built once per
-- revision, kept here verbatim, and served as-is. The ETag falls out of the
-- same bytes, which is what makes a 304 cheap and honest.
CREATE TABLE IF NOT EXISTS catalog_snapshots (
    channel VARCHAR(16) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    revision BIGINT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    etag VARCHAR(80) NOT NULL,
    document BYTEA NOT NULL,
    built_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel, platform)
);

-- One row, holding the number that says "the catalogue changed". Anything that
-- publishes, yanks or edits catalogue content bumps it; a snapshot built under
-- an older revision is rebuilt rather than served.
CREATE TABLE IF NOT EXISTS registry_state (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    revision BIGINT NOT NULL DEFAULT 1,
    CONSTRAINT registry_state_single_row CHECK (id)
);

INSERT INTO registry_state (id, revision) VALUES (TRUE, 1) ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS registry_state;
DROP TABLE IF EXISTS catalog_snapshots;
DROP TABLE IF EXISTS review_events;
DROP TABLE IF EXISTS external_registrations;
DROP TABLE IF EXISTS store_app_texts;
DROP TABLE IF EXISTS store_app_versions;
DROP TABLE IF EXISTS store_apps;
DROP TABLE IF EXISTS publishers;
