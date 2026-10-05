-- A project's documents are listed by project; without this index the
-- planner can only narrow by org, and on stale statistics (right after a
-- bulk filing) it joins every document and decision in the org (F30).
CREATE INDEX documents_by_project ON documents (org_id, project_id);
