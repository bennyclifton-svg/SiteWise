-- Design & Construct does not mean novated (owner, 2026-10-11): the client
-- elects to novate later. Novation becomes no | option | yes. Existing true
-- rows become option, because no client has elected yet.
DO $$ DECLARE constraint_name text; BEGIN
 FOR constraint_name IN SELECT conname FROM pg_constraint
  WHERE conrelid='packages'::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%novation%'
 LOOP
  EXECUTE format('ALTER TABLE packages DROP CONSTRAINT %I',constraint_name);
 END LOOP;
END $$;
ALTER TABLE packages ALTER COLUMN novation DROP DEFAULT;
ALTER TABLE packages ALTER COLUMN novation TYPE text USING CASE WHEN novation THEN 'option' ELSE 'no' END;
ALTER TABLE packages ALTER COLUMN novation SET DEFAULT 'no';
ALTER TABLE packages ADD CONSTRAINT packages_novation_state CHECK (novation IN ('no','option','yes'));
ALTER TABLE packages ADD CONSTRAINT packages_novation_services CHECK (novation='no' OR kind='services');
