-- Project facts live in Profile (owner decision D1, 10 October 2026): an RFP
-- reads them and an edit in the RFP writes there. Draft overrides of facts
-- and of fields with a profile home are dropped (D2: test data only).
-- Issued versions keep theirs; their settings are part of a frozen basis.
DELETE FROM report_settings s
USING report_versions v
WHERE v.org_id = s.org_id AND v.id = s.report_version_id AND v.status <> 'issued'
  AND (s.key LIKE 'driver:%' OR s.key IN ('field:client','field:address','field:lga','field:zone','field:lot_dp','field:program','field:sustainability'));
