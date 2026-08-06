-- +goose Up

-- What a document needed is a property of the document, not of today's
-- configuration. Until now required_signatures was counted live from
-- document_workflow_steps on every read, and the completion decision compared a
-- signature count taken under a row lock against a requirement read before the
-- lock was held. Three defects followed from that:
--
--   * a chain edited between the two reads could approve a document on fewer
--     signatures than its chain asked for;
--   * shortening a chain left an already-signed document stuck PENDING while the
--     screen painted a green "complete" badge beside the amber "Pending" one, and
--     no further signature could finish it;
--   * editing a chain rewrote the reported progress of documents decided months
--     earlier — an APPROVED contract would start claiming 1 of 3 signatures.
--
-- A document now carries its own copy of the chain, taken when it starts waiting
-- for approval, and its own requirement count. A later configuration change
-- cannot retroactively redefine what an in-flight or finished document needed.

ALTER TABLE document_records
    ADD COLUMN IF NOT EXISTS required_signatures SMALLINT NOT NULL DEFAULT 1;

-- The document's own approval chain. No rows means one open signature approves,
-- which is how a document of a type with no chain behaves.
CREATE TABLE IF NOT EXISTS document_approval_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES document_records(id) ON DELETE CASCADE,
    step_order SMALLINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    -- Empty means the step is open: anyone allowed to sign may take it. A
    -- registration number names the one citizen whose signature fills it.
    signer_reg_number VARCHAR(64) NOT NULL DEFAULT '',
    CONSTRAINT document_approval_steps_order_unique UNIQUE (document_id, step_order),
    CONSTRAINT document_approval_steps_order_positive CHECK (step_order >= 1)
);
CREATE INDEX IF NOT EXISTS idx_document_approval_steps_document
    ON document_approval_steps(document_id);

-- Which step each signature filled. Signatures fill the chain in order, so this
-- is also the signature's ordinal — stored because a ledger that cannot say
-- which approval a signature was is a poor record of an approval chain.
ALTER TABLE document_signatures
    ADD COLUMN IF NOT EXISTS step_order SMALLINT;

-- +goose StatementBegin
DO $$
BEGIN
    -- A decided document asked for exactly what it got: its requirement is the
    -- number of signatures it carries, at least one. Anything else would make
    -- history read wrong the moment a chain changes.
    UPDATE document_records d
       SET required_signatures = GREATEST(1, (
             SELECT count(*) FROM document_signatures s WHERE s.document_id = d.id))
     WHERE d.status IN ('APPROVED', 'REJECTED');

    -- A document still waiting takes its type's chain as it stands now, which is
    -- the best evidence available for what it was routed under.
    UPDATE document_records d
       SET required_signatures = GREATEST(1, (
             SELECT count(*) FROM document_workflow_steps w
              WHERE w.tenant_id = d.tenant_id AND w.doc_type = d.doc_type))
     WHERE d.status = 'PENDING_APPROVAL';

    INSERT INTO document_approval_steps (tenant_id, document_id, step_order, name, signer_reg_number)
    SELECT d.tenant_id, d.id, w.step_order, w.name, w.signer_reg_number
      FROM document_records d
      JOIN document_workflow_steps w
        ON w.tenant_id = d.tenant_id AND w.doc_type = d.doc_type
     WHERE d.status = 'PENDING_APPROVAL'
        ON CONFLICT DO NOTHING;

    -- Existing signatures filled the chain in the order they were given.
    WITH ordered AS (
        SELECT id, row_number() OVER (PARTITION BY document_id ORDER BY signed_at, id) AS n
          FROM document_signatures
    )
    UPDATE document_signatures s
       SET step_order = ordered.n
      FROM ordered
     WHERE ordered.id = s.id AND s.step_order IS NULL;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE document_signatures DROP COLUMN IF EXISTS step_order;
DROP TABLE IF EXISTS document_approval_steps;
ALTER TABLE document_records DROP COLUMN IF EXISTS required_signatures;
