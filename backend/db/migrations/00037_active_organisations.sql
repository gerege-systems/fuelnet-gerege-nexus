-- Working in more than one organisation at a time.
--
-- Odoo answers this with `allowed_company_ids`: a user ticks several companies
-- in one widget and every list is then filtered by `company_id IN (…)` instead
-- of a single equality, with one of them still being the company new records
-- are created in. The distinction it draws is the right one, and it is the
-- distinction this migration writes into the policies:
--
--     reading  — any of the organisations the session is active in
--     writing  — the one it is acting in
--
-- That asymmetry is the whole safety of the feature. Widening reads lets
-- somebody who works for a parent and two subsidiaries see one list across all
-- three; widening writes would let a row be created in an organisation the
-- operator is not looking at, which no screen could show them and no audit
-- trail would explain.
--
-- Where the set comes from is unchanged in kind: it is written on the session
-- row, and only after the same membership check that decides tenant_id today.
-- A session that never asks for more than one organisation behaves exactly as
-- it does now — app.allowed_tenants is unset, and the policy falls back to
-- app.current_tenant, which is the clause 00029 shipped.
--
-- `tenant_id IS NULL` keeps meaning "belongs to the platform, readable by
-- everyone" — the same idea as Odoo's `company_id = False`, and already how
-- the shared AI prompts are stored.

-- +goose Up

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS allowed_tenant_ids UUID[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN sessions.allowed_tenant_ids IS
    'Organisations this session reads across. Empty means only tenant_id, which is what every session started as. tenant_id is always the one it writes into.';

-- +goose StatementBegin
DO $rls$
DECLARE target RECORD;
BEGIN
    FOR target IN
        SELECT c.table_name
          FROM information_schema.columns c
          JOIN information_schema.tables t
            ON t.table_schema = c.table_schema AND t.table_name = c.table_name
         WHERE c.table_schema = 'public'
           AND c.column_name = 'tenant_id'
           AND t.table_type = 'BASE TABLE'
    LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', target.table_name);
        EXECUTE format('ALTER TABLE public.%I FORCE ROW LEVEL SECURITY', target.table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', target.table_name);
        -- The array arrives already in PostgreSQL's literal form ({a,b}), so it
        -- casts in one step. COALESCE is what keeps a session that asked for
        -- nothing on exactly the old behaviour.
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON public.%I TO gerege_nexus_app '
            'USING (tenant_id IS NULL OR tenant_id = ANY (COALESCE('
            '    NULLIF(current_setting(''app.allowed_tenants'', true), '''')::uuid[], '
            '    ARRAY[NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid]))) '
            'WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)',
            target.table_name);
    END LOOP;
END
$rls$;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DO $rls$
DECLARE target RECORD;
BEGIN
    FOR target IN
        SELECT c.table_name
          FROM information_schema.columns c
          JOIN information_schema.tables t
            ON t.table_schema = c.table_schema AND t.table_name = c.table_name
         WHERE c.table_schema = 'public'
           AND c.column_name = 'tenant_id'
           AND t.table_type = 'BASE TABLE'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', target.table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON public.%I TO gerege_nexus_app '
            'USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid) '
            'WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)',
            target.table_name);
    END LOOP;
END
$rls$;
-- +goose StatementEnd

ALTER TABLE sessions DROP COLUMN IF EXISTS allowed_tenant_ids;
