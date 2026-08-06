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
-- were created. That is in the past for all of them, which is the right answer:
-- an approval nobody collected is not collectable now.
UPDATE document_eid_sign_sessions
   SET expires_at = created_at + INTERVAL '2 minutes'
 WHERE expires_at IS NULL;

-- +goose Down
ALTER TABLE document_eid_sign_sessions DROP COLUMN IF EXISTS expires_at;
