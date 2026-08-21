-- The canary's own table, brought by the module rather than by the platform.
-- See pkg/nexus/migrations.go.

-- +goose Up
CREATE TABLE IF NOT EXISTS canary_quotes (
    id        uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    sku       text NOT NULL,
    price     bigint NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS canary_quotes;
