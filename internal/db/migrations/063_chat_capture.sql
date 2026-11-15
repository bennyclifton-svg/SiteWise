-- Undo for two more things chat can change at once
-- (docs/plans/2026-11-15-chat-agent.md, Task 3): naming or removing an
-- invited firm, and edits to the accommodation schedule.
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON package_invitations
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON accommodation_lines
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','updated_at');
