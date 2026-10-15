-- Tenant deletion removes both the specification and its actor attribution.
-- Deleting a referenced user independently must still fail.
ALTER TABLE specification_types ALTER CONSTRAINT specification_types_org_id_created_by_fkey DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE specification_assertions ALTER CONSTRAINT specification_assertions_org_id_actor_fkey DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE specification_history ALTER CONSTRAINT specification_history_org_id_actor_fkey DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE specification_issues ALTER CONSTRAINT specification_issues_org_id_created_by_fkey DEFERRABLE INITIALLY DEFERRED;

CREATE FUNCTION protect_specification_issue() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD;END IF;
 RAISE EXCEPTION 'issued specifications are immutable' USING ERRCODE='23514';
END;
$$;
CREATE TRIGGER specification_issue_immutable BEFORE UPDATE OR DELETE ON specification_issues FOR EACH ROW EXECUTE FUNCTION protect_specification_issue();

CREATE FUNCTION protect_specification_history() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD;END IF;
 RAISE EXCEPTION 'specification history is immutable' USING ERRCODE='23514';
END;
$$;
CREATE TRIGGER specification_history_immutable BEFORE UPDATE OR DELETE ON specification_history FOR EACH ROW EXECUTE FUNCTION protect_specification_history();
