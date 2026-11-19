-- Email intake (docs/plans/2026-11-14-email-intake.md, Task 2, as amended
-- by A2 in docs/plans/2026-11-17-correspondence-transmittals.md).
--
-- One inbound address per project. The local part is random and says
-- nothing about the org or the project: "p-" and 32 hex characters (128
-- bits). Rotating it is the PM's act.
CREATE TABLE project_mail_addresses (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 local_part text NOT NULL CHECK (local_part ~ '^p-[0-9a-f]{32}$'),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,project_id),
 UNIQUE (local_part),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);
INSERT INTO project_mail_addresses(org_id,project_id,local_part)
SELECT org_id,id,'p-' || replace(gen_random_uuid()::text,'-','') FROM projects;

-- One row per message received for a project. A filed message is a record
-- and is never edited or deleted (protect_mail_message below); a held one
-- waits for a member and may be discarded. Addresses and subject are kept
-- as text; the body is plain text, never HTML.
CREATE TABLE mail_messages (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 message_id text NOT NULL CHECK (length(message_id) BETWEEN 1 AND 998),
 webhook_token text NOT NULL CHECK (length(webhook_token) BETWEEN 1 AND 256),
 sender text NOT NULL CHECK (length(sender) <= 320),
 sender_name text NOT NULL DEFAULT '' CHECK (length(sender_name) <= 200),
 subject text NOT NULL DEFAULT '' CHECK (length(subject) <= 998),
 to_addrs text NOT NULL DEFAULT '' CHECK (length(to_addrs) <= 4000),
 cc_addrs text NOT NULL DEFAULT '' CHECK (length(cc_addrs) <= 4000),
 in_reply_to text NOT NULL DEFAULT '' CHECK (length(in_reply_to) <= 998),
 refs text NOT NULL DEFAULT '' CHECK (length(refs) <= 8000),
 body text NOT NULL DEFAULT '' CHECK (length(body) <= 20000),
 received_at timestamptz NOT NULL DEFAULT now(),
 status text NOT NULL CHECK (status IN ('filed','held')),
 held_reason text NOT NULL DEFAULT ''
  CHECK (held_reason IN ('','unknown_sender','unverified_sender','plan_limit','archived','held_full','busy')),
 decided_by uuid,
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,message_id),
 UNIQUE (webhook_token),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,decided_by) REFERENCES users(org_id,id) ON DELETE SET NULL (decided_by),
 CHECK ((status = 'held') = (held_reason <> ''))
);
CREATE INDEX mail_messages_project ON mail_messages(org_id,project_id,received_at DESC);
CREATE INDEX mail_messages_held ON mail_messages(org_id,project_id,received_at DESC) WHERE status='held';

-- The parts of a message: 0 is the body document, then each attachment.
-- A skipped part keeps its name and the reason, not its bytes.
CREATE TABLE mail_parts (
 org_id uuid NOT NULL,
 message_id uuid NOT NULL,
 n integer NOT NULL CHECK (n >= 0 AND n <= 100),
 filename text NOT NULL CHECK (length(filename) BETWEEN 1 AND 255),
 content_type text NOT NULL DEFAULT '' CHECK (length(content_type) <= 255),
 sha256 bytea CHECK (octet_length(sha256)=32),
 byte_size bigint NOT NULL DEFAULT 0 CHECK (byte_size >= 0),
 skipped text NOT NULL DEFAULT '' CHECK (skipped IN ('','inline_image','calendar','winmail','not_kept')),
 document_id uuid,
 PRIMARY KEY (org_id,message_id,n),
 FOREIGN KEY (org_id,message_id) REFERENCES mail_messages(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,document_id) REFERENCES documents(org_id,id) ON DELETE SET NULL (document_id),
 CHECK ((skipped = '') = (sha256 IS NOT NULL))
);
CREATE INDEX mail_parts_sha ON mail_parts(sha256) WHERE sha256 IS NOT NULL;

-- Every signed webhook token seen, whatever became of the post, so a
-- captured post cannot be replayed with another body inside the freshness
-- window (Mailgun signs only the timestamp and the token). Pruned after two
-- days; the webhook refuses timestamps older than one.
CREATE TABLE mail_webhook_tokens (
 token text PRIMARY KEY CHECK (length(token) BETWEEN 1 AND 256),
 seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mail_webhook_tokens_seen ON mail_webhook_tokens(seen_at);

-- Frozen means frozen: a filed message's received fields never change and
-- it is never deleted, except with its project or org. Held mail may be
-- discarded or expire. Only the decision columns move.
CREATE FUNCTION protect_mail_message() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' AND NOT EXISTS(SELECT 1 FROM orgs WHERE id=OLD.org_id) THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' AND current_setting('sitewise.deleting_project', true)='1' THEN RETURN OLD; END IF;
 IF TG_OP='DELETE' THEN
  IF OLD.status='held' THEN RETURN OLD; END IF;
  RAISE EXCEPTION 'a filed message is never deleted' USING ERRCODE='23514';
 END IF;
 IF (to_jsonb(OLD) - ARRAY['status','held_reason','decided_by']) IS DISTINCT FROM
    (to_jsonb(NEW) - ARRAY['status','held_reason','decided_by']) THEN
  RAISE EXCEPTION 'a received message is never edited' USING ERRCODE='23514';
 END IF;
 IF OLD.status='filed' AND NEW.status<>'filed' THEN
  RAISE EXCEPTION 'a filed message stays filed' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER protect_mail_message BEFORE UPDATE OR DELETE ON mail_messages
 FOR EACH ROW EXECUTE FUNCTION protect_mail_message();

-- A part keeps its bytes and name; only the document it became may be set,
-- or cleared when that document is deleted.
CREATE FUNCTION protect_mail_part() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN
  -- With its message (held, or the project going): the parent row is gone.
  IF EXISTS(SELECT 1 FROM mail_messages WHERE org_id=OLD.org_id AND id=OLD.message_id AND status='filed') THEN
   RAISE EXCEPTION 'a filed message keeps its parts' USING ERRCODE='23514';
  END IF;
  RETURN OLD;
 END IF;
 IF (to_jsonb(OLD) - 'document_id') IS DISTINCT FROM (to_jsonb(NEW) - 'document_id') THEN
  RAISE EXCEPTION 'a message part is never edited' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER protect_mail_part BEFORE UPDATE OR DELETE ON mail_parts
 FOR EACH ROW EXECUTE FUNCTION protect_mail_part();
