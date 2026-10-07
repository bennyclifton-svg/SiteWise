-- An issue remains immutable until the whole owning tenant is removed.
CREATE OR REPLACE FUNCTION protect_issued_report() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD;END IF;
 IF OLD.status='issued' THEN
  RAISE EXCEPTION 'issued report versions are immutable' USING ERRCODE='23514';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION protect_issued_report_content() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_status text;
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD;END IF;
 IF TG_OP<>'INSERT' THEN
  SELECT status INTO parent_status FROM report_versions WHERE org_id=OLD.org_id AND id=OLD.report_version_id FOR UPDATE;
  IF parent_status='issued' THEN RAISE EXCEPTION 'issued report content is immutable' USING ERRCODE='23514';END IF;
 END IF;
 IF TG_OP<>'DELETE' THEN
  SELECT status INTO parent_status FROM report_versions WHERE org_id=NEW.org_id AND id=NEW.report_version_id FOR UPDATE;
  IF parent_status='issued' THEN RAISE EXCEPTION 'issued report content is immutable' USING ERRCODE='23514';END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END;
$$;
