-- A replacement belongs to the same planning value, not merely the same
-- organisation. Null successors remain valid for withdrawals/scope changes.
-- V1 has one project per site, so the site+scope pair also fixes project
-- ownership. A future shared-site migration must preserve this guarantee.
ALTER TABLE profile_planning_values
 ADD CONSTRAINT profile_planning_values_lineage_identity
 UNIQUE (org_id,site_id,part_id,key,scope,id),
 DROP CONSTRAINT profile_planning_values_org_id_superseded_by_fkey,
 ADD CONSTRAINT profile_planning_values_successor_fkey
 FOREIGN KEY (org_id,site_id,part_id,key,scope,superseded_by)
 REFERENCES profile_planning_values(org_id,site_id,part_id,key,scope,id)
 DEFERRABLE INITIALLY DEFERRED;
