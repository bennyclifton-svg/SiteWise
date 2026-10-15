-- Project-owned specification definitions can exist before a work item or
-- consultant appointment. Display codes are deliberately not unique.
CREATE TABLE specification_types (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL DEFAULT gen_random_uuid(),
 content jsonb NOT NULL CHECK (jsonb_typeof(content)='object'),
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_by uuid NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(org_id,id),
 UNIQUE(org_id,project_id,id),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,created_by) REFERENCES users(org_id,id)
);

CREATE TABLE specification_applications (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL DEFAULT gen_random_uuid(),
 type_id uuid NOT NULL,
 part_id uuid,
 work_item_id uuid,
 content jsonb NOT NULL CHECK (jsonb_typeof(content)='object'),
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY(org_id,id),
 UNIQUE(org_id,project_id,type_id,id),
 FOREIGN KEY(org_id,project_id,type_id) REFERENCES specification_types(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,part_id) REFERENCES project_parts(org_id,id),
 FOREIGN KEY(org_id,project_id,work_item_id) REFERENCES work_items(org_id,project_id,id)
);

CREATE TABLE specification_assertions (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL DEFAULT gen_random_uuid(),
 type_id uuid NOT NULL,
 application_id uuid,
 attribute_key text NOT NULL,
 meaning text NOT NULL CHECK (meaning IN ('requirement','selection','capability','allowance','exclusion','observed')),
 value jsonb NOT NULL CHECK (jsonb_typeof(value)='object'),
 source jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(source)='object'),
 origin text NOT NULL CHECK (origin IN ('user','document')),
 status text NOT NULL CHECK (status IN ('proposed','accepted','rejected','superseded')),
 actor uuid NOT NULL,
 reason text NOT NULL DEFAULT '',
 import_key text,
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(org_id,id),
 FOREIGN KEY(org_id,project_id,type_id) REFERENCES specification_types(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,project_id,type_id,application_id) REFERENCES specification_applications(org_id,project_id,type_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,actor) REFERENCES users(org_id,id)
);
CREATE UNIQUE INDEX specification_assertion_import_uq ON specification_assertions(org_id,project_id,import_key) WHERE import_key IS NOT NULL;
CREATE UNIQUE INDEX specification_assertion_accepted_uq ON specification_assertions(org_id,project_id,type_id,COALESCE(application_id,'00000000-0000-0000-0000-000000000000'::uuid),attribute_key,meaning) WHERE status='accepted';

CREATE TABLE specification_relationships (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL DEFAULT gen_random_uuid(),
 from_id uuid NOT NULL,
 to_id uuid NOT NULL,
 kind text NOT NULL CHECK (kind IN ('component_of','finished_with','requires','matches','replaces')),
 content jsonb NOT NULL CHECK (jsonb_typeof(content)='object'),
 version bigint NOT NULL DEFAULT 1 CHECK (version>0),
 PRIMARY KEY(org_id,id),
 FOREIGN KEY(org_id,project_id,from_id) REFERENCES specification_types(org_id,project_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,project_id,to_id) REFERENCES specification_types(org_id,project_id,id) ON DELETE CASCADE,
 UNIQUE(org_id,project_id,from_id,to_id,kind),
 CHECK(from_id<>to_id)
);

CREATE TABLE specification_history (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id bigint GENERATED ALWAYS AS IDENTITY,
 entity_id uuid NOT NULL,
 entity_kind text NOT NULL,
 actor uuid NOT NULL,
 content jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(org_id,id),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,actor) REFERENCES users(org_id,id)
);

CREATE TABLE specification_issues (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL DEFAULT gen_random_uuid(),
 number integer NOT NULL CHECK(number>0),
 title text NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 200),
 purpose text NOT NULL CHECK(purpose IN ('preliminary','coordination','pricing')),
 snapshot jsonb NOT NULL CHECK(jsonb_typeof(snapshot)='object'),
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(org_id,id),
 UNIQUE(org_id,project_id,number),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,created_by) REFERENCES users(org_id,id)
);
ALTER TABLE project_revisions ADD COLUMN specifications bigint NOT NULL DEFAULT 0 CHECK(specifications>=0);
