# Profile reading feedback

12 October 2026. Lane A: truthful document-reading status.

The register tick and `read_documents` describe inclusion policy, not completed
reading. Changing that setting deliberately does not enqueue Jev work; Update
project profile remains the explicit request. The previous permanent “Reading
N of M” label made an idle or finished project look busy.

The API now returns `completed_documents`, scoped to the org and project and
counting included documents only. Both label and evidence jobs must be done,
and no extraction, label or evidence job may remain unfinished. Failed rereads
cannot count as complete. The UI shows inclusion separately from a native
progress bar and completed-document count, with queued, active, ready, failed
and complete states. Completion is not a claim that every passage was resolved;
source review and threshold notices remain visible. Existing SSE and pending
polling deliver progress without another service or model call.

The reported blank title was present among harvested cover candidates. Its
stored Jev confidence was 0.58, but `data/intake/thresholds.json` has an empty
title calibration list: no Jev title is accepted automatically. No title,
confidence threshold or extraction rule was changed. Blank Jev title decisions
now say “Title needs review” and explain how to enter the title. This preserves
[TypeSafe’s pre-parsed extraction pattern](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook):
code finds literal candidates and Jev picks; an unapproved pick is not promoted.

Validation:
- Store regression covers selected/text-ready, queued evidence, completed,
  failed reread, skipped completed documents and wrong org.
- Four browser regressions pass: failure/retry, missed completion SSE, bulk
  inclusion and exclusion, and multi-document progress through completion.
- Desktop and 390px mobile checked; no horizontal overflow.
- Frontend build and intake tests pass; no new dependencies.
- Live local profile GET, 40 samples: before p50 9.70 ms / p90 10.95 ms;
  after p50 11.21 ms / p90 12.74 ms. Budget: 50 / 150 ms.
- Development replay speed benchmark (20 files, two rounds): whole filing
  p50 598.9 ms / p90 1380.7 ms against 1000 / 2000 ms; profile GET
  p50 1.3 ms / p90 7.8 ms against 50 / 150 ms. Both pass. These are local
  replay timings, not live-provider or release evidence. Cold-round filing p90
  was 2222.1 ms; aggregate is the existing gate.
- The full benchmark exits 1: Spec Home profile edit p50 56.6 ms exceeds
  50 ms (p90 61.4 ms passes). Extraction and deterministic-rule component
  overruns were also reported, as target-VPS-only gates. No clean full speed
  gate is claimed; this task does not establish whether the edit overrun is
  pre-existing.
- The full store and intake packages pass. The full HTTP API package fails
  TestProfilePlacesInputsInBuildOrder on unassigned field homes. Running that
  test with the unchanged HEAD versions of both modified Go implementation
  files reproduces the failure, confirming it predates this fix.
- Reloaded the local dev server and verified the reported project shows one
  completed document, zero unread/pending/active/failed documents, and the new
  title-review message. Existing open browser pages need a refresh.
