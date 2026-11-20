-- Project mail addresses are named after the project (owner, 20 November
-- 2026): "ST13 House" is st13_house@sitewise.au, not "p-" and 32 random hex
-- characters. The address is no longer private; the sender check (DKIM in
-- Go) and holding mail from non-members are what keep it safe.
--
-- The rule for a name (store.mailLocalBase) is repeated here once, for the
-- addresses that already exist: lower case, letters and digits, every other
-- run one underscore, at most 40 characters, "project" when nothing is
-- left, "_project" after a name the domain keeps for its own mailboxes, and
-- "_2", "_3" ... when the name is taken.
ALTER TABLE project_mail_addresses DROP CONSTRAINT project_mail_addresses_local_part_check;
ALTER TABLE project_mail_addresses ADD CONSTRAINT project_mail_addresses_local_part_check
 CHECK (local_part ~ '^[a-z0-9][a-z0-9_-]{0,63}$');

DO $$
DECLARE
 r record;
 base text;
 cand text;
 n int;
BEGIN
 FOR r IN
  SELECT a.org_id, a.project_id, p.name
  FROM project_mail_addresses a
  JOIN projects p ON p.org_id = a.org_id AND p.id = a.project_id
  WHERE a.local_part ~ '^p-[0-9a-f]{32}$'
  ORDER BY p.created_at, a.project_id
 LOOP
  base := trim(both '_' from regexp_replace(lower(r.name), '[^a-z0-9]+', '_', 'g'));
  base := trim(both '_' from left(base, 40));
  IF base = '' THEN
   base := 'project';
  END IF;
  IF base IN ('abuse','accounts','admin','administrator','billing','contact','help','hello',
              'hostmaster','info','legal','mail','noreply','postmaster',
              'privacy','root','sales','security','support','team','webmaster') THEN
   base := base || '_project';
  END IF;
  n := 1;
  LOOP
   cand := CASE WHEN n = 1 THEN base ELSE base || '_' || n END;
   EXIT WHEN NOT EXISTS (SELECT 1 FROM project_mail_addresses WHERE local_part = cand);
   n := n + 1;
  END LOOP;
  UPDATE project_mail_addresses SET local_part = cand
  WHERE org_id = r.org_id AND project_id = r.project_id;
 END LOOP;
END $$;
