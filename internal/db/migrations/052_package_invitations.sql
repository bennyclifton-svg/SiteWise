-- A firm the PM names on a Procurement row before its quote arrives: the
-- firms an RFP or RFT goes to (owner, 11 November 2026;
-- docs/plans/2026-11-11-procurement-invited-firms.md). Nothing is priced,
-- compared or awarded from it; the firm's proposal, once on the row, takes
-- its cell. One nomination per firm per package, gone with its package.
CREATE TABLE package_invitations (
 org_id uuid NOT NULL,
 project_id uuid NOT NULL,
 package_id uuid NOT NULL,
 firm_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (org_id, package_id, firm_id),
 FOREIGN KEY (org_id, project_id, package_id) REFERENCES packages(org_id, project_id, id) ON DELETE CASCADE,
 FOREIGN KEY (org_id, firm_id) REFERENCES firms(org_id, id) ON DELETE CASCADE
);
CREATE INDEX package_invitations_project_idx ON package_invitations (org_id, project_id, created_at);
