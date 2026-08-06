-- +goose Up

-- One approval, one signature. The ledger already refused the same citizen twice
-- (`document_signatures_once_per_signer`), but nothing stopped two DIFFERENT citizens
-- being recorded against the same step of the same document — and when that happened
-- it happened silently, leaving the trail crediting a named step to whoever landed on
-- it second.
--
-- Two ways it could arise, both now fixed in the code, both invisible without this:
--
--   * migration 00014 §4 parked a left-over signature at "number of steps + n", which
--     on a chain whose numbers are not 1..n lands ON an existing step;
--   * recordSignature numbered a signature that filled no step "applied + 1", with
--     the same collision at runtime.
--
-- A constraint is worth more than both fixes. It turns a numbering mistake into a
-- failed write — loud, and before anything is attributed to the wrong person —
-- instead of a ledger that reads plausibly and says the wrong thing.

-- +goose StatementBegin
DO $$
DECLARE
    repaired INTEGER;
BEGIN
    -- Any duplicate that already exists is renumbered past the end of its document's
    -- chain, oldest kept in place. That is what §4 should have done with it: the
    -- signature is real and counts toward what the document holds, it just cannot
    -- claim an approval another signature already claimed.
    WITH ranked AS (
        SELECT id, document_id, step_order,
               row_number() OVER (PARTITION BY document_id, step_order
                                  ORDER BY signed_at, id) AS n
          FROM document_signatures
         WHERE step_order IS NOT NULL
    ),
    clashing AS (
        SELECT r.id, r.document_id,
               row_number() OVER (PARTITION BY r.document_id ORDER BY r.step_order, r.id) AS offset_n
          FROM ranked r
         WHERE r.n > 1
    )
    UPDATE document_signatures s
       SET step_order = GREATEST(
             COALESCE((SELECT max(x.step_order) FROM document_signatures x
                        WHERE x.document_id = c.document_id), 0),
             COALESCE((SELECT max(st.step_order) FROM document_approval_steps st
                        WHERE st.document_id = c.document_id), 0)) + c.offset_n
      FROM clashing c
     WHERE c.id = s.id;

    GET DIAGNOSTICS repaired = ROW_COUNT;
    IF repaired > 0 THEN
        RAISE NOTICE 'renumbered % signature(s) that shared an approval with another', repaired;
    END IF;
END $$;
-- +goose StatementEnd

-- NULL step_order is still allowed — 00014 fills them all in, but a row that somehow
-- arrives without one should not be forced onto an approval it did not fill. Postgres
-- treats NULLs as distinct in a unique index, which is the behaviour wanted here.
ALTER TABLE document_signatures
    ADD CONSTRAINT document_signatures_one_per_approval UNIQUE (document_id, step_order);

-- +goose Down
ALTER TABLE document_signatures
    DROP CONSTRAINT IF EXISTS document_signatures_one_per_approval;
