CREATE TABLE packages (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 kind text NOT NULL CHECK (kind IN ('services','works','supply')),
 works_scope text CHECK (works_scope IN ('head_contract','trade')),
 discipline_id text CHECK (length(trim(discipline_id)) BETWEEN 1 AND 200),
 title text NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 200),
 novation boolean NOT NULL DEFAULT false,
 lifecycle_status text NOT NULL CHECK (lifecycle_status IN ('proposed','planned','procuring','appointed','closed','cancelled')),
 origin text NOT NULL CHECK (origin IN ('document','user','calculation','assumption')),
 review_status text NOT NULL CHECK (review_status IN ('proposed','accepted_for_planning','verified','superseded')),
 meaning text NOT NULL DEFAULT 'stated' CHECK (meaning IN ('stated','requirement','allowance','forecast')),
 provenance jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(provenance)='object'),
 verified_by uuid,
 verified_at timestamptz,
 verification_basis text,
 source_proposal_key text,
 retired_at timestamptz,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,verified_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 CHECK ((kind='works' AND works_scope IS NOT NULL) OR (kind<>'works' AND works_scope IS NULL)),
 CHECK (NOT novation OR kind='services'),
 CHECK (review_status<>'verified' OR (verified_by IS NOT NULL AND verified_at IS NOT NULL AND COALESCE(length(trim(verification_basis)),0)>0))
);
CREATE UNIQUE INDEX packages_proposal_uq ON packages(org_id,project_id,source_proposal_key) WHERE source_proposal_key IS NOT NULL;

CREATE TABLE package_stages (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 stage_id text NOT NULL CHECK (length(trim(stage_id)) BETWEEN 1 AND 200),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 200),
 ordinal integer NOT NULL CHECK (ordinal>=0),
 novation_phase text NOT NULL DEFAULT 'none' CHECK (novation_phase IN ('none','pre','post')),
 origin text NOT NULL CHECK (origin IN ('document','user','calculation','assumption')),
 retired_at timestamptz,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,package_id,id),
 FOREIGN KEY (org_id,project_id,package_id) REFERENCES packages(org_id,project_id,id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX package_stages_label_uq ON package_stages(org_id,package_id,label) WHERE retired_at IS NULL;
