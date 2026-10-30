# Brief 02: Independent review of one domain map

To dispatch, replace `{TYPE_ID}` and `{SLUG}`.

---

You are an independent reviewer: a senior Australian PM who has delivered
this kind of job, and who also reads YAML and Go. Another agent wrote the
domain map in `testing/synthetic/types/{TYPE_ID}-{SLUG}/` from
`testing/synthetic/briefs/01-domain-map.md`. That map will become an answer
key, so your job is to find what is wrong or missing. Don't polish it.
Repository: `D:\AI Projects\sitewise`.

Do not edit `DOMAIN.md` or `shape.yaml`. Write
`testing/synthetic/types/{TYPE_ID}-{SLUG}/REVIEW.md`.

## Check, in this order

1. **The validator.** Run
   `python testing/synthetic/tools/validate_shape.py testing/synthetic/types/{TYPE_ID}-{SLUG}/shape.yaml`.
   Record the result.
2. **Built-in claims.** Check every claim marked `yes` or `partial` against
   the cited file *and* the selection logic in
   `testing/synthetic/reference/KNOWLEDGE_INDEX.md`. For example, a package
   default only fires for its work types and classes. Also spot-check at
   least five claims marked `no`: confirm the knowledge really has nothing.
3. **Completeness.** Does the map cover the whole job?
   - Every top-level system is in or out, with a reason.
   - Approvals, authority connections, hold points and certification.
   - Existing-building interfaces, where they apply.
   - The common triggers. Name any you would expect that are missing.
4. **Domain accuracy.** First read
   `testing/synthetic/reference/OWNER_RULINGS.md`. The owner's rulings
   override your own view; flag any place the map breaks one. Check
   `reference/CONVENTIONS.md` too. Then flag anything:
   - a consultant the job does not need, or a needed one that is missing;
   - logic in the wrong order;
   - an NSW pathway or authority that is wrong;
   - a rule number without `verified: false` that you cannot confirm;
   - any price, rate or duration that the repo does not hold.
5. **Testability.** Is each trigger's evidence written the way real
   documents phrase it, including indirect mentions? Is each negation
   realistic? Would each consequence be visible in the app's procurement,
   cost plan, program or RFP output?

## Write REVIEW.md

- Verdict: **PASS** (usable as is), **FIX** (specific changes listed) or
  **REDO** (wrong at the root).
- A numbered list of findings. Each one gives: the file and field, what is
  wrong, the evidence, and the exact change.
- The list of built-in claims you checked, with each one ✔ or ✘.

## Reply

Reply to the orchestrator in **100 words or fewer**: the verdict, the
number of findings, and the two most serious ones.
