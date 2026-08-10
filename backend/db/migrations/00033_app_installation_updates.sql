-- An installation can now be behind the catalogue, so it needs an opinion about
-- what should happen next.
--
-- auto_update is TRUE by default because that is what the platform already did
-- in effect: reinstalling an app took whatever the catalogue carried, and no
-- tenant was ever asked. Making it a column is what lets an administrator say
-- otherwise — an app whose new version widens the permissions it asks for, or a
-- third-party service a tenant wants to hold at a known-good version, stays
-- where it is until somebody decides to move it.
--
-- pinned_version is that decision written down: the version this tenant is held
-- at regardless of what the catalogue publishes. NULL means "follow the
-- catalogue", which is every existing row. It is nullable rather than defaulted
-- to installed_version precisely so the two states stay distinguishable —
-- "nobody has pinned this" is not the same fact as "this is pinned to the
-- version it happens to be on".

-- +goose Up
ALTER TABLE app_installations
    ADD COLUMN IF NOT EXISTS auto_update BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS pinned_version VARCHAR(32);

-- +goose Down
ALTER TABLE app_installations
    DROP COLUMN IF EXISTS pinned_version,
    DROP COLUMN IF EXISTS auto_update;
