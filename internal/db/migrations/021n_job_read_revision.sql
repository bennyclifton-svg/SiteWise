-- Which families of profile questions a finished label reading covered
-- (profile.ReadRevision). "Update project profile" reads a finished document
-- again when its revision is behind, so a question family added later (project
-- details, flags) reaches documents already read. Passages whose questions did
-- not change are answered from the call cache and cost no Jev call.
ALTER TABLE jobs ADD COLUMN read_revision text NOT NULL DEFAULT '';
