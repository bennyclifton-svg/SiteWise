-- Posting is a property of an existing revision. Let the composite foreign
-- key reject a missing reference, rather than hiding it behind a posting error.
CREATE OR REPLACE FUNCTION cost_posting_value() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM cost_item_revisions
   WHERE org_id=NEW.org_id AND plan_version_id=NEW.plan_version_id
     AND cost_item_id=NEW.cost_item_id AND NOT posting) THEN
  RAISE EXCEPTION 'non-posting cost item' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
