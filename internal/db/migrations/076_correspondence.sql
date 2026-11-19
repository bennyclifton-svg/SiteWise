-- Correspondence and transmittals over the mail tables
-- (docs/plans/2026-11-17-correspondence-transmittals.md, Data and Task 1).
-- A message is a record: In or Out, threaded, linked to a firm and a
-- package. What is owed is a register row, so there is no notice, due or
-- status column here, and Replied is computed at read, never stored.
ALTER TABLE mail_messages
 ALTER COLUMN webhook_token DROP NOT NULL,   -- a message SiteWise sent has none
 ADD COLUMN direction text NOT NULL DEFAULT 'in' CHECK (direction IN ('in','out')),
 ADD COLUMN thread_id uuid,                  -- the first message of its thread
 ADD COLUMN firm_id uuid,
 ADD COLUMN package_id uuid,
 ADD COLUMN linked_by text NOT NULL DEFAULT '' CHECK (linked_by IN ('','rule','user')),
 -- Jev's notice pick on the body document: a proposal, never a record.
 ADD COLUMN notice_read text NOT NULL DEFAULT ''
  CHECK (notice_read IN ('','none','rfi','direction','variation_claim','eot_delay','payment_claim','dispute','other_notice')),
 ADD COLUMN notice_confidence real CHECK (notice_confidence IS NULL OR (notice_confidence >= 0 AND notice_confidence <= 1)),
 -- amber only past an owner-approved cut-off; blank below it; grey unread.
 ADD COLUMN notice_band text NOT NULL DEFAULT '' CHECK (notice_band IN ('','amber','blank','grey')),
 ADD COLUMN pick_state text NOT NULL DEFAULT '' CHECK (pick_state IN ('','used','dismissed')),
 ADD COLUMN number integer CHECK (number > 0),      -- TR-001; sent by SiteWise only
 ADD COLUMN reply_to text NOT NULL DEFAULT '' CHECK (length(reply_to) <= 1000),  -- what a sent message's replies go to
 ADD COLUMN send_status text NOT NULL DEFAULT '' CHECK (send_status IN ('','queued','sent','failed')),
 ADD COLUMN send_attempts integer NOT NULL DEFAULT 0 CHECK (send_attempts >= 0),
 ADD COLUMN send_error text NOT NULL DEFAULT '' CHECK (length(send_error) <= 300),
 ADD COLUMN sent_at timestamptz,
 ADD COLUMN send_tried_at timestamptz,     -- the last attempt, for the sweep
 ADD COLUMN version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 ADD CONSTRAINT mail_sent_shape CHECK ((webhook_token IS NULL) = (number IS NOT NULL)
  AND (number IS NULL OR direction = 'out')
  AND ((number IS NULL) = (send_status = ''))),
 ADD FOREIGN KEY (org_id,firm_id) REFERENCES firms(org_id,id) ON DELETE SET NULL (firm_id),
 ADD FOREIGN KEY (org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id)
  ON DELETE SET NULL (package_id);
-- The backfill is the one write the frozen-record trigger must not see.
ALTER TABLE mail_messages DISABLE TRIGGER protect_mail_message;
UPDATE mail_messages SET thread_id = id WHERE thread_id IS NULL;
ALTER TABLE mail_messages ENABLE TRIGGER protect_mail_message;
ALTER TABLE mail_messages ALTER COLUMN thread_id SET NOT NULL;
CREATE UNIQUE INDEX mail_number_uq ON mail_messages(org_id,project_id,number) WHERE number IS NOT NULL;
CREATE INDEX mail_thread ON mail_messages(org_id,project_id,thread_id);
CREATE INDEX mail_queued ON mail_messages(send_tried_at) WHERE send_status = 'queued';
CREATE INDEX mail_parts_document ON mail_parts(org_id,document_id) WHERE document_id IS NOT NULL;
CREATE INDEX firms_email ON firms(org_id, lower(email)) WHERE email <> '';

-- A sent part keeps what went: the document's number, revision and title as
-- they were, so a later rename or delete never rewrites the transmittal.
ALTER TABLE mail_parts
 ADD COLUMN doc_number text NOT NULL DEFAULT '' CHECK (length(doc_number) <= 200),
 ADD COLUMN doc_revision text NOT NULL DEFAULT '' CHECK (length(doc_revision) <= 50),
 ADD COLUMN doc_title text NOT NULL DEFAULT '' CHECK (length(doc_title) <= 500);

-- Frozen means frozen (index, agreement 8): what was received or sent never
-- changes. Only the PM's two edits (package, the pick), the record's own
-- decision and send state, and Jev's pick may move.
CREATE OR REPLACE FUNCTION protect_mail_message() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE movable text[] := ARRAY['status','held_reason','decided_by','firm_id','package_id','linked_by',
 'notice_read','notice_confidence','notice_band','pick_state','send_status','send_attempts','send_error','sent_at','send_tried_at','version'];
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' THEN
  IF OLD.status='held' THEN RETURN OLD; END IF;
  RAISE EXCEPTION 'a filed message is never deleted' USING ERRCODE='23514';
 END IF;
 IF (to_jsonb(OLD) - movable) IS DISTINCT FROM (to_jsonb(NEW) - movable) THEN
  RAISE EXCEPTION 'a received or sent message is never edited' USING ERRCODE='23514';
 END IF;
 IF OLD.status='filed' AND NEW.status<>'filed' THEN
  RAISE EXCEPTION 'a filed message stays filed' USING ERRCODE='23514';
 END IF;
 IF OLD.send_status='sent' AND NEW.send_status<>'sent' THEN
  RAISE EXCEPTION 'a sent message stays sent' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;

-- The PM's two edits (package, the pick) are captured for Undo, as 063
-- does for other tables; the record's own columns (and version) moving
-- does not count as a change.
CREATE TRIGGER capture_change AFTER UPDATE ON mail_messages
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','status','held_reason','decided_by','firm_id',
  'notice_read','notice_confidence','notice_band','send_status','send_attempts','send_error','sent_at','send_tried_at');

-- The change log may name the mail area. Every area allowed before stays
-- (066 delivery, 070 reports); migrations are applied per file, so a
-- sibling migration that rewrites this check must keep 'mail' too.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_area_check
 CHECK (area IN ('cost_plan','profile','procurement','delivery','reports','variations','claims','mail'));
