CREATE TABLE proposals (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 site_id uuid NOT NULL,
 key text NOT NULL CHECK (length(key) BETWEEN 1 AND 1000),
 record_kind text NOT NULL CHECK (record_kind IN ('ic','cq','uc')),
 record_id text NOT NULL CHECK (length(record_id) BETWEEN 1 AND 200 AND position('|' IN record_id)=0),
 interface_id text CHECK (length(interface_id) BETWEEN 1 AND 200 AND position('|' IN interface_id)=0),
 proposal_index integer NOT NULL CHECK (proposal_index>=0),
 target_system_id text CHECK (length(target_system_id) BETWEEN 1 AND 200 AND position('|' IN target_system_id)=0),
 target_part_id uuid,
 kind text NOT NULL CHECK (kind IN ('investigation','work_item','discipline','approval','hold_point','obligation')),
 label text NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 2000),
 action text CHECK (action IN ('new','replace','upgrade','alter','repair','remove','retain','investigate')),
 reason jsonb NOT NULL CHECK (jsonb_typeof(reason)='object'),
 severity text NOT NULL DEFAULT '' CHECK (severity IN ('','life-safety','compliance','durability','cost','programme')),
 specificity integer NOT NULL CHECK (specificity BETWEEN 0 AND 3),
 rank integer NOT NULL CHECK (rank>0),
 critical boolean NOT NULL DEFAULT false,
 draft boolean NOT NULL,
 unaccepted_triggers boolean NOT NULL,
 inputs_fingerprint text NOT NULL CHECK (inputs_fingerprint ~ '^[0-9a-f]{64}$'),
 knowledge_version text NOT NULL CHECK (length(knowledge_version)>0),
 state text NOT NULL CHECK (state IN ('open','dismissed','accepted','reopened','addressed_by_evidence')),
 inputs_changed boolean NOT NULL DEFAULT false,
 PRIMARY KEY (org_id,project_id,key),
 FOREIGN KEY (org_id,project_id,site_id) REFERENCES projects(org_id,id,site_id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,site_id,target_part_id) REFERENCES project_parts(org_id,site_id,id),
 CHECK (record_id LIKE record_kind||'.%'),
 CHECK (key=record_id||'|'||COALESCE(interface_id,'')||'|'||COALESCE(target_system_id,'')||'|'||COALESCE(target_part_id::text,'')||'|'||proposal_index::text)
);

-- A relation, rather than an unchecked UUID array, enforces each trigger's
-- tenant and project. The API can aggregate it as trigger_work_item_ids.
CREATE TABLE proposal_triggers (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 proposal_key text NOT NULL,
 work_item_id uuid NOT NULL,
 PRIMARY KEY (org_id,project_id,proposal_key,work_item_id),
 FOREIGN KEY (org_id,project_id,proposal_key) REFERENCES proposals(org_id,project_id,key) ON DELETE CASCADE,
 FOREIGN KEY (org_id,project_id,work_item_id) REFERENCES work_items(org_id,project_id,id)
);

-- Decisions deliberately do not reference the replaceable projection: a
-- removed knowledge record must not erase the person's decision history.
CREATE TABLE proposal_decisions (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 proposal_key text NOT NULL CHECK (length(proposal_key) BETWEEN 1 AND 1000),
 record_id text NOT NULL CHECK (length(record_id) BETWEEN 1 AND 200),
 trigger_work_item_id uuid,
 decision text NOT NULL CHECK (decision IN ('accepted','dismissed')),
 inputs_fingerprint text NOT NULL CHECK (inputs_fingerprint ~ '^[0-9a-f]{64}$'),
 rationale text NOT NULL DEFAULT '' CHECK (length(rationale)<=200),
 actor uuid NOT NULL,
 decided_at timestamptz NOT NULL DEFAULT now(),
 created_record_type text CHECK (created_record_type='work_item'),
 created_record_id uuid,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,project_id,proposal_key),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY (org_id,actor) REFERENCES users(org_id,id),
 FOREIGN KEY (org_id,project_id,trigger_work_item_id) REFERENCES work_items(org_id,project_id,id),
 FOREIGN KEY (org_id,project_id,created_record_id) REFERENCES work_items(org_id,project_id,id),
 CHECK (split_part(proposal_key,'|',1)=record_id),
 CHECK ((decision='dismissed' AND created_record_type IS NULL AND created_record_id IS NULL)
     OR (decision='accepted' AND created_record_type IS NOT NULL AND created_record_type='work_item' AND created_record_id IS NOT NULL))
);
