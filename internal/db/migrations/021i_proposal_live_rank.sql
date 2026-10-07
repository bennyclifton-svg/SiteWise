-- Rank is a display ordinal of the current proposal set, not authoritative
-- evidence. Computing it on reads avoids rewriting every evidence payload
-- when a newly applicable proposal changes the order. This is the exact
-- severity/specificity/key comparator used by works.RankProposals.
ALTER TABLE proposals DROP COLUMN rank;

CREATE VIEW ranked_proposals AS
SELECT p.*, (row_number() OVER (
 PARTITION BY org_id,project_id
 ORDER BY CASE severity WHEN 'life-safety' THEN 0 WHEN 'compliance' THEN 1
 WHEN 'durability' THEN 2 WHEN 'cost' THEN 3 WHEN 'programme' THEN 4 ELSE 5 END,
 specificity DESC,key COLLATE "C"
))::integer AS rank
FROM proposals p;
