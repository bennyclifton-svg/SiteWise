-- A project the PM has put away: off the Projects list unless they ask for
-- archived ones, with everything in it kept. Archive is not delete; nothing
-- else reads this column (owner, 11 Nov 2026, projects page).
ALTER TABLE projects ADD COLUMN archived_at timestamptz;
