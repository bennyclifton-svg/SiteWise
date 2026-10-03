-- The user chooses which documents the project profile reads. Automatic
-- follows the document kind (data/profile/reading.json); read and skip are
-- the user's word. Text is still split for every document: the setting
-- controls Jev reading only.
ALTER TABLE documents ADD COLUMN profile_read text NOT NULL DEFAULT 'auto'
    CHECK (profile_read IN ('auto', 'read', 'skip'));
