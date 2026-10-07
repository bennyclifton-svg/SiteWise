# Exact package-default identifier mappings

Date: 6 October 2026. Lane A draft-knowledge correction.

Compared the frozen `consultant-rosters.json` complexity additions with the
existing SiteWise determinant definitions. Reconciled only identical named
concepts; no new interpretation, field, threshold or option was introduced.

| Source identifier | Existing SiteWise identifier |
| --- | --- |
| heritage_status: local_heritage_item | heritage_status: local_item |
| heritage_status: state_heritage_register | heritage_status: state_register |
| bushfire_exposure: bal_low | bal: BAL-LOW |
| bushfire_exposure: bal_12_5 | bal: BAL-12.5 |
| bushfire_exposure: bal_19 | bal: BAL-19 |
| bushfire_exposure: bal_29 | bal: BAL-29 |
| bushfire_exposure: bal_40 | bal: BAL-40 |
| bushfire_exposure: bal_fz | bal: BAL-FZ |

`conservation_area` already matches and remains unchanged. Consultant lists,
baselines, source citation and `status: draft` are preserved. The original
archived roster is unchanged. BAL identifiers here are catalogue tokens, not
newly verified regulatory thresholds. This does not establish bushfire-prone
land from BAL or approve the recommendation to engage any consultant.

Two mappings deliberately remain unresolved: contamination severity categories
are not equivalent to `potentially_contaminated_land`, and the three legacy
flood categories are not equivalent to `flood_hazard`. Both SiteWise facts are
boolean and cannot preserve those source distinctions. Do not collapse them
to true, fabricate enums, or infer them from numerical flood levels.

Strict checker: zero errors and two warnings (those unresolved fields), 2,000
covered dataset rows and zero pending. Knowledge/procurement tests pass
(3.438/1.086 s). Refreshed current K4 catalogue hashes. No new dependencies.

Runtime `BaselinePackages` still consumes baselines only; this data correction
does not enable complexity-based suggestions. Catalogue hashes change, so
existing freshness mechanisms can identify changed knowledge. No user-path
operations or Jev questions are added and no latency result is claimed. Existing
edit/rebuild failures, content review and later package gates remain open.
