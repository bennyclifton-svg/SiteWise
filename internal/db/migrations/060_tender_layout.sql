-- The manager's own layout of a tender comparison (owner, 15 Nov 2026),
-- seen by everyone on the project:
-- a line's slot is its place within its row, shared across firms, so like
-- lines sit side by side. NULL leaves the line to SiteWise's word matching.
-- A row's slots are written together, so the change log can undo them.
ALTER TABLE tender_lines ADD COLUMN slot integer CHECK (slot >= 0 AND slot < 1000);

-- The firms' left-to-right order in the package's price table. Firms not
-- named here follow, in the order their proposals were received.
ALTER TABLE packages ADD COLUMN firm_order uuid[] NOT NULL DEFAULT '{}';
