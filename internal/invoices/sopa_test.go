package invoices

import (
	"strings"
	"testing"
)

func ip(n int) *int { return &n }

// A verified QLD row with its Christmas spans, and 2026-27 holidays loaded:
// the date skips both (done means 4).
func TestScheduleDueAcrossChristmasVerified(t *testing.T) {
	act := &SOPAct{State: "QLD", ScheduleSection: "s 76(1)", ScheduleDays: 15, PaymentSection: "s 73(1)", PaymentDays: 10, Verified: true,
		Spans: [][2]string{{"12-22", "12-24"}, {"12-27", "12-31"}, {"01-02", "01-10"}}}
	in := SOPInput{State: "QLD", Act: act, Served: "2026-12-14", ServedConfirmed: true,
		Holidays: []string{"2026-12-25", "2026-12-28", "2027-01-01", "2027-01-26"}, Years: map[int]bool{2026: true, 2027: true}}
	d := ScheduleDue(in)
	// 15 business days from Mon 14 Dec: 15-18 Dec (4), 21 Dec (5); 22-24 and
	// 25 Dec off; 28-31 off (spans and the holiday); 1 Jan holiday; 2-10 Jan
	// off; then 11 Jan onward: 11-15 (10), 18-22 (15).
	if d.Date != "2027-01-22" || d.ToCheck || d.Source != "act" || d.Section != "s 76(1)" {
		t.Fatalf("%+v", d)
	}
}

// Unverified NSW with no contract period: the Act's figure, to check, and
// no excluded span applied, so the date can only be early.
func TestUnverifiedActIsToCheck(t *testing.T) {
	act := &SOPAct{State: "NSW", ScheduleSection: "s 14(4)", ScheduleDays: 10, PaymentSection: "s 11(1A)(a)", PaymentDays: 15, Source: "government guidance", Spans: [][2]string{{"12-27", "12-31"}}}
	in := SOPInput{State: "NSW", Act: act, Served: "2026-12-21", ServedConfirmed: true, Years: map[int]bool{2026: true, 2027: true}}
	d := ScheduleDue(in)
	// 22,23,24,25,28,29,30,31 Dec, 1,4 Jan: no span or holiday loaded.
	if d.Date != "2027-01-04" || !d.ToCheck || !strings.Contains(d.Words, "No contract period entered") || !strings.Contains(strings.Join(d.Notes, " "), "not verified") {
		t.Fatalf("%+v", d)
	}
	// With the contract's shorter period the contract's date wins.
	in.ScheduleDays, in.ScheduleBasis = ip(5), "business"
	if d := ScheduleDue(in); d.Date != "2026-12-28" || d.Source != "contract" || !d.ToCheck {
		t.Fatalf("contract shorter: %+v", d)
	}
	// A longer contract period cannot extend the Act's.
	in.ScheduleDays = ip(20)
	if d := ScheduleDue(in); d.Date != "2027-01-04" || d.Source != "act" || !strings.Contains(strings.Join(d.Notes, " "), "cannot extend") {
		t.Fatalf("contract longer: %+v", d)
	}
	// Calendar days are calendar days.
	in.ScheduleDays, in.ScheduleBasis = ip(7), "calendar"
	if d := ScheduleDue(in); d.Date != "2026-12-28" || d.Source != "contract" {
		t.Fatalf("calendar days: %+v", d)
	}
	if p := PaymentDue(in); p.Section != "s 11(1A)(a)" || !p.ToCheck {
		t.Fatalf("payment: %+v", p)
	}
}

func TestWeekendServiceAndMissingHolidays(t *testing.T) {
	act := &SOPAct{State: "NSW", ScheduleSection: "s 14(4)", ScheduleDays: 10, Verified: true}
	// Served Saturday 7 Nov: counting starts Monday 9th; ten business days is Friday 20th.
	in := SOPInput{State: "NSW", Act: act, Served: "2026-11-07", ServedConfirmed: true, Years: map[int]bool{2026: true}}
	if d := ScheduleDue(in); d.Date != "2026-11-20" || d.ToCheck {
		t.Fatalf("weekend service: %+v", d)
	}
	in.Years = nil
	if d := ScheduleDue(in); !d.ToCheck || !strings.Contains(strings.Join(d.Notes, " "), "serve by") {
		t.Fatalf("missing year: %+v", d)
	}
	in.Years, in.ServedConfirmed = map[int]bool{2026: true}, false
	if d := ScheduleDue(in); !d.ToCheck || !strings.Contains(strings.Join(d.Notes, " "), "dropped") {
		t.Fatalf("drop day: %+v", d)
	}
}

func TestUnknownDueDates(t *testing.T) {
	if d := ScheduleDue(SOPInput{Served: "2026-11-02"}); d.Date != "" || !strings.Contains(d.Words, "state") {
		t.Fatalf("no state: %+v", d)
	}
	if d := ScheduleDue(SOPInput{State: "WA", Served: "2026-11-02"}); d.Date != "" || !strings.Contains(d.Words, "no payment schedule period is recorded for WA") {
		t.Fatalf("no Act row, no contract: %+v", d)
	}
	d := ScheduleDue(SOPInput{State: "WA", Served: "2026-11-02", ServedConfirmed: true, ScheduleDays: ip(10), ScheduleBasis: "business", Years: map[int]bool{2026: true}})
	if d.Date != "2026-11-16" || !d.ToCheck || !strings.Contains(d.Words, "may set an earlier date") {
		t.Fatalf("contract only: %+v", d)
	}
}

func TestScheduleAlert(t *testing.T) {
	act := &SOPAct{State: "NSW", ScheduleSection: "s 14(4)", ScheduleDays: 10, Verified: true}
	in := SOPInput{State: "NSW", Act: act, Served: "2026-11-02", ServedConfirmed: true, Years: map[int]bool{2026: true}}
	due := ScheduleDue(in) // 16 November
	for today, want := range map[string]string{"2026-11-05": "", "2026-11-12": AlertSoon, "2026-11-16": AlertSoon, "2026-11-17": AlertOverdue} {
		if got := ScheduleAlert(in, due, false, today); got != want {
			t.Errorf("%s: %q, want %q", today, got, want)
		}
	}
	if ScheduleAlert(in, due, true, "2026-11-30") != "" {
		t.Error("a served schedule raises nothing")
	}
	if ScheduleAlert(in, DueDate{}, false, "2026-11-05") != AlertCheck {
		t.Error("an unknown date is to check")
	}
}

// A schedule's date passing with none served makes the claimed amount the
// amount counted only under the Act, and only on a row the owner has
// verified (AGENTS.md; plan 2026-11-26-payment-schedules, decision 12). A
// date from the contract's own period, or from an unverified row, warns.
func TestScheduleLapsedNeedsAVerifiedRowsOwnDate(t *testing.T) {
	verified := &SOPAct{State: "NSW", ScheduleSection: "s 14(4)", ScheduleDays: 10, Verified: true}
	in := SOPInput{State: "NSW", Act: verified, Served: "2026-11-02", ServedConfirmed: true, Years: map[int]bool{2026: true}}
	// The Act's date is 16 November.
	if ScheduleLapsed(in, false, "2026-11-16") {
		t.Error("the Act's date itself is still in time")
	}
	if !ScheduleLapsed(in, false, "2026-11-17") {
		t.Error("past a verified row's date with none served, the claim has lapsed")
	}
	if ScheduleLapsed(in, true, "2026-11-30") {
		t.Error("a served schedule never lapses")
	}
	unverified := *verified
	unverified.Verified = false
	in.Act = &unverified
	if ScheduleLapsed(in, false, "2026-11-30") {
		t.Error("an unverified row warns only")
	}
	// The contract asks for the schedule in five business days. Past that
	// day the screen warns, but the Act's own date has not passed.
	in.Act, in.ScheduleDays, in.ScheduleBasis = verified, ip(5), "business"
	due := ScheduleDue(in)
	if due.Date != "2026-11-09" || ScheduleAlert(in, due, false, "2026-11-10") != AlertOverdue {
		t.Fatalf("the contract's earlier date: %+v", due)
	}
	if ScheduleLapsed(in, false, "2026-11-10") {
		t.Error("past the contract's date but not the Act's, nothing has lapsed")
	}
	if ScheduleLapsed(SOPInput{State: "WA", Served: "2026-11-02", ScheduleDays: ip(10), ScheduleBasis: "business"}, false, "2026-12-30") {
		t.Error("with no Act row, the contract's period alone warns only")
	}
}
