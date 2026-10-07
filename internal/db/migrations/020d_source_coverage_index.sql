-- Coverage counts only completed reading outcomes. Keep those values in a
-- compact index instead of fetching each source's text and mapping metadata.
CREATE INDEX passage_sources_coverage_idx
 ON passage_sources(org_id,passage_id) INCLUDE(outcome)
 WHERE outcome<>'pending';
