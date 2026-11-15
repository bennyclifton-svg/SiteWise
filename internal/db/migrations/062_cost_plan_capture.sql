-- The first cost line of a project also creates its plan (a cost_plans row
-- pointing at the draft version). A marked change must capture that row
-- too, or undoing the change cannot remove the version it points at.
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON cost_plans
 FOR EACH ROW EXECUTE FUNCTION capture_change();
