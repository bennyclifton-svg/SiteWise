package reports

// KindPaymentSchedule is a progress claim's payment schedule
// (docs/plans/2026-11-17-variations-progress-claims.md, Task 9): one report
// per claim, issued and frozen, never a package's RFP or RFT. What it prints
// is in payment_schedule_sheet.go. Schedules issued before 27 November 2026
// keep the snapshot and PDF of the first layout they were issued with.
const KindPaymentSchedule = "payment_schedule"
