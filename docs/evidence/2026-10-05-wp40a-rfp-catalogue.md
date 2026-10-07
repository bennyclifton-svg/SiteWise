# WP-40a: draft RFP catalogue

Lane A. Outcome: responsibilities and the shared report assembler can reference
versioned RFP wording and stable template sections. No runtime AI, dependency,
service, legal requirement or verified rule number is added.

The template shape was documented in `knowledge/SCHEMA.md` before adding data.
`knowledge/reports/templates.yaml` contains the draft `tpl.rfp-capex` version 1,
with seven essential sections: brief; services; investigations; interfaces;
dates; fee return; proposal requirements. Seven matching draft clauses cite the
existing architecture/schema documents. They are requests for proposal content,
not approved obligations or an appointment. Fee wording creates no amount or
cost ledger; M1 must keep unavailable costing explicit.

Go startup loads and hashes clauses/templates, validates clause IDs, output
kinds, versions, statuses and required metadata, and rejects incompatible
template references, duplicate sections and missing essential flags. Clause
lookup returns an exact version; `ApprovedClause` still requires owner-reviewed
status. Tests prove every new fragment remains unapproved.

The Python checker validates template structure and cross-references with the
same section/output/version rules. `python tools/check_knowledge.py --strict`
passes with 0 errors and the three existing legacy package-field warnings
(`.tools/wp40a-knowledge-check.log`). All 30 Python tool tests pass, as do Go
knowledge and procurement suites. No owner review is claimed.

WP-40a's catalogue implementation is locally complete; RFT/PMP catalogues remain
WP-40b. Owner review is required before standard obligations or issue. Assembly,
citations, protected edits, issue and export belong to their later packages and
have not been verified by these catalogue tests.
