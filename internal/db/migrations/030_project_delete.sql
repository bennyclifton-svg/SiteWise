-- Deleting a project is the one act that removes its frozen cost baselines,
-- issued reports and issued specifications. Those stay immutable for every
-- other delete. DeleteProject sets this flag for its own transaction; a
-- missing setting is not a bypass.
CREATE OR REPLACE FUNCTION protect_cost_version() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE target uuid; tenant uuid; frozen boolean;
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 IF TG_TABLE_NAME='cost_plan_versions' THEN
  IF OLD.frozen_at IS NOT NULL THEN RAISE EXCEPTION 'frozen cost plan' USING ERRCODE='23514'; END IF;
 ELSE
  IF TG_OP='DELETE' THEN target:=OLD.plan_version_id; tenant:=OLD.org_id; ELSE target:=NEW.plan_version_id; tenant:=NEW.org_id; END IF;
  SELECT frozen_at IS NOT NULL INTO frozen FROM cost_plan_versions WHERE org_id=tenant AND id=target;
  IF frozen THEN RAISE EXCEPTION 'frozen cost plan' USING ERRCODE='23514'; END IF;
  IF TG_OP='UPDATE' AND (OLD.org_id<>NEW.org_id OR OLD.plan_version_id<>NEW.plan_version_id) THEN RAISE EXCEPTION 'cost ownership is immutable' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION protect_issued_report() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 IF OLD.status='issued' THEN
  RAISE EXCEPTION 'issued report versions are immutable' USING ERRCODE='23514';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION protect_issued_report_content() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_status text;
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 IF TG_OP<>'INSERT' THEN
  SELECT status INTO parent_status FROM report_versions WHERE org_id=OLD.org_id AND id=OLD.report_version_id FOR UPDATE;
  IF parent_status='issued' THEN RAISE EXCEPTION 'issued report content is immutable' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_OP<>'DELETE' THEN
  SELECT status INTO parent_status FROM report_versions WHERE org_id=NEW.org_id AND id=NEW.report_version_id FOR UPDATE;
  IF parent_status='issued' THEN RAISE EXCEPTION 'issued report content is immutable' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION protect_specification_issue() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 RAISE EXCEPTION 'issued specifications are immutable' USING ERRCODE='23514';
END $$;

CREATE OR REPLACE FUNCTION protect_specification_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 RAISE EXCEPTION 'specification history is immutable' USING ERRCODE='23514';
END $$;
