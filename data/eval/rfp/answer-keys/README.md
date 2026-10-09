# Consultant RFP selection answer keys

Each file is one scenario: the project facts a consultant RFP is resolved
from, any user settings, and what the resolver must select for each
discipline. `internal/rfp/answerkey_test.go` runs them. They test
*selection* (which stages, positions and decisions), independently of the
wording snapshots in `internal/rfp/testdata/`.

All keys are agent drafts (`reviewed: false`). Only the owner sets
`reviewed: true`. Do not edit an expectation to match the output without
the owner's approval; fix the discipline rules or raise the disagreement.

Expectations list what must hold, not every row. Keys:

- `stages_include`, `stages_option`, `stages_absent`, `stages_ask`
- `elements_include`, `elements_option`, `elements_absent`
- `positions`: item key → `I`, `O`, `X`, `ask` or `absent`
- `party`: item key → text the X party must contain
- `blocking`: check IDs that must block issue; `blocking_only` means no
  other blocking check may appear
- `checks`: non-blocking check IDs that must appear
- `scale`, `role` (role line keys that must appear), `role_absent`
- `sources`: why each expectation holds
