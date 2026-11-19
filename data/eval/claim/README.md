# The progress claim answer key

Status: draft and empty. The owner supplies it (variations plan, decision
7): 20 or more real head-contract progress claims, kept private in
`data/eval/claim/private/answer-key.json` (git-ignored). No case here is
invented. Until a key is reviewed and `data/claim/thresholds.json` is fitted
for question version `claim-1`, every claim pick Jev makes is stored blank
and never placed; only Go's rules place a row.

`go run ./cmd/invoice-eval -claims` checks the key's shape and counts it.
The live run, its replay and the fit follow the invoice key's
(`cmd/invoice-eval`) when the first cases arrive; the row options are then
shuffled per case, since Jev leans to the first option (jev-1.13).

## Format

```json
{
 "version": 1,
 "reviewed": false,
 "question_version": "claim-1",
 "corpus_root": "the private folder holding the claims",
 "cases": [
  {
   "id": "a short id",
   "path": "the claim's file, relative to corpus_root",
   "sha256": "the file's hash",
   "contract": {"lines": ["the contract's lines, in contract order"], "variations": ["V01 its title"]},
   "number": "the claim's number as printed, or null",
   "date": "YYYY-MM-DD, or null",
   "reference_date": "YYYY-MM-DD, or null",
   "rows": [{"label": "the row as printed", "line": "a contract line or variation from contract, or none"}]
  }
 ]
}
```
