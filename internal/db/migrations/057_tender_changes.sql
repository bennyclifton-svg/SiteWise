-- Tender line edits join the project change log, so each can be undone
-- (docs/plans/2026-11-14-tender-comparison-slice4.md, task 2).
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_area_check CHECK (area IN ('cost_plan','profile','procurement'));

-- submission_prices and submissions are not captured: both are rewritten
-- from the lines after every edit and every undo (refreshTender).
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON tender_lines
 FOR EACH ROW EXECUTE FUNCTION capture_change('updated_at');
