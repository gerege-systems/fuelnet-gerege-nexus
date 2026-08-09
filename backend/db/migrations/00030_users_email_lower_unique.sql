-- Sign-in looks an address up case-insensitively; the database has to agree.
--
-- handleLogin matches on `lower(u.email) = $1` so that somebody who capitalises
-- the first letter of their own address can still sign in. The only index on
-- users.email is the plain UNIQUE one, and a btree on email cannot answer a
-- query about lower(email): every sign-in attempt became a sequential scan of
-- the whole users table, in front of a bcrypt comparison.
--
-- The uniqueness was the other half of the same mismatch. UNIQUE(email) is
-- case-sensitive, so Bat@example.mn and bat@example.mn were two accounts, while
-- the lookup that finds them is case-insensitive and takes LIMIT 1 — which of
-- the two you signed into depended on membership order.
--
-- +goose Up

-- Fold the addresses that can be folded without colliding with an existing one.
-- +goose StatementBegin
UPDATE users u
   SET email = lower(u.email)
 WHERE u.email <> lower(u.email)
   AND NOT EXISTS (
       SELECT 1 FROM users other
        WHERE other.id <> u.id
          AND lower(other.email) = lower(u.email));
-- +goose StatementEnd

-- What is left is two real accounts whose addresses differ only in case. That
-- is a data question with a person behind it — which of the two is the account,
-- and what happens to the other one's documents — so it is refused here with
-- the addresses named, rather than resolved by a rule this file invented.
-- +goose StatementBegin
DO $dupes$
DECLARE collisions TEXT;
BEGIN
    SELECT string_agg(duplicated.address, ', ')
      INTO collisions
      FROM (
          SELECT lower(email) AS address
            FROM users
           GROUP BY lower(email)
          HAVING count(*) > 1
      ) AS duplicated;

    IF collisions IS NOT NULL THEN
        RAISE EXCEPTION
            'cannot enforce case-insensitive e-mail uniqueness: these addresses exist more than once, differing only in case: %',
            collisions
        USING HINT = 'Merge or rename the duplicate accounts, then run this migration again.';
    END IF;
END
$dupes$;
-- +goose StatementEnd

CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_key ON users (lower(email));

-- +goose Down
DROP INDEX IF EXISTS users_email_lower_key;
