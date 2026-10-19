-- The question strip's "Not sure" (plan docs/plans/2026-10-18-program.md
-- section 7.3): a dismissed question stays away until its Profile value
-- changes. Each entry maps the Profile key (det.<id>, hdr.work_type,
-- hdr.cond.<key> or plan.<key>) to the projection hash its whole-project
-- profile row had when the user said "Not sure" ('' when there was no row).
-- The dismissal holds only while the row's hash is unchanged, so any edit or
-- new evidence on that key asks again.
ALTER TABLE programs ADD COLUMN question_dismissed jsonb NOT NULL DEFAULT '{}'
 CHECK (jsonb_typeof(question_dismissed)='object');
