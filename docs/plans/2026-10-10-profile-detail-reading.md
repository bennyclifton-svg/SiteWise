# Profile details read from documents

**Goal.** "Update project profile" fills everything on the Profile tab that a
document can settle, so the user mobilises a project with less typing. The
principle is unchanged: the user chooses which documents are read (the
read/skip toggle in the register); Update reads those; the user's own word
always wins.

**Lane A** (the RFP prints these values). One Jev fan-out per passage (the
existing label call); nothing is judged while a user waits; every row is
scoped to `org_id`. No new dependency.

## Why it was blank

The refactor that moved site address, lot and DP, client, LGA, zone and the
28 RFP flags into Profile made them user-entered only
(`knowledge/profile/project_details.yaml`: "no document reading fills these").
Before it, the RFP filled them itself. Petersham's PPR states the address,
the client and the LGA, so the profile should have shown them.

## Inventory of the Profile tab (10 October 2026)

| Area | Read by Jev before | After |
|---|---|---|
| Building class, subclass, work type, 8 project conditions | Yes (header-routed passages) | Same |
| Scale fields (storeys, GFA, site area, units, bedrooms, car parks) | Yes | Same |
| Compliance determinants (about 60) | Yes | Same |
| 7 contract and consent facts | Yes | Same |
| Scope of works (systems in, out, by others) | Yes (evidence fan-out) | Same |
| Site address, lot and DP | No | Yes (`site.address`, `site.lot`) |
| Client, LGA, zone, sustainability target | No | Yes (`proj.client`, `proj.lga`, `proj.zone`, `proj.sustainability`) |
| 24 RFP flags with no determinant | No | Yes (`proj.flag.*`, `flag_readings.yaml`) |
| Program milestones, regulated design | No | Still user-entered (a judgement or professional advice) |
| Objections likely, super-flat floor, contract administration (flags) | No | Still user-entered (no document states them) |
| Parts (Building A and so on) | User-defined | Same |

## How a detail is read

1. Code finds verbatim candidates in a passage (`internal/profile/details.go`):
   street addresses, "Lot n DP n", the name after "Principal:" or "Prepared
   for", "X Council" or "X local government area", a zone code near "zone",
   a Green Star or NABERS target. A flag has no candidates: a keyword gate
   (`flag_readings.yaml`) decides whether the yes/no question is asked.
2. The questions join the existing label fan-out (no extra round trip).
   Jev picks one candidate or none
   ([pre-parsed extraction](https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook),
   [fan-out](https://docs.typesafe.ai/patterns/fan-out)). Code stores the
   candidate's text, never Jev's.
3. Reconcile shows the reading with its source. It uses a new `detail`
   confidence shape with a provisional floor of 0.6, like the other shapes
   (unapproved; no keyed evaluation yet).
4. The site address and lot are stored on the site record. The typed value
   wins; until one is typed the reading is shown, and Reset on a typed value
   returns to the reading. The RFP reads the typed value first, then the
   reading.

## Re-reading documents already read

A finished label job is stamped with `profile.ReadRevision`
(`jobs.read_revision`, migration `021n`). The next Update queues finished
documents behind the current revision once. Questions are unchanged for most
passages, so they are answered from the call cache and cost no Jev call; only
passages where a new detail question applies are asked again. The question
version is not bumped, which keeps the cache valid.

## Not verified

No Jev key is available to the agent, so the live answers and their
confidences on the Petersham PPR are unmeasured. The 0.6 floor for the
`detail` shape is a provisional copy of the other shapes' floor. The first
live run on Petersham is the first evidence of accuracy; the draft answer key
(`data/eval/profile/answer-keys/petersham.yaml`, `details:`) says what it
should fill.
