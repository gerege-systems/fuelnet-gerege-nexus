-- The state registry lookups become an app's, and their permissions say so.
--
-- `xyp.citizen.read` and `xyp.company.read` were registered by migration 00024
-- and granted to the administrator role alone. The lookups behind them have
-- moved out of the platform's route table into the `egov` app, so the codes
-- follow: `egov.citizen.read` and `egov.company.read`.
--
-- Renamed in place and with no alias, unlike the app ids in 00055 and 00056. A
-- permission code is internal — it is compared against the compiled module's
-- own list and against `role_permissions`, and nothing outside this deployment
-- ever sees one — so there is nobody to keep a second spelling working for.
-- Renaming in place is also what preserves every grant: role_permissions joins
-- on permissions(id), which does not move.
--
-- What is deliberately not here:
--
--   * the audit rows. `xyp.citizen_queried` events are a record of something
--     that happened under that name, and rewriting history to match a rename
--     would make the log a worse record than it is. The app's history screen
--     reads both prefixes;
--   * `egov.read`, the third permission the module declares. It is new, so
--     there is nothing to migrate: the installer registers it when the app is
--     installed, which EnsureDefaultApps does for every existing tenant on the
--     next boot.
--
-- The two lookups stay administrative after the move. The module marks them
-- AdminOnly, which stops the installer's default rule — anything ending `.read`
-- goes to every member — from widening on install what this migration narrowly
-- preserves.

-- +goose Up

-- An instance that reached the new codes first (a catalogue carrying egov
-- synced before this ran) already registered them. Merge the grants onto the
-- row that is about to be renamed, then drop the duplicate, so the rename does
-- not collide with an existing UNIQUE code and nobody loses an assignment.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, keep.id
  FROM role_permissions rp
  JOIN permissions dup  ON dup.id = rp.permission_id
  JOIN permissions keep ON keep.code = 'xyp' || substr(dup.code, length('egov') + 1)
 WHERE dup.code IN ('egov.citizen.read', 'egov.company.read')
ON CONFLICT DO NOTHING;

DELETE FROM permissions
 WHERE code IN ('egov.citizen.read', 'egov.company.read')
   AND EXISTS (SELECT 1 FROM permissions old
                WHERE old.code = 'xyp' || substr(permissions.code, length('egov') + 1));

UPDATE permissions
   SET code = 'egov.citizen.read',
       name = 'Query the citizen registry',
       description = 'Look up authoritative citizen data through ХУР'
 WHERE code = 'xyp.citizen.read';

UPDATE permissions
   SET code = 'egov.company.read',
       name = 'Query the company registry',
       description = 'Look up authoritative legal-entity data through ХУР'
 WHERE code = 'xyp.company.read';

-- +goose Down

UPDATE permissions
   SET code = 'xyp.citizen.read',
       name = 'Query citizen registry',
       description = 'Query authoritative citizen data through XYP'
 WHERE code = 'egov.citizen.read';

UPDATE permissions
   SET code = 'xyp.company.read',
       name = 'Query company registry',
       description = 'Query authoritative company data through XYP'
 WHERE code = 'egov.company.read';

-- egov.read has no pre-migration spelling to go back to. It is dropped rather
-- than renamed: on the way down the app that declares it is gone too, and a
-- permission nothing enforces is one more row in the access-control screen that
-- nobody can explain.
DELETE FROM role_permissions rp
 USING permissions p
 WHERE p.id = rp.permission_id AND p.code = 'egov.read';
DELETE FROM permissions WHERE code = 'egov.read';
