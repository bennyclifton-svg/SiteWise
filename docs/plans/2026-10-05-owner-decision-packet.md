# Owner decision packet: what unblocks the next wave

> **Current progress:** [NEXT-WAVE-STATUS.md](NEXT-WAVE-STATUS.md). On 7 October the owner authorized completing M2/M3 unattended; earlier sequencing/permission holds are historical. Acceptance evidence remains explicit.

**Historical decision packet.** Answers below are retained verbatim as evidence. Its “waiting”, “blocked” and gate descriptions are historical, not current instructions. The implementation plan §5 is the current decision table and §0.2 the current evidence summary. The later review amendment supersedes the green-band choice (D-32), services-only design allocation (D-33) and original sequence (D-31). Unedited recommendations are planning defaults, not proof of owner approval. This revision authorises no implementation.

For: Benny, on return. Written 5 October 2026 during the first unattended run. Full context for each decision is in `docs/plans/2026-10-04-next-wave-implementation-plan.md` §5.

**Where things stand:**

- **Done and merged:** WP-00 (profile replay gate restored).
- **Ready to merge after review:** WP-11 (sites; parts belong to the site) and WP-25 (works knowledge loaded; `works` and `system_existing` predicates evaluated in code).
- **Waiting on you:** almost everything after that.

Answer by writing **A**, **B** or your own words after each "Your answer". I recommend the first option in every case.

## 1. D-04: where building facts live (blocks WP-12, WP-15, then all of Stage 2)

A site outlasts its projects, so facts about the building should key to the site. Facts about the works stay with the project.

- **A (recommended).** Keep one user-values table. Each key is classified as site or project by a small registry file.
  - Site: building class, subclass and scale, `det.*`, existing-system condition.
  - Project: work type, conditions, `fact.*`, system presence, provider, notes and actions, scope.
- **B.** Separate site-value tables.

Also: `existing_building` ("Work to an existing building") describes the works, not the building. Should it be site or project?

Your answer:A

## 2. D-06: can a value you typed feed a regulatory derivation? (blocks WP-15)

Today, any value you enter can feed derivations such as Type of Construction, and the result shows green.

- **A (recommended).**
  - What you state as fact still feeds derivations, and they keep their current green colour, labelled "accepted for planning", not "verified".
  - Values you mark as an **assumption** or allowance never feed a derivation.
- **B.** Only verified inputs feed derivations. Most derived rows would go blank today.

Your answer:A

## 3. D-05: where an existing system's condition lives (blocks WP-20, which is all of Stage 2)

The PRG puts "existing condition" on the work item and also among site facts.

- **A (recommended).** On the site (`sys.<leaf>.condition`). The work item shows it, plus a project note. One record, and it survives to the next project.
- **B.** A copy on each work item.

Your answer:A

## 4. D-07: mixed jobs like Hale (extension plus refurbishment) (blocks WP-21)

- **A (recommended).** Keep one project work type for the header and answer keys. Let a part carry its own work type, for example the extension part `new` and the existing building `refurb`; default actions follow the part. The K4 records that test `work_type` start to work.
- **B.** Make work type multi-choice. This changes a Jev question and the answer keys.

Your answer:A

## 5. D-03: order of Stage 2 and Stage 3 (blocks WP-32 and WP-35)

The PRG asks for the package gap check in Stage 2, but packages arrive in Stage 3. First reports need dates, risks and approvals before Stage 6.

- **A (recommended).** Move the gap check to Stage 3, and build a minimal dates, risks and approvals register in Stage 3. Until then, package-type proposals can be dismissed but not accepted.
- **B.** Build a skeleton packages table in Stage 2.

Your answer:A

## 6. D-09: who must do what, per action (blocks WP-32)

**A (recommended)**, the rules:

- Physical work (new, replace, upgrade, alter, repair, remove) needs exactly one works package that installs or carries it out. "Supply and install" means the same package also supplies. An owner supply package alone is not an installer.
- Design is needed for new, replace, upgrade, alter and repair, from exactly one consultant package.
- Investigate needs one package that inspects or tests.
- Retain needs a works package that keeps the system operating or protects it.
- Groups inherit from their parent unless a child sets its own.

Your answer:A

## 7. Other open questions, one line each

| Decision | Question | My recommendation |
| - | - | - |
| D-02 | Is the primary user the owner-side PM who appoints consultants and contractors? | No, primary users are owner-side PM and D&C Contractor, keeping targer user audience larger  |
| D-13 | Forecast for the PMP variance | Computed: commitment, else estimate, else budget |
| D-15 / D-16 | PDF renderer; minimum font size | A one-day renderer spike; A4, 10 pt body, 8.5 pt minimum in tables |
| D-18 | Budget rule for large projects | Keep 50/150 ms on the bench project; large projects are governed by `profile_rebuild` 100/300 ms |
| D-22 | Copy Clerk's consultant rosters as draft default packages? | Yes, draft |
| D-23 | Who drafts the 0991 and 0777 work-item keys, and does "by ID only" allow the path and hash? | An agent drafts and you review; ID, path and hash |
| D-24 | Unforeseen conditions may attach to a stage or package kind, as K4 already does | Accept |
| D-29 | Tenant fit-out should raise a base-building capacity check | Re-type the fit-out edge as `supplies`, with a reviewed label |
| D-30 | "Test the control link" proposal should be a test obligation, not an investigate item | Yes |
| K0 | Approve the works-layer record shapes (and the 560 K4 records written on them) as the format | Approve the format; content stays draft |

## 8. Gates that need you, not code

- **Filing accuracy baseline (F28).** The intake gate stops until you accept a baseline (`go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -live -accept` after review). Held-out title accuracy in the latest run is 0.29, with 15 titles filled wrongly with high confidence. You may prefer to fix titles before accepting.you choose. 
- **Latency bench (F29).** 52 requests, probably the scanned no-text documents, have no recording, so the bench can't run locally. Re-record the OCR path, or exclude those documents from the bench? you choose. 
