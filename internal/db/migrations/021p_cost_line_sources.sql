-- system_id is also a frozen dimension on legacy work-item lines. Do not
-- reject or rewrite those snapshots when adding system-targeted lines.
DO $$ DECLARE constraint_name text; BEGIN
 SELECT conname INTO constraint_name FROM pg_constraint
 WHERE conrelid='cost_item_revisions'::regclass AND contype='c'
 AND pg_get_constraintdef(oid) LIKE '%package_stage_id IS NOT NULL%';
 IF constraint_name IS NOT NULL THEN
  EXECUTE format('ALTER TABLE cost_item_revisions DROP CONSTRAINT %I',constraint_name);
 END IF;
END $$;
ALTER TABLE cost_item_revisions ADD CONSTRAINT cost_line_target CHECK (
 (line_kind='works' AND (work_item_id IS NOT NULL OR system_id IS NOT NULL) AND package_stage_id IS NULL AND category IS NULL)
 OR (line_kind='fee' AND work_item_id IS NULL AND system_id IS NULL AND package_id IS NOT NULL AND category IS NULL)
 OR (line_kind='project_wide' AND work_item_id IS NULL AND system_id IS NULL AND package_id IS NULL AND package_stage_id IS NULL AND category IN ('contingency','escalation','authority_fees','exclusions','other'))
);
ALTER TABLE cost_item_revisions ADD COLUMN source_key text CHECK(source_key IS NULL OR length(source_key) BETWEEN 1 AND 200);
ALTER TABLE cost_item_revisions ADD COLUMN resolved_category text NOT NULL DEFAULT '';
ALTER TABLE cost_item_revisions ADD COLUMN scope_work_items jsonb NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(scope_work_items)='array');
CREATE UNIQUE INDEX cost_source_key_uq ON cost_item_revisions(org_id,plan_version_id,source_key) WHERE source_key IS NOT NULL;
CREATE TABLE cost_suggestion_dismissals (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 project_id uuid NOT NULL, source_key text NOT NULL CHECK(length(source_key) BETWEEN 1 AND 200),
 actor uuid NOT NULL, dismissed_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(org_id,project_id,source_key),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);
