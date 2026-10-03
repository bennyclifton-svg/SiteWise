# PPR cover title diagnosis

The reported upload stored its abbreviated filename as a green rule title.
Replaying the original PDF reproduced that decision: the reader recovered
the printed requirements heading, but the heading harvester rejected it.
Its text height was approximately 12.16 points against 9.74-point footer
text, below the 1.4x prominence filter. The filename was therefore the only
title candidate and the unique-value rule settled it before Jev was asked.

The focused change retains short, early, all-capital requirements headings
on page one even at body size, as existing handling does for legal headings.
It does not settle those headings by rule. They compete with the filename
inside the existing single fan-out, following TypeSafe's
[pre-parsed extraction pattern](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook).
No dependency, extra round trip, or confidence-threshold change was added.

The regression failed before the change and passes after it. It exercises
Harvest through Draft.Call, checking the literal heading reaches the title
question; mixed-case body prose and later pages remain excluded. Replaying
the original PDF now returns both title candidates and an unresolved title
rule. Revision and date decisions remain the same.

Intake, identity and store tests pass against the dedicated test database.
Candidate harvesting p50/p90 measured 3.490/3.794 ms (budget 5/10 ms), and
rules 0.313/0.314 ms (budget 1/1 ms). These are component measurements, not
whole-filing timing or live Jev accuracy evidence.
The full intake benchmark could not validate the 1,000/2,000 ms gate:
it exited with 14 Jev requests lacking recorded documents. New live
recordings are required before the full replay gate can pass.

Automatic title adoption remains limited by the existing empty title
calibration table. The candidate fix prevents false rule certainty but does
not authorise an uncalibrated Jev title. The running application and existing
upload have not been changed by this diagnosis.
