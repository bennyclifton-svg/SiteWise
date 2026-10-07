# Package-default value mapping validation

Date: 6 October 2026. Lane A catalogue-validation correction.

Outcome: expose unresolved consultant-default mappings before they are used.
The checker previously warned when a complexity field was unknown but accepted
any value for a recognised field. The draft defaults use `heritage_status` with
`local_heritage_item` and `state_heritage_register`; the determinant actually
defines `local_item` and `state_register`. A recognised field name therefore
did not prove that its trigger could match.

`check_catalogues` now checks listed values against declared determinant or
profile-condition options, and against `true`/`false` for boolean determinants.
Unmapped field/value names remain warnings, consistent with preserving the
explicit legacy draft catalogue for mapping. Scalar values, non-string list
members and malformed field names produce errors rather than being interpreted
as usable rules. Free-form fields without declared options do not acquire an
invented enumeration.

Regression coverage exercises valid determinant/condition/boolean values,
invalid values on each, an unknown field and malformed list/field shapes.
`python -m unittest discover -s tools -p 'test_*.py'`: 31 tests pass in 6.870 s.
`python tools/check_knowledge.py --strict`: zero errors, five warnings, 2,000
dataset rows and zero pending. The two additional warnings are the heritage
values above; the three existing unknown fields remain. `git diff --check`
passes with the existing line-ending warnings.

No catalogue data, mapping, Jev question or runtime suggestion changed.
`BaselinePackages` still consumes supported building-class/work-type baselines
only; it does not activate legacy complexity additions. The mapping and content
review required for those additions remain open. No new dependency or timing
claim: this developer-side validation is outside user-facing paths, whose
existing budget failures remain unresolved. This is not WP-30 or whole-wave
completion.
