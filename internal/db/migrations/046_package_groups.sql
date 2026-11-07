-- Combined engagements, slice 1 (docs/plans/2026-11-07-combined-engagements-slice1.md).
-- A group is a services package with no discipline that holds several
-- consultant packages under one appointment. Members keep their own rows,
-- disciplines and RFP choices, so uncombining loses nothing.
ALTER TABLE packages ADD COLUMN role text NOT NULL DEFAULT 'single' CHECK (role IN ('single','group'));
ALTER TABLE packages ADD COLUMN group_id uuid;
ALTER TABLE packages ADD CONSTRAINT packages_group_fk FOREIGN KEY (org_id, project_id, group_id) REFERENCES packages(org_id, project_id, id);
-- A group is a consultant appointment with no discipline of its own and no
-- group above it; a member is a consultant discipline row. One group_id
-- column means a discipline row belongs to at most one group.
ALTER TABLE packages ADD CONSTRAINT packages_group_shape CHECK (role <> 'group' OR (kind = 'services' AND discipline_id IS NULL AND group_id IS NULL));
ALTER TABLE packages ADD CONSTRAINT packages_member_shape CHECK (group_id IS NULL OR (kind = 'services' AND role = 'single' AND group_id <> id));
CREATE INDEX packages_group_idx ON packages (org_id, project_id, group_id) WHERE group_id IS NOT NULL;

-- A member may only point at a group row; a CHECK cannot see the other row.
-- A missing target is left to the foreign key, which names itself.
CREATE FUNCTION package_group_target() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.group_id IS NOT NULL AND EXISTS(SELECT 1 FROM packages
   WHERE org_id=NEW.org_id AND project_id=NEW.project_id AND id=NEW.group_id AND role<>'group') THEN
  RAISE EXCEPTION 'package group must be a group row' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER package_group_target BEFORE INSERT OR UPDATE OF group_id ON packages FOR EACH ROW EXECUTE FUNCTION package_group_target();

-- A proposal moved from a member to its group remembers the member, so
-- uncombining returns it there.
ALTER TABLE submissions ADD COLUMN group_member_id uuid;
ALTER TABLE submissions ADD CONSTRAINT submissions_group_member_fk FOREIGN KEY (org_id, project_id, group_member_id) REFERENCES packages(org_id, project_id, id);
ALTER TABLE submissions ADD CONSTRAINT submissions_group_member_package CHECK (group_member_id IS NULL OR package_id IS NOT NULL);
