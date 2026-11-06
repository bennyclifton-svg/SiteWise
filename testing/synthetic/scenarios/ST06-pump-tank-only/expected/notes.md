# ST06-pump-tank-only notes

The base without the level 3 fit-out: level 3 remains leased to the existing tenant and is not affected, every floor
stays tenanted, one CDC under the Fire Safety Code and one AS 4000 contract. 18 documents in 4 waves. Only
`proj.budget_stated` differs from the base ($2,700,000 to $1,200,000, the fire services figure), which the trigger
now lists.

## What changed

- d01 says "refurbishing the base-building fire services" to anchor `refurb`, and carries the trigger's evidence in place of "the level 3 fit-out is delivered with the fire services upgrade";
  its fit-out funding, landlord's-works and interim-occupation sentences, the second contract and the interior
  designer go.
- Removed: d10 (interior designer's proposal) and d15 (level 3 concept report).
- d05, d07, d08, d09 and d11 drop their level 3 scope and fees; d12 and d14 drop level 3; d20 (hazmat invoice)
  follows the smaller survey fee. d05 stages 1 and 2 are unchanged, so d18 stands.

## Profile

- `proj.budget_stated` is $1,200,000: removing the fit-out removes its $1,500,000. d01 restates the budget.
- `hdr.cond.operational_constraints` is now `live_environment` in fact (every floor tenanted); the trigger now lists
  it, and the key accepts it and partial_occupation, as in the base.

## Expected noise (shape `why`)

The fit-out concept, design development, documentation and tender rows and the three strip-out rows still appear in
the app's program; the shape lists them as removed, so they score as extras.

## Traps

Base traps carry over. The fit-out-amenities and affected-part negations (d10, d15) leave with those documents; both
triggers need a fit-out, so nothing is lost.
