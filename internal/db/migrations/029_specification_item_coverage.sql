-- Item responsibility survives reclassification. A physical-system label must
-- never allow a second competing coverage decision for the same item.
CREATE UNIQUE INDEX specification_coverage_item_uq ON specification_coverage(org_id,project_id,type_id) WHERE type_id IS NOT NULL;
