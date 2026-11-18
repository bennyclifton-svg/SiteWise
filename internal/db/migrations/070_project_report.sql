-- The project report (docs/plans/2026-11-17-monthly-report.md, Task 1): one
-- project-level report kind beside the PMP, one per project.
ALTER TABLE reports DROP CONSTRAINT reports_kind_check;
ALTER TABLE reports ADD CONSTRAINT reports_kind_check CHECK (kind IN ('pmp','rfp','rft','monthly'));
ALTER TABLE reports DROP CONSTRAINT reports_check;
ALTER TABLE reports ADD CONSTRAINT reports_package_check CHECK (kind IN ('pmp','monthly') OR package_id IS NOT NULL);
CREATE UNIQUE INDEX reports_one_monthly ON reports(org_id, project_id) WHERE kind = 'monthly';

-- A comment written from chat has Undo (Task 10): report_edits is captured,
-- and the change log may name the reports area. A later save bumps version
-- and updated_at without that counting as a change.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_area_check CHECK (area IN ('cost_plan','profile','procurement','reports'));
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON report_edits
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','updated_at');
