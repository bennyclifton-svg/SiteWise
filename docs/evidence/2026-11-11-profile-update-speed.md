# Profile update speed: evidence

Date: 11 November 2026. Owner question: is "Update project profile" slower
since the run-time feedback work (`docs/evidence/2026-11-10-run-time-feedback.md`),
and can it be made faster? Lane A (the Jev client is shared with filing).
Not committed: the shared tree holds other sessions' uncommitted work in
`internal/jobs/worker.go` and `cmd/sitewise/main.go`.

## Answer

The feedback did not slow updates. The same documents read about 16% faster
on 11 Nov than on 10 Nov (before the feedback build was live). The update
felt slower because it read more: two more documents, and every passage was
asked again after the 11 Nov question change. The time goes to Jev calls, and
on the dev machine those were capped by its upload speed. Compressing request
bodies and reading sixteen passages at a time instead of eight makes a read
2.3 times faster on the dev machine, with filing's calls unchanged.

## 1. Was it the feedback?

House Class 1a, same documents, read on 10 Nov at 18:20 (old page; this
session's build went live at 22:43) and on 11 Nov at 15:32 (new page). Job
times from `events` (each job ends when the next begins; one job runs at a
time):

| Document | Passages | Label 10 Nov | Label 11 Nov | Evidence 10 Nov | Evidence 11 Nov |
|---|---|---|---|---|---|
| Structural Plan (set) | 673 | 31.6 s | 27.1 s | 29.7 s | 22.3 s |
| Sheet 2 | 214 | 10.6 s | 8.8 s | 12.6 s | 12.8 s |
| Sheet 5 | 136 | 6.4 s | 5.8 s | 5.2 s | 3.3 s |
| Sheet 4 | 116 | 6.0 s | 5.1 s | 4.0 s | 3.4 s |
| Sheet 3 | 106 | 5.3 s | 4.7 s | 4.3 s | 4.1 s |
| Sheet 6 | 71 | 4.0 s | 3.2 s | 3.5 s | 2.6 s |

Both passes of these six documents: 123.3 s on 10 Nov, 103.1 s on 11 Nov.
The 11 Nov update took 2 min 52 s against 2 min 03 s (first job finished to
last) because it also read Stamped Architectural Plans (927 passages,
67.5 s) and Sheet 26 (1.4 s).
During it the worker kept eight calls in flight 83% of the time with no idle
gap over 0.5 s, so nothing the page did held it back. The page's reads on
this project: profile 18 ms (417 KB), progress 2 ms, finds 1–4 ms, next 9 ms.

Why every passage was asked again: the 11 Nov Accommodation work moved
`QuestionVersion` to profile-15 and `ReadRevision` to read-15
(`internal/profile/questions.go`). The call cache key is a hash of the whole
call, version included (`passageFingerprint`, `internal/jobs/source_reading.go`),
so a version change re-asks every passage, including those whose questions
did not change. One update made 3,874 Jev calls for nine documents.

## 2. Where the time goes

Jev itself answered a label call in 0.31 s (p50) at eight in flight. The
read rate is calls in flight over latency, so the first test was more calls
in flight.

| Test (live) | Calls/s | Label p50 | Filing's own calls during the read |
|---|---|---|---|
| 8 in flight, plain JSON (before) | 24.9 | 0.31 s | not measured |
| 16 in flight, plain JSON | 24.9 | 0.62 s | 545–836 ms |
| 16 in flight, gzip | 60.1 | 0.25 s | 198–371 ms (p50 245 ms) |

Filing's calls with nothing else running took 215–300 ms.

Sixteen plain calls doubled each call's time and read no faster, and
filing's calls queued behind them. A throwaway probe (`-tags liveprobe`,
since deleted) found why:

- Same throughput over HTTP/2 (one connection) and HTTP/1.1 (one per call):
  22.7 vs 19.5 calls/s at 8, 23.0 vs 19.0 at 16. Not the connection.
- Small calls (~1,140 tokens) scaled: 37.7 calls/s at 8, 69.2 at 16, latency
  flat at ~200 ms. Not a per-key call limit.
- This machine uploads 597–601 KB/s (Cloudflare speed test; downloads
  3.3 MB/s). A real label call is ~23.4 KB of JSON, and 25 a second is the
  whole upload.

So on the dev machine a profile read was bound by its upload speed. The VPS
(sitewise.au, Singapore) has a data-centre link; its speed was not measured.

## 3. What changed

- **Compressed requests.** `jev.Options.CompressRequests` gzips request bodies
  of 1 KiB or more at the fastest level and sets `Content-Encoding: gzip`.
  TypeSafe accepts it (https://docs.typesafe.ai/api/errors.md, Compression:
  gzip, x-gzip or zstd; an unsupported encoding returns 415, a malformed body
  400). Only the server turns it on (`cmd/sitewise/main.go`); recording and
  replay tools hash the plain body, and they keep sending it. The Structural
  Plan's 673 real label calls: 23.4 KB to 6.5 KB each at p50 (3.6x). Each
  call is compressed once; a background retry resends the same bytes.
- **Sixteen passages at a time.** `passageWorkers` in `internal/jobs/worker.go`
  is the Jev client's background allowance (`TotalSlots - InteractiveReserve`,
  24 - 8). Filing keeps its eight reserved slots. Specification import uses
  the same pool.

What calls what: nothing new. The same calls, the same answers, fewer bytes.
No new dependency (`compress/gzip` is in the standard library).

What could fail: if TypeSafe stopped accepting gzip, every compressed call
would fail with 415 and filing would show grey; the circuit breaker opens and
background reads retry with backoff. Turning the option off restores plain
bodies. On a slower upload than this one, sixteen compressed calls fill more
of the link and filing's calls would queue sooner.

## 4. Result

Scratch copy of the Structural Plan set (673 passages, fresh passages so
every call was a real Jev call), project "Speed check 16 (scratch)":

| | Before (11 Nov 15:32) | After |
|---|---|---|
| Label pass | 27.1 s | 11.3 s |
| Evidence pass | 22.3 s | 7.4 s |
| Read, request to done | 49.4 s | 21.0 s |

Scaled the same way, House Class 1a's 2 min 52 s update would take about
75 s. Filing calls made during the read took 198–371 ms.

Tests: `TestCompressedClientSendsLargeBodiesAsGzip`,
`TestSmallBodiesAndPlainClientsStayUncompressed` (internal/jev),
`TestEachPassageFillsTheBackgroundAllowance` (internal/jobs). Full Go suite:
31 packages pass. Failing before this change: `TestProgramRoutes`,
`TestCommittedBudgets`, `TestMeridianAnswerKey`. `TestCompute2000Budget` and
`TestProfileFindsReadsRecentPassagesOfOneProject` missed their timing budgets
while the whole suite loaded the machine, and pass run alone. One
`cmd/intake-bench` run failed on a program-import upload (500) and passed on
rerun; it replays plain bodies, so it does not exercise this change.

## 5. Not changed (open, for the owner)

- **A split drawing set is read twice.** Setting a set to read also sets its
  sheets (`SetProfileReading`), and `readableSQL` still reads the set itself
  (status `split`). House Class 1a's Structural Plan was read as the set (673
  passages) and again as its six sheets (679 passages): 49 s of the 172 s
  update. The set gave 1,303 readings and its sheets 1,201; 106 of the same
  readings came from both, so those values count one page twice. Reading
  the sheets and not the set would remove both costs. This changes what the
  profile reads, so it is a lane A decision.
- **A question change re-asks everything.** Keying the call cache on the
  questions and the passage, not the version label, would re-ask only
  passages whose questions changed. Changing the key needs a one-time
  compatible lookup, or the change itself forces one more full re-read.
- **Drawings are many small passages.** The two drawing sets were 1,600 of
  the 2,297 passages the 11 Nov update read; each is one full label call of
  ~6,600 tokens.
