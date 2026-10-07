-- Rebuilds still compute every row; unchanged saved values need no rewrite.
-- Existing rows have no hash and are rewritten on their first rebuild.
ALTER TABLE profile_rows ADD COLUMN projection_hash text NOT NULL DEFAULT '';
ALTER TABLE profile_rows ADD CONSTRAINT profile_projection_hash_shape
 CHECK (projection_hash = '' OR projection_hash ~ '^[a-f0-9]{64}$');
