-- Program (plan docs/plans/2026-10-18-program.md section 5). These tables
-- store inputs only: durations, links, constraints and the calendar. Dates,
-- float and the critical path are computed on every read.
CREATE TABLE programs (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 start_date date,
 start_origin text CHECK (start_origin IN ('user','document','profile')),
 skeleton_id text,
 library_version text NOT NULL,
 stage_revision bigint NOT NULL DEFAULT 1 CHECK (stage_revision>0),
 working_week smallint NOT NULL DEFAULT 31 CHECK (working_week BETWEEN 1 AND 127),
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_by_surface text NOT NULL CHECK (created_by_surface IN ('program','procurement','cost_plan')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);

CREATE TABLE program_activities (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 program_id uuid NOT NULL,
 parent_id uuid,
 ordinal integer NOT NULL CHECK (ordinal>=0),
 kind text NOT NULL CHECK (kind IN ('stage','activity','milestone','hold_point')),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 200),
 catalogue_stage_id text,
 package_id uuid, package_stage_id uuid, work_item_id uuid,
 fragment_id text, interface_id text,
 systems text[] NOT NULL DEFAULT '{}',
 duration_days integer CHECK (duration_days BETWEEN 0 AND 3650),
 duration_low integer, duration_high integer,
 duration_origin text CHECK (duration_origin IN ('user','document','profile','library','calculation')),
 duration_source jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(duration_source)='object'),
 constraint_kind text NOT NULL DEFAULT 'asap' CHECK (constraint_kind IN ('asap','snet','mso')),
 constraint_date date,
 review_status text NOT NULL CHECK (review_status IN ('proposed','accepted')),
 origin text NOT NULL CHECK (origin IN ('library','user','document')),
 source_key text,
 dismissed_reason text,
 retired_at timestamptz,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,parent_id) REFERENCES program_activities(org_id,id),
 CHECK ((kind='stage') = (parent_id IS NULL)),
 CHECK (kind='stage' OR catalogue_stage_id IS NULL),
 CHECK (kind<>'stage' OR catalogue_stage_id IS NOT NULL),
 CHECK ((duration_low IS NULL) = (duration_high IS NULL)),
 CHECK (duration_low IS NULL OR (duration_low <= duration_high)),
 CHECK ((constraint_kind='asap') = (constraint_date IS NULL)),
 CHECK (kind<>'milestone' OR duration_days=0),
 CHECK (kind='hold_point' OR dismissed_reason IS NULL)
);
CREATE UNIQUE INDEX program_activities_source_uq ON program_activities(org_id,program_id,source_key)
 WHERE source_key IS NOT NULL AND retired_at IS NULL;
-- The store loads one program's rows in one query; foreign keys from
-- children and links are checked by parent.
CREATE INDEX program_activities_program_idx ON program_activities(org_id,program_id);
CREATE INDEX program_activities_parent_idx ON program_activities(org_id,parent_id) WHERE parent_id IS NOT NULL;

CREATE TABLE program_links (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 program_id uuid NOT NULL,
 pred_id uuid NOT NULL,
 succ_id uuid NOT NULL,
 type text NOT NULL CHECK (type IN ('FS','SS','FF','SF')),
 lag_days integer NOT NULL DEFAULT 0 CHECK (lag_days BETWEEN -365 AND 365),
 lag_unit text NOT NULL DEFAULT 'working' CHECK (lag_unit IN ('working','calendar')),
 origin text NOT NULL CHECK (origin IN ('chain','library','user','document','interface')),
 interface_id text,
 source_key text,
 retired_at timestamptz,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,pred_id) REFERENCES program_activities(org_id,id),
 FOREIGN KEY (org_id,succ_id) REFERENCES program_activities(org_id,id),
 CHECK (pred_id<>succ_id),
 CHECK (origin<>'chain' OR (type='FS' AND lag_days=0))
);
CREATE UNIQUE INDEX program_links_pair_uq ON program_links(org_id,pred_id,succ_id) WHERE retired_at IS NULL;
CREATE INDEX program_links_program_idx ON program_links(org_id,program_id);
CREATE INDEX program_links_succ_idx ON program_links(org_id,succ_id);

CREATE TABLE program_calendar_days (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 program_id uuid NOT NULL,
 day date NOT NULL,
 kind text NOT NULL CHECK (kind IN ('public_holiday','rdo','shutdown','working_override')),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 200),
 origin text NOT NULL CHECK (origin IN ('library','user')),
 PRIMARY KEY (org_id,program_id,day),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE
);

CREATE TABLE program_ops (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 program_id uuid NOT NULL,
 seq bigint NOT NULL,
 actor uuid NOT NULL,
 ops jsonb NOT NULL,
 inverse jsonb NOT NULL,
 undone_at timestamptz,
 at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,program_id,seq),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE
);
-- Undo and redo read one user's latest batches.
CREATE INDEX program_ops_actor_idx ON program_ops(org_id,program_id,actor,seq);

CREATE TABLE program_changes (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 program_id uuid NOT NULL,
 source text NOT NULL CHECK (source IN ('profile','work_items','procurement','library')),
 kind text NOT NULL CHECK (kind IN ('add','remove','rename','duration','link')),
 target_activity_id uuid,
 source_key text NOT NULL,
 proposal jsonb NOT NULL,
 summary text NOT NULL CHECK (length(trim(summary)) BETWEEN 1 AND 300),
 default_choice text NOT NULL CHECK (default_choice IN ('adopt','keep')),
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','adopted','kept','superseded')),
 decided_by uuid, decided_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX program_changes_pending_uq ON program_changes(org_id,program_id,source_key) WHERE status='pending';

CREATE TABLE program_readings (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 program_id uuid NOT NULL,
 source text NOT NULL CHECK (source IN ('program_file','proposal_r7','hold_point_evidence')),
 document_id uuid, passage_id uuid,
 target_activity_id uuid,
 catalogue_stage_id text,
 field text NOT NULL,
 value text,
 parsed jsonb NOT NULL DEFAULT '{}',
 band text NOT NULL CHECK (band IN ('green','amber','blank','grey')),
 decided_by text NOT NULL CHECK (decided_by IN ('rule','jev','user')),
 confidence double precision,
 question_version text,
 status text NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed','accepted','dismissed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE
);

-- Link existing package stages to program stages, and pin each to a stage revision.
ALTER TABLE package_stages ADD COLUMN program_stage_id uuid;
ALTER TABLE package_stages ADD COLUMN stage_revision bigint;
ALTER TABLE package_stages ADD CONSTRAINT package_stages_program_stage_fk
 FOREIGN KEY (org_id,program_stage_id) REFERENCES program_activities(org_id,id);

-- The RFP draft record is the draft report version (internal/store/reports.go):
-- it records the program stage revision its stage rows were built from (7.6).
ALTER TABLE report_versions ADD COLUMN stage_revision bigint;
