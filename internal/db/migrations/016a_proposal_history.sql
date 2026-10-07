-- Retain the decision-time inputs so a reopened dismissal can explain its
-- change after the replaceable projection has been rebuilt.
ALTER TABLE proposal_decisions ADD COLUMN inputs_snapshot jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(inputs_snapshot)='object');
