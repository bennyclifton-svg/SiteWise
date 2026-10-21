ALTER TABLE schedule_items ADD COLUMN comparison_label text CHECK (length(trim(comparison_label)) BETWEEN 1 AND 500);
ALTER TABLE schedule_items ADD COLUMN comparison_kind text CHECK (comparison_kind IN ('stage','option'));
ALTER TABLE schedule_items ADD COLUMN comparison_custom boolean NOT NULL DEFAULT false;

-- Preserve the preceding mapping for undo, and prevent a background reread
-- from resurrecting source fees already combined by the user.
CREATE TABLE comparison_alignment (
 org_id uuid NOT NULL,
 submission_id uuid NOT NULL,
 prices jsonb NOT NULL,
 decisions jsonb NOT NULL,
 version bigint NOT NULL,
 PRIMARY KEY (org_id, submission_id),
 FOREIGN KEY (org_id, submission_id) REFERENCES submissions(org_id,id) ON DELETE CASCADE
);
