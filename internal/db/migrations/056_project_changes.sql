-- One row per change applied through the operations entry point. Rows hold
-- each record the change wrote, before and after, so it can be undone.
CREATE TABLE project_changes (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 project_id uuid NOT NULL,
 id uuid NOT NULL,
 n bigint GENERATED ALWAYS AS IDENTITY,
 actor uuid NOT NULL,
 via text NOT NULL CHECK (via IN ('chat','screen')),
 turn_id uuid,
 area text NOT NULL CHECK (area IN ('cost_plan','profile')),
 ops jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(ops)='array'),
 undone_at timestamptz,
 at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);
CREATE INDEX project_changes_recent ON project_changes(org_id,project_id,n DESC);

CREATE TABLE project_change_rows (
 org_id uuid NOT NULL,
 change_id uuid NOT NULL,
 n bigint GENERATED ALWAYS AS IDENTITY,
 tbl text NOT NULL,
 op text NOT NULL CHECK (op IN ('INSERT','UPDATE','DELETE')),
 old jsonb,
 new jsonb,
 PRIMARY KEY (org_id,change_id,n),
 FOREIGN KEY (org_id,change_id) REFERENCES project_changes(org_id,id) ON DELETE CASCADE,
 CHECK ((op='INSERT')=(old IS NULL)), CHECK ((op='DELETE')=(new IS NULL))
);

-- The mark is transaction-local JSON: {"id","org","project","actor","via",
-- "turn","area"}. With no mark this returns at once, so a screen write pays
-- one setting lookup per row and nothing else. Trigger arguments name the
-- columns a later write may bump without it counting as a change.
CREATE FUNCTION capture_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 mark text := current_setting('sitewise.change', true);
 c jsonb; o jsonb; n jsonb; r jsonb; ignore text[] := '{}';
BEGIN
 IF mark IS NULL OR mark = '' THEN RETURN NULL; END IF;
 c := mark::jsonb;
 FOR i IN 0..TG_NARGS-1 LOOP ignore := ignore || TG_ARGV[i]; END LOOP;
 IF TG_OP <> 'INSERT' THEN o := to_jsonb(OLD); END IF;
 IF TG_OP <> 'DELETE' THEN n := to_jsonb(NEW); END IF;
 IF TG_OP = 'UPDATE' AND (o - ignore) = (n - ignore) THEN RETURN NULL; END IF;
 r := COALESCE(n, o);
 IF r->>'org_id' <> c->>'org' OR (r ? 'project_id' AND r->>'project_id' <> c->>'project') THEN
  RAISE EXCEPTION 'change crosses its project' USING ERRCODE='23514';
 END IF;
 INSERT INTO project_changes(org_id,project_id,id,actor,via,turn_id,area)
 VALUES ((c->>'org')::uuid,(c->>'project')::uuid,(c->>'id')::uuid,(c->>'actor')::uuid,
  c->>'via',NULLIF(c->>'turn','')::uuid,c->>'area')
 ON CONFLICT (org_id,id) DO NOTHING;
 INSERT INTO project_change_rows(org_id,change_id,tbl,op,old,new)
 VALUES ((c->>'org')::uuid,(c->>'id')::uuid,TG_TABLE_NAME,TG_OP,o,n);
 RETURN NULL;
END $$;

CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_items
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_item_revisions
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_values
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON scope_cost_links
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_suggestion_dismissals
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_plan_versions
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON profile_user_values
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','updated_at');
