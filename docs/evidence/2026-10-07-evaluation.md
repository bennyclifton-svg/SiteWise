# Evaluation tooling and corpus preparation — 7 October 2026

Lane A evidence tooling, with a Lane C Windows runner repair. No production
service or dependency added. Filing remains subject to 1,000/2,000 ms p50/p90;
this work changes no production request path and makes no new latency claim.

## Runner repair

`tools/check.ps1` selects `npm.cmd` on Windows and puts TEMP/TMP beneath the
ignored workspace `tmp` directory. This avoids the recorded npm.ps1 execution
failure and external-temp permission failures without skipping any gate.

Playwright builds and runs an owned e2e binary. Building alone did not resolve
the Windows teardown hang: all 22 existing cases passed their assertions, but
the server had to be stopped manually. A subsequent fix adds a test-only
shutdown endpoint, restricts the entire test server to loopback, and calls the
endpoint from global teardown before Playwright attempts Windows process-tree
cleanup. This route exists only in the test binary. The focused saved-report
browser scenario then passed and exited automatically: **1 passed, 11.2 s**.
The full suite still needs its final integrated run after concurrent changes.
`npm.cmd --prefix web run build` passed at the initial tooling checkpoint.

## Source/Hale recordings and workload

Live refresh was attempted with the key loaded from `.env` without printing it.
The sandbox denied the network connection. Automatic approval review then
rejected the escalated source/Hale run: it considered the general Jev-call
authorization insufficiently explicit for these private excerpts to TypeSafe.
No workaround or alternate transmission was attempted; recordings remain
stale and cannot establish current accuracy. Parent was given the exact
rejection and command. Replaying old questions is not a replacement.

The tool follows [TypeSafe API](https://docs.typesafe.ai/api),
[fan-out](https://docs.typesafe.ai/patterns/fan-out),
[confidence](https://docs.typesafe.ai/confidence), and
[Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).
Model remains pinned to `jev-1.13.0`. No private data was sent after rejection.

`profile-eval -measure` now reports every request rather than aborting on the
first oversized request, and still exits unsuccessfully if any exceed the
existing budget. It uses recorded labels to construct current calls locally;
it does not make calls, create answers, or claim accuracy. With the initial
WP-27 signal fan-out, three requests fail the unchanged preflight:

| Case | Questions | Request bytes | Estimated tokens |
| --- | ---: | ---: | ---: |
| source / bankstown-page9 | 282 | 253,743 | 63,436 |
| Hale / hale-door | 296 | 251,973 | 62,994 |
| Hale / hale-rainwater | 335 | 245,983 | 61,496 |

Commands: `go run ./cmd/profile-eval -measure` and the same with
`-cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json`.
Both exit 1. Text-free detailed counts are in ignored `tmp/source-measure.txt`
and `tmp/hale-measure.txt`. These are preflight estimates, not provider tokens,
latency, or live cache measurements. No request bound was relaxed. The
four-bytes-per-token estimator remains an empirical English-fixture estimate,
not a proven upper bound for arbitrary inputs. This limitation was escalated
to the implementation lead and WP-27 owner.

**Superseding measurement after WP-27 routing correction:** signals now follow
the accepted passage labels and their ancestors only, not speculative
unlabelled leaf questions. Both commands exit 0. Source: 14 requests, 876
questions total, maximum 224 questions / 232,858 bytes. Hale: 22 requests,
1,605 questions total, maximum 227 questions / 224,918 bytes. Zero requests
exceed the unchanged preflight. The ignored logs now contain this later run.
This resolves the initial signal-expansion size failure, not stale accuracy
recordings or the estimator's general upper-bound limitation.

## Petersham private review packet

Prepared ignored `data/eval/profile/private/petersham/manifest.json`, per-page
local extraction files, and `draft-expectations.json`. The bounded selection
pins **13 PDFs / 421 pages**: PPR, electrical specification, mechanical
design/specification, Section J, discipline design certificates, waterproofing,
primary fire engineering and access reports. Every original file was rehashed
after preparation. Source documents remain outside the repository. Neither
extracted text nor the private review packet is publishable.

The packet has seven preliminary work expectations, eight candidate critical
obligations, and three cross-document conflict cases. Every expectation stays
draft/unreviewed; part labels need canonical reconciliation. This is a review
packet, not a complete extraction key or actual system output. It cannot pass
the scored gate. Certificates are statements about a specified design and
revision, never independent confirmation of construction compliance. Code
references remain unverified without the instrument itself.

Useful trust cases include: design scope excluded by one consultant while
still required by the PPR; a Section J report limited to one part and building
fabric; inconsistent building particulars; and differing preliminary versus
later NCC design bases. No generic BCA report was identified conclusively;
the PPR references one, so its existence is not silently assumed or denied.

## 0777 source identity

The `0777-city-tower-fitout` folder is accessible. Its README is a template and
its PMP is an agent-authored draft, not an owner-reviewed source. Previous
visual inspection recorded that its CC-04 drawing is residential rather than
the commercial fit-out described by that PMP (WP-28 evidence, 5 October).
Do not infer a commercial-fitout gold key from that design set or treat the
draft PMP as authoritative proof. This blocks project-specific scored
acceptance, not generic implementation of RFT/PMP behavior.

## Bounded Petersham current-model attempt

Five new cases were prepared from the explicitly supplied Petersham PPR corpus:
PPR condenser requirements, a mechanical designer's scope exclusion,
carpark-ventilation control requirements, electrical contractual scope, and the
limited Section J assessment boundary. The payload totals 1,372 characters;
contact details, individual names and addresses were omitted. Exact source-PDF
and excerpt hashes remain in ignored `private/petersham/bounded-case-provenance.json`.
The two positive readings and one forbidden reading are draft assertions only;
this exploratory set is not an owner-reviewed accuracy key or a replacement
for source/Hale regression recordings.

A specific request to transmit those five excerpts to TypeSafe was rejected by
automatic approval review before execution. The reason was that supplying the
private corpus and authorizing Jev calls did not specifically authorize this
sensitive payload to that destination. No transfer, model output, or scoring
occurred. No workaround or retry was used. The full current-model Petersham
usability check therefore remains unverified, with local cases ready for an
explicitly approved run.


### Final K3/K4 catalogue preflight

After the final catalogue freeze (862 failure modes / 16 compliance questions),
source and Hale `-measure` both exit 0 with unchanged strict-labelled workload:
source 14 requests / 876 questions, largest 224 questions / 232,858 bytes;
Hale 22 requests / 1,605 questions, largest 227 questions / 224,918 bytes.
Largest individual-context estimates are respectively 1,330 and 1,180 tokens.
Ignored text-free logs: `tmp/source-measure-final.txt` and
`tmp/hale-measure-final.txt`. These use recorded labels to size current requests,
make no API calls and establish no new detector accuracy, accepted source/Hale
replay, actual provider token counts, or private Petersham usability claim.
