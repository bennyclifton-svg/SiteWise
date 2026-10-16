-- Procurement from fee proposal to award
-- (docs/plans/2026-10-16-procurement-to-award-implementation.md).
-- Money is integer cents in AUD, ex GST; null is unknown. Every key is
-- composite with org_id, so no row can point into another org.

-- One package's returnable schedule as rows. The RFP prints them, proposals
-- are read into them, the comparison lays them out and award copies its
-- price items into the cost plan. A row a refresh no longer produces is
-- retired, never deleted: prices and cost lines may still point at it.
CREATE TABLE schedule_items (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 key text NOT NULL CHECK (key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 kind text NOT NULL CHECK (kind IN ('stage','element','option','reimbursable','rate','criterion')),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 500),
 detail text NOT NULL DEFAULT '' CHECK (length(detail) <= 4000),
 base boolean NOT NULL DEFAULT false,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 -- rfp: the package's RFP draft or issue; default: the discipline's RFP with
 -- no settings; generic: a discipline with no RFP knowledge.
 source text NOT NULL CHECK (source IN ('rfp','default','generic')),
 retired_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, package_id, key),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id) ON DELETE CASCADE,
 CHECK (base = (kind IN ('stage','element','reimbursable')))
);
CREATE INDEX schedule_items_project_idx ON schedule_items (org_id, project_id, package_id, ordinal);

-- The firm directory is shared by all of an organisation's projects. A firm
-- first seen on a proposal is a draft until a user confirms it.
CREATE TABLE firms (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
 -- Lower-case letters and digits only, without Pty Ltd and the like: the
 -- name rule matches on it.
 name_key text NOT NULL CHECK (length(name_key) BETWEEN 1 AND 200),
 abn text CHECK (abn ~ '^[0-9]{11}$'),
 address text NOT NULL DEFAULT '' CHECK (length(address) <= 500),
 contact text NOT NULL DEFAULT '' CHECK (length(contact) <= 200),
 email text NOT NULL DEFAULT '' CHECK (length(email) <= 254),
 phone text NOT NULL DEFAULT '' CHECK (length(phone) <= 50),
 status text NOT NULL CHECK (status IN ('draft','confirmed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id)
);
CREATE UNIQUE INDEX firms_abn_uq ON firms (org_id, abn) WHERE abn IS NOT NULL;
CREATE INDEX firms_name_idx ON firms (org_id, name_key);

-- A submission is one firm's proposal or tender for one package. It is
-- opened when its document is filed; package and firm may still be unknown
-- while it is read. Deleting the document leaves the submission and its
-- prices (an award may rest on them).
CREATE TABLE submissions (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid,
 firm_id uuid,
 document_id uuid,
 filename text NOT NULL CHECK (length(trim(filename)) BETWEEN 1 AND 255),
 kind text CHECK (kind IN ('fee_proposal','tender')),
 status text NOT NULL CHECK (status IN ('reading','received','needs_review','awarded','unsuccessful','superseded','withdrawn','not_proposal')),
 received_on date NOT NULL,
 superseded_by uuid,
 -- The total the firm printed for its base fee, as printed, and on which basis.
 stated_total_cents bigint,
 gst_basis text NOT NULL DEFAULT 'unknown' CHECK (gst_basis IN ('ex','inc','unknown')),
 -- Plain flags code raised while reading: rates only, options A and B, no text.
 flags jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(flags) = 'array'),
 retry_after timestamptz,
 read_attempts integer NOT NULL DEFAULT 0 CHECK (read_attempts >= 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, project_id, id),
 FOREIGN KEY (org_id, project_id) REFERENCES projects(org_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id),
 FOREIGN KEY (org_id, firm_id) REFERENCES firms(org_id, id),
 FOREIGN KEY (org_id, project_id, document_id) REFERENCES documents(org_id, project_id, id) ON DELETE SET NULL (document_id),
 FOREIGN KEY (org_id, project_id, superseded_by) REFERENCES submissions(org_id, project_id, id),
 CHECK (superseded_by IS DISTINCT FROM id),
 CHECK ((status = 'superseded') = (superseded_by IS NOT NULL))
);
CREATE UNIQUE INDEX submissions_document_uq ON submissions (org_id, document_id) WHERE document_id IS NOT NULL;
CREATE INDEX submissions_package_idx ON submissions (org_id, project_id, package_id, received_on);
CREATE INDEX submissions_retry_idx ON submissions (retry_after) WHERE retry_after IS NOT NULL;

-- One judgement per field, the same shape as filing decisions. A user's
-- decision is final; proposed_value keeps what rule or Jev said before it.
CREATE TABLE submission_decisions (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 field text NOT NULL CHECK (length(field) BETWEEN 1 AND 220),
 value text NOT NULL DEFAULT '' CHECK (length(value) <= 2000),
 band text NOT NULL CHECK (band IN ('green','amber','blank','grey')),
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev','user')),
 question_version text NOT NULL DEFAULT '',
 confidence double precision,
 proposed_value text,
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, submission_id, field),
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE
);

-- The proposal split into sections by its headings, with the page each
-- starts on. Prices and non-price summaries cite these.
CREATE TABLE submission_passages (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 heading text NOT NULL DEFAULT '' CHECK (length(heading) <= 500),
 page integer NOT NULL DEFAULT 0 CHECK (page >= 0),
 body text NOT NULL CHECK (length(body) <= 20000),
 PRIMARY KEY (org_id, submission_id, ordinal),
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE
);

-- One cell of the price table. item_key null is an item the firm offered
-- unasked; it stays out of totals until award. amount is per hour for a rate.
CREATE TABLE submission_prices (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 item_key text CHECK (item_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 label text NOT NULL DEFAULT '' CHECK (length(label) <= 500),
 amount_cents bigint,
 state text NOT NULL CHECK (state IN ('priced','not_priced','included','excluded')),
 -- How the amount was reached, in words: "6% of the construction budget",
 -- "quoted inc GST, ÷ 1.1".
 basis text NOT NULL DEFAULT '' CHECK (length(basis) <= 500),
 page integer NOT NULL DEFAULT 0 CHECK (page >= 0),
 passage text NOT NULL DEFAULT '' CHECK (length(passage) <= 2000),
 band text NOT NULL CHECK (band IN ('green','amber','blank','grey')),
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev','user')),
 ordinal integer NOT NULL DEFAULT 0 CHECK (ordinal >= 0),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, id),
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 CHECK (state = 'priced' OR amount_cents IS NULL)
);
CREATE UNIQUE INDEX submission_prices_item_uq ON submission_prices (org_id, submission_id, item_key) WHERE item_key IS NOT NULL;

-- A non-price criterion for one firm: the passages that answer it, a
-- summary and the PM's rating. The rating is only ever set by a user.
CREATE TABLE submission_responses (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 item_key text NOT NULL CHECK (item_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 passages integer[] NOT NULL DEFAULT '{}',
 summary text NOT NULL DEFAULT '' CHECK (length(summary) <= 1000),
 rating text CHECK (rating IN ('strong','acceptable','concern')),
 rated_by uuid,
 band text NOT NULL DEFAULT 'blank' CHECK (band IN ('green','amber','blank','grey')),
 decided_by text NOT NULL DEFAULT 'rule' CHECK (decided_by IN ('rule','jev','user')),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, submission_id, item_key),
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, rated_by) REFERENCES users(org_id, id) DEFERRABLE INITIALLY DEFERRED,
 CHECK (rating IS NULL OR rated_by IS NOT NULL)
);

-- The one-page report per package: the recommendation the PM edits and
-- signs off. output holds a generated draft and its checks when a model is
-- enabled; input_sha256 is what it was drafted from.
CREATE TABLE comparison_reports (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 input_sha256 text NOT NULL DEFAULT '' CHECK (input_sha256 = '' OR input_sha256 ~ '^[0-9a-f]{64}$'),
 model text NOT NULL DEFAULT '' CHECK (length(model) <= 200),
 output jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(output) = 'object'),
 recommendation text NOT NULL DEFAULT '' CHECK (length(recommendation) <= 4000),
 recommendation_by uuid,
 signed_off_by uuid,
 signed_off_at timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, package_id),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, recommendation_by) REFERENCES users(org_id, id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY (org_id, signed_off_by) REFERENCES users(org_id, id) DEFERRABLE INITIALLY DEFERRED,
 CHECK ((signed_off_by IS NULL) = (signed_off_at IS NULL))
);

-- An award writes the firm's fee breakdown into the cost plan. parent_before
-- is the package line as it was, so undo restores it exactly.
CREATE TABLE awards (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 parent_item_id uuid NOT NULL,
 parent_created boolean NOT NULL,
 parent_before jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(parent_before) = 'object'),
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 undone_at timestamptz,
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, project_id, id),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, submission_id) REFERENCES submissions(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, parent_item_id) REFERENCES cost_items(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, created_by) REFERENCES users(org_id, id) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX awards_live_uq ON awards (org_id, package_id) WHERE undone_at IS NULL;

-- Each child cost line names the schedule items (or offered extras) it
-- holds. A line combining Stages 3 to 5 has three rows; that is how an
-- invoice for "Stage 4" finds it.
CREATE TABLE cost_line_sources (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 award_id uuid NOT NULL,
 cost_item_id uuid NOT NULL,
 item_key text CHECK (item_key ~ '^[A-Za-z0-9_:.\-]{1,200}$'),
 price_id uuid,
 label text NOT NULL DEFAULT '' CHECK (length(label) <= 500),
 ordinal integer NOT NULL DEFAULT 0 CHECK (ordinal >= 0),
 PRIMARY KEY (org_id, id),
 FOREIGN KEY (org_id, project_id, award_id) REFERENCES awards(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, project_id, cost_item_id) REFERENCES cost_items(org_id, project_id, id) ON DELETE CASCADE,
 CHECK ((item_key IS NULL) <> (price_id IS NULL))
);
CREATE INDEX cost_line_sources_line_idx ON cost_line_sources (org_id, project_id, cost_item_id);

-- An awarded package's line is a group whose children carry the contract;
-- the group keeps the budget the PM set. Only a budget may sit on a group.
CREATE OR REPLACE FUNCTION cost_posting_value() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.metric <> 'budget' AND EXISTS(SELECT 1 FROM cost_item_revisions
   WHERE org_id=NEW.org_id AND plan_version_id=NEW.plan_version_id
     AND cost_item_id=NEW.cost_item_id AND NOT posting) THEN
  RAISE EXCEPTION 'non-posting cost item' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
