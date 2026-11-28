-- The project report in the owner's format
-- (docs/plans/2026-11-26-project-report-golden-format.md, section 7).

-- The period a draft reports on, proposed at its first Update; and the
-- draft as Go generated it, before the PM's rows are laid over it, so a
-- row write lays them again without running the loader.
ALTER TABLE report_versions
 ADD COLUMN period_from date,
 ADD COLUMN period_to date,
 ADD COLUMN period_label text CHECK (length(period_label) <= 80),
 ADD COLUMN generated jsonb;

-- What the PM owns in the project report's tables. source_key null: the
-- PM's own row. source_key set: laid over the generated row of that key.
CREATE TABLE report_rows (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 report_id uuid NOT NULL,
 table_id text NOT NULL CHECK (length(table_id) BETWEEN 1 AND 100),
 source_key text CHECK (length(source_key) BETWEEN 1 AND 200),
 position double precision,
 cells jsonb NOT NULL DEFAULT '{}',
 hidden boolean NOT NULL DEFAULT false,
 sources jsonb NOT NULL DEFAULT '[]',
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, report_id, table_id, source_key),
 FOREIGN KEY (org_id, project_id, report_id) REFERENCES reports(org_id, project_id, id) ON DELETE CASCADE
);
CREATE INDEX report_rows_report ON report_rows (org_id, report_id);

-- The PM's row changes have one Undo, in the reports area.
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON report_rows
 FOR EACH ROW EXECUTE FUNCTION capture_change('version', 'updated_at');

-- Images this report may serve (slice 5).
CREATE TABLE report_images (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 report_id uuid NOT NULL,
 sha256 bytea NOT NULL,
 media_type text NOT NULL CHECK (media_type IN ('image/png','image/jpeg')),
 byte_size bigint NOT NULL CHECK (byte_size > 0),
 created_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, report_id, sha256),
 FOREIGN KEY (org_id, project_id, report_id) REFERENCES reports(org_id, project_id, id) ON DELETE CASCADE
);
