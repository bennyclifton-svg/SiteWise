-- The ask-first review (plan docs/plans/2026-10-18-program.md section 7.4).
--
-- basis: the review does not raise again a difference the user kept until
-- its inputs change. It is a hash of the change as raised (its kind, source
-- key and the ops Adopt would apply, with row and link ids written as their
-- source keys so an unrelated adopt does not alter it). A difference whose
-- basis equals the latest row for its source key, when that row was kept, is
-- not raised. Null on rows written before this column, which never match.
--
-- ordinal: the change's place in the review's diff order. Adopt applies
-- changes in this order so a stage is added before its activities and an
-- activity lands after the sibling generation put before it.
ALTER TABLE program_changes ADD COLUMN basis text;
ALTER TABLE program_changes ADD COLUMN ordinal integer NOT NULL DEFAULT 0 CHECK (ordinal>=0);
