# Project profile: first live smoke run (3 October 2026)

Branch `profile/impl`. One document, live Jev (`jev-1.13.0`), provisional
floors (`-profile-provisional`), a fresh database, `serve` on a spare port.
This is a smoke test of the pipeline, not an accuracy result: there are no
reviewed answer keys yet (task P02/O1).

**Input:** Hale `Design/Design Brief [P1].pdf` (21 pages, warehouse
extension). Uploaded through the API; filing, full text, labelling, evidence
and profile rebuild all ran in the background.

**Result:** profile built 30 s after upload with no Jev errors.

| Area | Read | Against my draft reading of the brief |
|---|---|---|
| Building type | warehouse (amber, 4 sources) | correct (warehouse or logistics) |
| Work type | extend (amber, 3 sources) | correct |
| Water supply | reticulated (amber) | correct (town water) |
| Sewer | reticulated (amber, asserted "required") | correct ("connection to mains sewer") |
| Systems evidenced as included | sprinklers, vehicle doors, external walls, roof coverings, slab on ground, cold and hot water, sanitary, joinery, lighting, stairs | correct |
| Systems evidenced as included, doubtful | electrical.ev-charging | wrong: the brief's "MHE charging area" is forklift battery charging, not EV charging |
| Missed absences | solar PV and natural gas are "Not applicable" in the brief | not read as *not included*; no row at all |
| Suggested (not evidence) | 30 systems from warehouse × extend | shown as "Typical", as designed |

**Defect found and fixed in this run:** every evidence call had been rejected
before sending, because YAML read the fire pilot's `true:`/`false:` criteria
keys as booleans. The background worker had never run in `serve` before this
branch, so it was latent. Commit `02bf86b`; test `TestEvidenceQuestionsEncode`.

**Follow-ups for P17 (tuning):**

1. Absence detection. "Not applicable" lines sit under a heading such as
   "Solar PV System". The passage is short and may not reach the label floor
   or the right leaf. Check the label answers for those passages first.
2. Leaf boundary: add "forklift and materials-handling battery charging" to
   `electrical.ev-charging` excludes (or `electrical.process-hazardous`
   describes).
3. Run the five-project answer keys (P02) through a harness (P16) before
   tuning anything.

Reproduce: `smoke.ps1` in the session scratchpad started the server with
`sitewise serve -addr 127.0.0.1:8090 -profile-provisional` against a
throwaway database, then uploaded the file and polled
`GET /api/projects/{id}/profile`.
