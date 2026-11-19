-- Variations, progress claims and payment schedules
-- (docs/plans/2026-11-17-variations-progress-claims.md, Task 2). Money is
-- integer cents ex GST; null is unknown. Variation sums, contract sum to
-- date, certified to date and due dates are computed at read, never stored.

-- A progress claim is an invoice of its own kind; a tax invoice for the
-- certified money names the claim it settles and is not counted again.
ALTER TABLE invoices DROP CONSTRAINT invoices_kind_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_kind_check CHECK (kind IN ('invoice','credit_note','not_invoice','progress_claim'));
ALTER TABLE invoices ADD COLUMN settles_claim_id uuid;
ALTER TABLE invoices ADD CONSTRAINT invoices_settles_claim_fk FOREIGN KEY (org_id, project_id, settles_claim_id) REFERENCES invoices(org_id, project_id, id);
ALTER TABLE invoices ADD CONSTRAINT invoices_settles_not_self CHECK (settles_claim_id IS DISTINCT FROM id);

-- The contract's own periods, typed once per contract; null is not entered.
ALTER TABLE awards ADD COLUMN schedule_days smallint CHECK (schedule_days BETWEEN 0 AND 60);
ALTER TABLE awards ADD COLUMN schedule_basis text NOT NULL DEFAULT 'business' CHECK (schedule_basis IN ('business','calendar'));
ALTER TABLE awards ADD COLUMN payment_days smallint CHECK (payment_days BETWEEN 0 AND 120);
ALTER TABLE awards ADD COLUMN payment_basis text NOT NULL DEFAULT 'business' CHECK (payment_basis IN ('business','calendar'));

-- A variation to a live contract. It holds cents that flow into the cost
-- plan through its line, so it carries the capture trigger for Undo.
CREATE TABLE variations (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 award_id uuid NOT NULL,
 cost_item_id uuid NOT NULL,
 number text NOT NULL CHECK (number ~ '^[A-Za-z0-9][A-Za-z0-9 ./-]{0,19}$'),
 title text NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 200),
 details text NOT NULL DEFAULT '' CHECK (length(details) <= 4000),
 status text NOT NULL CHECK (status IN ('proposed','submitted','approved','rejected','withdrawn')),
 claimed_cents bigint,
 estimate_cents bigint,
 approved_cents bigint,
 time_claimed_days integer CHECK (time_claimed_days BETWEEN 0 AND 3650),
 submitted_on date,
 decided_on date,
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, project_id, id),
 FOREIGN KEY (org_id, project_id) REFERENCES projects(org_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id),
 FOREIGN KEY (org_id, project_id, award_id) REFERENCES awards(org_id, project_id, id),
 FOREIGN KEY (org_id, project_id, cost_item_id) REFERENCES cost_items(org_id, project_id, id),
 FOREIGN KEY (org_id, created_by) REFERENCES users(org_id, id) DEFERRABLE INITIALLY DEFERRED,
 CHECK ((status = 'approved') = (approved_cents IS NOT NULL))
);
CREATE UNIQUE INDEX variations_number_uq ON variations (org_id, package_id, upper(number));
CREATE INDEX variations_project_idx ON variations (org_id, project_id, package_id);

-- A document (a filed letter or message) linked to a variation.
CREATE TABLE variation_documents (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 variation_id uuid NOT NULL,
 document_id uuid NOT NULL,
 PRIMARY KEY (org_id, variation_id, document_id),
 FOREIGN KEY (org_id, project_id, variation_id) REFERENCES variations(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, document_id) REFERENCES documents(org_id, project_id, id) ON DELETE CASCADE
);

-- One row of a progress claim's table: the builder's three amounts as
-- printed, the line or variation it charges for, and the PM's assessment
-- to date (null is not assessed) with the reason when it is less.
CREATE TABLE progress_claim_lines (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 invoice_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 kind text NOT NULL CHECK (kind IN ('contract','variation','retention')),
 label text NOT NULL DEFAULT '' CHECK (length(label) <= 500),
 page integer CHECK (page > 0),
 cost_item_id uuid,
 variation_id uuid,
 mapped_by text NOT NULL CHECK (mapped_by IN ('rule','jev','user')),
 band text NOT NULL CHECK (band IN ('green','amber','blank','grey')),
 to_date_cents bigint,
 previous_cents bigint,
 this_claim_cents bigint,
 assessed_to_date_cents bigint,
 reason text NOT NULL DEFAULT '' CHECK (length(reason) <= 1000),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, invoice_id, ordinal),
 FOREIGN KEY (org_id, project_id, invoice_id) REFERENCES invoices(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, cost_item_id) REFERENCES cost_items(org_id, project_id, id),
 FOREIGN KEY (org_id, project_id, variation_id) REFERENCES variations(org_id, project_id, id),
 CHECK (cost_item_id IS NULL OR variation_id IS NULL),
 CHECK (kind <> 'retention' OR (cost_item_id IS NULL AND variation_id IS NULL))
);
CREATE INDEX progress_claim_lines_project_idx ON progress_claim_lines (org_id, project_id, invoice_id);

-- A payment schedule is a report of its own kind, one per claim. Every kind
-- 070 allowed stays (migrations may run out of number order).
ALTER TABLE reports DROP CONSTRAINT reports_kind_check;
ALTER TABLE reports ADD CONSTRAINT reports_kind_check CHECK (kind IN ('pmp','rfp','rft','monthly','payment_schedule'));
ALTER TABLE reports ADD COLUMN invoice_id uuid;
ALTER TABLE reports ADD CONSTRAINT reports_invoice_fk FOREIGN KEY (org_id, project_id, invoice_id) REFERENCES invoices(org_id, project_id, id);
ALTER TABLE reports ADD CONSTRAINT reports_invoice_check CHECK ((kind = 'payment_schedule') = (invoice_id IS NOT NULL));
CREATE UNIQUE INDEX reports_one_schedule ON reports (org_id, invoice_id) WHERE kind = 'payment_schedule';

-- Writes from chat and the screen have one Undo. Every area 070 allowed
-- stays.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_area_check CHECK (area IN ('cost_plan','profile','procurement','delivery','reports','variations','claims'));
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON variations
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','updated_at');
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON variation_documents
 FOR EACH ROW EXECUTE FUNCTION capture_change();
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON progress_claim_lines
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');

-- Frozen means frozen: once a claim's payment schedule is issued its rows
-- are never changed, by a screen, by chat or by Undo. A delete cascading
-- from the project (a nested trigger) is let through.
CREATE FUNCTION protect_issued_claim_lines() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
 o uuid := COALESCE(NEW.org_id, OLD.org_id);
 inv uuid := COALESCE(NEW.invoice_id, OLD.invoice_id);
BEGIN
 IF pg_trigger_depth() > 1 THEN
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
 END IF;
 IF EXISTS (SELECT 1 FROM reports r JOIN report_versions v ON v.org_id = r.org_id AND v.report_id = r.id AND v.status = 'issued'
            WHERE r.org_id = o AND r.kind = 'payment_schedule' AND r.invoice_id = inv) THEN
  RAISE EXCEPTION 'an issued payment schedule''s claim rows are frozen' USING ERRCODE = '23514';
 END IF;
 IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER claim_lines_issued BEFORE INSERT OR UPDATE OR DELETE ON progress_claim_lines
 FOR EACH ROW EXECUTE FUNCTION protect_issued_claim_lines();
