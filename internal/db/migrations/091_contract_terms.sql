-- Payment schedules in the owner's format
-- (docs/plans/2026-11-26-payment-schedules.md, slice 4): what a schedule's
-- header and retention lines print, typed once.

-- Per contract: the contract's own date (never the award's), and its
-- retention as a rate of a base. Null is not entered: SiteWise has no
-- default rate and no release rule.
ALTER TABLE awards ADD COLUMN contract_date date;
ALTER TABLE awards ADD COLUMN retention_bp integer CHECK (retention_bp BETWEEN 0 AND 10000);
ALTER TABLE awards ADD COLUMN retention_base text CHECK (retention_base IN ('works','let','to_date'));

-- Per project: the principal and the principal's representative, as the
-- schedule names them. Company numbers are kept as typed; Go warns, and
-- does not refuse, when one is not an ABN or a nine-digit ACN.
ALTER TABLE delivery_settings ADD COLUMN principal_name text CHECK (length(principal_name) <= 200);
ALTER TABLE delivery_settings ADD COLUMN principal_abn text CHECK (length(principal_abn) <= 40);
ALTER TABLE delivery_settings ADD COLUMN principal_email text CHECK (length(principal_email) <= 320);
ALTER TABLE delivery_settings ADD COLUMN assessor_org text CHECK (length(assessor_org) <= 200);
ALTER TABLE delivery_settings ADD COLUMN assessor_company_no text CHECK (length(assessor_company_no) <= 40);
ALTER TABLE delivery_settings ADD COLUMN assessor_name text CHECK (length(assessor_name) <= 200);
