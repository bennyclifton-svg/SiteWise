# Inline project name

The project header uses a native text input with the existing title typography.
It stays mounted so a click places the caret naturally; hover and focus add only
an underline. Enter or leaving the field saves, Escape discards the draft, and
composition input is not interrupted by Enter. Blank names are rejected and
surrounding spaces are trimmed. A failed save keeps the draft with retry guidance.

Lane A: a persisted project identity edit that reduces navigation and time to
decision. Only the project name changes. No Jev call or new dependency.
PATCH /projects/{id} checks origin, session membership and org ownership, then
updates the name in one SQL statement. The header and project switcher update
after success. Missing and other-organisation projects return 404.

The path uses the existing profile_edit budget: p50 50 ms / p90 150 ms.
The Playwright project-name spec gates 25 real HTTP writes against that budget,
and checks caret placement, persistence, cancellation, blank input, save failure
and retry, origin rejection and organisation isolation. Local measurements on
28 October 2026: p50 1.8 ms / p90 2.3 ms, including the browser round trip.
Desktop and mobile screenshots were inspected. Production network latency is
not measured by this local check.
