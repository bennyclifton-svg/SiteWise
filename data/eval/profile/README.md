# Profile evaluation

`manifest.json` lists the private research documents (paths under
`SITEWISE_PROFILE_CORPUS`, default `D:/AI Projects/Test Data`) with SHA-256
hashes. Never copy the documents into the repo.

`answer-keys/<project>.yaml` are **agent drafts** (`reviewed: false`). Only the
owner changes that. Scoring rules for the harness (plan task P16):

- `accept: [...]`: a filled value must be one of these; a blank is a miss.
- `blank_ok: true`: a blank is correct. With `accept: []`, any fill is a
  wrong value.
- `assertion`: the expected assertion (stated, required, allowance).
- `systems`: `presence` is included, not_included or not_stated (expected
  blank); `provider` when the document says who supplies it.
- Systems not listed are not scored.
- Mornington is three tenders for one house: `tenders: differ` marks values
  the profile should show as "Tenderers differ".

Gates: no wrong green; report the share correct without user action against
the 80% target.

Version-2 work-item keys use exact system/action/part matching instead. See
[work extraction evaluation](WORKS.md) for the offline command, source and
version pins, precision/recall rules and owner-review gate. The 0991 key is a
bounded draft for two drawings; 0777 awaits correction of a mismatched design
set. Neither is an approved quality result.
