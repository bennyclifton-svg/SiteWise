# WP-X1: actor attribution belongs to the organisation

Lane C. Outcome: a planning or work write cannot attribute an authoritative
record to another organisation's user, and database writes cannot bypass the
actor relationship. This protects answer trust. Scope is actor attribution;
no verification UI, membership deletion workflow or additional service.
Existing CRUD budgets stay 100/250 ms and profile edit stays 50/150 ms.

The semantic review found five actor columns absent from the foreign-key
inventory: profile value author/verifier, planning value author, and work
verifier/retirement actor. Other new-wave package, delivery, decision and report
actor columns already have composite constraints. The earlier sweep correctly
proved existing constraints, but could not prove these missing relationships.

The store-level reproduction accepted foreign and missing users through
`SetUserValue`, `SetPlanningValue`, `CreateWorkItem` and the scope picker.
HTTP handlers obtain actor IDs from the authenticated user; this finding does
not demonstrate that an HTTP caller could spoof that identity.

Migration `020a_actor_scope.sql` adds five `(org_id, actor_column)` foreign
keys to `users(org_id,id)`. They preserve audit references, reject independent
deletion of a referenced user, and defer checks so whole-organisation deletion
can remove both sides atomically. Existing invalid attribution fails migration
rather than being silently reassigned or deleted. Calculated planning values
may retain a null human author.

The four store write paths now reuse the existing tenant-scoped actor check
before mutation. Empty human actors return not-found. Any supplied calculation
author must also belong to the organisation. This adds one local existence
query on these writes; no Jev call or dependency is added. Work provenance
retains its historical JSON actor snapshot, with the live actor validated
before writing it.

The regression matrix covers site and project values, planning, work creation
and scope changes with foreign, missing and empty actors. Rejections preserve
values, history, work, profile rows/builds, revisions and events; valid actors
still succeed. The foreign-key inventory explicitly grows from 51 to 56 and
its database sweep exercises each new constraint. Anonymous calculations and
audit-reference deletion semantics have separate tests.

Validation on 6 October 2026: the complete serial Go suite passes
(`.tools/wpx1-actor-scope-all-tests.log`). It includes 18 actor rejection cases,
valid-actor controls, anonymous calculation, actor retention, whole-org
deletion and the pinned 56-constraint database sweep. `git diff --check` passes.

The unchanged local replay workload is saved as
`bench/results/2026-10-06-actor-scope-diagnostic.json`. Work writes measured
p50/p90 6.2/6.7 ms against 100/250 ms; profile rebuild 28.2/29.4 ms against
100/300 ms; filing 477.7/637.4 ms against 1000/2000 ms. The full gate still
fails: Spec Home edits measured 50.200/51.7 ms against 50/150 ms. This remains
a failure despite the small margin; run variation is not proof of a speed
improvement. Local component overruns and target-VPS checks remain recorded.

This closes the actor gap, not the whole semantic relationship review or
WP-X1; M2 issue recovery and other milestone/release gates remain outstanding.
