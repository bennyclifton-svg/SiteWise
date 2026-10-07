ALTER TABLE proposal_decisions DROP CONSTRAINT proposal_decisions_created_record_type_check;
ALTER TABLE proposal_decisions DROP CONSTRAINT proposal_decisions_check1;
ALTER TABLE proposal_decisions DROP CONSTRAINT proposal_decisions_org_id_project_id_created_record_id_fkey;
ALTER TABLE proposal_decisions ADD CONSTRAINT proposal_decisions_created_record_type_check CHECK (created_record_type IN ('work_item','package'));
ALTER TABLE proposal_decisions ADD CONSTRAINT proposal_decisions_created_record_consistent CHECK (
 (decision='dismissed' AND created_record_type IS NULL AND created_record_id IS NULL)
 OR (decision='accepted' AND created_record_type IS NOT NULL AND created_record_id IS NOT NULL));
-- Conditional FK anchors keep the polymorphic public identity tenant/project
-- safe without storing a second editable copy of the target ID.
ALTER TABLE proposal_decisions ADD COLUMN created_work_item_id uuid GENERATED ALWAYS AS (CASE WHEN created_record_type='work_item' THEN created_record_id END) STORED;
ALTER TABLE proposal_decisions ADD COLUMN created_package_id uuid GENERATED ALWAYS AS (CASE WHEN created_record_type='package' THEN created_record_id END) STORED;
ALTER TABLE proposal_decisions ADD FOREIGN KEY (org_id,project_id,created_work_item_id) REFERENCES work_items(org_id,project_id,id) DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE proposal_decisions ADD FOREIGN KEY (org_id,project_id,created_package_id) REFERENCES packages(org_id,project_id,id) DEFERRABLE INITIALLY DEFERRED;
