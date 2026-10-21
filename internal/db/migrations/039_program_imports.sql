-- Reading A: a program file dropped on Program (plan
-- docs/plans/2026-10-18-program.md section 10.2).
--
-- One row per imported document and program. The file is filed through the
-- normal intake path; this row marks the document for the program_import
-- read that follows filing, holds what code harvested from it (the file's
-- rows, verbatim, with code-parsed dates), which targets were not asked
-- ("Not matched"), which answers are grey and when to ask them again, and
-- the checks code raised (an assumed day/month order). Matched rows become
-- program_readings; nothing here changes the program.
CREATE TABLE program_imports (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 program_id uuid NOT NULL,
 document_id uuid NOT NULL,
 filename text NOT NULL CHECK (length(trim(filename)) BETWEEN 1 AND 500),
 status text NOT NULL DEFAULT 'filing' CHECK (status IN ('filing','reading','read','not_read','dismissed')),
 not_read text,
 file_rows jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(file_rows)='array'),
 not_matched jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(not_matched)='array'),
 grey_targets jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(grey_targets)='array'),
 checks jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(checks)='array'),
 read_attempts integer NOT NULL DEFAULT 0 CHECK (read_attempts>=0),
 retry_after timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 UNIQUE (org_id,program_id,document_id),
 FOREIGN KEY (org_id,program_id) REFERENCES programs(org_id,id) ON DELETE CASCADE
);
CREATE INDEX program_imports_document ON program_imports(org_id,document_id);
CREATE INDEX program_readings_document ON program_readings(org_id,program_id,document_id);
