-- +goose Up

-- An approval was redeemable for as long as its row survived. expires_at came
-- back from eID, was handed to the caller, and then thrown away: nothing stored
-- it and nothing checked it. The only cleanup ran when somebody started another
-- session for the same document, so a session a citizen approved days ago — its
-- id still sitting in a proxy log or a browser's network panel — could be posted
-- to the poll endpoint and turned into a signature dated today.
ALTER TABLE document_eid_sign_sessions
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

-- Rows that predate the column have no provider deadline to fall back on, so they
-- get the two minutes an eID request is normally given, measured from when they
-- were created. The poll allows a further two minutes of grace for clock skew, so
-- a session started in the last four minutes stays collectable — anything older
-- does not, which is the right answer for an approval nobody came back for.
UPDATE document_eid_sign_sessions
   SET expires_at = created_at + INTERVAL '2 minutes'
 WHERE expires_at IS NULL;

-- +goose Down
ALTER TABLE document_eid_sign_sessions DROP COLUMN IF EXISTS expires_at;
