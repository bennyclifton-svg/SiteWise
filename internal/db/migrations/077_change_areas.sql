-- Repairs and additions after 075 and 076 (mail). Those two files were
-- edited in place on the branch before it merged; they are back to their
-- first versions, and everything added since is here, written so it runs
-- on a database that applied either version (security review LOW-6).

-- The change log's areas, whole, after both 073 (variations, claims) and
-- 076 (mail): a database that ran 076 before 073 existed would otherwise
-- lose 'mail' when 073 rewrote this check. Pending files apply in name
-- order, so this always runs after 073; it came in with the merge that
-- brought 073, so no database ran it without 073.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_area_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_area_check
 CHECK (area IN ('cost_plan','profile','procurement','delivery','reports','variations','claims','mail'));

-- What a sent message's replies go to, and the outbox's last attempt.
ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS reply_to text NOT NULL DEFAULT '' CHECK (length(reply_to) <= 1000);
ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS send_tried_at timestamptz;
DROP INDEX IF EXISTS mail_queued;
CREATE INDEX mail_queued ON mail_messages(send_tried_at) WHERE send_status = 'queued';

-- The raw message's hash, so a second message with the same Message-ID but
-- other bytes is held rather than dropped (review HIGH-1), and which daily
-- pool a send drew on (review MEDIUM-3).
ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS raw_sha256 bytea CHECK (raw_sha256 IS NULL OR octet_length(raw_sha256) = 32);
ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS send_pool text NOT NULL DEFAULT '' CHECK (send_pool IN ('','free','paid'));
CREATE INDEX IF NOT EXISTS mail_sent_day ON mail_messages(received_at) WHERE number IS NOT NULL;

-- Held reasons: every earlier one, and duplicate (a repeated Message-ID
-- with other bytes).
ALTER TABLE mail_messages DROP CONSTRAINT IF EXISTS mail_messages_held_reason_check;
ALTER TABLE mail_messages ADD CONSTRAINT mail_messages_held_reason_check
 CHECK (held_reason IN ('','unknown_sender','unverified_sender','plan_limit','archived','held_full','busy','duplicate'));

-- Frozen means frozen: only the decision, the PM's two edits, Jev's pick
-- and the send state move.
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

-- Only the PM's two edits (package, the pick) are captured for Undo.
DROP TRIGGER IF EXISTS capture_change ON mail_messages;
CREATE TRIGGER capture_change AFTER UPDATE ON mail_messages
 FOR EACH ROW EXECUTE FUNCTION capture_change('version','status','held_reason','decided_by','firm_id',
  'notice_read','notice_confidence','notice_band','send_status','send_attempts','send_error','sent_at','send_tried_at');

-- Bytes the mail code stored and no part names yet, per request, so one
-- request never clears another's (reviews H2, LOW-5). Cleared once named
-- or removed; a sweep removes what a crash left behind.
DROP TABLE IF EXISTS mail_staged_blobs;
CREATE TABLE IF NOT EXISTS mail_blob_stages (
 sha256 bytea NOT NULL CHECK (octet_length(sha256)=32),
 request text NOT NULL CHECK (length(request) BETWEEN 1 AND 64),
 staged_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (sha256, request)
);
CREATE INDEX IF NOT EXISTS mail_blob_stages_at ON mail_blob_stages(staged_at);
