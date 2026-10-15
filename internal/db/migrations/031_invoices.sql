-- Invoice ledger (docs/plans/2026-10-16-invoice-intake-implementation.md).
-- The ledger is the one source of the cost plan's invoiced to date: nothing
-- here is a cached total. Money is integer cents in AUD; null is unknown.

-- The supplier rule matches an invoice's issuer ABN to a package. It is set
-- only when a user confirms an invoice's supplier, never by Jev.
ALTER TABLE packages ADD COLUMN supplier_abn text CHECK (supplier_abn ~ '^[0-9]{11}$');

CREATE TABLE invoices (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 file_id uuid NOT NULL,
 filename text NOT NULL CHECK (length(trim(filename)) BETWEEN 1 AND 255),
 status text NOT NULL CHECK (status IN ('reading','filed','needs_review','not_read','void','superseded')),
 kind text CHECK (kind IN ('invoice','credit_note','not_invoice')),
 number text CHECK (length(number) <= 100),
 invoice_date date,
 -- The month the invoice is certified in: the first day of that month.
 period date NOT NULL CHECK (extract(day FROM period) = 1),
 package_id uuid,
 issuer_abn text CHECK (issuer_abn ~ '^[0-9]{11}$'),
 currency char(3) NOT NULL DEFAULT 'AUD' CHECK (currency ~ '^[A-Z]{3}$'),
 ex_gst_cents bigint,
 gst_cents bigint,
 total_cents bigint,
 not_read_reason text NOT NULL DEFAULT '',
 -- Distinct invoice numbers seen in the file; more than one is flagged.
 invoice_count integer NOT NULL DEFAULT 1 CHECK (invoice_count >= 0),
 superseded_by uuid,
 duplicate_ok boolean NOT NULL DEFAULT false,
 retry_after timestamptz,
 -- Background re-checks after a grey read; bounded so a dead question stops.
 read_attempts integer NOT NULL DEFAULT 0 CHECK (read_attempts >= 0),
 uploaded_by uuid,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, project_id, id),
 -- The same bytes in the same project are one invoice: a re-drop is "Already filed".
 UNIQUE (org_id, project_id, file_id),
 FOREIGN KEY (org_id, project_id) REFERENCES projects(org_id, id) ON DELETE CASCADE,
 -- No cascade: deleting a register document must not take an invoice's bytes.
 FOREIGN KEY (org_id, file_id) REFERENCES files(org_id, id),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id),
 FOREIGN KEY (org_id, project_id, superseded_by) REFERENCES invoices(org_id, project_id, id),
 CHECK (superseded_by IS DISTINCT FROM id),
 CHECK ((status = 'superseded') = (superseded_by IS NOT NULL))
);
CREATE INDEX invoices_project_idx ON invoices (org_id, project_id, created_at DESC);
CREATE INDEX invoices_retry_idx ON invoices (retry_after) WHERE retry_after IS NOT NULL;

-- One row per activity line an invoice is charged to. A null line is the
-- Unallocated row. The line id is the stable cost item, so a rename or a new
-- draft keeps the allocation; line_label keeps the name a removed line had.
CREATE TABLE invoice_allocations (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 invoice_id uuid NOT NULL,
 cost_item_id uuid,
 amount_cents bigint,
 line_label text NOT NULL DEFAULT '',
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev','user')),
 band text NOT NULL CHECK (band IN ('green','amber','blank','grey')),
 ordinal integer NOT NULL DEFAULT 0 CHECK (ordinal >= 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, invoice_id, ordinal),
 FOREIGN KEY (org_id, project_id, invoice_id) REFERENCES invoices(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, cost_item_id) REFERENCES cost_items(org_id, project_id, id) ON DELETE CASCADE
);
CREATE INDEX invoice_allocations_line_idx ON invoice_allocations (org_id, project_id, cost_item_id);

-- One judgement per field per invoice, the same shape as filing decisions.
-- proposed_value keeps what rule or Jev said after a user corrects it, so a
-- correction is a labelled example for threshold calibration.
CREATE TABLE invoice_decisions (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 invoice_id uuid NOT NULL,
 field text NOT NULL CHECK (length(field) BETWEEN 1 AND 40),
 value text CHECK (length(value) <= 500),
 band text NOT NULL CHECK (band IN ('green','amber','blank','grey')),
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev','user')),
 question_version text,
 confidence double precision,
 proposed_value text CHECK (length(proposed_value) <= 500),
 proposed_by text CHECK (proposed_by IN ('rule','jev')),
 proposed_confidence double precision,
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, invoice_id, field),
 FOREIGN KEY (org_id, project_id, invoice_id) REFERENCES invoices(org_id, project_id, id) ON DELETE CASCADE
);
CREATE INDEX invoice_decisions_project_idx ON invoice_decisions (org_id, project_id);
