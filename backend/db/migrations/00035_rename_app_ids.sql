-- The platform's own apps stop calling themselves examples.
--
-- `io.example.*` was placeholder vocabulary from the first week — the reverse
-- domain of nobody, borrowed the way `example.com` is borrowed — and it has
-- been the primary key of every app in the store ever since. These are Gerege
-- Nexus's own apps and they now say so: `io.gerege.nexus.*`.
--
-- It is a rename of a primary key, so it is a data migration and not a search
-- and replace. Three tables carry the id, and both foreign keys are ON UPDATE
-- NO ACTION, which is why they come off and go back on around the update rather
-- than the rows being rewritten in place — parent first would orphan the
-- children, children first would point at rows that do not exist yet.
--
-- The migrations before this one are left as they were. They already ran on
-- every deployment in the field, and a fresh database is expected to seed the
-- old ids at 00004 and 00006 and then arrive here, which is exactly what makes
-- this migration equally true of a database created yesterday and one created
-- next year.

-- +goose Up

ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_fkey;
ALTER TABLE app_versions DROP CONSTRAINT IF EXISTS app_versions_app_id_fkey;

-- The registry is renamed first, so between the two deployments an instance can
-- sync a catalogue that already carries the new ids and file them as apps it
-- has never seen. The store then offers those alongside the ones the tenant
-- already has, and somebody can install one.
--
-- All of that is undone here rather than collided with. Without it the rename
-- of the old row would meet a primary key that already exists, the migration
-- would fail, and the deployment would stop — which is a bad way to find out
-- about a five-minute window.
--
-- An installation made under the premature id is the same app: it is moved back
-- onto the row that carries the tenant's history, or dropped where that tenant
-- already had one.
DELETE FROM app_installations dup
 WHERE dup.app_id LIKE 'io.gerege.nexus.%'
   AND EXISTS (SELECT 1 FROM app_installations kept
                WHERE kept.tenant_id = dup.tenant_id
                  AND kept.app_id = 'io.example.' || substr(dup.app_id, length('io.gerege.nexus.') + 1));

UPDATE app_installations
   SET app_id = 'io.example.' || substr(app_id, length('io.gerege.nexus.') + 1)
 WHERE app_id LIKE 'io.gerege.nexus.%'
   AND EXISTS (SELECT 1 FROM apps old
                WHERE old.id = 'io.example.' || substr(app_installations.app_id, length('io.gerege.nexus.') + 1));

DELETE FROM app_versions v
 WHERE v.app_id LIKE 'io.gerege.nexus.%'
   AND EXISTS (SELECT 1 FROM apps old
                WHERE old.id = 'io.example.' || substr(v.app_id, length('io.gerege.nexus.') + 1));

DELETE FROM apps a
 WHERE a.id LIKE 'io.gerege.nexus.%'
   AND EXISTS (SELECT 1 FROM apps old
                WHERE old.id = 'io.example.' || substr(a.id, length('io.gerege.nexus.') + 1));

UPDATE apps
   SET id = 'io.gerege.nexus.' || substr(id, length('io.example.') + 1)
 WHERE id LIKE 'io.example.%';

UPDATE app_installations
   SET app_id = 'io.gerege.nexus.' || substr(app_id, length('io.example.') + 1)
 WHERE app_id LIKE 'io.example.%';

UPDATE app_versions
   SET app_id = 'io.gerege.nexus.' || substr(app_id, length('io.example.') + 1)
 WHERE app_id LIKE 'io.example.%';

-- The stored manifest names the app inside itself, and that copy is what an
-- upgrade compares against to decide whether a new version asks for more than
-- the installed one. Left behind, it would read as a different app.
UPDATE app_versions
   SET manifest = jsonb_set(manifest, '{id}',
                            to_jsonb('io.gerege.nexus.' || substr(manifest ->> 'id', length('io.example.') + 1)))
 WHERE manifest ->> 'id' LIKE 'io.example.%';

-- Nothing writes this table today, but it is the one other place an app names
-- another app, and leaving it stale would make the first dependency ever
-- recorded here unresolvable.
UPDATE app_dependencies
   SET dependency_app_id = 'io.gerege.nexus.' || substr(dependency_app_id, length('io.example.') + 1)
 WHERE dependency_app_id LIKE 'io.example.%';

ALTER TABLE app_installations
    ADD CONSTRAINT app_installations_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
ALTER TABLE app_versions
    ADD CONSTRAINT app_versions_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_fkey;
ALTER TABLE app_versions DROP CONSTRAINT IF EXISTS app_versions_app_id_fkey;

UPDATE apps
   SET id = 'io.example.' || substr(id, length('io.gerege.nexus.') + 1)
 WHERE id LIKE 'io.gerege.nexus.%';

UPDATE app_installations
   SET app_id = 'io.example.' || substr(app_id, length('io.gerege.nexus.') + 1)
 WHERE app_id LIKE 'io.gerege.nexus.%';

UPDATE app_versions
   SET app_id = 'io.example.' || substr(app_id, length('io.gerege.nexus.') + 1)
 WHERE app_id LIKE 'io.gerege.nexus.%';

UPDATE app_versions
   SET manifest = jsonb_set(manifest, '{id}',
                            to_jsonb('io.example.' || substr(manifest ->> 'id', length('io.gerege.nexus.') + 1)))
 WHERE manifest ->> 'id' LIKE 'io.gerege.nexus.%';

UPDATE app_dependencies
   SET dependency_app_id = 'io.example.' || substr(dependency_app_id, length('io.gerege.nexus.') + 1)
 WHERE dependency_app_id LIKE 'io.gerege.nexus.%';

ALTER TABLE app_installations
    ADD CONSTRAINT app_installations_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
ALTER TABLE app_versions
    ADD CONSTRAINT app_versions_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;
