# WP-21: existing systems and part-level work type

Date: 5 October 2026. Lane A. Implemented in the supplied `main` working tree
on base/head `80a5a4244623109b5da70b14859c303a46e056b8`, after WP-20. No commit,
merge or deployment. Independent review and integration acceptance remain.
Contracts: plan D-07/D-27, §4.3/4.5, plan commit
`ceb67518a03701a7df948fe2f384cce816e0e404`.

## Outcome

For refurbishment, remediation and advisory work, applied document presence
with no applicable action produces `sys.<system>.existing = present` on the
site, retaining all document provenance. It no longer creates a work item or
compatibility scope row by itself. The original presence reading remains
unchanged for audit and factual answer keys. New and extension work retain
their existing presence-to-scope behaviour.

The pure profile build resolves the project work type from the whole part and
uses a stated part work type as an override only for that part. This also fixes
an ordering defect: the first alphabetically sorted part's header could
previously activate project-wide scope defaults. Proposed work actions use the
same override rules. User scope choices remain final and default to the part's
action; resetting one hands control back to the normal routing.

Low-confidence or conflicting presence creates neither existing-system facts
nor work. Existing site user values are not overwritten. The versioned profile
endpoint accepts `present` and `absent` for the existing-system key. The scope
registry already assigned that key to the site and `hdr.work_type` to the
project, with a part ID; no registry or schema change was needed.

No migration, service, dependency, model call, question wording, threshold,
knowledge status or factual answer key changed. The action question and its
own threshold remain WP-22; predicate evaluation remains WP-25. A rule-supplied
action exercises the routing boundary in this package's tests.

## Verification

- Full Go suite with the dedicated test database: passed.
- Source replay 15/15, Hale replay 12/12, zero forbidden applied readings.
- Ninety combinations of project work type, part override and action absence,
  silence or presence pass. They cover a new extension part on a refurbishment
  project, the reverse override, inherited type and all five work types.
- Existing scope tests remain green. New tests cover unresolved/conflicting
  presence, user scope, site user precedence, project-default isolation and
  database persistence of site provenance. The store test confirms resetting
  a choice does not revive existing-system presence as proposed work.
- WP-20's browser/build/OCR/strict-knowledge/intake checks were already green;
  this package changes no assets, extraction, knowledge data or filing calls.
- Local endpoint/end-to-end workload gate passed. p50/p90 in milliseconds:
  profile edit 35.0/113.9 (50/150 budget), rebuild 69.2/73.6 (100/300),
  works read 1.0/1.9 and write 8.3/10.6 (100/250), filing 334.2/879.7
  (1,000/2,000). Exact results are in `bench/results/2026-10-05-wp21.json`.
  Development component limits remain over for identity extraction
  (69.3/360.3 versus 80/250), deterministic rules (1.3/4.0 versus 1/1)
  and Jev admission/request (237.5/1,200.1 versus 350/800, with deliberate
  degraded calls). Target-VPS component and live release evidence remain
  outstanding; the local gate does not waive them.

```powershell
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
go test -p 1 ./...
go run ./cmd/profile-eval
go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
go run ./cmd/intake-bench -out bench/results/2026-10-05-wp21.json -samples-out .tools/wp21-samples.json
```

## Traceability

Implemented in code, awaiting review/integration: NW-REQ-067, 075, 080, 108,
116, 117. AT-29 routing and AT-05 factual replay plus synthetic mixed-part
fixture pass. NW-REQ-005/315 remain partial until the wider work-item/proposal
and owner-reviewed case gates pass. NW-REQ-376's part override is implemented;
its knowledge predicate portion belongs to WP-25. No row is marked Verified.

The one-project-per-site restriction remains. Facts are document-owned and the
new site row is a recomputable projection, consistent with WP-15; it is not
copied into a second authoritative fact store. Production readiness still
requires the wave's migration, reviewer, owner-key and VPS gates.
