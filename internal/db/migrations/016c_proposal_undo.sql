ALTER TABLE proposal_decisions ADD COLUMN created_record_version bigint CHECK (created_record_version>0);
UPDATE proposal_decisions SET created_record_version=1 WHERE decision='accepted';
ALTER TABLE proposal_decisions ADD COLUMN undone_at timestamptz;
ALTER TABLE proposal_decisions ADD COLUMN undo_history jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(undo_history)='array');
