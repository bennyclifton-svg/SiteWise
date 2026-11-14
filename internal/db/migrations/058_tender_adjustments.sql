-- One addition or subtraction for a package: a row of the second table.
CREATE TABLE tender_adjustments (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 300),
 note text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
 origin text NOT NULL CHECK (origin IN ('system','user')),
 reason text NOT NULL CHECK (reason IN ('gap','allowance','option','saving','taken_out','other')),
 row_key text CHECK (row_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 -- Stable for a system row: line:<line id>, gap:<row key> or option:<item key>.
 suggestion_key text CHECK (length(suggestion_key) BETWEEN 1 AND 300),
 -- A system row the manager removed. It is kept so it does not come back.
 dismissed_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id) ON DELETE CASCADE,
 CHECK ((origin = 'system') = (suggestion_key IS NOT NULL))
);
CREATE UNIQUE INDEX tender_adjustments_key_uq ON tender_adjustments (org_id, package_id, suggestion_key) WHERE suggestion_key IS NOT NULL;
CREATE INDEX tender_adjustments_package_idx ON tender_adjustments (org_id, package_id, ordinal);

-- One firm's signed amount on one adjustment. Only applied amounts count.
-- A suggested gap has no amount until the manager chooses one.
CREATE TABLE tender_adjustment_amounts (
 org_id uuid NOT NULL,
 adjustment_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 project_id uuid NOT NULL,
 amount_cents bigint,
 state text NOT NULL CHECK (state IN ('suggested','applied')),
 -- firm: the firm's own printed figure. none: a gap with no figure yet.
 source text NOT NULL CHECK (source IN ('firm','others_highest','others_average','typed','none')),
 line_id uuid,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, adjustment_id, submission_id),
 FOREIGN KEY (org_id, adjustment_id) REFERENCES tender_adjustments(org_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 CHECK (state = 'suggested' OR amount_cents IS NOT NULL),
 CHECK ((source = 'none') = (amount_cents IS NULL))
);

-- Adds and subs join the project change log, as tender lines did.
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON tender_adjustments
 FOR EACH ROW EXECUTE FUNCTION capture_change('updated_at');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON tender_adjustment_amounts
 FOR EACH ROW EXECUTE FUNCTION capture_change('updated_at');
