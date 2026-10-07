-- Avoid parsing unchanged evidence JSON on every deterministic rebuild. Rank
-- is compared independently so a rank-only change needs no large JSON upsert.
ALTER TABLE proposals ADD COLUMN projection_hash text NOT NULL DEFAULT '';
ALTER TABLE proposals ADD CONSTRAINT proposal_projection_hash_shape
 CHECK (projection_hash = '' OR projection_hash ~ '^[a-f0-9]{64}$');
