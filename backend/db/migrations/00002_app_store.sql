-- +goose Up
CREATE TABLE IF NOT EXISTS apps (
    id VARCHAR(128) PRIMARY KEY, -- e.g. io.example.contacts
    slug VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon_url TEXT NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT 'General',
    visibility VARCHAR(32) NOT NULL DEFAULT 'public',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id VARCHAR(128) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    platform_constraint VARCHAR(128) NOT NULL DEFAULT '>=0.1.0',
    manifest JSONB NOT NULL,
    package_url TEXT,
    package_sha256 TEXT,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (app_id, version)
);

CREATE TABLE IF NOT EXISTS app_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_version_id UUID NOT NULL REFERENCES app_versions(id) ON DELETE CASCADE,
    dependency_app_id VARCHAR(128) NOT NULL,
    version_constraint VARCHAR(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS app_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    app_id VARCHAR(128) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    installed_version VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'installed',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, app_id)
);

CREATE TABLE IF NOT EXISTS installation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL REFERENCES app_installations(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_installations_tenant ON app_installations(tenant_id);
CREATE INDEX idx_app_versions_app ON app_versions(app_id);

-- +goose Down
DROP TABLE IF EXISTS installation_events;
DROP TABLE IF EXISTS app_installations;
DROP TABLE IF EXISTS app_dependencies;
DROP TABLE IF EXISTS app_versions;
DROP TABLE IF EXISTS apps;
