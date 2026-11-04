-- Email and password accounts. One person is one org with one owner.
-- Older invite-only users keep a null hash and are not sign-in names, so the
-- unique index covers only users who can sign in with a password.
ALTER TABLE users ADD COLUMN password_hash bytea;

CREATE UNIQUE INDEX users_login_email_uq ON users (lower(email))
    WHERE password_hash IS NOT NULL;

-- A reset link is presented without an org id, so the hash is unique.
CREATE TABLE password_resets (
    org_id uuid NOT NULL,
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, id),
    UNIQUE (token_hash),
    CHECK (octet_length(token_hash) = 32),
    FOREIGN KEY (org_id, user_id) REFERENCES users (org_id, id) ON DELETE CASCADE
);
