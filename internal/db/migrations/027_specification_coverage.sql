CREATE TABLE specification_coverage (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 id uuid NOT NULL,
 system_id text NOT NULL,
 type_id uuid,
 content jsonb NOT NULL CHECK(jsonb_typeof(content)='object'),
 version bigint NOT NULL DEFAULT 1 CHECK(version>0),
 updated_by uuid NOT NULL,
 PRIMARY KEY(org_id,id),
 FOREIGN KEY(org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE,
 FOREIGN KEY(org_id,project_id,type_id) REFERENCES specification_types(org_id,project_id,id),
 FOREIGN KEY(org_id,updated_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE UNIQUE INDEX specification_coverage_scope_uq ON specification_coverage(org_id,project_id,system_id,COALESCE(type_id,'00000000-0000-0000-0000-000000000000'::uuid));
