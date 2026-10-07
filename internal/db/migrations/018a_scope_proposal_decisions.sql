ALTER TABLE proposal_decisions DROP CONSTRAINT proposal_decisions_created_record_type_check;
ALTER TABLE proposal_decisions ADD CONSTRAINT proposal_decisions_created_record_type_check CHECK (created_record_type IN ('work_item','package','package_scope_item'));
ALTER TABLE proposal_decisions ADD COLUMN created_scope_item_id uuid GENERATED ALWAYS AS (CASE WHEN created_record_type='package_scope_item' THEN created_record_id END) STORED;
ALTER TABLE proposal_decisions ADD FOREIGN KEY (org_id,project_id,created_scope_item_id) REFERENCES package_scope_items(org_id,project_id,id) DEFERRABLE INITIALLY DEFERRED;
