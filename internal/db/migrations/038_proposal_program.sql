-- Reading B: the R7 Program returnable in a proposal (plan
-- docs/plans/2026-10-18-program.md section 10.3). The proposal_program read
-- runs after the proposal read finishes, as its own background job with its
-- own question version. These columns are that job's state on the
-- submission it reads: reading, read, or not read and why; which package
-- stages came back grey, and when to ask them again. Its proposals are
-- program_readings with source 'proposal_r7'; nothing here changes the
-- program, and a reading stays hidden until the submission is awarded.
ALTER TABLE submissions ADD COLUMN program_read text CHECK (program_read IN ('reading','read','not_read'));
ALTER TABLE submissions ADD COLUMN program_not_read text;
ALTER TABLE submissions ADD COLUMN program_grey jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(program_grey)='array');
ALTER TABLE submissions ADD COLUMN program_read_attempts integer NOT NULL DEFAULT 0 CHECK (program_read_attempts>=0);
ALTER TABLE submissions ADD COLUMN program_retry_after timestamptz;
