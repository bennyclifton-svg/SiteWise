# WP-42a: report citations

Date: 2026-10-05. Status: implementation in progress, local evidence only.

Outcome: saved report values retain readable provenance outside an app session.
Lane A; supports answer trust. No export engine, issue flow or cost ledger in
this slice. Existing report budgets remain read/write 100/250 ms and assembly
300/1000 ms. No Jev request, new service or dependency.

`internal/reports/citations.go` assigns explicit E (evidence), U (user input),
C (calculation) and A (assumption) references to report blocks. Each reference
saves its source object and printable detail. Evidence detail includes filename,
document number, revision, page, location and section; absent details say “not
recorded”. Nested saved document sources are included, sorted and deduplicated.
The review status and meaning remain separate from origin. A planning acceptance
does not turn an assumption into evidence. Every assumption used in the report
is included in `material_assumptions` until a narrower materiality rule is agreed.

Protected wording receives a separate U reference with its editor and text,
while retaining the generated source reference. Missing source blocks retain a
U reference and an explicit review warning. Citation metadata does not affect
generated-content hashes, preventing false protected-edit conflicts.

Refresh and edit save references and sections in the same transaction. GET
reads saved references with the saved draft, scoped by organisation/version;
it does not reconstruct them from current documents or knowledge. The existing
issued-content triggers protect this table for the later issue implementation.

The integration test exposed an existing silent marshal failure in interface
blocks: YAML question criteria were included in a report source snapshot. The
assembler now snapshots the relationship fields actually used and rejects a
calculated basis that cannot be encoded, instead of saving an empty basis.

Pure tests cover printable source detail, all labels, protected edits, missing
sources, malformed bases, determinism and hash stability. Store tests verify
saved references, every block's reference resolution and persisted editor
provenance. Report, store and HTTP suites passed (`.tools/wp42-tests.log`).

The reduced local diagnostic passed all user-path speed gates
(`bench/results/2026-10-05-wp42-citations-diagnostic.json`). Report read p50/p90
was 4.5/4.7 ms (40 samples), write 3.0/5.5 ms (80), and assembly 8.2/8.9 ms
(40), within their respective 100/250, 100/250 and 300/1000 ms budgets. Filing
was 445.5/624.8 ms. This basic report workload and 20-file, two-pass filing run
does not establish larger-corpus, VPS or release performance.

User/profile assumptions now include the recorded author, timestamp and rationale.
The lookup matches scope, part, key and version within the current project/site;
an explicitly stale profile cannot borrow the author of a newer edit or another
scope. Protected report edits retain their database timestamp through refresh,
including orphaned edits. Calculation references print saved clause ID/version/
status, rule results, allocation inputs, interface details and proposal triggers/
determinants. Evidence excerpts are retained in the printable reference.

Report/store/HTTP suites passed again (`.tools/wp42-audit-tests.log`), including
an assumption round trip with author/time/rationale and an independent check
against newer-version/wrong-scope attribution. Pure tests additionally verify
actual catalogue serialization and saved calculation inputs. Missing author or
time remains explicitly “not recorded”; the existing package-stage schema does
not record an author/time, so this change does not fabricate those fields.

After the audit lookup, the reduced benchmark again passed every user-path
gate (`bench/results/2026-10-05-wp42-audit-diagnostic.json`): report read
5.0/5.6 ms, write 3.2/6.3 ms, assembly 9.3/10.0 ms p50/p90. Filing remained
463.6/624.7 ms. The same local/basic-workload limitations apply.

Remaining: citation detail UI and greyscale checks (WP-45a),
larger saved-corpus report evidence, export references (later package), and
owner-reviewed clauses. No WP-42 completion claim.
