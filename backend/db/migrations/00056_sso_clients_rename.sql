-- The developer portal stops pretending to be one.
--
-- `developer_portal` named the wrong thing twice. There is a real developer
-- portal in this ecosystem — developer.gerege.mn, backed by the publisher
-- studio — where a third party submits an app to the store. This is not that:
-- it is CRUD over the OAuth2 clients registered against this platform's own
-- OIDC provider, used by whoever runs an organisation's integrations, and
-- nobody involved is a developer. An administrator looking for one and finding
-- the other had nothing in the name to tell them they were in the wrong
-- product.
--
-- So: `io.gerege.nexus.developer_portal` becomes `io.gerege.nexus.sso_clients`,
-- slug `developer_portal` becomes `sso-clients`, and the permissions become
-- `sso_clients.read` / `sso_clients.manage`.
--
-- The shape is 00055's, for the same reasons set out there, minus two sections
-- it does not need: this app registers no reports, so no report key moves, and
-- the app is not in DefaultApps, so no sweep has an opinion about it. What it
-- does have that 00055 did not is a slug containing an underscore becoming one
-- with a hyphen — the store URL is keyed by it.
--
-- Nothing here touches `oauth2_clients`. The clients this app manages are
-- keyed by their own client_id and have never carried the app's id; renaming
-- the screen does not rename anybody's integration.

-- +goose Up

ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_fkey;
ALTER TABLE app_versions DROP CONSTRAINT IF EXISTS app_versions_app_id_fkey;

-- An instance that synced a catalogue carrying the new id before this migration
-- ran already has both rows; the tenant's own installation is the one with the
-- history, so the premature row goes.
DELETE FROM app_installations dup
 WHERE dup.app_id = 'io.gerege.nexus.sso_clients'
   AND EXISTS (SELECT 1 FROM app_installations kept
                WHERE kept.tenant_id = dup.tenant_id
                  AND kept.app_id = 'io.gerege.nexus.developer_portal');

DELETE FROM app_versions v
 WHERE v.app_id = 'io.gerege.nexus.sso_clients'
   AND EXISTS (SELECT 1 FROM apps old WHERE old.id = 'io.gerege.nexus.developer_portal');

DELETE FROM apps a
 WHERE a.id = 'io.gerege.nexus.sso_clients'
   AND EXISTS (SELECT 1 FROM apps old WHERE old.id = 'io.gerege.nexus.developer_portal');

UPDATE apps
   SET id = 'io.gerege.nexus.sso_clients',
       slug = 'sso-clients',
       icon_url = CASE WHEN icon_url = '/icons/developer_portal.png'
                       THEN '/icons/sso-clients.png' ELSE icon_url END
 WHERE id = 'io.gerege.nexus.developer_portal';

UPDATE app_installations
   SET app_id = 'io.gerege.nexus.sso_clients'
 WHERE app_id = 'io.gerege.nexus.developer_portal';

UPDATE app_versions
   SET app_id = 'io.gerege.nexus.sso_clients'
 WHERE app_id = 'io.gerege.nexus.developer_portal';

-- The stored manifest carries the id and the permission codes, and an upgrade
-- diffs the catalogue against it: left stale, every renamed permission would
-- read as newly requested and hold the upgrade for an approval nobody can give.
UPDATE app_versions
   SET manifest = replace(replace(replace(manifest::text,
                    '"io.gerege.nexus.developer_portal"', '"io.gerege.nexus.sso_clients"'),
                    '"developer.read"', '"sso_clients.read"'),
                    '"developer.manage"', '"sso_clients.manage"')::jsonb
 WHERE manifest::text LIKE '%io.gerege.nexus.developer_portal%'
    OR manifest::text LIKE '%"developer.read"%'
    OR manifest::text LIKE '%"developer.manage"%';

UPDATE app_dependencies
   SET dependency_app_id = 'io.gerege.nexus.sso_clients'
 WHERE dependency_app_id = 'io.gerege.nexus.developer_portal';

ALTER TABLE app_installations
    ADD CONSTRAINT app_installations_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
ALTER TABLE app_versions
    ADD CONSTRAINT app_versions_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

-- Permissions, renamed in place so every role that holds one keeps holding it.
-- The merge-then-delete is for the instance that already registered the new
-- codes by installing under the new id.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, keep.id
  FROM role_permissions rp
  JOIN permissions dup  ON dup.id = rp.permission_id
  JOIN permissions keep ON keep.code = 'developer' || substr(dup.code, length('sso_clients') + 1)
 WHERE dup.code IN ('sso_clients.read', 'sso_clients.manage')
ON CONFLICT DO NOTHING;

DELETE FROM permissions
 WHERE code IN ('sso_clients.read', 'sso_clients.manage')
   AND EXISTS (SELECT 1 FROM permissions old
                WHERE old.code = 'developer' || substr(permissions.code, length('sso_clients') + 1));

UPDATE permissions
   SET code = 'sso_clients.read',
       name = 'Read SSO Clients'
 WHERE code = 'developer.read';
UPDATE permissions
   SET code = 'sso_clients.manage',
       name = 'Manage SSO Clients'
 WHERE code = 'developer.manage';

-- The module kill switch, whose key is derived from the app id.
ALTER TABLE feature_flag_overrides DROP CONSTRAINT IF EXISTS feature_flag_overrides_flag_key_fkey;

DELETE FROM feature_flags
 WHERE key = 'module.io.gerege.nexus.sso_clients.disabled'
   AND EXISTS (SELECT 1 FROM feature_flags old
                WHERE old.key = 'module.io.gerege.nexus.developer_portal.disabled');

UPDATE feature_flags SET key = 'module.io.gerege.nexus.sso_clients.disabled'
 WHERE key = 'module.io.gerege.nexus.developer_portal.disabled';
UPDATE feature_flag_overrides SET flag_key = 'module.io.gerege.nexus.sso_clients.disabled'
 WHERE flag_key = 'module.io.gerege.nexus.developer_portal.disabled';

DELETE FROM feature_flag_overrides o
 WHERE NOT EXISTS (SELECT 1 FROM feature_flags f WHERE f.key = o.flag_key);

ALTER TABLE feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_flag_key_fkey
    FOREIGN KEY (flag_key) REFERENCES feature_flags(key) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_fkey;
ALTER TABLE app_versions DROP CONSTRAINT IF EXISTS app_versions_app_id_fkey;

UPDATE apps
   SET id = 'io.gerege.nexus.developer_portal',
       slug = 'developer_portal',
       icon_url = CASE WHEN icon_url = '/icons/sso-clients.png'
                       THEN '/icons/developer_portal.png' ELSE icon_url END
 WHERE id = 'io.gerege.nexus.sso_clients';

UPDATE app_installations
   SET app_id = 'io.gerege.nexus.developer_portal'
 WHERE app_id = 'io.gerege.nexus.sso_clients';

UPDATE app_versions
   SET app_id = 'io.gerege.nexus.developer_portal'
 WHERE app_id = 'io.gerege.nexus.sso_clients';

UPDATE app_versions
   SET manifest = replace(replace(replace(manifest::text,
                    '"io.gerege.nexus.sso_clients"', '"io.gerege.nexus.developer_portal"'),
                    '"sso_clients.read"', '"developer.read"'),
                    '"sso_clients.manage"', '"developer.manage"')::jsonb
 WHERE manifest::text LIKE '%io.gerege.nexus.sso_clients%'
    OR manifest::text LIKE '%"sso_clients.read"%'
    OR manifest::text LIKE '%"sso_clients.manage"%';

UPDATE app_dependencies
   SET dependency_app_id = 'io.gerege.nexus.developer_portal'
 WHERE dependency_app_id = 'io.gerege.nexus.sso_clients';

ALTER TABLE app_installations
    ADD CONSTRAINT app_installations_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
ALTER TABLE app_versions
    ADD CONSTRAINT app_versions_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

UPDATE permissions
   SET code = 'developer.read',
       name = 'Read Developer Apps'
 WHERE code = 'sso_clients.read';
UPDATE permissions
   SET code = 'developer.manage',
       name = 'Manage Developer Apps'
 WHERE code = 'sso_clients.manage';

ALTER TABLE feature_flag_overrides DROP CONSTRAINT IF EXISTS feature_flag_overrides_flag_key_fkey;

UPDATE feature_flags SET key = 'module.io.gerege.nexus.developer_portal.disabled'
 WHERE key = 'module.io.gerege.nexus.sso_clients.disabled';
UPDATE feature_flag_overrides SET flag_key = 'module.io.gerege.nexus.developer_portal.disabled'
 WHERE flag_key = 'module.io.gerege.nexus.sso_clients.disabled';

ALTER TABLE feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_flag_key_fkey
    FOREIGN KEY (flag_key) REFERENCES feature_flags(key) ON DELETE CASCADE;
