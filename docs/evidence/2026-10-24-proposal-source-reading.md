# Proposal source-reading repair

Lane A: preserve the source offer, manual decisions and money. No new dependency,
provider, schema, cost-plan write or award action. Existing filing remains separate.

Reproduced with Markdown regression fixtures: a fixed fee followed by “staged per
engagement letter” vanished as a unit rate; stage-table GST headings were ignored;
bold signature titles entered company candidates; explicit exclusion headings were
missed. Source quotes repeated in forwarded email appeared as additional offers.

The proposal harvester now recognises Markdown headings and cells, reads tax from
the fee column, separates signature roles from company names, and retains explicit
back-references without adding them as another offer. Code reconciles stage
breakdowns with their printed total. Unit-rate detection requires a unit after
“per”. Equal amounts for distinct tasks remain distinct.

Source scope, exclusions, qualifications, conditions and disbursement wording are
preserved as excerpts in the consultant comparison. Coverage v2 adds partial scope
and asks about the requested stage/extent. It remains an uncalibrated suggestion;
partial coverage cannot confirm scope or money. Existing calibrated field questions
are unchanged; the request version includes coverage-2 to separate cached reads.

This follows TypeSafe's [fan-out pattern](https://docs.typesafe.ai/patterns/fan-out),
[pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
[confidence rules](https://docs.typesafe.ai/confidence) and
[Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13): one
background call per source, literal candidates, code arithmetic, no generated prose.

A reread does not supersede a related source merely because its firm is now known.
Manual package and price decisions remain protected. Multiple source documents for
one firm are labelled as such, with a warning not to add their fees.

Validation: proposal unit suite; focused store/API proposal, comparison, award and
attachment regressions; dedicated reread preservation regression; source-reading,
repository-link and reassignment browser scenarios; TypeScript/Vite build. Dark-theme
source comparison screenshot inspected. Three project readings saved and checked
against original source documents; twelve repository documents remain.

Local live Jev rereads, 21 samples: p50 332.3 ms, p90 388.7 ms. These measure existing
source extraction/context plus one Jev fan-out, not a fresh upload/filing. The recorded
proposal path budget is 3,000/6,000 ms including filing. Final save readings including
commit took 338.5–490.5 ms. Saved price reads, 20 samples: p50 4.4 ms, p90 6.7 ms
against the named 150/300 ms budget (also below the general 100/250 ms read target).
Repository link/move browser gates also passed at 10.0/14.6 and 13.4/15.0 ms,
respectively, against 100/250 ms. Private requests, source files and before/after
snapshots remain in ignored tmp/.

During browser polling while the background read was committing, an existing
price-read transaction returned a serialization error once. The source-reading
scenario now waits for the board's received state before inspecting the completed
price table. Concurrent price-read retry behaviour was not changed in this repair.
