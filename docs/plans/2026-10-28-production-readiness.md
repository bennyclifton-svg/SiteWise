# Production readiness: review, decisions and implementation

Date: 28 October 2026. Scope: critique of the supplied agent review, followed
by implementation of substantiated cleanup. The assistant proposal is excluded
at the owner's explicit direction. This is not permission to replace the VPS
or delete existing production data.

## Outcome and boundaries

Make existing work reachable, legible and honest, including on narrow screens;
recover from load failures; provide a real way to end a session. This improves
time-to-decision and trust without changing filing or commercial decisions.
UI work is lane B; session termination is lane C. Existing money, accepted
values, issue history and Jev policy are lane A regression boundaries.
Preserve the extensive pre-existing uncommitted changes in this checkout.
No new dependency, service, model, invented metadata or pricing.

Budgets in milliseconds: filing 1000/2000 p50/p90; project documents 50/150;
auth 100/250; cost/procurement/specification reads and writes 100/250 unless
the named entry in `bench/budgets.json` is tighter; profile edits retain their
recorded budget. UI navigation and filtering are local and make no AI call.
Run the repository benchmark; distinguish replay measurements from live VPS
evidence. No budget is relaxed to make a test pass.

## Critique and disposition

The review is a useful usability inventory, not a production-readiness audit.
Screenshots are observations of one build, not proof of current behavior.
It mixes genuine defects, already-built protections and product expansion.

| Review item | Decision and reasoning |
| --- | --- |
| 1. Dev status | Partially valid. Server already limits `/dev/build` to dev-login. Fix client polling/error handling so an unconfirmed production outage cannot display developer instructions, and stop polling when absent. |
| 2. Composer / Chats | Accept removal. They imply functionality that does not exist and consume space. No replacement command surface. |
| 3. Document titles | Accept filename fallback with explicit review wording. Filename is display only, never an accepted document title; preserve source suggestions and acceptance. |
| 4. Project Plan | Already disabled in current code. Keep the product's navigation placeholder and label it as planned. |
| 5. Project list | Rows already have a large link and deletion already confirms. Add local name search, readable Open, and a separate actions disclosure. Do not invent metadata absent from the project API or add per-project fetches. |
| 6. Profile width | Accept pane-based container layout rather than a viewport-only breakpoint. Add non-colour selection and source-count help. Preserve provenance and editable drafts. |
| 7. Design authority | Accept. PRODUCT already specifies Instrument Sans and current cockpit. Update DESIGN from actual token mappings; no second palette or brand redesign. The named root `.impeccable/design.json` is absent. |
| 8. Mobile shell | Accept reachable navigation, project heading and bounded table scrolling. Keep existing editing available; a read-only mobile product is unnecessary scope. |
| 9. Assistant | Excluded by owner. |
| Home search / Recent tabs | Search accepted; Recent requires a defined real timestamp and is deferred. |
| New-project dialog / autocomplete | Defer. Existing short inline form is faster; autocomplete adds a service and a new workflow. |
| User/org/settings menu | Implement sign-out with server-side session revocation. Org switching/settings need defined permissions and workflows; do not ship empty destinations. |
| Switcher clipping | Accept bounded searchable disclosure and persistent All projects access, with Escape and keyboard access. |
| Header download/theme | Label existing download toolbar; align theme control size. Current toolbar already exists. |
| Header status chips | Defer API expansion and redundant status density. Never invent last-reading or completeness data. |
| Profile review banner/primary button | Keep current source-review and action placement. Do not collapse distinct uncertainty states or override the recent navigation design's intentionally equal controls. |
| Repository empty columns | Keep stable editable columns: missing is not necessarily not applicable. Do not infer n/a from a file extension. |
| Repository delete on hover | Reject hover-only access (touch/keyboard discoverability). Existing confirmation/bulk actions remain. |
| Filing detail empty-state/download styling | Small consistency changes only where they preserve provenance and recovery. No removal of meaningful uncertainty. |
| Cost sticky columns/scroll cue | Accepted after reproducing sticky Item cells intercepting Budget edits in a narrow pane. Release horizontal freezing below 700px pane width while preserving row-control positioning and header behavior; make scrollbars visible. No financial logic change. |
| Cost category/dash/quantity cosmetics | Defer broad restyling; preserve actionable links and known/unknown semantics. |
| Program empty-state illustration/completeness | Reject invented percentages/skeleton Gantt. Existing honest guidance is preferable to fake data. |
| Procurement upload labels/callouts | Existing workflow remains; defer redesign of every firm cell pending a focused interaction brief. |
| Procurement hover-only Create/Prices | Reject: important actions must remain discoverable and available on touch. |
| Specifications add-button repetition | Defer; adding to a group may be useful without opening it. Fix reachability/overflow first. |
| Typography/spacing/colour | Fix the touched controls' readability and neutral navigation selection. Do not globally enlarge every dense table or remove evidence colours. |
| Loading/errors/SSE | Fix confirmed silent project-load failure with retry. Existing progress components remain; no duplicate toast/progress architecture. Capture console/network failures in browser checks. |
| Accessibility | Visible skip-link focus, meaningful disclosure semantics, mobile keyboard navigation, non-colour choices, and readable touched controls are required. No claim of complete WCAG certification. |

## Implementation sequence

1. Remove misleading chrome; harden diagnostics detection; label placeholder
   and download action; fix switcher, mobile navigation and focus behavior.
2. Fix document-title fallback and pane-aware Profile layout/help/selection.
3. Add project search, safe actions disclosure, load-error retry and sign-out.
   Revoke only the authenticated session, require same-origin mutation,
   expire the HttpOnly cookie, retain other sessions; surface failures.
4. Reconcile visual documentation with runtime tokens.
5. Build/typecheck, frontend tests, Go correctness/security tests, browser
   regression suite, strict knowledge checks, replay accuracy and latency
   gates. Record exact failures and distinguish existing issues from changes.

Done means the accepted changes work with mouse and keyboard, 390px/mobile
and desktop layouts remain navigable, filenames never become accepted titles,
failed loads can retry, sign-out invalidates the session, and test evidence is
saved. Edge cases: long names, many projects, empty/no-match lists, network
failure, unknown diagnostics endpoint, narrow desktop panes, blank titles,
failed logout, cross-origin requests and another user's session.

## Deployment boundary

Existing `docs/operations.md` already records the production gates: clean VPS
configuration, live latency/accuracy and an off-site restore rehearsal before
inviting users. Local UI cleanup cannot prove these. Evaluate the available
build/tests and report remaining gates; do not describe the app as deployment
ready while required checks are missing or failing. No infrastructure rebuild
or production deployment is part of this cleanup.

## Results

Implementation and measured verification are recorded in
`docs/evidence/2026-10-28-production-readiness.md` after execution.

Additional defects substantiated during execution: live Profile refresh could
overwrite a typed draft; an absolutely positioned live region made the desktop
page thousands of pixels too tall; newly added appointment flags without a
layout mapping disappeared from Profile; background worker setup copied a
`sync.Once`. Fix these narrowly, with regression checks. Portability fixes
normalise CRLF in a YAML test fixture, make Python rendering dependencies lazy,
and bring the synthetic RFP fixture into agreement with its validator.

The Program import test isolates its empty-stage scenario from the newly
reviewed library. Browser assertions follow the explicitly removed composer
and collapsed mobile navigation. These changes must not relax accuracy gates,
change trusted answers, or conceal unrelated failing tests.

## Remaining release work discovered by verification

The accepted review cleanup is implemented. The application is **not yet
release-ready**. Follow the evidence document for measured results and exact
remaining failures; this section is pending, not represented as completed:

1. Reconcile Program's current generation/award rules with its route and
   browser fixtures. Preserve independent assertions for editing, linking,
   holds, import acceptance, printing and stage synchronisation. Review new
   screenshot baselines visually against the current shell before accepting.
2. Resolve the remaining procurement/award, proposal scope, saved-report and
   specification browser failures, distinguishing stale selectors from actual
   behavior defects. Run the complete browser suite green, with explicit
   justification for existing skipped coverage.
3. Refresh the private Jev recordings for current question versions and
   restore the missing proposal answer key. Resolve the Meridian wrong-green
   regression against its source evidence. Do not fabricate ground truth or
   reduce accuracy requirements.
4. Resolve `spec_home_profile_edit` p50 54.779ms against its 50ms budget and
   repeat timing in isolation. Obtain a complete passing local gate, then perform live VPS latency and
   accuracy checks and the operational restore rehearsal before deployment.
