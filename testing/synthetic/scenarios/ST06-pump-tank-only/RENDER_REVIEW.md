# ST06-pump-tank-only: render review

Reviewer: independent read, brief `04r-render-review.md`, 7 November 2026.
Variant of ST06-base (corpus final and reviewed): the level 3 fit-out is removed.

## Verdict: PASS

Four findings, all advisory (realism). None changes a key, a figure or a trigger hit, and none needs to be
fixed before the run.

## 1. Lint

`python testing/synthetic/tools/lint_corpus.py testing/synthetic/scenarios/ST06-pump-tank-only` gives
`ok`, exit 0.

## 2. Copies and rewrites against the base

- Byte-identical to base (`cmp`): d02, d03, d04, d06, d13, d16, d17, d19. That matches `changed_from_base`,
  which lists none of them.
- Rewritten: d05, d07, d08, d09, d11, d12, d14, d18 and d20 (same filenames), and d01 (new filename without
  "and level 3 fit-out"). I diffed each against the base. Every change is one that `changed_from_base`
  names, and nothing else moved.
- Removed: d10 (Marlowe proposal) and d15 (level 3 concept report). Neither file is in the corpus.
- Leftovers: I searched the corpus for `level 3`, `fit-out`, `Marlowe`, `interior`, `incoming`,
  `new tenant`, `heads of terms`, `landlord`, `vacant`, `second`, `both`, `two packages`/`contracts`/
  `certificates`, `each contract`, `2,700,000` and `1,500,000`. The only hits are wanted ones:
  - the trigger's evidence in d01 line 22 and d12 lines 91-92;
  - Plenum's office address, "Level 3, 40 Herbert Street" (d08 line 3, which `fees.json` also keys);
  - "both pumpsets" (d16, d17) and "both sized by the engineer" (d01).
- Budget: d01 states one figure, "Approved budget: $1,200,000 excluding GST (fire services upgrade)". No
  other document states a budget. That matches `profile.json` `proj.budget_stated` = 120000000 and
  `rfps.json` `budget_stated_cents` = 120000000. No file in `expected/` still carries 270000000, a fit-out
  line or d10/d15.

## 3. What a PM concludes from the documents (written before opening `expected/`)

- **Building:** a 1988 6-storey office building over one basement car park level. Class 5 with Class 7a
  (d03, d09). Rise in storeys 6. Sprinklered throughout to AS 2118.1. Deemed-to-satisfy, no performance
  solutions.
- **Work:** alteration/refurbishment of base-building fire services. Electric duty, diesel standby and
  jockey pumps, a 150 kL modular tank in the store, the pump room enlarged into the store, and the backflow
  device upgraded. Hydrants unchanged. No tenancy works, and every floor stays tenanted (a live
  environment).
- **Pathway:** one CDC under the Fire Safety Code. No panel and no cl 4.6.
- **Contract:** traditional, fully documented, AS 4000 lump sum, one stage. Single owner. Mid quality.
  Urban constrained access (no on-street loading). Standard environment.
- **Site:** North Sydney Council, E2, Lot 12 DP 1093476. Not heritage, not flood, not bushfire, no
  contamination record, no rail, no classified road. Reticulated water from the 200 mm main. Nothing is
  excavated, and there are no transfers, no OSD and no heavy vehicles.
- **Consultants:** fire/hydraulic, structural, electrical, mechanical, hazmat and certifier. No interior
  designer.
- **Fees (ex GST):** Calder $90,000; Ferrule $24,000; Kilnworth $18,700 inc GST, which is $17,000 ex and
  $190/h ex; Plenum $12,000; Lintel $13,000; Sentinel $3,300 (laboratory analysis included).
- **Invoices:**
  - CFH-INV-0412: $18,500 + $1,850 = $20,350.
  - FCE-2611: $7,500 + $375 = $7,875. The GST is 5%, which is wrong; that is the base's `gst_rate` trap.
  - SHS-INV-5530: $2,600 + $260 = $2,860.

## 4. Comparison with `expected/`

- `profile.json`: every value matches the reading above. `hdr.cond.operational_constraints` accepts
  `live_environment`, which is what the documents now say.
- `fees.json`: the totals add up, the stage amounts match the tables, the Kilnworth ex-GST normalisation is
  right, and the Lintel exclusion and the Sentinel "included" line match.
- `invoices.json`: matches all three invoices, including the `gst_rate` flag on d19.
- `document_register.json`: d01's title and number match its title block. The other rewritten clean
  documents (d07, d09, d11, d14) keep the base titles, numbers and dates, which match the register.

No differences found.

## 5. Stray triggers

I searched for every `shape.yaml` `triggers[].evidence` phrase (car spaces, DA, easement, performance
solution, road opening, hydrant upgrade, asbestos, audible or above-roof exhaust, Ausgrid, agreement for
lease, fire safety order, toilets, heritage, strata, roof or crane, affected part, fire-engineered,
interim occupation) and every DOMAIN.md §10 trap (basement, water main, fire storage, structural…fire,
flood, contaminat, `stages? [1-9a-c]`, stair, kitchen, Sydney Water asset, lodge).

- Of the trigger evidence, only `pump-tank-only` (this scenario's trigger) is present, in d01 and d12.
- Every §10 hit is a base trap already planned in the manifest, in a byte-identical copy or in an
  unchanged sentence of d05 or d12:
  - Stage 1-8 in d05;
  - Stage 1 cut-over and flooding in d12;
  - the stairs in d03;
  - water main and Sydney Water asset in d04 and d12;
  - "structural engineer will confirm the slab for the fire water storage tank" in d06.
- The negations ("no car spaces", "no crane", "not audible", "no performance solution") are base text and
  still hold.

No unplanned hits.

## 6. Title blocks

- d01 (adversarial): the label lines, `Ref: CPH-CS41-B01`, Client and Site come before the first `##`, and
  the `#` heading is identical to the Title.
- The clean documents (d07, d09, d11, d14, d18, d20) each have label lines before the first `##` and one `#`
  heading that matches `Title:`. Their values match the register.
- d08 (messy) keeps its letterhead and Markdown-table title block, as its tier says.
- d05 and d12 (adversarial) are unchanged in form.

## 7. Findings (advisory)

1. **d12 `Fire services options report CFH-R-26-019-01.md`, lines 91-92**, "Level 3 remains leased to the
   existing tenant and is not affected."
   - *What is wrong:* nothing else in an engineer's options report mentions level 3. Singling out one
     floor with no context reads as written for a test.
   - *Change (optional):* keep the sentence word for word, since it is trigger evidence, and give it a
     reason in the line before it. For example, replace "The scope is limited to the fire pump and tank
     upgrade; no tenancy works are included." with "The scope is limited to the fire pump and tank
     upgrade; no tenancy works are included. Level 3 was re-let in January 2026 on its existing fit-out."
     The evidence sentence then follows naturally.
   - Not required for scoring.
2. **d01 `Owner brief - 41 Callowhill Street fire services upgrade.md`, lines 22-23**, "Level 3 remains
   leased to the existing tenant and is not affected by these works."
   - *What is wrong:* the same oddity as finding 1, straight after "Every floor is leased and occupied."
   - *Change (optional):* add a reason after "Every floor is leased and occupied.", such as "Level 3 was
     re-let in January 2026 on its existing fit-out." Leave the evidence sentence unchanged. Keep the
     same date in d12 if finding 1 is taken up.
3. **d01 line 37**, "We are refurbishing the base-building fire services by upgrading the sprinkler water
   supply of the existing building."
   - *What is wrong:* slightly stilted for an owner. It is deliberate: notes.md says it anchors `refurb`.
   - *Change:* none. It is recorded only so a later reviewer does not "fix" it and lose the anchor.
4. **d09 `Fee proposal - Lintel Building Certifiers - 41 Callowhill Street.md`, line 22**, "On that
   arrangement the certifier confirmed the tank and pump house meet the Fire Safety Code development
   standards."
   - *What is wrong:* the certifier refers to itself in the third person.
   - *Change (optional):* "On that arrangement we confirmed the tank and pump house meet the Fire Safety
     Code development standards."
   - This text is unchanged from the reviewed base and carries no key, so it is outside this variant's
     rewrite. Apply it to the base too if it is taken up.

## 8. Realism

The rewritten fee proposals read like real proposals cut back to a pump-room job. Each one states its
narrower scope as a plain negation:

- d07: "no other electrical work in the building is included";
- d08: "the air conditioning serving the occupied floors is not touched";
- d05: "existing … layouts on the floors are not altered".

The reduced fees fit the work: about $159,000 ex GST of consultant fees against a $1.2 million
construction budget is high, but believable for a small, live-building fire services job. The d09
sentence "Whether a fire safety schedule and a final fire safety certificate are needed for this alteration
will be confirmed at assessment" is a sensible certifier's hedge, and it matches the trigger's authority
note in `shape.yaml`. Apart from findings 1 and 2, no document reads as written for a test.
