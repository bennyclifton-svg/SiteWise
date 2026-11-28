-- A contract's breakdown (docs/plans/2026-11-26-payment-schedules.md,
-- slice 3): the lines a payment schedule prints for a live award. A line
-- belongs to the award and rolls up to one of its cost lines, so Claimed to
-- Date per cost line is unchanged. Works lines and provisional sums are
-- sections of it. Money is integer cents ex GST; null is unknown.
--
-- Numbered 090: the tender work on another branch holds 080 to 084, and
-- migrations are applied by name, in any order.
CREATE TABLE contract_breakdown_lines (
 org_id uuid NOT NULL,
 id uuid NOT NULL,
 project_id uuid NOT NULL,
 award_id uuid NOT NULL,
 cost_item_id uuid NOT NULL,
 section text NOT NULL CHECK (section IN ('works','provisional')),
 ordinal integer NOT NULL CHECK (ordinal >= 0),
 label text NOT NULL CHECK (length(btrim(label)) BETWEEN 1 AND 500),
 -- The PM's note beside the label: "actual 30.31 tonne".
 note text NOT NULL DEFAULT '' CHECK (length(note) <= 300),
 -- Total let, or the provisional sum.
 amount_cents bigint,
 -- A provisional sum the PM marked settled at what was completed.
 final_at timestamptz,
 -- award: the line that stood for a whole cost line when the contract was
 -- let, and still stands for claim rows that name only the cost line.
 -- tender: split from the awarded firm's own lines. user: the PM's.
 source text NOT NULL CHECK (source IN ('award','tender','user')),
 -- Retired, not deleted: issued schedules keep the line.
 retired_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
 PRIMARY KEY (org_id, id),
 UNIQUE (org_id, project_id, id),
 FOREIGN KEY (org_id, project_id) REFERENCES projects(org_id, id) ON DELETE CASCADE,
 -- Deferred: a project's awards and cost rows go in one transaction with
 -- its breakdown, in whichever order the cascade takes.
 FOREIGN KEY (org_id, project_id, award_id) REFERENCES awards(org_id, project_id, id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY (org_id, project_id, cost_item_id) REFERENCES cost_items(org_id, project_id, id) DEFERRABLE INITIALLY DEFERRED,
 CHECK (final_at IS NULL OR section = 'provisional')
);
CREATE INDEX contract_breakdown_award_idx ON contract_breakdown_lines (org_id, project_id, award_id, ordinal);

-- A claim row placed on a breakdown line. Null is a row that names only its
-- cost line: it sits on that cost line's 'award' line, worked out at read,
-- so no issued (frozen) row is touched here.
ALTER TABLE progress_claim_lines ADD COLUMN breakdown_line_id uuid;
ALTER TABLE progress_claim_lines ADD CONSTRAINT progress_claim_lines_breakdown_fk
 FOREIGN KEY (org_id, project_id, breakdown_line_id) REFERENCES contract_breakdown_lines(org_id, project_id, id) DEFERRABLE INITIALLY DEFERRED;

-- Every live contract gets one line per cost line, in the award's order,
-- with the label and contract value the draft plan holds.
INSERT INTO contract_breakdown_lines (org_id, id, project_id, award_id, cost_item_id, section, ordinal, label, amount_cents, source)
SELECT x.org_id, gen_random_uuid(), x.project_id, x.award_id, x.cost_item_id, 'works',
 (row_number() OVER (PARTITION BY x.org_id, x.award_id ORDER BY x.ord, x.cost_item_id) - 1)::integer,
 left(COALESCE(NULLIF(btrim(x.label), ''), 'Contract line'), 500), x.amount_cents, 'award'
FROM (
 SELECT s.org_id, s.project_id, s.award_id, s.cost_item_id, min(s.ordinal) AS ord,
  COALESCE(max(r.label), max(s.label)) AS label, max(round(v.amount * 100))::bigint AS amount_cents
 FROM cost_line_sources s
 JOIN awards a ON a.org_id = s.org_id AND a.id = s.award_id AND a.undone_at IS NULL
 LEFT JOIN cost_plans cp ON cp.org_id = s.org_id AND cp.project_id = s.project_id
 LEFT JOIN cost_item_revisions r ON r.org_id = s.org_id AND r.plan_version_id = cp.draft_version_id AND r.cost_item_id = s.cost_item_id
 LEFT JOIN cost_values v ON v.org_id = s.org_id AND v.plan_version_id = cp.draft_version_id AND v.cost_item_id = s.cost_item_id
  AND v.metric = 'commitment' AND v.value_state = 'known'
 GROUP BY s.org_id, s.project_id, s.award_id, s.cost_item_id
) x;

-- The PM's edits to a breakdown have one Undo, in the claims area.
CREATE TRIGGER capture_change AFTER INSERT OR UPDATE OR DELETE ON contract_breakdown_lines
 FOR EACH ROW EXECUTE FUNCTION capture_change('version');
