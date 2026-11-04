-- Professional indemnity a letter states (Walsh review, 2026-10-27): only
-- the cover limit and the last day of the stated period, read in Go from the
-- submission's text. The insurer and policy number stay in the text. An
-- expired period shows "request a current certificate"; nothing is decided
-- from it.
ALTER TABLE submissions ADD COLUMN pi_limit_cents bigint CHECK (pi_limit_cents > 0);
ALTER TABLE submissions ADD COLUMN pi_expires_on date;
