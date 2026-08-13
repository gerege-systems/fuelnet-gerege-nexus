-- The organisation app stops calling itself the core.
--
-- `core` was the first app this platform had and it took the name of the thing
-- underneath it. That was survivable while there was one repository; it stops
-- being survivable the moment the platform is published as an SDK, because then
-- "core" is both the module that holds departments and people *and* the package
-- every product imports, and no amount of documentation fixes a name collision
-- in an import path. So: `io.gerege.nexus.core` becomes
-- `io.gerege.nexus.organisation`, slug `core` becomes `organisation`, and the
-- two permissions it declares are renamed with it.
--
-- It is a rename of a primary key, so it is a data migration and not a search
-- and replace. Migration 00035 did this once already, for the whole catalogue,
-- and the shape here is deliberately the same: the two foreign keys are
-- ON UPDATE NO ACTION, so they come off and go back on around the update rather
-- than the rows being rewritten in place.
--
-- Five other places carry one of these names and none of them is a foreign key,
-- which is exactly why they are easy to forget:
--
--   * `app_versions.manifest` — the stored copy of what the installed version
--     asked for. It is what an upgrade diffs the catalogue against, so a stale
--     copy would report every renamed permission as newly requested and hold
--     the upgrade for an administrator who has nothing to approve;
--   * `permissions.code` — renamed in place, which is what keeps every existing
--     grant: `role_permissions` joins on the id, not the code;
--   * `report_schedules.report_key` and `report_grants.report_key` — a schedule
--     naming a report that no longer exists silently stops producing, and a
--     grant naming one silently stops sharing;
--   * `feature_flags.key` — the module kill switch is `module.<id>.disabled`,
--     so a switch thrown before this migration would be left holding down an
--     app id that no longer exists.
--
-- Everything is written to be true of a database that has never carried the old
-- names as well as one that has: a fresh database seeds the old ids at 00004
-- and arrives here, and an instance that has already synced a catalogue under
-- the new ones meets the same UPDATE finding nothing to do.

-- +goose Up

ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_fkey;
ALTER TABLE app_versions DROP CONSTRAINT IF EXISTS app_versions_app_id_fkey;

-- An instance that synced a catalogue carrying the new id before this migration
-- ran already has both rows. The new one has no history — it was created by a
-- sync minutes ago — so the tenant's own installation is what survives, and the
-- premature row is removed rather than collided with. Without this the rename
-- below would meet a primary key that already exists and the deployment would
-- stop.
DELETE FROM app_installations dup
 WHERE dup.app_id = 'io.gerege.nexus.organisation'
   AND EXISTS (SELECT 1 FROM app_installations kept
                WHERE kept.tenant_id = dup.tenant_id
                  AND kept.app_id = 'io.gerege.nexus.core');

DELETE FROM app_versions v
 WHERE v.app_id = 'io.gerege.nexus.organisation'
   AND EXISTS (SELECT 1 FROM apps old WHERE old.id = 'io.gerege.nexus.core');

DELETE FROM apps a
 WHERE a.id = 'io.gerege.nexus.organisation'
   AND EXISTS (SELECT 1 FROM apps old WHERE old.id = 'io.gerege.nexus.core');

-- The slug moves with the id. It is UNIQUE and it is what a store URL is keyed
-- by, so leaving it behind would put the app at /store/apps/core under a name
-- nothing else uses any more.
UPDATE apps
   SET id = 'io.gerege.nexus.organisation',
       slug = 'organisation',
       icon_url = CASE WHEN icon_url = '/icons/core.png' THEN '/icons/organisation.png' ELSE icon_url END
 WHERE id = 'io.gerege.nexus.core';

UPDATE app_installations
   SET app_id = 'io.gerege.nexus.organisation'
 WHERE app_id = 'io.gerege.nexus.core';

UPDATE app_versions
   SET app_id = 'io.gerege.nexus.organisation'
 WHERE app_id = 'io.gerege.nexus.core';

-- The manifest names the app and its permissions inside itself. Rewritten as
-- text because the two live at different depths — one scalar at the root, two
-- inside an array of objects — and a pair of jsonb_set calls would have to know
-- the array index. The tokens are quoted, so nothing matches by accident.
UPDATE app_versions
   SET manifest = replace(replace(replace(manifest::text,
                    '"io.gerege.nexus.core"', '"io.gerege.nexus.organisation"'),
                    '"core.read"', '"organisation.read"'),
                    '"core.manage"', '"organisation.manage"')::jsonb
 WHERE manifest::text LIKE '%io.gerege.nexus.core%'
    OR manifest::text LIKE '%"core.read"%'
    OR manifest::text LIKE '%"core.manage"%';

-- Nothing declares a dependency on this app today, which is one of the reasons
-- it stopped being undeletable. Updated anyway: the day something does, the
-- edge must not be the one row still naming the old id.
UPDATE app_dependencies
   SET dependency_app_id = 'io.gerege.nexus.organisation'
 WHERE dependency_app_id = 'io.gerege.nexus.core';

ALTER TABLE app_installations
    ADD CONSTRAINT app_installations_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
ALTER TABLE app_versions
    ADD CONSTRAINT app_versions_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

-- Permissions. Renamed in place so every role that holds one keeps holding it:
-- role_permissions references permissions(id), and the id does not move.
--
-- The delete first is the same defence as above — an instance that installed the
-- app under its new id already registered the new codes, and a rename onto an
-- existing UNIQUE code would fail. The row being dropped is the one nothing is
-- granted through yet, and its grants are re-pointed at the surviving row first
-- so an early adopter does not lose an assignment.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, keep.id
  FROM role_permissions rp
  JOIN permissions dup  ON dup.id = rp.permission_id
  JOIN permissions keep ON keep.code = 'core' || substr(dup.code, length('organisation') + 1)
 WHERE dup.code IN ('organisation.read', 'organisation.manage')
ON CONFLICT DO NOTHING;

DELETE FROM permissions
 WHERE code IN ('organisation.read', 'organisation.manage')
   AND EXISTS (SELECT 1 FROM permissions old
                WHERE old.code = 'core' || substr(permissions.code, length('organisation') + 1));

UPDATE permissions SET code = 'organisation.read'   WHERE code = 'core.read';
UPDATE permissions SET code = 'organisation.manage' WHERE code = 'core.manage';

-- The two reports this app registers. A schedule or a grant naming the old key
-- would keep its row and quietly match nothing.
UPDATE report_schedules SET report_key = 'organisation.user_activity'
 WHERE report_key = 'core.user_activity';
UPDATE report_schedules SET report_key = 'organisation.headcount_by_unit'
 WHERE report_key = 'core.headcount_by_unit';

UPDATE report_grants SET report_key = 'organisation.user_activity'
 WHERE report_key = 'core.user_activity';
UPDATE report_grants SET report_key = 'organisation.headcount_by_unit'
 WHERE report_key = 'core.headcount_by_unit';

-- The module kill switch. Its key is derived from the app id, and the override
-- table points at it with ON DELETE CASCADE and no ON UPDATE, so the constraint
-- comes off for the rename the same way the app's did.
ALTER TABLE feature_flag_overrides DROP CONSTRAINT IF EXISTS feature_flag_overrides_flag_key_fkey;

DELETE FROM feature_flags
 WHERE key = 'module.io.gerege.nexus.organisation.disabled'
   AND EXISTS (SELECT 1 FROM feature_flags old
                WHERE old.key = 'module.io.gerege.nexus.core.disabled');

UPDATE feature_flags SET key = 'module.io.gerege.nexus.organisation.disabled'
 WHERE key = 'module.io.gerege.nexus.core.disabled';
UPDATE feature_flag_overrides SET flag_key = 'module.io.gerege.nexus.organisation.disabled'
 WHERE flag_key = 'module.io.gerege.nexus.core.disabled';

-- An override left pointing at a flag that is no longer there would fail the
-- constraint being restored. It can only exist on an instance that had both.
DELETE FROM feature_flag_overrides o
 WHERE NOT EXISTS (SELECT 1 FROM feature_flags f WHERE f.key = o.flag_key);

ALTER TABLE feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_flag_key_fkey
    FOREIGN KEY (flag_key) REFERENCES feature_flags(key) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_fkey;
ALTER TABLE app_versions DROP CONSTRAINT IF EXISTS app_versions_app_id_fkey;

UPDATE apps
   SET id = 'io.gerege.nexus.core',
       slug = 'core',
       icon_url = CASE WHEN icon_url = '/icons/organisation.png' THEN '/icons/core.png' ELSE icon_url END
 WHERE id = 'io.gerege.nexus.organisation';

UPDATE app_installations
   SET app_id = 'io.gerege.nexus.core'
 WHERE app_id = 'io.gerege.nexus.organisation';

UPDATE app_versions
   SET app_id = 'io.gerege.nexus.core'
 WHERE app_id = 'io.gerege.nexus.organisation';

UPDATE app_versions
   SET manifest = replace(replace(replace(manifest::text,
                    '"io.gerege.nexus.organisation"', '"io.gerege.nexus.core"'),
                    '"organisation.read"', '"core.read"'),
                    '"organisation.manage"', '"core.manage"')::jsonb
 WHERE manifest::text LIKE '%io.gerege.nexus.organisation%'
    OR manifest::text LIKE '%"organisation.read"%'
    OR manifest::text LIKE '%"organisation.manage"%';

UPDATE app_dependencies
   SET dependency_app_id = 'io.gerege.nexus.core'
 WHERE dependency_app_id = 'io.gerege.nexus.organisation';

ALTER TABLE app_installations
    ADD CONSTRAINT app_installations_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
ALTER TABLE app_versions
    ADD CONSTRAINT app_versions_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

UPDATE permissions SET code = 'core.read'   WHERE code = 'organisation.read';
UPDATE permissions SET code = 'core.manage' WHERE code = 'organisation.manage';

UPDATE report_schedules SET report_key = 'core.user_activity'
 WHERE report_key = 'organisation.user_activity';
UPDATE report_schedules SET report_key = 'core.headcount_by_unit'
 WHERE report_key = 'organisation.headcount_by_unit';

UPDATE report_grants SET report_key = 'core.user_activity'
 WHERE report_key = 'organisation.user_activity';
UPDATE report_grants SET report_key = 'core.headcount_by_unit'
 WHERE report_key = 'organisation.headcount_by_unit';

ALTER TABLE feature_flag_overrides DROP CONSTRAINT IF EXISTS feature_flag_overrides_flag_key_fkey;

UPDATE feature_flags SET key = 'module.io.gerege.nexus.core.disabled'
 WHERE key = 'module.io.gerege.nexus.organisation.disabled';
UPDATE feature_flag_overrides SET flag_key = 'module.io.gerege.nexus.core.disabled'
 WHERE flag_key = 'module.io.gerege.nexus.organisation.disabled';

ALTER TABLE feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_flag_key_fkey
    FOREIGN KEY (flag_key) REFERENCES feature_flags(key) ON DELETE CASCADE;
