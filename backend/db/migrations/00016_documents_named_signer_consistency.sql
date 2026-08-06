-- +goose Up

-- require_named_signer used to be checked at signing time, and its guard only
-- asked that SOME step name a signer. A tenant could therefore store the flag
-- against a chain with an open step, or with one citizen named twice — pairs that
-- under the new rules cannot be saved at all, because a named step now binds on its
-- own and every step must name a distinct citizen for the flag to mean anything.
--
-- The flag is no longer consulted when a signature is applied, so leaving such a
-- pair stored would have the screen advertise "Named signer only" over a chain that
-- lets anyone fill its open steps. Enforcing it at signing time instead would strand
-- every document of that type, which is the outcome the save-time guards exist to
-- prevent.
--
-- Worse, a named step now binds whether or not the flag is set — the old code only
-- looked at the chain when the flag was on, and treated its registration numbers as
-- an unordered set. So a chain stored under the old rules naming ONE citizen at TWO
-- steps used to mean "two signatures from anyone" and was completable; under the new
-- rules the second step is owed to a citizen who has already signed, and one citizen
-- signs a document once. Every document of that type — the ones migration 00014 has
-- just copied it onto, and every one created afterwards — could never be approved.
-- Clearing the flag does not help, because the flag is no longer what binds.
--
-- So the chain is normalised first: where a citizen is named more than once, the
-- later steps are opened. The tenant asked for that many approvals and still gets
-- them; the repeat is simply fillable by somebody else, which is the only reading
-- that leaves the chain completable.
-- +goose StatementBegin
DO $$
BEGIN
    WITH repeated AS (
        SELECT id, row_number() OVER (PARTITION BY tenant_id, doc_type, signer_reg_number
                                      ORDER BY step_order) AS n
          FROM document_workflow_steps
         WHERE signer_reg_number <> ''
    )
    UPDATE document_workflow_steps w
       SET signer_reg_number = ''
      FROM repeated
     WHERE repeated.id = w.id AND repeated.n > 1;

    -- Chains already copied onto documents by 00014 need the same repair, or those
    -- documents stay stranded however the type is configured afterwards.
    WITH repeated AS (
        SELECT id, row_number() OVER (PARTITION BY document_id, signer_reg_number
                                      ORDER BY step_order) AS n
          FROM document_approval_steps
         WHERE signer_reg_number <> ''
    )
    UPDATE document_approval_steps s
       SET signer_reg_number = ''
      FROM repeated
     WHERE repeated.id = s.id AND repeated.n > 1;
END $$;
-- +goose StatementEnd

-- With the chains repaired, the flag is cleared wherever the chain beneath it still
-- could not satisfy it — which now means a chain with no steps, or one carrying an
-- open step (including a step this migration has just opened).
UPDATE document_signature_policies p
   SET require_named_signer = FALSE, updated_at = NOW()
 WHERE p.require_named_signer
   AND (
        -- no steps at all: nobody could sign
        NOT EXISTS (SELECT 1 FROM document_workflow_steps w
                     WHERE w.tenant_id = p.tenant_id AND w.doc_type = p.doc_type)
        -- a step naming nobody: it could never be filled under the flag
        OR EXISTS (SELECT 1 FROM document_workflow_steps w
                    WHERE w.tenant_id = p.tenant_id AND w.doc_type = p.doc_type
                      AND w.signer_reg_number = '')
        -- one citizen named twice: they sign a document once, so the second
        -- step could never be filled
        OR EXISTS (SELECT w.signer_reg_number
                     FROM document_workflow_steps w
                    WHERE w.tenant_id = p.tenant_id AND w.doc_type = p.doc_type
                      AND w.signer_reg_number <> ''
                    GROUP BY w.signer_reg_number HAVING count(*) > 1)
   );

-- +goose Down

-- Nothing to undo. The flag was cleared, and a repeated signer opened, because the
-- configuration could not be completed by anybody; restoring either would restore a
-- state no screen will save and no citizen could satisfy.
SELECT 1;
