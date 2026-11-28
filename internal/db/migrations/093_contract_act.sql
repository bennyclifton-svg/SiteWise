-- Whether a contract's payment schedules name the Security of Payment Act
-- (docs/plans/2026-11-26-payment-schedules.md, slice 7, owner decision 8).
-- Null is the default by kind: on for works, off for services, because
-- whether a consultant's invoice is a payment claim under the Act is the
-- PM's call, not SiteWise's.
ALTER TABLE awards ADD COLUMN act_applies boolean;
