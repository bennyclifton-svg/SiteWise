-- Chat conversations (docs/plans/2026-11-15-chat-agent.md, Task 5).
-- A conversation is one project and one user: another member of the org
-- cannot read it. Every table carries org_id and goes with its project.

CREATE TABLE chat_conversations (
 org_id uuid NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 user_id uuid NOT NULL,
 started_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,project_id) REFERENCES projects(org_id,id) ON DELETE CASCADE
);
CREATE INDEX chat_conversations_recent ON chat_conversations(org_id,project_id,user_id,started_at DESC);

-- One typed message and what came of it. answer is what the strip shows
-- again on reload: the checked text and the cards.
CREATE TABLE chat_turns (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 conversation_id uuid NOT NULL,
 n bigint GENERATED ALWAYS AS IDENTITY,
 text text NOT NULL,
 status text NOT NULL DEFAULT 'running' CHECK (status IN ('running','done','stopped','failed')),
 outside boolean NOT NULL DEFAULT false,
 model text NOT NULL DEFAULT '',
 rounds int NOT NULL DEFAULT 0,
 in_tokens bigint NOT NULL DEFAULT 0,
 out_tokens bigint NOT NULL DEFAULT 0,
 cache_read_tokens bigint NOT NULL DEFAULT 0,
 cache_write_tokens bigint NOT NULL DEFAULT 0,
 first_ms int NOT NULL DEFAULT 0,
 latency_ms int NOT NULL DEFAULT 0,
 answer jsonb NOT NULL DEFAULT '{}',
 at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,conversation_id) REFERENCES chat_conversations(org_id,id) ON DELETE CASCADE
);
CREATE INDEX chat_turns_order ON chat_turns(org_id,conversation_id,n);

-- The provider's messages of a turn, each kept whole as JSON.
CREATE TABLE chat_blocks (
 org_id uuid NOT NULL,
 turn_id uuid NOT NULL,
 ordinal int NOT NULL,
 block jsonb NOT NULL,
 PRIMARY KEY (org_id,turn_id,ordinal),
 FOREIGN KEY (org_id,turn_id) REFERENCES chat_turns(org_id,id) ON DELETE CASCADE
);

-- A change that waits for the PM's click.
CREATE TABLE chat_prepared (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 turn_id uuid NOT NULL,
 area text NOT NULL,
 ops jsonb NOT NULL CHECK (jsonb_typeof(ops)='array'),
 summary text NOT NULL DEFAULT '',
 why text NOT NULL DEFAULT '',
 state text NOT NULL DEFAULT 'waiting' CHECK (state IN ('waiting','applied','dismissed')),
 result jsonb,
 decided_at timestamptz,
 PRIMARY KEY (org_id,id),
 FOREIGN KEY (org_id,turn_id) REFERENCES chat_turns(org_id,id) ON DELETE CASCADE
);
