-- The contact register leaves the platform for the commerce distribution.
--
-- Migration 00059 folded Contacts into the Directory a few hours ago, on the
-- argument that who an organisation is made of and who it deals with are one
-- subject. That was half right, and the half it got wrong is the half that
-- decides where code lives: departments and staff are something every
-- organisation has, and customers are something a *business* has. The first
-- belongs to a platform every deployment runs; the second belongs to a product.
--
-- So `io.gerege.nexus.contacts` is an app again, built and shipped by
-- commerce-gerege-nexus, and this migration undoes the naming half of 00059 —
-- nothing else needs undoing:
--
--   * the `contacts` table stays. It was created by 00003, it has run on every
--     deployment in the field, and the module reading it is the same code at a
--     different import path;
--   * the grants stay. 00059 gave organisation.read/manage to every role that
--     held contacts.read/manage, and taking them back would remove a permission
--     from roles that may have been edited since — an administrator can drop
--     one they do not want, and cannot restore one they were never told had
--     gone. The contacts.* codes are registered again by the installer the
--     first time a deployment installs the app from the commerce catalogue;
--   * the screens stay. The shell is one image serving every deployment and
--     carries the union of first-party pages (ECOSYSTEM_GIT_STRATEGY §2.3);
--     without the module they are unlisted in the menu and refused by the API.
--
-- What is left is the two sentences an administrator reads in Access control,
-- which 00059 widened to mention contacts and which are now wrong.

-- +goose Up

UPDATE permissions
   SET name = 'Read Organisation',
       description = 'View the organisation profile, its departments and its people'
 WHERE code = 'organisation.read';

UPDATE permissions
   SET name = 'Manage Organisation',
       description = 'Edit the organisation profile, its departments and its people'
 WHERE code = 'organisation.manage';

-- +goose Down

UPDATE permissions
   SET name = 'Read Directory',
       description = 'View the organisation profile, its departments, its people and its contacts'
 WHERE code = 'organisation.read';

UPDATE permissions
   SET name = 'Manage Directory',
       description = 'Edit the organisation profile, its departments, its people and its contacts'
 WHERE code = 'organisation.manage';
