# WP-K4 vehicle-clearance record ownership

Date: 6 October 2026. Status: implemented bounded merge-pass correction.

## Outcome and scope

Moved `fm.vehicle-clearance-lost-under-structural-beam` from fire to structure.
The two source rows, B2-0916 and B2-0917, describe car-park/loading-dock clearance
lost under a structural beam. The record itself and the Pass B report identify
fire placement as an artefact of keyword triage. Structure is the appropriate
file owner for this beam coordination failure. Correct ownership helps future
knowledge review find the physical constraint.

Lane A knowledge maintenance. No new record, detector, routing, dependency,
approval, or legal requirement. The complete record is preserved, including
its stable ID, access-egress attachment, structure/access-egress `runs_on`,
question wording, severity, sources and draft status. The historical note about
fire placement is retained. The corresponding two ledger entries moved from
the fire overlay to the structure overlay. Original dataset and triage files
are unchanged.

## Validation

- Parsed every failure-mode YAML record before and after: all 859 keyed
  records compare equal, with no duplicate IDs in the baseline. The change
  is file ownership only; it does not resolve whether the attachment, severity
  or cited technical standard is correct.
- `python tools/check_knowledge.py --strict`: zero errors, 2,000 dataset rows,
  zero pending. Three existing package-default mapping warnings remain.
- `go test -p 1 ./internal/knowledge ./internal/profile ./internal/works`:
  passed in 3.422 / 7.355 / 1.786 seconds respectively.
- Refreshed `docs/unforeseen/k4-current-audit.json` against the same pre-K4
  baseline. This record was added after that baseline; the inventory of
  changed pre-K4 records is unchanged.

The loader sorts evidence by ID after reading all cluster files, so relocation
does not add or reorder runtime questions. File-content knowledge fingerprints
can change, as expected for a catalogue edit. Filing budgets remain p50
1,000 ms / p90 2,000 ms; profile edit 50/150 ms and rebuild 100/300 ms.
No new timing measurement is claimed for this content-preserving relocation.
The failing full-fixture latency result remains open.

WP-K4 is not complete: other flagged records, overlaps, loose merges, contract
wording and owner review remain outstanding. Neither the checker nor this
relocation promotes any record to reviewed.
