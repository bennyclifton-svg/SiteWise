-- The decline measure (docs/plans/2026-11-18-chat-agency.md, Task 7). Go
-- sets declined when a turn's answer carries the fixed words "I can't do
-- that from chat yet": the share of such turns, week by week, is the
-- measure of what PMs ask chat for that it cannot do. The words stay in
-- the turn's own row; only counts are read across orgs (owner decision 3).
ALTER TABLE chat_turns ADD COLUMN declined boolean NOT NULL DEFAULT false;
