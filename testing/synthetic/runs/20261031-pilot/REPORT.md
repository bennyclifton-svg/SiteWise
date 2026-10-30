# Pilot run 20261031-pilot: T01 new house

Three scenarios through a separate local SiteWise (`harness/serve.sh`,
`:8091`, live Jev): ST01-base, ST01-cdc-pathway, ST01-trees-affected.
About 15 s per wave; one 160 s wave in the trees run. Findings are in
`../../FINDINGS.md`. Per-scenario detail: `<scenario>/field_score.txt` and
`shape_score.txt` (JSON alongside).

| Area | Base | CDC | Trees |
|---|---|---|---|
| Register: kind, title, status | 14/14 | 13/13 | 16/16 |
| Register: discipline | 12/14 | 10/13 | 14/16 |
| Profile values correct (of keyed) | 44/60 | 44/60 | 45/60 |
| Trigger values read right | n/a | planning = cdc | both tree flags = yes |
| Procurement matched | 7/11 | 6/10 | 8/12 |
| Cost plan lines matched | 24/34 | 22/34 | 25/36 |
| Program activities matched | 51/83 | 49/81 | 53/85 |
| Program skeleton | generic (want new-traditional-da) | generic | generic |
| RFPs found | 7/10 | 6/9 | 8/11 |
| Fee totals right | 4/5 | 4/5 | 5/6 |

The profile counts include values that are missed by design and conflicts
that include the true value; see the score files.
