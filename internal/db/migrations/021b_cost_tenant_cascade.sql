-- Complete tenant cascade ownership for the commercial tables. Frozen records
-- remain immutable while their owning tenant exists.
DO $$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['cost_plan_versions','cost_plans','cost_items','cost_item_revisions','cost_values','scope_cost_links'] LOOP
  EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY(org_id) REFERENCES orgs(id) ON DELETE CASCADE',t,t||'_org_cascade');
 END LOOP;
END $$;
CREATE OR REPLACE FUNCTION protect_cost_version() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE target uuid; tenant uuid; frozen boolean;
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD;END IF;
 IF TG_TABLE_NAME='cost_plan_versions' THEN
  IF OLD.frozen_at IS NOT NULL THEN RAISE EXCEPTION 'frozen cost plan' USING ERRCODE='23514'; END IF;
 ELSE
  IF TG_OP='DELETE' THEN target:=OLD.plan_version_id;tenant:=OLD.org_id;ELSE target:=NEW.plan_version_id;tenant:=NEW.org_id;END IF;
  SELECT frozen_at IS NOT NULL INTO frozen FROM cost_plan_versions WHERE org_id=tenant AND id=target;
  IF frozen THEN RAISE EXCEPTION 'frozen cost plan' USING ERRCODE='23514';END IF;
  IF TG_OP='UPDATE' AND (OLD.org_id<>NEW.org_id OR OLD.plan_version_id<>NEW.plan_version_id) THEN RAISE EXCEPTION 'cost ownership is immutable' USING ERRCODE='23514';END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;
END $$;
