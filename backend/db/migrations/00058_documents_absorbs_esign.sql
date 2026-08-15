-- PDF E-Sign stops being an app of its own and becomes part of Documents.
--
-- Two cards in the store, two installations, two menu entries and two
-- permission namespaces, for one question a person actually has: where are my
-- documents and who has signed them. Nobody adopts a signature on its own —
-- they adopt it because something has to be signed. So the rails moved inside
-- `io.gerege.nexus.documents`, which keeps its id; `io.gerege.nexus.esign` is
-- removed from the catalogue and from this database.
--
-- This is an absorption, not the rename 00055 did, and the difference is the
-- whole risk: a rename moves rows and everything they carry travels with them,
-- while a deletion has to be *preceded* by every transfer, in the right order,
-- or somebody loses access to a signature they were relying on this morning.
-- So the order below is deliberate and every step is written to be true on a
-- database that has both apps, one of them, or neither:
--
--   1. permissions   — the three codes exist before anything is granted them;
--   2. grants        — every role holding esign.X also holds documents.X;
--   3. installations — every tenant that had the PDF app has the documents app;
--   4. reports       — schedules and shares follow the renamed report keys;
--   5. only then     — the app row goes, taking its installations and versions
--                      with it through the cascades already on those tables.
--
-- What is deliberately *not* carried across: `module.io.gerege.nexus.esign.disabled`.
-- A tenant that had switched the PDF app off gets the rails back, because the
-- alternative — mapping that switch onto the documents module — would silently
-- turn off the register, the approval chains and retention as well. Losing a
-- kill switch is visible the moment somebody looks at the menu; the other way
-- round takes a working app away from people who never asked for that.

-- +goose Up

-- 1. The three surviving codes. Normally the installer writes these when an app
--    is installed, but the grants below have to join to them, and an instance
--    where the documents app was never installed would have none of them yet.
INSERT INTO permissions (id, code, name, description) VALUES
  (gen_random_uuid(), 'documents.read',   'Read Documents',   'View documents, uploaded PDFs, signature status and the signature log'),
  (gen_random_uuid(), 'documents.manage', 'Manage Documents', 'Create documents, upload PDFs, route them for approval, run signing batches, and configure templates, approval chains, signature policies, signing rails, stamp placement and retention rules'),
  (gen_random_uuid(), 'documents.sign',   'Sign Documents',   'Apply an eID / DAN / HSM digital signature or reject a document')
ON CONFLICT (code) DO NOTHING;

-- 2. Every grant, carried one to one. This is the step that decides whether an
--    administrator has to reconstruct anything on Monday morning, and the
--    mapping is exact — esign.read → documents.read, and so on — because the
--    codes were renamed rather than redesigned.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, new.id
  FROM role_permissions rp
  JOIN permissions old ON old.id = rp.permission_id
  JOIN permissions new ON new.code = 'documents' || substr(old.code, length('esign') + 1)
 WHERE old.code IN ('esign.read', 'esign.sign', 'esign.manage')
ON CONFLICT DO NOTHING;

-- The old codes go once nothing is reached through them. role_permissions
-- references permissions(id) ON DELETE CASCADE, so the dead grants leave with
-- them; the live ones were written above and are a different row.
DELETE FROM permissions WHERE code IN ('esign.read', 'esign.sign', 'esign.manage');

-- 3. A tenant that had the PDF app and not the documents app would otherwise
--    wake up with neither. Installed in whatever state their PDF installation
--    was in: somebody who had disabled it does not get an app switched on by a
--    migration, and somebody using it does not have to notice anything.
INSERT INTO app_installations (tenant_id, app_id, installed_version, status, enabled, installed_at)
SELECT e.tenant_id, 'io.gerege.nexus.documents', '2.0.0', e.status, e.enabled, e.installed_at
  FROM app_installations e
 WHERE e.app_id = 'io.gerege.nexus.esign'
   AND NOT EXISTS (SELECT 1 FROM app_installations d
                    WHERE d.tenant_id = e.tenant_id
                      AND d.app_id = 'io.gerege.nexus.documents')
   AND EXISTS (SELECT 1 FROM apps a WHERE a.id = 'io.gerege.nexus.documents')
ON CONFLICT (tenant_id, app_id) DO NOTHING;

-- A tenant that had both, with the PDF app on and the documents app switched
-- off, was signing PDFs this morning. Leaving the documents row disabled would
-- take that away — `enabled` is what the app gate reads before it mounts a
-- route, and the routes are the same routes.
UPDATE app_installations d
   SET enabled = TRUE, status = 'installed', updated_at = NOW()
  FROM app_installations e
 WHERE d.tenant_id = e.tenant_id
   AND d.app_id = 'io.gerege.nexus.documents'
   AND e.app_id = 'io.gerege.nexus.esign'
   AND e.enabled
   AND NOT d.enabled;

-- 4. The two reports the rails register. A schedule naming a key that no longer
--    exists stops producing silently, and a grant naming one stops sharing
--    silently; both are the kind of failure nobody reports because nothing
--    appears to be wrong.
UPDATE report_schedules SET report_key = 'documents.signatures_by_rail' WHERE report_key = 'esign.signatures_by_rail';
UPDATE report_schedules SET report_key = 'documents.signer_activity'   WHERE report_key = 'esign.signer_activity';
UPDATE report_grants    SET report_key = 'documents.signatures_by_rail' WHERE report_key = 'esign.signatures_by_rail';
UPDATE report_grants    SET report_key = 'documents.signer_activity'   WHERE report_key = 'esign.signer_activity';

-- 5. The app itself. app_installations and app_versions reference apps(id)
--    ON DELETE CASCADE, so this is also what removes the rows that named it.
--    The tables holding actual work — esign_documents, esign_signature_logs,
--    esign_sign_sessions and the batches — are untouched by any of this and are
--    read by the same code as before, under a different app's menu.
DELETE FROM app_dependencies WHERE dependency_app_id = 'io.gerege.nexus.esign';
DELETE FROM apps WHERE id = 'io.gerege.nexus.esign';

DELETE FROM feature_flag_overrides WHERE flag_key = 'module.io.gerege.nexus.esign.disabled';
DELETE FROM feature_flags WHERE key = 'module.io.gerege.nexus.esign.disabled';

-- +goose Down
--
-- The app row comes back and so do the permissions, because a rollback that
-- left an administrator's grants pointing at nothing would be worse than the
-- thing it was rolling back. What cannot come back is which tenants had the PDF
-- app *separately*: that fact left with the deleted installations, and this
-- migration is not able to invent it. So every tenant holding the documents app
-- gets the PDF app back — the superset, which is the safe direction: an app
-- switched on that somebody has to switch off, rather than work nobody can
-- reach.

INSERT INTO apps (id, slug, name, description, icon_url, category, visibility)
VALUES ('io.gerege.nexus.esign', 'esign', 'PDF E-Sign',
        'Upload PDF documents and sign them with Mongolian digital signatures via the Gerege eSign HSM service.',
        '/icons/esign.png', 'Productivity', 'public')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, code, name, description) VALUES
  (gen_random_uuid(), 'esign.read',   'View E-Sign Documents', 'View uploaded and signed PDF documents and the signature log'),
  (gen_random_uuid(), 'esign.sign',   'Sign Documents',        'Sign PDF documents with a digital signature'),
  (gen_random_uuid(), 'esign.manage', 'Manage E-Sign',         'Upload documents, run batches and configure signing')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, old.id
  FROM role_permissions rp
  JOIN permissions new ON new.id = rp.permission_id
  JOIN permissions old ON old.code = 'esign' || substr(new.code, length('documents') + 1)
 WHERE new.code IN ('documents.read', 'documents.sign', 'documents.manage')
ON CONFLICT DO NOTHING;

INSERT INTO app_installations (tenant_id, app_id, installed_version, status, enabled, installed_at)
SELECT d.tenant_id, 'io.gerege.nexus.esign', '2.0.0', d.status, d.enabled, d.installed_at
  FROM app_installations d
 WHERE d.app_id = 'io.gerege.nexus.documents'
ON CONFLICT (tenant_id, app_id) DO NOTHING;

UPDATE report_schedules SET report_key = 'esign.signatures_by_rail' WHERE report_key = 'documents.signatures_by_rail';
UPDATE report_schedules SET report_key = 'esign.signer_activity'    WHERE report_key = 'documents.signer_activity';
UPDATE report_grants    SET report_key = 'esign.signatures_by_rail' WHERE report_key = 'documents.signatures_by_rail';
UPDATE report_grants    SET report_key = 'esign.signer_activity'    WHERE report_key = 'documents.signer_activity';
