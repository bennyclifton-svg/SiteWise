CREATE TABLE reports (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 kind text NOT NULL CHECK (kind IN ('pmp','rfp','rft')),
 package_id uuid,
 title text NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 200),
 current_draft_version_id uuid,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id) DEFERRABLE INITIALLY DEFERRED,
 CHECK (kind='pmp' OR package_id IS NOT NULL)
);

CREATE TABLE report_versions (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 report_id uuid NOT NULL,
 project_id uuid NOT NULL,
 number integer NOT NULL CHECK (number>0),
 status text NOT NULL CHECK (status IN ('draft','issued')),
 reporting_date date NOT NULL,
 previous_issue_id uuid,
 source_revisions jsonb NOT NULL CHECK (jsonb_typeof(source_revisions)='object'),
 template_id text NOT NULL CHECK (length(trim(template_id)) BETWEEN 1 AND 200),
 template_version integer NOT NULL CHECK (template_version>0),
 sections jsonb NOT NULL CHECK (jsonb_typeof(sections)='array'),
 budget_disclosed boolean NOT NULL DEFAULT false,
 snapshot jsonb,
 snapshot_sha256 text CHECK (snapshot_sha256 ~ '^[0-9a-f]{64}$'),
 export_file_sha256 bytea CHECK (octet_length(export_file_sha256)=32),
 issued_at timestamptz,
 issued_by uuid,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,report_id,id),
 UNIQUE (org_id,report_id,number),
 FOREIGN KEY (org_id,project_id,report_id) REFERENCES reports(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,project_id,report_id,previous_issue_id) REFERENCES report_versions(org_id,project_id,report_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY (org_id,issued_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 CHECK (previous_issue_id IS NULL OR previous_issue_id<>id),
 CHECK (status<>'issued' OR (snapshot IS NOT NULL AND snapshot_sha256 IS NOT NULL AND issued_at IS NOT NULL AND issued_by IS NOT NULL))
);
ALTER TABLE reports ADD FOREIGN KEY (org_id,project_id,id,current_draft_version_id) REFERENCES report_versions(org_id,project_id,report_id,id) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE report_edits (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 report_version_id uuid NOT NULL,
 target_id text NOT NULL CHECK (length(target_id) BETWEEN 1 AND 500),
 text text NOT NULL CHECK (length(text)<=20000),
 base_content_sha256 text NOT NULL CHECK (base_content_sha256 ~ '^[0-9a-f]{64}$'),
 user_id uuid NOT NULL,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,report_version_id,target_id),
 FOREIGN KEY (org_id,report_version_id) REFERENCES report_versions(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,user_id) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE report_references (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 report_version_id uuid NOT NULL,
 citation_id text NOT NULL CHECK (citation_id ~ '^[EUCA][1-9][0-9]*$'),
 label text NOT NULL CHECK (label IN ('E','U','C','A')),
 anchor_id text NOT NULL CHECK (length(anchor_id) BETWEEN 1 AND 500),
 basis jsonb NOT NULL CHECK (jsonb_typeof(basis)='object'),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,report_version_id,citation_id),
 FOREIGN KEY (org_id,report_version_id) REFERENCES report_versions(org_id,id) ON DELETE CASCADE,
 CHECK (left(citation_id,1)=label)
);

CREATE FUNCTION protect_issued_report() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.status='issued' THEN
  RAISE EXCEPTION 'issued report versions are immutable' USING ERRCODE='23514';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER report_version_immutable BEFORE UPDATE OR DELETE ON report_versions FOR EACH ROW EXECUTE FUNCTION protect_issued_report();

CREATE FUNCTION protect_issued_report_content() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_status text;
BEGIN
 IF TG_OP<>'INSERT' THEN
  SELECT status INTO parent_status FROM report_versions WHERE org_id=OLD.org_id AND id=OLD.report_version_id FOR UPDATE;
  IF parent_status='issued' THEN
   RAISE EXCEPTION 'issued report content is immutable' USING ERRCODE='23514';
  END IF;
 END IF;
 IF TG_OP<>'DELETE' THEN
  SELECT status INTO parent_status FROM report_versions WHERE org_id=NEW.org_id AND id=NEW.report_version_id FOR UPDATE;
  IF parent_status='issued' THEN
   RAISE EXCEPTION 'issued report content is immutable' USING ERRCODE='23514';
  END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER report_edits_immutable BEFORE INSERT OR UPDATE OR DELETE ON report_edits FOR EACH ROW EXECUTE FUNCTION protect_issued_report_content();
CREATE TRIGGER report_references_immutable BEFORE INSERT OR UPDATE OR DELETE ON report_references FOR EACH ROW EXECUTE FUNCTION protect_issued_report_content();
