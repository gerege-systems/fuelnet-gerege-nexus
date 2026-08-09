-- +goose Up
-- The login role used by the service should be granted this role, but must not
-- own the tables. Each request transaction sets app.current_tenant before it
-- touches tenant data.
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'gerege_nexus_app') THEN
        CREATE ROLE gerege_nexus_app NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO gerege_nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO gerege_nexus_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gerege_nexus_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO gerege_nexus_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO gerege_nexus_app;

DO $rls$
DECLARE row RECORD;
BEGIN
    FOR row IN
        SELECT c.table_name
          FROM information_schema.columns c
         WHERE c.table_schema='public' AND c.column_name='tenant_id'
    LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', row.table_name);
        EXECUTE format('ALTER TABLE public.%I FORCE ROW LEVEL SECURITY', row.table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', row.table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON public.%I TO gerege_nexus_app USING (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant'', true), '''')::uuid)',
            row.table_name);
    END LOOP;
END $rls$;

-- +goose Down
DO $rls$
DECLARE row RECORD;
BEGIN
    FOR row IN
        SELECT c.table_name
          FROM information_schema.columns c
         WHERE c.table_schema='public' AND c.column_name='tenant_id'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON public.%I', row.table_name);
        EXECUTE format('ALTER TABLE public.%I DISABLE ROW LEVEL SECURITY', row.table_name);
    END LOOP;
END $rls$;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM gerege_nexus_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM gerege_nexus_app;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM gerege_nexus_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM gerege_nexus_app;
REVOKE USAGE ON SCHEMA public FROM gerege_nexus_app;
DROP ROLE IF EXISTS gerege_nexus_app;
