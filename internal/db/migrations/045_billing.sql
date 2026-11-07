-- Billing, verified email and terms (docs/design/2026-11-07-billing.md).
-- Stripe is the source of truth for a paid plan; these columns are its copy,
-- written only by a signed webhook or the operator's set-plan command.
ALTER TABLE orgs
    ADD COLUMN plan text NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'starter', 'team')),
    ADD COLUMN billing_status text NOT NULL DEFAULT '' CHECK (length(billing_status) <= 40),
    ADD COLUMN billing_interval text NOT NULL DEFAULT '' CHECK (billing_interval IN ('', 'month', 'year')),
    ADD COLUMN paid_until timestamptz,
    ADD COLUMN cancel_at_period_end boolean NOT NULL DEFAULT false,
    ADD COLUMN stripe_customer_id text UNIQUE CHECK (length(stripe_customer_id) <= 255),
    ADD COLUMN stripe_subscription_id text CHECK (length(stripe_subscription_id) <= 255),
    -- Created time of the newest Stripe event applied: an older event that
    -- arrives late never overwrites a newer state.
    ADD COLUMN billing_event_at timestamptz;

-- Only self sign-up starts unverified. Users made any other way (invite
-- era, accepted team invites, operators) were reached at their address.
ALTER TABLE users
    ADD COLUMN email_verified boolean NOT NULL DEFAULT true,
    ADD COLUMN terms_accepted_at timestamptz,
    ADD COLUMN terms_version text NOT NULL DEFAULT '' CHECK (length(terms_version) <= 40);

-- A verification link is presented without an org id, so the hash is unique.
CREATE TABLE email_verifications (
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

-- Documents read per calendar month (Sydney). Deleting a document does not
-- give the count back.
CREATE TABLE org_usage (
    org_id uuid NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    month date NOT NULL CHECK (extract(day FROM month) = 1),
    documents integer NOT NULL DEFAULT 0 CHECK (documents >= 0),
    PRIMARY KEY (org_id, month)
);

-- Stripe may deliver one event more than once; its id is applied once.
CREATE TABLE stripe_events (
    id text PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 255),
    type text NOT NULL CHECK (length(type) <= 100),
    received_at timestamptz NOT NULL DEFAULT now()
);
