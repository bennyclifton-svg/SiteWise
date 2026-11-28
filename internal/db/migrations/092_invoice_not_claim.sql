-- An invoice the PM says is not for a progress claim
-- (docs/plans/2026-11-26-payment-schedules.md, slice 6). On a contract
-- with a claim still waiting for its tax invoice, an invoice no rule places
-- is held out of the totals until the PM places it (owner decision 2);
-- "not for a claim" makes it count as an ordinary invoice.
ALTER TABLE invoices ADD COLUMN not_claim boolean NOT NULL DEFAULT false;
