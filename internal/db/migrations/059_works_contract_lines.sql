-- An awarded builder's or trade's price sits in the cost plan as works
-- lines under its package. Such a contract line names no work item and no
-- system, so the works rule admits a line that names its package instead.
ALTER TABLE cost_item_revisions DROP CONSTRAINT cost_line_target;
ALTER TABLE cost_item_revisions ADD CONSTRAINT cost_line_target CHECK (
 (line_kind='works' AND (work_item_id IS NOT NULL OR system_id IS NOT NULL OR package_id IS NOT NULL) AND package_stage_id IS NULL AND category IS NULL)
 OR (line_kind='fee' AND work_item_id IS NULL AND system_id IS NULL AND package_id IS NOT NULL AND category IS NULL)
 OR (line_kind='project_wide' AND work_item_id IS NULL AND system_id IS NULL AND package_id IS NULL AND package_stage_id IS NULL AND category IN ('contingency','escalation','authority_fees','exclusions','other'))
);
