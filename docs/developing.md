# Developing SiteWise

Date: 3 October 2026. Distilled from a conversation with Guillermo Rauch
(CEO of Vercel) and applied to this repo. The binding rules for an agent
are the short section in `AGENTS.md`. This note is the why, and the prompts
to paste when starting work.

## Direction

Anyone can now ship a lot of software. The advantage is velocity (speed
aimed at a few problems), taste, and proof. Speed without a direction
produces work a user can tell was one-shotted.

The human job moved up one layer. Typing the implementation is the manual
labour. What stays with the owner is creative control, risk control, and
deciding what gets built. People still buy from humans: the story, the
taste, and a product that feels considered.

Speed and velocity are different. Speed is how many changes land. Velocity
is whether those changes move the product toward the thing only this
product should be great at. A clear "this is broken" signal should become
a fix quickly. Taste calls (a new workflow, a confused screen) stay with
the owner until they are a decision an agent can implement.

Own the few problems with a real advantage. Buy or reuse the rest.
Infrastructure the world already scrutinises is safer than a private
rewrite. Pour effort into the product judgment: what "good" means, what
must never regress, and the story.

The model will not pick the tradeoffs. It will spend a long time on a tiny
detail, or write a test that only checks a constant. Set the axioms: what
must be fast, what must be correct, what may be rough. Spend attention on
the edges of the system (performance budgets, security, the behaviour a
user actually hits). After a change, understand what just happened: what
calls what, what got slower, whether a public behaviour changed. That is
enough. Reading every line is not the job, and shipping blind is not either.

Slowness gets replaced. Tools should start fast, the loop should feel
real-time while a person is in it, and background work is for tasks
already decided. "I think it's fast" fails once users are on another
machine or a bad network. Production truth (error rate, slow sessions,
whether anyone uses the page) belongs in the loop.

Security is the debt that will sort products. A short happy-path program
hides a large space of corner cases. Get ahead of that on the critical
path, prefer code that is safe by construction there, and be able to say
how it was checked.

Two stretches of that conversation were sponsor reads. The build-versus-buy
point stands. Switching this repo onto a sponsor's stack does not.

## What this means here

SiteWise already matches the part worth keeping. Speed is the first
objective, with budgets that fail the build. Jev judges; Go does parsing,
lookups, and control flow. The advantage is the building: systems,
determinants, NCC rules, interfaces, failure modes. A faster generic
document app is a different product.

Hold this:

- Go deep on intake, filing, and the building model. A feature earns its
  place when it makes a project faster to understand or a rule faster to
  trust. A feature that does not move filing speed, answer accuracy, or
  time-to-decision is a side quest.
- Reuse anything that is not that. Magic-link auth, Postgres, Caddy, and
  the Jev client are infrastructure. Do not grow a second database, a chat
  agent, a vector store, or a home-grown payments or email system.
- Keep two speeds of work. Clear defects (wrong chip, slow upload, a
  cross-org leak, a broken budget) get fixed as themselves. Product
  judgment stays with the owner until it is a decision.
- Hold the bar of a consultant who already knows the NCC: faster than
  their current habit, and trustworthy on the numbers. Model output often
  looks impressive only next to nothing.

The owner sets the budget, the correctness bar, and the "do not touch
this" list. These magnitudes are enough to push back:

| Thing | Roughly |
|---|---|
| A click should feel instant | under 100 ms |
| Filing budget already set | typical under 1 s, slow under 2 s |
| One Jev call | a few hundred ms; never a chain of them on a click |
| A round trip across the world | about 150–200 ms |
| Reading a file on this machine's disk | milliseconds |

If a page takes a second and the user is not waiting on Jev, something
local is wrong. The prompt should say "make this path faster" and name
the budget.

## Three lanes

Say which lane a change is in before any code.

**Lane A, the product core.** Filing, org isolation, rule numbers,
anything a user trusts. An existing test or answer key must still pass,
and the speed budget must still pass. Unverified NCC numbers stay
unverified. No new library unless the alternative is genuinely hard, and
the reason is written down.

**Lane B, the edges.** A label, an empty state, a layout. Move fast. Name
the speed budget and "do not break filing."

**Lane C, security and other people's code.** Login, file access, anything
downloaded into the build. Look for missing-file, wrong-org, and
unexpected-input cases. Prefer a small dependency that many people already
depend on over a new one this repo would be the only watcher of.

After a burst of fast shipping, write the decisions down in `docs/design/`.
Five days of features with no note of what must stay true is the moment to
stop and name the axioms.

## Prompt run sheets

Paste one of these and fill the brackets. Coding starts once the brackets
are real.

### New feature

```text
Feature: [one sentence, the user outcome]
Why this, not a side quest: [how it helps filing speed, answer trust, or time-to-decision on a real project]
Lane: [A core / B edge / C security]

Out of scope:
- Do not add a new service, database, AI provider, or dependency unless you stop and say why the current stack cannot do it.
- Jev is the only model. Code parses, computes, looks up, and controls flow. One Jev fan-out for this state. No serial Jev calls on a click.
- Do not invent rule numbers. Unverified rules stay verified: false.

Speed (this is a gate, not a wish):
- This path must stay within [p50 / p90, e.g. filing 1s / 2s].
- Name the budget for the new work in milliseconds.
- If you cannot meet it, stop and say what you would have to trade off. Do not silently add a round trip, a retry, or a heavier library.

Correctness:
- Done means: [the click, the file, the answer key, or the empty/error case I will see]
- Must still pass: [existing speed benchmark, org-isolation test, Hale/Petersham/Newham field if relevant]
- Edge cases to handle: [missing file, duplicate hash, Jev timeout, wrong org, scanned PDF with no text]
- Tests should prove those edges. Do not add tests that only restate a constant.

Security:
- Every row stays scoped to org_id.
- [Anything else: auth, file read, user-supplied names]

After you implement:
1. Run the speed check and report the numbers against the budget.
2. In plain language: what changed, what calls what, what could fail in production.
3. List any dependency or behaviour you added that I would have to trust.
4. Do not widen the feature. If you see a follow-up, name it and leave it.
```

### Obvious defect

```text
This is a defect, not a redesign. [what the user saw, with the screenshot or the slow number]
Fix only that. Keep the filing budget (p50 ≤ 1s, p90 ≤ 2s) and org isolation.
Show me the before/after timing and the case that was broken.
```

### Choose direction before any code

```text
Do not write code. I am choosing direction.
Here is the user signal: [what they did, how often, how slow, how wrong]
Say whether this is an objective defect or a taste decision.
Give me two options that respect the speed budget, and what each would make worse.
Recommend one. I will turn the winner into a build prompt.
```

The last prompt protects velocity. The scarce step is deciding what
"better" means, then keeping the next hour of work pointed there.
