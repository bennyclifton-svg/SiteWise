-- Tender comparison (docs/design/2026-11-14-tender-comparison.md).

-- One switch per org covers every model use: the org agrees that its
-- project text may be sent to the model provider. Null is off.
ALTER TABLE orgs
 ADD COLUMN model_enabled_at timestamptz,
 ADD COLUMN model_enabled_by uuid;

-- Bumped by every write to a package's tender tables; an edit that names a
-- stale version is refused.
ALTER TABLE packages ADD COLUMN tender_version bigint NOT NULL DEFAULT 1 CHECK (tender_version > 0);

-- The tender pass on one submission. '' is never asked, or the pass is off.
ALTER TABLE submissions
 ADD COLUMN tender_state text NOT NULL DEFAULT '' CHECK (tender_state IN ('','pending','reading','read','failed')),
 ADD COLUMN tender_attempts integer NOT NULL DEFAULT 0 CHECK (tender_attempts >= 0),
 ADD COLUMN tender_retry_after timestamptz,
 ADD COLUMN tender_error text NOT NULL DEFAULT '' CHECK (length(tender_error) <= 200),
 ADD COLUMN tender_prompt text NOT NULL DEFAULT '' CHECK (length(tender_prompt) <= 40),
 ADD COLUMN tender_basis text NOT NULL DEFAULT '' CHECK (tender_basis IN ('','fixed','estimate','cost_plus','rates_only','unknown')),
 -- The firm's printed price for its base offer, ex GST, and how it was chosen.
 ADD COLUMN tender_sum_cents bigint,
 ADD COLUMN tender_sum_how text NOT NULL DEFAULT '' CHECK (length(tender_sum_how) <= 200),
 ADD COLUMN tender_sum_by text NOT NULL DEFAULT '' CHECK (tender_sum_by IN ('','rule','user')),
 -- The sum the pass itself chose, kept so that undoing the manager's own
 -- tender sum can put it back.
 ADD COLUMN tender_sum_read_cents bigint,
 ADD COLUMN tender_sum_read_how text NOT NULL DEFAULT '' CHECK (length(tender_sum_read_how) <= 200),
 ADD COLUMN tender_flags jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tender_flags) = 'array'),
 ADD COLUMN tender_notes jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tender_notes) = 'array'),
 ADD COLUMN tender_exclusions jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(tender_exclusions) = 'array'),
 ADD COLUMN tender_read_at timestamptz,
 -- The manager's acceptance of the reading (AGENTS.md rule 1).
 ADD COLUMN tender_checked_at timestamptz,
 ADD COLUMN tender_checked_by uuid,
 ADD CONSTRAINT submissions_tender_checked CHECK ((tender_checked_at IS NULL) = (tender_checked_by IS NULL));
CREATE INDEX submissions_tender_due_idx ON submissions (tender_retry_after) WHERE tender_state IN ('pending','failed');

-- One line a firm printed. amount_cents is ex GST; printed is the firm's
-- own figure as text. A line with no amount is priced-but-unknown,
-- included, excluded or not priced.
CREATE TABLE tender_lines (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 page integer NOT NULL DEFAULT 0 CHECK (page >= 0),
 label text NOT NULL CHECK (length(label) <= 500),
 printed text NOT NULL DEFAULT '' CHECK (length(printed) <= 80),
 amount_cents bigint,
 gst text NOT NULL DEFAULT 'unknown' CHECK (gst IN ('ex','inc','unknown')),
 state text NOT NULL CHECK (state IN ('priced','included','excluded','not_priced')),
 kind text NOT NULL CHECK (kind IN ('base','ps','pc','option','saving','rate','subtotal','total','gst')),
 -- sum: its own part of the price. inside: already in another line.
 -- outside: extra to the price. ("where" is a reserved word.)
 placed text NOT NULL CHECK (placed IN ('sum','inside','outside')),
 heading text NOT NULL DEFAULT '' CHECK (length(heading) <= 300),
 row_key text CHECK (row_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 note text NOT NULL DEFAULT '' CHECK (length(note) <= 300),
 counted boolean NOT NULL DEFAULT false,
 flags jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(flags) = 'array'),
 decided_by text NOT NULL CHECK (decided_by IN ('model','rule','user')),
 split_of uuid,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 CHECK (state = 'priced' OR amount_cents IS NULL),
 CHECK (NOT counted OR amount_cents IS NOT NULL)
);
CREATE INDEX tender_lines_submission_idx ON tender_lines (org_id, submission_id, ordinal);
