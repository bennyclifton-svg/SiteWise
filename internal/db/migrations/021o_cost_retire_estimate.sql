-- Estimate is retired: budget is the one planning figure. A draft's known
-- estimate moves into an empty or unknown budget; a known budget wins. Frozen
-- baselines keep their estimate rows (the trigger forbids changing them) and
-- the metric CHECK still allows them.
INSERT INTO cost_values(org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,low,high,as_of,origin,review_status,meaning,rationale,provenance)
SELECT e.org_id,e.project_id,e.plan_version_id,e.cost_item_id,'budget',e.value_state,e.amount,e.low,e.high,e.as_of,e.origin,e.review_status,e.meaning,e.rationale,
 e.provenance||jsonb_build_object('moved_from','estimate','migration','021o')
FROM cost_values e JOIN cost_plan_versions v ON v.org_id=e.org_id AND v.id=e.plan_version_id AND v.status='draft'
WHERE e.metric='estimate' AND e.value_state='known'
ON CONFLICT(org_id,plan_version_id,cost_item_id,metric) DO UPDATE
SET value_state=excluded.value_state,amount=excluded.amount,low=excluded.low,high=excluded.high,as_of=excluded.as_of,origin=excluded.origin,
 review_status=excluded.review_status,meaning=excluded.meaning,rationale=excluded.rationale,provenance=excluded.provenance,version=cost_values.version+1
WHERE cost_values.value_state='unknown';
DELETE FROM cost_values e USING cost_plan_versions v
WHERE v.org_id=e.org_id AND v.id=e.plan_version_id AND v.status='draft' AND e.metric='estimate';
