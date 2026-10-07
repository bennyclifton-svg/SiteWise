-- Explicit author input: unknown is not an assertion that layout is unchanged.
ALTER TABLE work_items ADD COLUMN layout_change text NOT NULL DEFAULT 'unknown'
 CHECK (layout_change IN ('unknown','yes','no'));
ALTER TABLE work_items ADD CONSTRAINT work_layout_change_scope CHECK (
 layout_change='unknown' OR
 (system_id='interiors.walls-linings' AND action IN ('new','alter','remove'))
);
