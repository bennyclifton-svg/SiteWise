ALTER TABLE schedule_items DROP CONSTRAINT schedule_items_comparison_kind_check;
ALTER TABLE schedule_items ADD CONSTRAINT schedule_items_comparison_kind_check CHECK (comparison_kind IN ('stage','option','rate'));
ALTER TABLE schedule_items ADD COLUMN comparison_unit text NOT NULL DEFAULT '' CHECK (comparison_unit IN ('','hour','meeting','inspection','approval','report','review','revision'));
