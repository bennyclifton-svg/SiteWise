-- Who engages a consultant (owner decision 3, 13 October 2026): the client
-- directly, or the lead design consultant as a subconsultant whose fee is in
-- the lead's price. NULL means not decided; the lead's RFP asks for it.
ALTER TABLE packages ADD COLUMN engaged_by text;
ALTER TABLE packages ADD CONSTRAINT packages_engaged_by_state CHECK (engaged_by IN ('client','lead_consultant'));
ALTER TABLE packages ADD CONSTRAINT packages_engaged_by_services CHECK (engaged_by IS NULL OR kind='services');
