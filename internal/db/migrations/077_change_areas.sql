-- The change log's areas, whole, after both 073 (variations, claims) and
-- 076 (mail) whichever applied first: migrations are tracked per file, so
-- a database that ran 076 before 073 existed would otherwise lose 'mail'
-- when 073 rewrote this check. This file runs after both.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_area_check
 CHECK (area IN ('cost_plan','profile','procurement','delivery','reports','variations','claims','mail'));
