-- Tenant deletion cascades through several independent roots. Check composite
-- references at transaction end, after all tenant-owned rows have cascaded.
DO $$
DECLARE r record;
BEGIN
 FOR r IN SELECT conrelid::regclass AS tbl,conname FROM pg_constraint WHERE contype='f' AND conrelid IN ('cost_plan_versions'::regclass,'cost_plans'::regclass,'cost_items'::regclass,'cost_item_revisions'::regclass,'cost_values'::regclass,'scope_cost_links'::regclass) AND confrelid<>'orgs'::regclass LOOP
  EXECUTE format('ALTER TABLE %s ALTER CONSTRAINT %I DEFERRABLE INITIALLY DEFERRED',r.tbl,r.conname);
 END LOOP;
END $$;
