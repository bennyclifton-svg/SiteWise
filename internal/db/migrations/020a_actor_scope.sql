-- Actor attribution is a tenant-owned relationship, including site values
-- without a project. Refuse deleting a referenced user independently; deleting
-- a whole organisation still removes both sides in the same transaction.
-- Existing invalid attribution fails migration rather than being rewritten.
ALTER TABLE profile_user_values
 ADD CONSTRAINT profile_user_values_org_id_user_id_fkey
 FOREIGN KEY (org_id,user_id) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 ADD CONSTRAINT profile_user_values_org_id_verified_by_fkey
 FOREIGN KEY (org_id,verified_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE profile_planning_values
 ADD CONSTRAINT profile_planning_values_org_id_user_id_fkey
 FOREIGN KEY (org_id,user_id) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE work_items
 ADD CONSTRAINT work_items_org_id_verified_by_fkey
 FOREIGN KEY (org_id,verified_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED,
 ADD CONSTRAINT work_items_org_id_retired_by_fkey
 FOREIGN KEY (org_id,retired_by) REFERENCES users(org_id,id) DEFERRABLE INITIALLY DEFERRED;
