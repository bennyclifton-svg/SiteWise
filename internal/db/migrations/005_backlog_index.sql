-- Health reads the age of unfinished work on every probe. Finished jobs
-- accumulate, so the index covers only queued and leased rows.
CREATE INDEX jobs_unfinished_idx ON jobs (kind, created_at)
    WHERE status IN ('queued', 'leased');
