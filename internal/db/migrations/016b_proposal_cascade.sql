-- Keep audit references restrictive when deleting a user or work item alone,
-- but check them after the complete organisation/project cascade has finished.
ALTER TABLE proposal_decisions ALTER CONSTRAINT proposal_decisions_org_id_actor_fkey DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE proposal_decisions ALTER CONSTRAINT proposal_decisions_org_id_project_id_trigger_work_item_id_fkey DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE proposal_decisions ALTER CONSTRAINT proposal_decisions_org_id_project_id_created_record_id_fkey DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE proposal_triggers ALTER CONSTRAINT proposal_triggers_org_id_project_id_work_item_id_fkey DEFERRABLE INITIALLY DEFERRED;
