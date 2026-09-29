ALTER TABLE invites ADD COLUMN role text NOT NULL DEFAULT 'member';

-- A magic link is presented without an org id, so the hash cannot be ambiguous.
CREATE UNIQUE INDEX invites_token_hash_uq ON invites (token_hash);
