-- One person outside the org sees one shared thing; an invited firm may send
-- its quote (docs/plans/2026-11-17-guest-access.md). No user, membership,
-- invite or session: never a seat, never admitted by a member route. The
-- token is kept only as its SHA-256. document_ids has no key: a deleted
-- document drops off. Not captured: sharing is not undone.
CREATE TABLE guest_grants (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
 kind text NOT NULL CHECK (kind IN ('documents','report','package')),
 shared_with text NOT NULL CHECK (length(trim(shared_with)) BETWEEN 1 AND 254),
 document_ids uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(document_ids) <= 200),
 report_id uuid,
 package_id uuid,
 firm_id uuid,
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 opens integer NOT NULL DEFAULT 0 CHECK (opens >= 0),
 last_opened_at timestamptz,
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,created_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY (org_id,project_id,report_id) REFERENCES reports(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id) ON DELETE CASCADE,
 -- A firm's link lives and dies with its invited cell; a corrected name keeps it.
 FOREIGN KEY (org_id,package_id,firm_id) REFERENCES package_invitations(org_id,package_id,firm_id)
  ON DELETE CASCADE ON UPDATE CASCADE,
 CHECK (CASE kind
  WHEN 'documents' THEN cardinality(document_ids)>0 AND report_id IS NULL AND package_id IS NULL AND firm_id IS NULL
  WHEN 'report'    THEN report_id IS NOT NULL AND cardinality(document_ids)=0 AND package_id IS NULL AND firm_id IS NULL
  WHEN 'package'   THEN package_id IS NOT NULL AND firm_id IS NOT NULL AND report_id IS NULL AND cardinality(document_ids)=0 END)
);
CREATE INDEX guest_grants_project ON guest_grants(org_id,project_id,created_at DESC);

-- A guest's quote is recorded with the grant as actor; no member can undo it.
ALTER TABLE project_changes DROP CONSTRAINT project_changes_via_check;
ALTER TABLE project_changes ADD CONSTRAINT project_changes_via_check CHECK (via IN ('chat','screen','guest'));
