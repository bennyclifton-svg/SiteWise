# Bankstown CC-11 Retry OCR delivery delay

Lane A, focused defect: an explicit OCR retry appeared to take 10–30 seconds.

The original document's durable event times on 3 October 2026 show:

| Stage | Sydney time | Elapsed |
|---|---|---|
| Queued | 20:56:19.142 | — |
| Reading | 20:56:20.162 | 1.021 s queue wait |
| Classifying | 20:56:23.899 | 3.737 s render/OCR |
| Filed | 20:56:24.148 | 0.249 s classification/commit |

Server completion was 5.006 seconds after queueing. Three read-only replays
of the same content-hash PDF took 3.850, 3.768 and 3.821 seconds for OCR.
These are diagnostic samples, not p50/p90 gate evidence.

The earlier 625/723 ms feasibility trial used two manually selected crops
and a different segmentation mode on CC-01 through CC-05. The deployed
reader reads generic perimeter bands. Its complete retry path additionally
includes queueing, classification, saving and browser delivery.

The notification defect is reproducible: background RunOCR commits progress
and filing events without waking the HTTP event broker. The retry endpoint
wakes it only when queueing. The browser's polling fallback was restricted
to already-filed detail recovery, excluding pending initial OCR and retries.
A connected browser could remain on Queued until an unrelated wake or reload.

The fix wires the OCR runner to the same org-scoped event broker, waking it
after committed reading/classifying states and on completion/failure. The
browser fallback now checks every active OCR document every two seconds,
stopping at terminal state. No new OCR pass, model call, confidence policy,
dependency, or change to extracted fields is introduced.

Verification: the browser regression failed before the fix (server state
complete, browser still Queued) and passed after. All eight browser tests,
intake/HTTP API/command tests and web build pass. Callback tests prove the
committed progress and terminal states are readable when notified, and that
wrong-org and already-complete calls do not notify. Existing correction and
review-only tests pass. Normal filing retains its 1000/2000 ms budget;
background OCR retains 10000/20000 ms. Provider failures still produce the
existing review/failure state; missing notifications have a bounded polling
fallback while the document is active.

The installed OCR gate passed after the change: 20 upload filings p50/p90 3183/3201 ms and 20 detail recoveries 3182/3189 ms, including 2000 ms polling allowance and replayed 350 ms Jev latency. These public-fixture timings do not replace the actual CC-11 diagnostic above.
