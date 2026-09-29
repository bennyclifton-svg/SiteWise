# Instant Intake Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** File a text-layer PDF, DOCX or XLSX into a chosen project within p50 ≤ 1 s and p90 ≤ 2 s after upload completion, preserving uncertainty and user corrections.

**Architecture:** One Go process serves an embedded React SPA, API, SSE and durable workers. PostgreSQL 17 and content-addressed files share the host. Code handles parsing, identity, ordering, persistence and the knowledge graph; one pinned Jev fan-out makes unresolved intake judgments.

**Tech Stack:** Go 1.27.1, PostgreSQL 17, pgx/sqlc, Vite/React, PDFium, Go ZIP/XML readers, Caddy/systemd. No runtime LLM, embeddings, agent loop, Supabase, Docker or pgvector.

---

This plan follows the agreed foundation design; it does not reopen the stack.
Fire is the first background knowledge pilot, including hydraulic fire water,
electrical supply, structural loading and smoke-control interfaces. Intake
classification still spans all document kinds and disciplines.

The existing knowledge is draft. Primary verification is independent of owner
review. `derives.pending` must return unknown. Baseline NCC tables cannot be
applied without their edition, jurisdiction, physical scope and exceptions.

## Development setup and evidence

Go is installed at `D:/AI Projects/sitewise/.tools/go/bin/go.exe`; `go version`
and a compile/run smoke test passed. The official Windows amd64 1.27.1 archive
was verified against SHA-256
`a3911b5e0e1b1053f25ed0675f4c1c6aad1e2bfcf253df2b9be4caabd2edd95d`.
The ignored `.tools` directory is local tooling, not deployment content.

PowerShell for subsequent work:

```powershell
$env:PATH = "$PWD/.tools/go/bin;$env:PATH"
$env:GOCACHE = "$PWD/.tools/go-cache"
$env:GOPATH = "$PWD/.tools/go-path"
$env:GOTOOLCHAIN = 'local'
go version
python tools/check_knowledge.py --strict
python -m unittest discover -s tools -p 'test_*.py'
```

Verify the available PostgreSQL 17 service and Node tooling before installing
anything else. Never assume an existing local database is disposable. Use a
dedicated `sitewise_test` database for tests and explicit environment variables.

Reference data located in the frozen sibling repo:

- `../clerk/data/taxonomy/disciplines.json`: 57 entries confirmed; copy only
  classification IDs, labels and aliases, excluding PM doctrine fields.
- `../clerk/data/eval/jev/design-intake-{hale,petersham,newham}.yaml`.
- `../clerk/data/eval/jev/answer-keys/{hale,petersham,newham}-design-docs.yaml`
  and their source/provenance files; Hale invoice key is separate.
- Source corpus locations are hints in the manifests, not verified local paths.
  Resolve them and match hashes before starting live evaluation.

Some answer-key labels are model judgments or unreviewed transcriptions; some
register revisions post-date the actual file. Preserve these flags. Do not
calibrate automated supersession against a later register revision as if it
were the supplied document's revision. Copy data, never reference-app code.

## Binding Jev contract

Pin `jev-1.13.0`, confirmed in [models](https://docs.typesafe.ai/models).
Use the [HTTP API](https://docs.typesafe.ai/api): authenticated JSON POST to
`https://api.typesafe.ai/v1/systemone`, structured `state` and a question map.
Validate response model, question IDs, answer type, allowed options and finite
probabilities before applying an answer. Malformed/missing fields stay unresolved.

Ask kind, discipline, lifecycle and unresolved identity/supersession questions
together, including discipline speculatively; code decides relevance afterward
([fan-out](https://docs.typesafe.ai/patterns/fan-out)). Harvest identity candidates
first and copy selected values verbatim
([pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook)).
No follow-up Jev call waits on another intake answer. Supersession candidates
are the union of same-number matches for all harvested number candidates,
scoped to the project, with code validating against the selected number later.
If that union exceeds the bounded option limit, leave supersession unresolved.

Calibrate green/amber/blank per question, version and option shape. Supersession
has the strictest action band; unknown thresholds disable automatic action.
Noul supplies probability, not a confidence field
([confidence](https://docs.typesafe.ai/confidence),
[routing](https://docs.typesafe.ai/patterns/confidence-routing)). No example
threshold from documentation becomes a production default.

Jev sees relevant identity text only. It does not count, compare dates, order
revisions, calculate compliance or generate titles
([limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13),
[building with System One](https://docs.typesafe.ai/concepts/how-to-build-with-system-one)).

## Latency gates

Measure from receipt of the final upload byte to durable decisions being
available through the SSE event. Include queue time, errors and degraded paths;
do not report only successful Jev calls or add component percentiles together.

| Path / component | p50 | p90 |
|---|---:|---:|
| Identity text extraction | 80 ms | 250 ms |
| Candidate harvesting | 5 ms | 10 ms |
| Deterministic field rules | 1 ms | 1 ms |
| Jev admission + request | 350 ms | 800 ms |
| Commit + SSE enqueue | 5 ms | 15 ms |
| Whole intake, including degraded outcome | 1,000 ms | 2,000 ms |
| Project/document list API, proposed initial budget | 50 ms | 150 ms |
| Field correction API, proposed initial budget | 50 ms | 150 ms |
| Project/invite/auth-token consumption API, proposed initial budget | 100 ms | 250 ms |
| Health/speed API, proposed initial budget | 25 ms | 100 ms |
| SSE reconnect catch-up, proposed initial budget | 100 ms | 250 ms |

The non-intake budgets are starting engineering targets; benchmark them on the
target host and record the chosen workload. Do not silently relax a gate. Upload
network duration and email delivery are externally variable: measure them
separately; gate bounded server handling, not third-party delivery time.

### Task 1: Bootstrap configuration and the failing speed gate

**Files:** create `go.mod`, `cmd/sitewise/main.go`, `internal/config/config.go`,
`internal/config/config_test.go`, `internal/latency/gate.go`,
`internal/latency/gate_test.go`, `bench/budgets.json`, `tools/check.ps1`.

1. Write tests for missing DB/file/auth/Jev configuration and reject a moving
   Jev model alias. Add nearest-rank percentile cases and a sample with good
   median but failing p90. Run `go test ./internal/config ./internal/latency`;
   expect failure until implemented.
2. Implement typed configuration, secret-presence errors without values, and
   a gate that returns a nonzero exit on insufficient samples or failed budgets.
   The percentile function sorts a copy and selects `ceil(p*n)-1`; empty input
   is an error. Store durations in integer microseconds.
3. Add `tools/check.ps1` to run strict knowledge validation, Go tests and the
   benchmark command, exiting on the first nonzero code. Do not allow skipped
   dependencies to print a successful overall build.
4. Re-run the tests; expect PASS. Commit the bootstrap and tests together.

### Task 2: Tenant schema and SQL boundary

**Files:** create `internal/db/migrations/001_init.sql`, `internal/db/queries.sql`,
`sqlc.yaml`, `internal/store/store.go`, `internal/store/isolation_test.go`.

1. Write DB integration tests that two orgs cannot read, update, download,
   supersede, stream or enqueue each other's documents. Include forged project
   IDs and same file hashes across orgs. Run
   `go test ./internal/store -run TestOrgIsolation -count=1`; expect failure.
2. Create orgs, users, memberships, invites, sessions, projects, files,
   documents, decisions, passages and jobs. Users are tenant-scoped in v1;
   memberships/sessions carry org scope. Foreign keys use `(org_id, id)` for
   tenant relationships. Unique membership, content hash and document identity
   constraints are scoped explicitly.
3. Every repository call takes org ID from authenticated context, never from a
   request body. Use SQL conditions for tenancy, not post-query filtering.
   Enforce migrations transactionally under an advisory lock at startup.
4. Generate sqlc code, then run `go test ./internal/store -count=1`; expect PASS
   against PostgreSQL 17. Commit; justify pgx and sqlc in the message.

### Task 3: Invitation and magic-link authentication

**Files:** create `internal/auth/tokens.go`, `internal/auth/handlers.go`,
`internal/auth/auth_test.go`, `internal/httpapi/projects.go`.

1. Write tests for expired/replayed tokens, wrong-org invites and project access
   before membership. Run `go test ./internal/auth`; expect failure.
2. Generate cryptographically random, expiring single-use tokens; persist token
   hashes. Consume token and membership/session creation atomically. Store only
   session IDs in secure, HttpOnly, SameSite cookies. Add CSRF/origin checks for
   mutations and bounded request sizes. Log no tokens.
3. Provide an explicit local bootstrap admin command and a test mail sink.
   Production startup requires configured mail delivery; no public signup.
   Project creation binds ownership to the session's org.
4. Run auth tests and tenant integration tests; expect PASS. Commit.

### Task 4: Durable upload and content-addressed storage

**Files:** create `internal/files/store.go`, `internal/files/store_test.go`,
`internal/intake/upload.go`, `internal/intake/upload_test.go`.

1. Test interrupted streams, hash collisions in metadata, concurrent duplicate
   uploads and DB failure after file rename. Run `go test ./internal/files
   ./internal/intake`; expect failure before implementation.
2. Stream bounded bytes to a temporary file while hashing. Flush and atomically
   rename to the hash path on the same filesystem. Treat an existing matching
   hash as immutable. Paths come from hashes, never uploaded filenames.
3. Commit the org/project file association and durable intake job in one DB
   transaction. Re-drop within the same project returns the existing filing;
   another authorized project may associate the same blob independently.
   Cross-org existence is never disclosed. Recover orphan blobs separately.
4. Persist unreadable/scanned files with a visible `not_filed` reason. A duplicate
   race must create one association/job, not two. Run tests; expect PASS. Commit.

### Task 5: Identity extraction and parser budget spike

**Files:** create `internal/identity/pdf.go`, `docx.go`, `xlsx.go`, `identity.go`,
`identity_test.go`, `identity_bench_test.go`, `testdata/identity/manifest.json`.

1. Assemble small fixtures covering rotated title blocks, empty/scanned PDF,
   malformed ZIP, large shared strings, merged spreadsheet cells and DOCX tables.
   Test text/provenance extraction before field judgment; expect failure.
2. Measure warm PDFium identity-page extraction before choosing its Go binding.
   [go-pdfium](https://github.com/klippa-app/go-pdfium) offers implementation
   choices; investigate its embedded WASM path to preserve one deployable Go
   executable. Confirm licensing, startup memory and cold/warm timing. If it
   cannot meet budget, document the measured reason before changing packaging.
3. Implement DOCX/XLSX extraction with bounded `archive/zip` and `encoding/xml`
   readers; respect decompressed limits, cancel checks, sheet/shared-string
   references and identity-region limits. Never evaluate formulas or execute
   macros. Cached cell values may be evidence, with their provenance.
4. Run `go test ./internal/identity` and
   `go test ./internal/identity -run '^$' -bench Identity -count=5`.
   Expect correct identity text and the 80/250 ms gate on the target benchmark
   host. Commit dependency choice and measured justification.

### Task 6: Candidate harvesting and deterministic identity

**Files:** create `internal/intake/candidates.go`, `revision.go`, `rules.go`,
`candidates_test.go`, `revision_test.go`, `data/intake/kinds.json`,
`data/intake/disciplines.json`, `data/intake/lifecycle.json`.

1. Write cases for multiple numbers, title-block versus filename disagreement,
   missing revisions, date-like numbers, `P10` versus `P2`, and incompatible
   revision series. Expect tests to fail before implementation.
2. Copy the 57 discipline identities/aliases as data with source hashes. Define
   the agreed 16 kinds and lifecycle labels from available data; reconcile a
   missing kind list explicitly before calibrating rather than inventing labels
   during extraction. Keep consultant disciplines a filing view.
3. Harvest candidates with byte/page/cell provenance; preserve literal display
   values and separate normalized comparison values. Settle fields only under
   explicit unambiguous rules. Never guess a revision or compare unrelated
   series as a total order.
4. Run `go test ./internal/intake -run 'Candidate|Revision|Rule'` and harvester
   benchmarks; expect PASS and ≤5/10 ms. Commit.

### Task 7: Bounded Jev client and response validation

**Files:** create `internal/jev/client.go`, `types.go`, `admission.go`,
`client_test.go`, `testdata/jev/` recorded responses.

1. Use `httptest` for success, invalid choices, wrong model/type, missing answers,
   429, 529, slow headers/body and queue saturation. Assert one HTTP attempt and
   one 1.2 s deadline spanning admission and transport.
2. Implement a reused `http.Transport`, warm HTTP/2 connection, bounded response
   body, context cancellation and circuit breaker. Start with 24 total in-flight
   slots; reserve interactive capacity and prevent background admission from
   consuming it. Measure and configure the reservation; do not promise 24 calls
   can all meet the latency gate merely because slots exist.
3. Return partial validated answers without inventing missing ones. Emit latency,
   tokens, question/model version and numerical uncertainty to structured logs;
   never log credentials or whole document text. Background retries use backoff;
   interactive requests have no retry.
4. Run `go test ./internal/jev -count=1`; expect PASS. The live credential test is
   separate and opt-in. Commit.

### Task 8: Filing transaction, bands and supersession

**Files:** create `internal/intake/file.go`, `questions.go`, `decisions.go`,
`file_test.go`, `internal/store/decisions.sql`, `data/intake/thresholds.json`.

The implementation boundary is deliberately small:

```go
type Candidate struct {
    ID, Value, Source string
}
type Decision struct {
    Field, Value, Band, DecidedBy, QuestionVersion string
    Confidence *float64
}
```

1. Write end-to-end service tests with recorded Jev responses: all-rule filing
   sends zero requests; ambiguous filing sends one; deadline returns grey with
   rule values; blank does not become an invented value; correction wins over
   a delayed response; supersession stays within org/project/number series.
2. Build one request from identity state and unresolved fields. Apply rules and
   calibrated bands, then atomically store decisions, document status and
   background jobs. Use optimistic version checks to protect user decisions.
3. Supersession is a link to a prior immutable document, not a file overwrite.
   Reject self/cyclic links and conflicting candidate numbers. Only apply the
   strict calibrated band and compatible code revision ordering; otherwise
   flag the candidate for review without hiding either version.
4. Run `go test ./internal/intake ./internal/store`; expect PASS. Commit.

### Task 9: Durable events and background fire pilot

**Files:** create `internal/events/sse.go`, `sse_test.go`,
`internal/jobs/worker.go`, `worker_test.go`, `internal/knowledge/load.go`,
`load_test.go`, `evaluate.go`, `evaluate_test.go`.

1. Test reconnect after commit-before-send crash, org-isolated event IDs,
   duplicate job delivery, expired leases and interactive priority under backlog.
   Test missing tables, unreviewed/pending derivations, mixed-class scopes and
   contradictory determinant evidence returning unknown.
2. Use a durable event cursor (a decision/document sequence or outbox), so SSE
   delivery is recoverable and at least once; clients de-duplicate. Slow clients
   cannot block commits. Lease jobs with transactional skip-locked selection,
   bounded retries and explicit failure status.
3. Background stages are full text/passages/index, system labeling, then scoped
   evidence questions. Top-system nouls and speculative leaf choices share one
   fan-out per labeling state; evidence questions share one fan-out per labeled
   passage state. Different background states do not add foreground round trips.
4. Load the building hierarchy and canonical interfaces; compute graph edges in
   code. Preserve scope/provenance for determinant conflicts. Draft pilot output
   is evidence coverage, never a claim the building complies. Run
   `go test ./internal/events ./internal/jobs ./internal/knowledge`; expect PASS.
   Commit and justify the YAML parser if required by the Go loader.

### Task 10: React drop zone and corrections

**Files:** create `web/package.json`, `web/src/App.tsx`, `web/src/Project.tsx`,
`web/src/DocumentRow.tsx`, `web/src/api.ts`, `web/tests/intake.spec.ts`,
`internal/httpapi/server.go`, `web/embed.go`.

1. Write a Playwright flow: authenticate an invited test user, create a project,
   drop each supported format, observe chips via SSE, correct a field, reconnect
   and verify the correction remains. Assert no access from another org.
2. Build a chosen-project drop zone, visible upload/filing progress, readable
   green/amber/blank/grey labels and stored-but-not-filed state. Colour must have
   text/icon meaning. A grey response must not look like completed AI judgment.
3. Embed the production Vite output in the Go binary. Keep API/event errors
   recoverable without re-uploading an already stored file. Handle keyboard
   access and focus after updates. Use the applicable frontend design skill at
   implementation time; avoid an unsolicited dashboard redesign.
4. Run `npm --prefix web run build` and `npm --prefix web run test:e2e`; expect
   PASS against the Go server with recorded Jev and the test DB. Commit.

### Task 11: Accuracy calibration and build-failing latency evaluation

**Files:** create `cmd/intake-eval/main.go`, `cmd/intake-bench/main.go`,
`data/eval/intake/manifest.json`, `internal/eval/metrics.go`,
`internal/eval/metrics_test.go`, `.github/workflows/check.yml` (if hosted on GitHub).

1. Validate source-file hashes and answer-key quality. Partition train/calibration
   and held-out evaluation by project/file family to prevent near-duplicate
   revisions leaking across splits. Exclude null/unscored fields and separate
   unreviewed/model-generated labels from authoritative ones.
2. Record exact Jev requests/responses, pinned model, question versions and option
   shapes. Measure per-field accuracy, coverage, amber rate and false confident
   actions. Fit thresholds on calibration only; preserve them as data. Set no
   universal confidence cut-off. Require no incorrect automatic supersession
   in the held-out gate; disclose sample size and uncertainty.
3. Run `go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay`
   and `go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json
   -budgets bench/budgets.json`. Expect nonzero on missing corpus, too few
   samples, field regression or budget breach. Include target hardware,
   concurrency, warm/cold mix, background load and degraded cases in results.
4. Separate deterministic replay CI from nightly live Jev measurement. A mock
   proves behavior, not the provider's real latency. Run the live gate on the
   intended VPS before release. Commit scripts, fixtures and results metadata;
   keep private corpus documents out of public Git history.

### Task 12: Host packaging, health and restore rehearsal

**Files:** create `deploy/sitewise.service`, `deploy/Caddyfile`,
`deploy/backup.sh`, `docs/operations.md`, `internal/httpapi/health.go`,
`health_test.go`, `internal/httpapi/speed.go`.

1. Test health for DB failure, stale Jev probe, backlog age and degraded circuit.
   The speed endpoint uses recorded histograms and must not make an interactive
   Jev call. Test missing secrets preventing startup.
2. Package the binary and static content; configure local PostgreSQL 17, disk
   directories and least-privilege service user. Keep public health minimal;
   expose detailed queue/speed status only to authorized users.
3. Write clean-VPS deployment steps, off-site continuous PostgreSQL backup and
   nightly content-hash file backup. Rehearse restoration to an isolated host,
   checking document counts, blob hashes and tenant boundaries before invitations.
4. Run the full gates and produce the exact deployment/change checklist. The
   implementation plan does not itself execute the irreversible VPS rebuild.
   Carry over only authorized secrets/keys when deployment is undertaken.
   Commit operational artifacts and restore evidence.

## Done means

The supported corpus files reach a durable visible state within the intake
gate; uncertainty is explicit; correction and supersession preserve history;
all tenant access paths pass isolation tests; interruption/reconnect/retry is
safe; one Jev call serves each intake state; the live VPS measurement and
restore rehearsal pass before an invite is issued. No benchmark of the old
application and no Supabase backup is part of this work.

This document is a plan. Apart from the local Go installation and knowledge
validation, these implementation tasks and tests have not yet been run.
