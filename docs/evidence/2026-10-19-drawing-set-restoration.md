# Missing drawing sheets after re-upload

Reproduced on Hale: the retained architectural set reported a completed
15-page expansion, but only the link for Sheet 15 remained. Deletion events
confirmed removed children. The list API reproduced the discrepancy, ruling
out a browser-only filter issue. Re-upload deduplication returned the existing
split source without scheduling expansion.

Lane A defect fix. Expansion views now compare completion against actual child
links and expose an incomplete set for review. Deleting children refreshes the
surviving source through the existing sheets event. An explicit re-upload or
retry schedules recovery; restarting alone does not undo deliberate deletions.
Recovery skips existing pages and publishes missing pages atomically, retaining
surviving document IDs, decisions and corrections. Restored PDFs automatically
receive the existing background visual-review job. No new dependency or model
call on the foreground filing path. Existing filing budget remains 1000/2000 ms;
list and correction budgets remain 50/150 ms.

The store regression failed before the fix (complete after child deletion),
then passed. HTTP regression covers actual two-page upload, split, edit,
deletion and re-upload, with no duplicate source or surviving-edit replacement.
Drawing/schedule admission, report retention, atomic publication and tenant
isolation tests pass. Existing list/correction speed gates pass; deletion
p50 4.67 ms, p90 6.78 ms against 500/1000 ms budgets.

Rebuilt and restarted the local app. Retried the real Hale original through
the authenticated API: all 15 pages now appear as filed children. Compared
Sheet 15's fields before and after recovery: unchanged. Title review remains
explicit; sheet restoration does not automatically accept visual suggestions.

Prevention: test the lifecycle across deletion and duplicate re-upload, rather
than testing initial splitting and duplicate uploads only. A stored completion
flag cannot stand in for the current child count.
