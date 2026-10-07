CREATE TABLE cost_plan_versions (
 org_id uuid NOT NULL, project_id uuid NOT NULL, id uuid NOT NULL,
 revision integer NOT NULL CHECK(revision>0), status text NOT NULL CHECK(status IN ('draft','baseline','superseded')),
 currency char(3) NOT NULL DEFAULT 'AUD' CHECK(currency ~ '^[A-Z]{3}$'), tax_basis text NOT NULL CHECK(tax_basis IN ('ex_tax','inc_tax')),
 tax_rate numeric(5,4) CHECK(tax_rate BETWEEN 0 AND 1),price_date date,coverage text NOT NULL DEFAULT '',funding_target numeric(18,2),
 funding_target_provenance jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(funding_target_provenance)='object'),
 frozen_at timestamptz,version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 PRIMARY KEY(org_id,id),UNIQUE(org_id,project_id,id),UNIQUE(org_id,project_id,revision),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id),
 CHECK((status='draft')=(frozen_at IS NULL))
);
CREATE UNIQUE INDEX cost_plan_draft_uq ON cost_plan_versions(org_id,project_id) WHERE status='draft';
CREATE TABLE cost_plans (
 org_id uuid NOT NULL,project_id uuid NOT NULL,draft_version_id uuid NOT NULL,baseline_version_id uuid,
 PRIMARY KEY(org_id,project_id),FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id),
 FOREIGN KEY(org_id,project_id,draft_version_id) REFERENCES cost_plan_versions(org_id,project_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(org_id,project_id,baseline_version_id) REFERENCES cost_plan_versions(org_id,project_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE cost_items (
 org_id uuid NOT NULL,project_id uuid NOT NULL,id uuid NOT NULL,created_in_version_id uuid NOT NULL,
 PRIMARY KEY(org_id,id),UNIQUE(org_id,project_id,id),
 FOREIGN KEY(org_id,project_id,created_in_version_id) REFERENCES cost_plan_versions(org_id,project_id,id)
);
CREATE TABLE cost_item_revisions (
 org_id uuid NOT NULL,project_id uuid NOT NULL,plan_version_id uuid NOT NULL,cost_item_id uuid NOT NULL,parent_item_id uuid,
 code text NOT NULL DEFAULT '',label text NOT NULL CHECK(length(trim(label)) BETWEEN 1 AND 500),
 line_kind text NOT NULL CHECK(line_kind IN ('works','fee','project_wide')), category text,work_item_id uuid,package_id uuid,package_stage_id uuid,
 posting boolean NOT NULL DEFAULT true,quantity numeric,unit text,rate numeric(18,4),rate_basis text,excluded boolean NOT NULL DEFAULT false,
 origin text NOT NULL CHECK(origin IN ('document','user','calculation','assumption')),review_status text NOT NULL DEFAULT 'accepted_for_planning',
 meaning text NOT NULL CHECK(meaning IN ('stated','requirement','allowance','forecast')),rationale text NOT NULL DEFAULT '',provenance jsonb NOT NULL DEFAULT '{}',
 system_id text,part_id uuid,version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 PRIMARY KEY(org_id,plan_version_id,cost_item_id),
 FOREIGN KEY(org_id,project_id,plan_version_id) REFERENCES cost_plan_versions(org_id,project_id,id),
 FOREIGN KEY(org_id,project_id,cost_item_id) REFERENCES cost_items(org_id,project_id,id),
 FOREIGN KEY(org_id,plan_version_id,parent_item_id) REFERENCES cost_item_revisions(org_id,plan_version_id,cost_item_id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(org_id,project_id,work_item_id) REFERENCES work_items(org_id,project_id,id),
 FOREIGN KEY(org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id),
 FOREIGN KEY(org_id,project_id,package_id,package_stage_id) REFERENCES package_stages(org_id,project_id,package_id,id),
 CHECK(parent_item_id IS DISTINCT FROM cost_item_id),
 CHECK((quantity IS NULL)=(rate IS NULL)),
 CHECK((line_kind='works' AND work_item_id IS NOT NULL AND package_stage_id IS NULL AND category IS NULL)
 OR (line_kind='fee' AND work_item_id IS NULL AND package_id IS NOT NULL AND package_stage_id IS NOT NULL AND category IS NULL)
 OR (line_kind='project_wide' AND work_item_id IS NULL AND package_id IS NULL AND package_stage_id IS NULL AND category IN ('contingency','escalation','authority_fees','exclusions','other')))
);
CREATE TABLE cost_values (
 org_id uuid NOT NULL,project_id uuid NOT NULL,plan_version_id uuid NOT NULL,cost_item_id uuid NOT NULL,
 metric text NOT NULL CHECK(metric IN ('budget','estimate','commitment','claimed_to_date')),value_state text NOT NULL CHECK(value_state IN ('known','unknown')),
 amount numeric(18,2),low numeric(18,2),high numeric(18,2),as_of date,
 origin text NOT NULL CHECK(origin IN ('document','user','calculation','assumption')),review_status text NOT NULL DEFAULT 'accepted_for_planning',
 meaning text NOT NULL CHECK(meaning IN ('stated','requirement','allowance','forecast')),rationale text NOT NULL DEFAULT '',provenance jsonb NOT NULL DEFAULT '{}',version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 PRIMARY KEY(org_id,plan_version_id,cost_item_id,metric),
 FOREIGN KEY(org_id,project_id,plan_version_id) REFERENCES cost_plan_versions(org_id,project_id,id),
 FOREIGN KEY(org_id,plan_version_id,cost_item_id) REFERENCES cost_item_revisions(org_id,plan_version_id,cost_item_id),
 CHECK(metric<>'claimed_to_date' OR as_of IS NOT NULL),CHECK((low IS NULL)=(high IS NULL)),CHECK(low<=high),CHECK(amount IS NULL OR low IS NULL OR amount BETWEEN low AND high),
 CHECK((value_state='unknown' AND amount IS NULL AND low IS NULL AND high IS NULL) OR (value_state='known' AND (amount IS NOT NULL OR low IS NOT NULL)))
);
CREATE TABLE scope_cost_links (
 org_id uuid NOT NULL,project_id uuid NOT NULL,plan_version_id uuid NOT NULL,package_scope_item_id uuid NOT NULL,cost_item_id uuid NOT NULL,
 PRIMARY KEY(org_id,project_id,plan_version_id,package_scope_item_id,cost_item_id),
 FOREIGN KEY(org_id,project_id,plan_version_id) REFERENCES cost_plan_versions(org_id,project_id,id),
 FOREIGN KEY(org_id,project_id,package_scope_item_id) REFERENCES package_scope_items(org_id,project_id,id),
 FOREIGN KEY(org_id,plan_version_id,cost_item_id) REFERENCES cost_item_revisions(org_id,plan_version_id,cost_item_id)
);
CREATE FUNCTION protect_cost_version() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE target uuid; tenant uuid; frozen boolean;
BEGIN
 IF TG_TABLE_NAME='cost_plan_versions' THEN
  IF OLD.frozen_at IS NOT NULL THEN RAISE EXCEPTION 'frozen cost plan' USING ERRCODE='23514'; END IF;
 ELSE
  IF TG_OP='DELETE' THEN target:=OLD.plan_version_id;tenant:=OLD.org_id;ELSE target:=NEW.plan_version_id;tenant:=NEW.org_id;END IF;
  SELECT frozen_at IS NOT NULL INTO frozen FROM cost_plan_versions WHERE org_id=tenant AND id=target;
  IF frozen THEN RAISE EXCEPTION 'frozen cost plan' USING ERRCODE='23514';END IF;
  IF TG_OP='UPDATE' AND (OLD.org_id<>NEW.org_id OR OLD.plan_version_id<>NEW.plan_version_id) THEN RAISE EXCEPTION 'cost ownership is immutable' USING ERRCODE='23514';END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;
END $$;
CREATE TRIGGER cost_version_frozen BEFORE UPDATE OR DELETE ON cost_plan_versions FOR EACH ROW EXECUTE FUNCTION protect_cost_version();
CREATE TRIGGER cost_item_frozen BEFORE INSERT OR UPDATE OR DELETE ON cost_item_revisions FOR EACH ROW EXECUTE FUNCTION protect_cost_version();
CREATE TRIGGER cost_value_frozen BEFORE INSERT OR UPDATE OR DELETE ON cost_values FOR EACH ROW EXECUTE FUNCTION protect_cost_version();
CREATE TRIGGER cost_links_frozen BEFORE INSERT OR UPDATE OR DELETE ON scope_cost_links FOR EACH ROW EXECUTE FUNCTION protect_cost_version();
CREATE FUNCTION cost_posting_value() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM cost_item_revisions WHERE org_id=NEW.org_id AND plan_version_id=NEW.plan_version_id AND cost_item_id=NEW.cost_item_id AND posting) THEN RAISE EXCEPTION 'non-posting cost item' USING ERRCODE='23514';END IF;RETURN NEW;
END $$;
CREATE TRIGGER cost_value_posting BEFORE INSERT OR UPDATE ON cost_values FOR EACH ROW EXECUTE FUNCTION cost_posting_value();
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM cost_plans) OR EXISTS(SELECT 1 FROM cost_plan_versions) THEN RAISE EXCEPTION 'cost migration expected empty additive tables';END IF;
END $$;
