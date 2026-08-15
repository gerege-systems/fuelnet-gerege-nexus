-- Contacts stops being an app of its own and becomes part of the Directory.
--
-- Two cards in the store for one subject cut in half: who this organisation is
-- made of, and who it deals with. An administrator who installed one and not
-- the other had half a directory, and nothing in the store said which half was
-- missing. So `io.gerege.nexus.contacts` is removed and its register is part of
-- `io.gerege.nexus.organisation`, which keeps its id and is now called
-- Directory.
--
-- The shape is 00058's, for the reason given there: an absorption has to
-- transfer everything *before* it deletes anything, or somebody loses access
-- they were relying on this morning. Permissions, then grants, then the app row
-- and the rows that cascade from it.
--
-- Two things are different here, and both are because this app is installed by
-- default (appinstaller.DefaultApps):
--
--   * every tenant already has the surviving app, so unlike 00058 there is no
--     tenant to install it for — only disabled ones to switch back on;
--   * the contact register therefore arrives everywhere, including at tenants
--     that never installed Contacts. That is a product decision the migration
--     carries out rather than makes: a directory with no outside half was the
--     thing being fixed.

-- +goose Up

-- 1. The surviving codes, in case a database has the app installed nowhere yet.
INSERT INTO permissions (id, code, name, description) VALUES
  (gen_random_uuid(), 'organisation.read',   'Read Directory',   'View the organisation profile, its departments, its people and its contacts'),
  (gen_random_uuid(), 'organisation.manage', 'Manage Directory', 'Edit the organisation profile, its departments, its people and its contacts')
ON CONFLICT (code) DO NOTHING;

-- Their descriptions widened with the merge; a database that already had them
-- would otherwise keep describing an app that no longer stops at its own staff.
UPDATE permissions SET name = 'Read Directory',
       description = 'View the organisation profile, its departments, its people and its contacts'
 WHERE code = 'organisation.read';
UPDATE permissions SET name = 'Manage Directory',
       description = 'Edit the organisation profile, its departments, its people and its contacts'
 WHERE code = 'organisation.manage';

-- 2. Every grant, one to one. Anybody who could read contacts can read the
--    directory; anybody who could edit them can edit it.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, new.id
  FROM role_permissions rp
  JOIN permissions old ON old.id = rp.permission_id
  JOIN permissions new ON new.code = 'organisation' || substr(old.code, length('contacts') + 1)
 WHERE old.code IN ('contacts.read', 'contacts.manage')
ON CONFLICT DO NOTHING;

DELETE FROM permissions WHERE code IN ('contacts.read', 'contacts.manage');

-- 3. A tenant using Contacts with the Directory switched off would lose the
--    register: `enabled` is what the app gate reads, and the contact routes are
--    mounted by the Directory module now.
UPDATE app_installations d
   SET enabled = TRUE, status = 'installed', updated_at = NOW()
  FROM app_installations c
 WHERE d.tenant_id = c.tenant_id
   AND d.app_id = 'io.gerege.nexus.organisation'
   AND c.app_id = 'io.gerege.nexus.contacts'
   AND c.enabled
   AND NOT d.enabled;

-- A tenant with Contacts and no Directory row at all would be left with
-- neither until the next boot. EnsureDefaultApps would put it back — it is a
-- default app — but "a sweep will fix it when the process restarts" is not
-- something a migration should leave behind when one INSERT settles it here.
INSERT INTO app_installations (tenant_id, app_id, installed_version, status, enabled, installed_at)
SELECT c.tenant_id, 'io.gerege.nexus.organisation', '2.0.0', c.status, c.enabled, c.installed_at
  FROM app_installations c
 WHERE c.app_id = 'io.gerege.nexus.contacts'
   AND NOT EXISTS (SELECT 1 FROM app_installations d
                    WHERE d.tenant_id = c.tenant_id
                      AND d.app_id = 'io.gerege.nexus.organisation')
   AND EXISTS (SELECT 1 FROM apps a WHERE a.id = 'io.gerege.nexus.organisation')
ON CONFLICT (tenant_id, app_id) DO NOTHING;

-- 4. The app itself. app_installations and app_versions cascade from apps(id).
--    The `contacts` table is untouched and is read by the same code as before,
--    under a different app's menu.
DELETE FROM app_dependencies WHERE dependency_app_id = 'io.gerege.nexus.contacts';
DELETE FROM apps WHERE id = 'io.gerege.nexus.contacts';

DELETE FROM feature_flag_overrides WHERE flag_key = 'module.io.gerege.nexus.contacts.disabled';
DELETE FROM feature_flags WHERE key = 'module.io.gerege.nexus.contacts.disabled';

-- +goose Down
--
-- As in 00058: which tenants had Contacts *separately* left with the deleted
-- rows and cannot be invented, so a rollback gives it back to every tenant
-- holding the Directory. The safe direction — an app to switch off rather than
-- work nobody can reach.
--
-- The same is true one level down, and it is worth saying because it is visible
-- in Access control: a role that held only the contacts codes before the merge
-- comes out of a round trip holding both pairs. Up cannot tell which of its
-- organisation grants were its own and which it was given, so Down gives it
-- back everything it could have had. An administrator can remove a permission;
-- they cannot restore one they were never told had gone.

INSERT INTO apps (id, slug, name, description, icon_url, category, visibility)
VALUES ('io.gerege.nexus.contacts', 'contacts', 'Contacts',
        'Manage business contacts, customers, and vendors.',
        '/icons/contacts.png', 'CRM', 'public')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, code, name, description) VALUES
  (gen_random_uuid(), 'contacts.read',   'Read Contacts',   'View contacts list'),
  (gen_random_uuid(), 'contacts.manage', 'Manage Contacts', 'Create and edit contacts')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, old.id
  FROM role_permissions rp
  JOIN permissions new ON new.id = rp.permission_id
  JOIN permissions old ON old.code = 'contacts' || substr(new.code, length('organisation') + 1)
 WHERE new.code IN ('organisation.read', 'organisation.manage')
ON CONFLICT DO NOTHING;

INSERT INTO app_installations (tenant_id, app_id, installed_version, status, enabled, installed_at)
SELECT d.tenant_id, 'io.gerege.nexus.contacts', '1.0.0', d.status, d.enabled, d.installed_at
  FROM app_installations d
 WHERE d.app_id = 'io.gerege.nexus.organisation'
ON CONFLICT (tenant_id, app_id) DO NOTHING;

UPDATE permissions SET name = 'Read Organisation',
       description = 'View the organisation profile, its departments and its people'
 WHERE code = 'organisation.read';
UPDATE permissions SET name = 'Manage Organisation',
       description = 'Edit the organisation profile, its departments and its people'
 WHERE code = 'organisation.manage';
